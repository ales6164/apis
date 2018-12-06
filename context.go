package apis

import (
	"encoding/json"
	"github.com/ales6164/client"
	gContext "github.com/gorilla/context"
	"golang.org/x/net/context"
	"google.golang.org/appengine"
	"google.golang.org/appengine/log"
	"io/ioutil"
	"net/http"
)

type Context struct {
	*client.Client
	context.Context
	hasReadBody bool
	body        []byte
}

func NewContext(r *http.Request) Context {
	gaeCtx := appengine.NewContext(r)
	// restore from request
	if c1, ok := gContext.GetOk(r, "context"); ok {
		if c, ok := c1.(Context); ok {
			c.Context = gaeCtx
			return c
		}
	}
	clientReq := client.New(gaeCtx, r)
	return Context{
		Client:  clientReq,
		Context: gaeCtx,
	}
}

// reads body once and stores contents
func (ctx Context) Body() []byte {
	if !ctx.hasReadBody {
		ctx.body, _ = ioutil.ReadAll(ctx.HttpRequest.Body)
		ctx.HttpRequest.Body.Close()
		ctx.hasReadBody = true
	}
	return ctx.body
}

func (ctx Context) User() (*client.User, error) {
	return ctx.Client.Session.GetUser(ctx)
}

func (ctx Context) HasScope(scopes ...string) bool {
	if ctx.Client.IsPublic {
		return true
	}
	for _, s := range scopes {
		for _, r := range ctx.Client.Session.Scopes {
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
