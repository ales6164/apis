package apis

import (
	"encoding/json"
	"github.com/ales6164/apis/errors"
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
	// restore from request
	if c1, ok := gContext.GetOk(r, "context"); ok {
		if c, ok := c1.(Context); ok {
			return c
		}
	}
	gaeCtx := appengine.NewContext(r)
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

/*func (ctx Context) Id() string {
	return mux.Vars(ctx.Client.HttpRequest)["id"]
}

func (ctx Context) SetNamespace(namespace string) (Context, error) {
	var err error
	ctx.Context, err = appengine.Namespace(ctx, namespace)
	return ctx, err
}*/

func (ctx Context) HasScope(scopes ...string) bool {
	if len(scopes) == 0 {
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

func (ctx *Context) Print(w http.ResponseWriter, result interface{}, headerPair ...string) {
	w.Header().Set("Content-Type", "application/json")
	var headerKey string
	for i, headerEl := range headerPair {
		if i%2 == 0 {
			headerKey = headerEl
			continue
		}
		w.Header().Set(headerKey, headerEl)
	}
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

func (ctx *Context) PrintError(w http.ResponseWriter, err error, descriptors ...string) {
	/*ctx.ClientRequest.Error = err.Error()
	for i, d := range descriptors {
		ctx.ClientRequest.Error += `\n[descriptor"` + strconv.Itoa(i) + `","` + d + `"]`
	}
	log.Errorf(ctx, "context error: %s", ctx.ClientRequest.Error)
	ctx.ClientRequest.Body = ctx.Body()
	datastore.Put(ctx, ctx.clientRequestKey, ctx.ClientRequest)*/
	log.Errorf(ctx, "context error: %s", err, descriptors)
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
