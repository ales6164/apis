package apis

import (
	"cloud.google.com/go/datastore"
	"encoding/json"
	"github.com/ales6164/apis/errors"
	"github.com/ales6164/apis/kind"
	"github.com/dgrijalva/jwt-go"
	gcontext "github.com/gorilla/context"
	"github.com/gorilla/mux"
	"golang.org/x/net/context"
	"google.golang.org/appengine"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

type Context struct {
	R                 *Route
	r                 *http.Request
	hasReadAuthHeader bool
	IsAuthenticated   bool
	Client            *datastore.Client
	context.Context
	CountryCode string // 2 character string; gb, si, ... -- ISO 3166-1 alpha-2
	UserEmail   string
	UserKey     *datastore.Key
	Role        Role
	*body
}

type body struct {
	hasReadBody bool
	body        []byte
}

func (R *Route) NewContext(r *http.Request) Context {
	cc := r.Header.Get("X-Custom-Country")
	if len(cc) == 0 {
		cc = r.Header.Get("X-AppEngine-Country")
	}
	ctx := appengine.NewContext(r)
	c, _ := datastore.NewClient(ctx, R.ProjectID)
	return Context{
		R:           R,
		r:           r,
		Client:      c,
		CountryCode: cc,
		Role:        PublicRole,
		Context:     ctx,
		body:        &body{hasReadBody: false},
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

func (ctx Context) Language() string {
	if _, ok := ctx.R.a.allowedTranslations[ctx.CountryCode]; ok {
		return ctx.CountryCode
	}
	return ctx.R.a.options.DefaultLanguage
}

/*func (ctx Context) SetGroup(group string) (Context, error) {
	var err error
	ctx.Context, err = appengine.Namespace(ctx, group)
	return ctx, err
}*/

// AdminRole has all permissions
func (ctx Context) HasPermission(k *kind.Kind, scope Scope) bool {
	if ctx.Role == AdminRole {
		return true
	}
	if val1, ok := ctx.R.a.permissions[ctx.Role]; ok {
		if val2, ok := val1[scope]; ok {
			if val3, ok := val2[k]; ok && val3 {
				if ctx.R.roles != nil {
					if _, ok := ctx.R.roles[ctx.Role]; ok {
						return true
					}
				} else {
					return true
				}
			}
		}
	}
	return false
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
		ctx.UserKey = datastore.NameKey("_user", userEmail, nil)
	} else {
		ctx.UserEmail = ""
		ctx.Role = PublicRole
		ctx.UserKey = nil
	}

	return ctx.IsAuthenticated, ctx
}

func NewToken(user *User) *jwt.Token {
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
	Token   *Token                 `json:"token"`
	User    *User                  `json:"user"`
	Profile map[string]interface{} `json:"profile"`
}

func (ctx *Context) Print(w http.ResponseWriter, result interface{}) {
	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(result)
}

func (ctx *Context) PrintResult(w http.ResponseWriter, result map[string]interface{}) {
	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(result)
}

func (ctx *Context) PrintAuth(w http.ResponseWriter, token *Token, user *User) {
	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(AuthResult{
		User:  user,
		Token: token,
	})
}

func (ctx *Context) PrintError(w http.ResponseWriter, err error) {
	log.Printf("context error: %v", err)
	if err == errors.ErrUnathorized {
		w.WriteHeader(http.StatusUnauthorized)
	} else if err == errors.ErrForbidden {
		w.WriteHeader(http.StatusForbidden)
	} else if _, ok := err.(*errors.Error); ok {
		w.WriteHeader(http.StatusBadRequest)
	} else {
		w.WriteHeader(http.StatusInternalServerError)
	}
	w.Write([]byte(err.Error()))
}
