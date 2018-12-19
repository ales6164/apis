package apis

import (
	"github.com/gorilla/mux"
	"google.golang.org/appengine/datastore"
	"net/http"
	"strings"
)

type Apis struct {
	*mux.Router
	*Options
	auth             *Auth
	hasAuth          bool
	collectionRouter *mux.Router
	kinds            map[string]*Kind
}

type Options struct {
	Roles map[string][]string
}

func New(options *Options) *Apis {
	if options == nil {
		options = &Options{}
	}

	if options.Roles == nil {
		options.Roles = map[string][]string{}
	}

	a := &Apis{
		Options: options,
		Router:  mux.NewRouter(),
		kinds:   map[string]*Kind{},
	}

	a.Router.Use(func(handler http.Handler) http.Handler {
		return http.Handler(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if origin := req.Header.Get("Origin"); origin != "" {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
				w.Header().Set("Access-Control-Allow-Headers",
					"Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, Cache-Control, "+
						"X-Requested-With, X-Total-Count, Link")
				w.Header().Set("Access-Control-Expose-Headers",
					"Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, Cache-Control, "+
						"X-Requested-With, X-Total-Count, Link")
			}
			if req.Method == "OPTIONS" {
				return
			}
			handler.ServeHTTP(w, req)
		}))
	})

	a.collectionRouter = a.Router.PathPrefix(joinPath("{collection}")).Subrouter()

	if a.Roles == nil {
		a.Roles = map[string][]string{}
	}

	//a.Handle("/iam", IAMKind)

	return a
}

func (a *Apis) SetAuth(auth *Auth) {
	a.auth = auth
	auth.a = a
	a.hasAuth = auth != nil
}

func (a *Apis) SetRole(name string, scopes ...string) {
	a.Roles[name] = append(a.Roles[name], scopes...)
}

