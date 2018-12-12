package kind

import (
	"encoding/json"
	gorilla "github.com/gorilla/context"
	"golang.org/x/net/context"
	"google.golang.org/appengine"
	"google.golang.org/appengine/log"
	"io/ioutil"
	"net/http"
)

type Context struct {
	context.Context
	request      *http.Request
	scopes       []string
	hasScopes    bool
	isProtected  bool
	namespace    string
	hasNamespace bool
	hasReadBody  bool
	body         []byte
}

func NewContext(r *http.Request) (ctx Context) {
	var err error
	ctx = Context{request: r}

	if _scopes, ok := gorilla.GetOk(r, "scopes"); ok {
		ctx.isProtected=true
		if scopes, ok := _scopes.([]string); ok {
			ctx.scopes = scopes
			ctx.hasScopes = true
		}
	}

	if _namespace, ok := gorilla.GetOk(r, "namespace"); ok {
		if namespace, ok := _namespace.(string); ok && len(namespace) > 0 {
			gaeCtx := appengine.NewContext(r)
			if ctx.Context, err = appengine.Namespace(gaeCtx, namespace); err == nil {
				ctx.Context = gaeCtx
				ctx.namespace = namespace
				ctx.hasNamespace = true
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
	if !ctx.isProtected {
		return true
	}
 	for _, s := range scopes {
		for _, r := range ctx.scopes {
			if r == s {
				return true
			}
		}
	}
	return false
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
		ctx.PrintError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(bs)
}

func (ctx *Context) PrintError(w http.ResponseWriter, err string, code int, descriptors ...string) {
	log.Errorf(ctx, "context error: %s", err, descriptors)
	http.Error(w, err, code)
}
