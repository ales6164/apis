package apis

import (
	"encoding/json"
	"github.com/dgrijalva/jwt-go"
	"golang.org/x/net/context"
	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"
	"io/ioutil"
	"net/http"
)

type Context struct {
	context.Context
	a           *Apis
	r           *http.Request
	w           http.ResponseWriter
	hasReadBody bool
	body        []byte
	session     *Session
}

func (a *Apis) NewContext(w http.ResponseWriter, r *http.Request, scopes ...string) (ctx Context, ok bool) {
	var token *jwt.Token
	ctx = Context{Context: appengine.NewContext(r), w: w, r: r, a: a}
	if a.hasAuth {
		token, _ = a.auth.middleware.CheckJWT(w, r)
	}
	var err error
	ctx.session, err = StartSession(ctx, token)
	if err != nil {
		ctx.PrintError(err.Error(), http.StatusForbidden)
		return ctx, false
	}
	if len(scopes) > 0 {
		if ctx.HasScope(scopes...) {
			return ctx, true
		}
		ctx.PrintError(http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return ctx, false
	}
	return ctx, true
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
	return ctx.session.HasScope(scopes...)
}

func (ctx Context) Member() *datastore.Key {
	return ctx.session.Member
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