func (a *Apis) RegisterKind(k *Kind) {
	a.kinds[k.Path] = k

	// QUERY
	a.HandleFunc(joinPath(k.Path), func(w http.ResponseWriter, r *http.Request) {
		ctx, err := a.NewContext(w, r)
		if err != nil {
			ctx.PrintError(err.Error(), http.StatusForbidden)
			return
		}
		if ok := ctx.HasScope(k.ScopeReadOnly, k.ScopeReadWrite, k.ScopeFullControl); ok {
			k.QueryHandler(ctx)
		} else {
			ctx.PrintError(http.StatusText(http.StatusForbidden), http.StatusForbidden)
		}
	}).Methods(http.MethodGet)

	// POST
	a.HandleFunc(joinPath(k.Path), func(w http.ResponseWriter, r *http.Request) {
		ctx, err := a.NewContext(w, r)
		if err != nil {
			ctx.PrintError(err.Error(), http.StatusForbidden)
			return
		}
		if ok := ctx.HasScope(k.ScopeReadWrite, k.ScopeFullControl); ok {
			k.PostHandler(ctx, nil)
		} else {
			ctx.PrintError(http.StatusText(http.StatusForbidden), http.StatusForbidden)
		}
	}).Methods(http.MethodPost)

	// GET
	a.HandleFunc(joinPath(k.Path, "{key}"), func(w http.ResponseWriter, r *http.Request) {
		ctx, err := a.NewContext(w, r)
		if err != nil {
			ctx.PrintError(err.Error(), http.StatusForbidden)
			return
		}
		if ok := ctx.HasScope(k.ScopeReadOnly, k.ScopeReadWrite, k.ScopeFullControl); ok {
			var key *datastore.Key
			vars := mux.Vars(r)
			if encodedKey, ok := vars["key"]; ok {
				if key, err = datastore.DecodeKey(encodedKey); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
			}
			k.GetHandler(ctx, key)
		} else {
			ctx.PrintError(http.StatusText(http.StatusForbidden), http.StatusForbidden)
		}
	}).Methods(http.MethodGet)

	// GET with path
	a.HandleFunc(joinPath(k.Path, "{key}", `{path:[a-zA-Z0-9=\-\/]+}`), func(w http.ResponseWriter, r *http.Request) {
		ctx, err := a.NewContext(w, r)
		if err != nil {
			ctx.PrintError(err.Error(), http.StatusForbidden)
			return
		}
		if ok := ctx.HasScope(k.ScopeReadOnly, k.ScopeReadWrite, k.ScopeFullControl); ok {
			var key *datastore.Key
			var path []string
			vars := mux.Vars(r)
			if encodedKey, ok := vars["key"]; ok {
				if key, err = datastore.DecodeKey(encodedKey); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
			}
			if _path, ok := vars["path"]; ok {
				path = strings.Split(_path, "/")
			}
			k.GetHandler(ctx, key, path...)
		} else {
			ctx.PrintError(http.StatusText(http.StatusForbidden), http.StatusForbidden)
		}
	}).Methods(http.MethodGet)

	// PUT
	a.HandleFunc(joinPath(k.Path, "{key}"), func(w http.ResponseWriter, r *http.Request) {
		ctx, err := a.NewContext(w, r)
		if err != nil {
			ctx.PrintError(err.Error(), http.StatusForbidden)
			return
		}
		if ok := ctx.HasScope(k.ScopeReadWrite, k.ScopeFullControl); ok {
			var key *datastore.Key
			vars := mux.Vars(r)
			if encodedKey, ok := vars["key"]; ok {
				if key, err = datastore.DecodeKey(encodedKey); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
			}
			k.PutHandler(ctx, key)
		} else {
			ctx.PrintError(http.StatusText(http.StatusForbidden), http.StatusForbidden)
		}
	}).Methods(http.MethodPut)

	// PUT with path
	a.HandleFunc(joinPath(k.Path, "{key}", `{path:[a-zA-Z0-9=\-\/]+}`), func(w http.ResponseWriter, r *http.Request) {
		ctx, err := a.NewContext(w, r)
		if err != nil {
			ctx.PrintError(err.Error(), http.StatusForbidden)
			return
		}
		if ok := ctx.HasScope(k.ScopeReadWrite, k.ScopeFullControl); ok {
			var key *datastore.Key
			var path []string
			vars := mux.Vars(r)
			if encodedKey, ok := vars["key"]; ok {
				if key, err = datastore.DecodeKey(encodedKey); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
			}
			if _path, ok := vars["path"]; ok {
				path = strings.Split(_path, "/")
			}
			k.PutHandler(ctx, key, path...)
		} else {
			ctx.PrintError(http.StatusText(http.StatusForbidden), http.StatusForbidden)
		}
	}).Methods(http.MethodPut)

	// DELETE
	a.HandleFunc(joinPath(k.Path, "{key}"), func(w http.ResponseWriter, r *http.Request) {
		ctx, err := a.NewContext(w, r)
		if err != nil {
			ctx.PrintError(err.Error(), http.StatusForbidden)
			return
		}
		if ok := ctx.HasScope(k.ScopeDelete, k.ScopeFullControl); ok {
			var key *datastore.Key
			vars := mux.Vars(r)
			if encodedKey, ok := vars["key"]; ok {
				if key, err = datastore.DecodeKey(encodedKey); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
			}
			k.DeleteHandler(ctx, key)
		} else {
			ctx.PrintError(http.StatusText(http.StatusForbidden), http.StatusForbidden)
		}
	}).Methods(http.MethodDelete)

	// DELETE with path
	a.HandleFunc(joinPath(k.Path, "{key}", `{path:[a-zA-Z0-9=\-\/]+}`), func(w http.ResponseWriter, r *http.Request) {
		ctx, err := a.NewContext(w, r)
		if err != nil {
			ctx.PrintError(err.Error(), http.StatusForbidden)
			return
		}
		if ok := ctx.HasScope(k.ScopeDelete, k.ScopeFullControl); ok {
			var key *datastore.Key
			var path []string
			vars := mux.Vars(r)
			if encodedKey, ok := vars["key"]; ok {
				if key, err = datastore.DecodeKey(encodedKey); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
			}
			if _path, ok := vars["path"]; ok {
				path = strings.Split(_path, "/")
			}
			k.DeleteHandler(ctx, key, path...)
		} else {
			ctx.PrintError(http.StatusText(http.StatusForbidden), http.StatusForbidden)
		}
	}).Methods(http.MethodDelete)

	// TODO: HANDLE collection permission checking right inside handlefunc
	/*a.collectionRouter.HandleFunc(joinPath(k.Path), func(w http.ResponseWriter, r *http.Request) {
		ctx := NewContext(w, r)
		k.QueryHandler(ctx)
	}).Methods(http.MethodGet)*/
	/*a.Handle("{collection}"+path, h)
	a.Handle("{collection}"+path+`/{key}`, h)
	a.Handle("{collection}"+path+`/{key}/{path:[a-zA-Z0-9=\-\/]+}`, h)*/
}
