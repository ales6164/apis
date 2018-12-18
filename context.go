package apis

import (
	"encoding/json"
	"github.com/dgrijalva/jwt-go"
	gorilla "github.com/gorilla/context"
	"golang.org/x/net/context"
	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"
	"io/ioutil"
	"net/http"
)

type Context struct {
	context.Context
	r               *http.Request
	w               http.ResponseWriter
	hasReadBody     bool
	body            []byte
	auth            *Auth
	session         *Session
	isAuthenticated bool
}

func NewContext(w http.ResponseWriter, r *http.Request) (ctx Context) {
	ctx = Context{Context: appengine.NewContext(r), w: w, r: r}
	if _auth, ok := gorilla.GetOk(r, "auth"); ok {
		if a, ok := _auth.(*Auth); ok {
			ctx.auth = a
			if _token, ok := gorilla.GetOk(r, "token"); ok {
				if token, ok := _token.(*jwt.Token); ok {
					var err error
					ctx.session, err = GetSession(ctx, token)
					ctx.isAuthenticated = err == nil
				}
			}
		}
	}
	return ctx
}

// reads body once and stores contents
func (ctx Context) Body() []byte {
	if !ctx.hasReadBody {
		ctx.body, _ = ioutil.ReadAll(ctx.r.Body)
		ctx.r.Body.Close()
		ctx.hasReadBody = true
	}
	return ctx.body
}

func (ctx Context) HasScope(scopes ...string) bool {
	if ctx.isAuthenticated {

	}
	return ctx.session.HasScope(scopes...)
}

func (ctx Context) User() *datastore.Key {
	return ctx.session.Subject
}

/**
RESPONSE
*/

func (ctx *Context) PrintJSON(result interface{}, statusCode int, headerPair ...string) {
	ctx.w.Header().Set("Content-Type", "application/json")
	var headerKey string
	for i, headerEl := range headerPair {
		if i%2 == 0 {
			headerKey = headerEl
			continue
		}
		if len(headerEl) > 0 {
			ctx.w.Header().Set(headerKey, headerEl)
		}
	}
	ctx.w.WriteHeader(statusCode)
	json.NewEncoder(ctx.w).Encode(result)
}

func (ctx *Context) PrintStatus(s string, c int, descriptors ...string) {
	log.Errorf(ctx, "context error: ", descriptors)
	ctx.w.WriteHeader(c)
	ctx.w.Write([]byte(s))
}

func (ctx *Context) PrintError(s string, c int, descriptors ...string) {
	log.Errorf(ctx, "context error: ", descriptors)
	http.Error(ctx.w, s, c)
}
