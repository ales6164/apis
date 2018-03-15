package apis

import (
	"golang.org/x/net/context"
	gcontext "github.com/gorilla/context"
	"google.golang.org/appengine"
	"io/ioutil"
	"net/http"
	"github.com/dgrijalva/jwt-go"
	"time"
	"encoding/json"
	"github.com/gorilla/mux"
	"google.golang.org/appengine/datastore"
	"github.com/ales6164/apis/kind"
	"google.golang.org/appengine/log"
)

type Context struct {
	a                 *Apis
	r                 *http.Request
	hasReadAuthHeader bool
	IsAuthenticated   bool
	context.Context
	UserEmail         string
	UserKey           *datastore.Key
	Role              Role
	*body
}

type body struct {
	hasReadBody bool
	body        []byte
}

func (a *Apis) NewContext(r *http.Request) Context {
	return Context{
		a:       a,
		r:       r,
		Role:    PublicRole,
		Context: appengine.NewContext(r),
		body:    &body{hasReadBody: false},
	}
}

func (ctx Context) Body() []byte {
	if !ctx.body.hasReadBody {
		ctx.body.body, _ = ioutil.ReadAll(ctx.r.Body)
		ctx.r.Body.Close()
		ctx.body.hasReadBody = true
	}
	return ctx.body.body
}

func (ctx Context) Id() string {
	return mux.Vars(ctx.r)["id"]
}

func (ctx Context) SetGroup(group string) (Context, error) {
	var err error
	ctx.Context, err = appengine.Namespace(ctx, group)
	return ctx, err
}

func (ctx Context) HasPermission(k *kind.Kind, scope ...Scope) (Context, error) {
	if val1, ok := ctx.a.permissions[ctx.Role]; ok {
		if val2, ok := val1[k]; ok {
			for _, s := range scope {
				if val3, ok := val2[s]; ok && val3 {
					return ctx, nil
				}
			}
		}
	}
	return ctx, ErrForbidden
}

// Authenticates user
func (ctx Context) Authenticate() (bool, Context) {
	ctx.hasReadAuthHeader = true
	var isAuthenticated, isExpired bool
	var userEmail, role string

	tkn := gcontext.Get(ctx.r, "auth")
	if tkn != nil {
		token := tkn.(*jwt.Token)
		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			if err := claims.Valid(); err == nil {
				if userEmail, ok = claims["sub"].(string); ok && len(userEmail) > 0 {
					isAuthenticated = true
				}
				if role, ok = claims["rol"].(string); !ok || len(role) == 0 {
					isAuthenticated = false
				}
			}
		}
	}

	ctx.IsAuthenticated = isAuthenticated && !isExpired
	if ctx.IsAuthenticated {
		ctx.UserEmail = userEmail
		ctx.Role = Role(role)
		ctx.UserKey = datastore.NewKey(ctx, "_user", userEmail, 0, nil)
	} else {
		ctx.UserEmail = ""
		ctx.Role = ""
		ctx.UserKey = nil
	}

	return ctx.IsAuthenticated, ctx
}

func NewToken(user *user) *jwt.Token {
	var exp = time.Now().Add(time.Hour * 72).Unix()
	return jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"aud": "api",
		"nbf": time.Now().Add(-time.Minute).Unix(),
		"exp": exp,
		"iat": time.Now().Unix(),
		"iss": "sdk",
		"sub": user.Email,
		"rol": user.Role,
	})
}

/**
RESPONSE
 */

type Token struct {
	Id        string `json:"id"`
	ExpiresAt int64  `json:"expiresAt"`
}

type AuthResult struct {
	Token *Token `json:"token"`
	User  *User  `json:"user"`
}

func (ctx *Context) PrintResult(w http.ResponseWriter, result interface{}) {
	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(result)
}

func (ctx *Context) PrintAuth(w http.ResponseWriter, user *user, token *Token) {
	w.Header().Set("Content-Type", "application/json")

	var out = AuthResult{
		User:  &User{user.Email, user.Role},
		Token: token,
	}

	json.NewEncoder(w).Encode(out)
}

func (ctx *Context) PrintError(w http.ResponseWriter, err error) {
	log.Errorf(ctx, "context error: %v", err)
	if err == ErrUnathorized {
		w.WriteHeader(http.StatusUnauthorized)
	} else if err == ErrForbidden {
		w.WriteHeader(http.StatusForbidden)
	} else if _, ok := err.(*Error); ok {
		w.WriteHeader(http.StatusBadRequest)
	} else {
		w.WriteHeader(http.StatusInternalServerError)
	}
	w.Write([]byte(err.Error()))
}
