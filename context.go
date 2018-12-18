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
	request     *http.Request
	hasReadBody bool
	body        []byte
	auth        *Auth
	session     *Session
}

func NewContext(r *http.Request) (ctx Context) {
	ctx = Context{Context: appengine.NewContext(r), request: r}
	if _auth, ok := gorilla.GetOk(r, "auth"); ok {
		if a, ok := _auth.(*Auth); ok {
			ctx.auth = a
			if _token, ok := gorilla.GetOk(r, "token"); ok {
				if token, ok := _token.(*jwt.Token); ok {
					ctx.session, _ = GetSession(ctx, token)
				}
			}
		}
	}
	return ctx
}

// reads body once and stores contents
func (ctx Context) Body() []byte {
	if !ctx.hasReadBody {
		ctx.body, _ = ioutil.ReadAll(ctx.request.Body)
		ctx.request.Body.Close()
		ctx.hasReadBody = true
	}
	return ctx.body
}

func (ctx Context) HasScope(scopes ...string) bool {
	return ctx.session.HasScope(scopes...)
}

func (ctx Context) User() *datastore.Key {
	return ctx.session.Subject
}

/**
RESPONSE
*/

func (ctx *Context) Print(w http.ResponseWriter, result interface{}, statusCode int, headerPair ...string) {
	w.Header().Set("Content-Type", "application/json")
	var headerKey string
	for i, headerEl := range headerPair {
		if i%2 == 0 {
			headerKey = headerEl
			continue
		}
		if len(headerEl) > 0 {
			w.Header().Set(headerKey, headerEl)
		}
	}
	w.WriteHeader(statusCode)
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
		ctx.PrintError(w, http.StatusInternalServerError)
		return
	}
	w.Write(bs)
}

func (ctx *Context) PrintError(w http.ResponseWriter, code int, descriptors ...string) {
	log.Errorf(ctx, "context error: ", descriptors)
	http.Error(w, http.StatusText(code), code)
}
