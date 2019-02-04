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
	a                    *Apis
	r                    *http.Request
	w                    http.ResponseWriter
	hasReadBody          bool
	body                 []byte
	session              *Session
	hasIncludeMetaHeader bool
	authError            error
	sessError            error
}

func (a *Apis) NewContext(w http.ResponseWriter, r *http.Request) (ctx Context) {
	ctx = Context{Context: appengine.NewContext(r), w: w, r: r, a: a, hasIncludeMetaHeader: len(r.Header.Get("X-Include-Meta")) > 0}
	var token *jwt.Token
	if ctx.a.hasAuth {
		token, ctx.authError = ctx.a.Auth.middleware.CheckJWT(ctx.w, ctx.r)
	}
	ctx.session, ctx.sessError = StartSession(ctx, token)
	return ctx
}

func (ctx Context) HasAccess(rules Rules, scopes ...string) bool {
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
}

// reads body once and stores contents
func (ctx Context) Body() []byte {
	if !ctx.hasReadBody {
		ctx.body, _ = ioutil.ReadAll(ctx.r.Body)
		_ = ctx.r.Body.Close()
		ctx.hasReadBody = true
	}
	return ctx.body
}

func (ctx Context) Member() *datastore.Key {
	return ctx.session.Member
}

func (ctx Context) User() string {
	return ctx.session.Member.StringID()
}

func (ctx Context) IsAuthenticated() bool {
	return ctx.session.IsAuthenticated
}

func (ctx Context) ExtendSession(seconds int64) error {
	return ctx.session.Extend(ctx, seconds)
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
