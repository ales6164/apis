package iam

import (
	"encoding/json"
	"github.com/dgrijalva/jwt-go"
	gctx "github.com/gorilla/context"
	"golang.org/x/net/context"
	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"
	"io/ioutil"
	"net/http"
)

type Context struct {
	*IAM
	context.Context
	def                     context.Context
	r                       *http.Request
	w                       http.ResponseWriter
	hasReadBody             bool
	body                    []byte
	token                   *jwt.Token
	session                 *Session
	HasIncludeMetaHeader    bool
	HasResolveMetaRefHeader bool
}

func (iam *IAM) NewContext(w http.ResponseWriter, r *http.Request) (ctx Context) {
	gae := appengine.NewContext(r)
	ctx = Context{IAM: iam, Context: gae, def: gae, w: w, r: r, HasIncludeMetaHeader: len(r.Header.Get("X-Include-Meta")) > 0, HasResolveMetaRefHeader: len(r.Header.Get("X-Resolve-Meta-Ref")) > 0}
	if t, ok := gctx.Get(r, "token").(*jwt.Token); ok {
		ctx.token = t
	}
	var err error
	ctx.session, err = startSession(ctx, ctx.token)
	if err != nil {
		log.Errorf(ctx, "context start session error: %s", err.Error())
	}
	return ctx
}

func (ctx Context) SetNamespace(ns string) (Context, error) {
	var err error
	ctx.Context, err = appengine.Namespace(ctx.Default(), ns)
	return ctx, err
}

/*func (ctx Context) HasAccess(rules Rules, scopes ...string) bool {
	var ruleScopes []string
	if ctx.session.IsValid {
		var sessRoles []string
		sessRoles = append(sessRoles, ctx.session.Roles...)
		if ctx.session.IsAuthenticated {
			sessRoles = append(sessRoles, AllAuthenticatedUsers)
		} else {
			sessRoles = append(sessRoles, AllUsers)
		}

		for key, value := range rules.Permissions {
			if ContainsScope(sessRoles, key) {
				ruleScopes = append(ruleScopes, value...)
			}
		}
		for _, s := range scopes {
			for _, s2 := range ruleScopes {
				if s == s2 {
					return true
				}
			}
		}
	}
	return false
}*/

// reads body once and stores contents
func (ctx Context) Body() []byte {
	if !ctx.hasReadBody {
		ctx.body, _ = ioutil.ReadAll(ctx.r.Body)
		_ = ctx.r.Body.Close()
		ctx.hasReadBody = true
	}
	return ctx.body
}

func (ctx Context) Default() context.Context {
	return ctx.def
}

func (ctx Context) Member() *datastore.Key {
	return ctx.session.stored.Subject
}

func (ctx Context) User() string {
	return ctx.session.stored.Subject.StringID()
}

func (ctx Context) IsAuthenticated() bool {
	return ctx.session.IsAuthenticated
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
	if err := json.NewEncoder(ctx.w).Encode(result); err != nil {
		http.Error(ctx.w, err.Error(), http.StatusInternalServerError)
	}
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
