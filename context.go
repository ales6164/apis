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
	"github.com/ales6164/apis/errors"
)

type Context struct {
	R                 *Route
	r                 *http.Request
	hasReadAuthHeader bool
	IsAnonymous       bool
	IsAuthenticated   bool
	context.Context
	CountryCode       string // 2 character string; gb, si, ... -- ISO 3166-1 alpha-2
	UserEmail         string
	UserKey           *datastore.Key
	Role              Role
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
	return Context{
		R:           R,
		r:           r,
		CountryCode: cc,
		Role:        PublicRole,
		Context:     appengine.NewContext(r),
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

func (ctx Context) SetGroup(group string) (Context, error) {
	var err error
	ctx.Context, err = appengine.Namespace(ctx, group)
	return ctx, err
}

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
	var isAuthenticated, isExpired, isAnonymous bool
	var userEncodedKey, userEmail, role string

	tkn := gcontext.Get(ctx.r, "auth")
	if tkn != nil {
		token := tkn.(*jwt.Token)
		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			if err := claims.Valid(); err == nil {
				if userEmail, ok = claims["sub"].(string); ok && len(userEmail) > 0 {
					isAuthenticated = true
					if userEncodedKey, ok = claims["uid"].(string); ok && len(userEncodedKey) > 0 {
						if ctx.UserKey, err = datastore.DecodeKey(userEncodedKey); err == nil {
							isAuthenticated = true
						} else {
							isAuthenticated = false
						}
					} else {
						isAuthenticated = false
					}
				}
				if role, ok = claims["rol"].(string); !ok || len(role) == 0 {
					isAuthenticated = false
				}
				if isAnonymous, ok = claims["ann"].(bool); ok && isAnonymous {
					isAuthenticated = false
				}
			}
		}
	}

	ctx.IsAuthenticated = isAuthenticated && !isExpired
	ctx.UserEmail = userEmail
	if ctx.IsAuthenticated {
		ctx.Role = Role(role)
	} else {
		ctx.Role = PublicRole
		ctx.IsAnonymous = isAnonymous
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
		"uid": user.Id.Encode(),
		"sub": user.Email,
		"ann": user.IsAnonymous,
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

func (ctx *Context) PrintBytes(w http.ResponseWriter, result []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.Write(result)
}

func (ctx *Context) PrintResult(w http.ResponseWriter, result map[string]interface{}) {
	w.Header().Set("Content-Type", "application/json")

	bs, err := json.Marshal(result)
	if err != nil {
		ctx.PrintError(w, err)
		return
	}
	w.Write(bs)
}

func (ctx *Context) PrintAuth(w http.ResponseWriter, token *Token, user *User) {
	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(AuthResult{
		User:  user,
		Token: token,
	})
}

func (ctx *Context) PrintError(w http.ResponseWriter, err error) {
	log.Errorf(ctx, "context error: %v", err)
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
