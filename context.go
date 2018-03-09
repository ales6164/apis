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
	a               *Apis
	r               *http.Request
	IsAuthenticated bool
	context.Context
	UserEmail       string
	UserKey         *datastore.Key
	UserGroup       UserGroup
	*body
}

type body struct {
	hasReadBody bool
	body        []byte
}

func (a *Apis) NewContext(r *http.Request) Context {
	return Context{
		a:         a,
		r:         r,
		UserGroup: public,
		Context:   appengine.NewContext(r),
		body:      &body{hasReadBody: false},
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

func (ctx Context) HasPermission(e *kind.Kind, scope Scope) (Context, error) {
	if val1, ok := ctx.a.permissions[ctx.UserGroup]; ok {
		if val2, ok := val1[e.Name]; ok {
			if val3, ok := val2[scope]; ok && val3 {
				return ctx, nil
			} else if val3, ok := val2["*"]; ok && val3 {
				return ctx, nil
			}
		} else if val2, ok := val1["*"]; ok {
			if val3, ok := val2[scope]; ok && val3 {
				return ctx, nil
			} else if val3, ok := val2["*"]; ok && val3 {
				return ctx, nil
			}
		}
	}

	return ctx, ErrForbidden
}

// Authenticates user
func (ctx Context) Authenticate() (bool, Context) {
	var isAuthenticated, isExpired bool
	var userEmail, userGroup string

	tkn := gcontext.Get(ctx.r, "auth")
	if tkn != nil {
		token := tkn.(*jwt.Token)
		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			if err := claims.Valid(); err == nil {
				if userGroup, ok = claims["gro"].(string); ok && len(userGroup) > 0 {
				}
				if userEmail, ok = claims["sub"].(string); ok && len(userEmail) > 0 {
					isAuthenticated = true
				}
			} /*else if exp, ok := claims["exp"].(float64); ok {
				// check if it's less than a week old
				if time.Now().Unix()-int64(exp) < time.Now().Add(time.Hour * 24 * 7).Unix() {
					if projectNamespace, ok = claims["pro"].(string); ok && len(projectNamespace) > 0 {
						hasProjectNamespace = true
					}
					if userEmail, ok = claims["sub"].(string); ok && len(userEmail) > 0 {
						isAuthenticated = true
						isExpired = true
					}
				}
			}*/
		}
	}

	ctx.IsAuthenticated = isAuthenticated && !isExpired
	if ctx.IsAuthenticated {
		ctx.UserEmail = userEmail
		ctx.UserGroup = UserGroup(userGroup)
		ctx.UserKey = datastore.NewKey(ctx, "User", userEmail, 0, nil)
	} else {
		ctx.UserEmail = ""
		ctx.UserGroup = ""
		ctx.UserKey = nil
	}

	return ctx.IsAuthenticated, ctx
}

// Authenticates user; if token is expired, returns a renewed unsigned *jwt.Token
/*func (ctx Context) Renew() (Context, *jwt.Token) {
	var isAuthenticated, hasProjectNamespace bool
	var userEmail, projectNamespace string
	var unsignedToken *jwt.Token

	tkn := gcontext.Get(ctx.r, "auth")
	if tkn != nil {
		token := tkn.(*jwt.Token)

		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {

			if err := claims.Valid(); err == nil {
				if projectNamespace, ok = claims["pro"].(string); ok && len(projectNamespace) > 0 {
					hasProjectNamespace = true
				}
				if userEmail, ok = claims["sub"].(string); ok && len(userEmail) > 0 {
					isAuthenticated = true
				}
			} else if exp, ok := claims["exp"].(float64); ok {
				// check if it's less than a week old
				if time.Now().Unix()-int64(exp) < time.Now().Add(time.Hour * 24 * 7).Unix() {
					if projectNamespace, ok = claims["pro"].(string); ok && len(projectNamespace) > 0 {
						hasProjectNamespace = true
					}
					if userEmail, ok = claims["sub"].(string); ok && len(userEmail) > 0 {
						isAuthenticated = true
					}
				}
			}
		}
	}

	ctx.IsAuthenticated = isAuthenticated
	ctx.User = userEmail

	vars := mux.Vars(ctx.r)
	newProjectNamespace := vars["project"]
	if len(newProjectNamespace) > 0 {
		ctx.HasProjectAccess = true
		ctx.Project = newProjectNamespace
	} else {
		ctx.HasProjectAccess = hasProjectNamespace
		ctx.Project = projectNamespace
	}

	// issue a new token
	if isAuthenticated {
		unsignedToken = NewToken(ctx.User)
	}

	return ctx, unsignedToken
}*/

func NewToken(user *user) *jwt.Token {
	var exp = time.Now().Add(time.Hour * 72).Unix()
	return jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"aud": "api",
		"nbf": time.Now().Add(-time.Minute).Unix(),
		"exp": exp,
		"iat": time.Now().Unix(),
		"iss": "sdk",
		"sub": user.Email,
		"gro": user.Group,
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
		User:  &User{user.Email, user.Group},
		Token: token,
	}

	json.NewEncoder(w).Encode(out)
}

func (ctx *Context) PrintError(w http.ResponseWriter, err error) {
	log.Errorf(ctx, "context error: %v", err)
	if err == ErrUnathorized {
		w.WriteHeader(http.StatusUnauthorized)
	} else if _, ok := err.(*Error); ok {
		w.WriteHeader(http.StatusBadRequest)
	} else {
		w.WriteHeader(http.StatusInternalServerError)
	}
	w.Write([]byte(err.Error()))
}
