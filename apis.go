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
	authRouter       *mux.Router
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

	a.authRouter = a.Router.PathPrefix(joinPath("auth")).Subrouter()
	a.collectionRouter = a.Router.PathPrefix(joinPath("{collection}")).Subrouter()

	if a.Roles == nil {
		a.Roles = map[string][]string{}
	}

	return a
}

func (a *Apis) SetAuth(auth *Auth) {
	a.auth = auth
	auth.a = a
	a.hasAuth = auth != nil
	for _, p := range auth.providers {
		a.authRouter.HandleFunc(joinPath(p.GetName(), "login"), func(w http.ResponseWriter, r *http.Request) {
			ctx, err := a.NewContext(w, r)
			if err != nil {
				ctx.PrintError(err.Error(), http.StatusForbidden)
				return
			}
			p.Login(ctx)
		}).Methods(http.MethodPost)
		a.authRouter.HandleFunc(joinPath(p.GetName(), "register"), func(w http.ResponseWriter, r *http.Request) {
			ctx, err := a.NewContext(w, r)
			if err != nil {
				ctx.PrintError(err.Error(), http.StatusForbidden)
				return
			}
			p.Register(ctx)
		}).Methods(http.MethodPost)
	}
}

func (a *Apis) SetRole(name string, scopes ...string) {
	a.Roles[name] = append(a.Roles[name], scopes...)
}

func (a *Apis) RegisterKind(k *Kind) {
	a.kinds[k.Path] = k

	// QUERY
	a.Handle(joinPath(k.Path), serve(func(w http.ResponseWriter, r *http.Request) {
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
	})).Methods(http.MethodGet, http.MethodOptions)

	// POST
	a.Handle(joinPath(k.Path), serve(func(w http.ResponseWriter, r *http.Request) {
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
	})).Methods(http.MethodPost, http.MethodOptions)

	// GET
	a.Handle(joinPath(k.Path, "{key}"), serve(func(w http.ResponseWriter, r *http.Request) {
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
	})).Methods(http.MethodGet, http.MethodOptions)

	// GET with path
	a.Handle(joinPath(k.Path, "{key}", `{path:[a-zA-Z0-9=\-\/]+}`), serve(func(w http.ResponseWriter, r *http.Request) {
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
	})).Methods(http.MethodGet, http.MethodOptions)

	// PUT
	a.Handle(joinPath(k.Path, "{key}"), serve(func(w http.ResponseWriter, r *http.Request) {
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
	})).Methods(http.MethodPut, http.MethodOptions)

	// PUT with path
	a.Handle(joinPath(k.Path, "{key}", `{path:[a-zA-Z0-9=\-\/]+}`), serve(func(w http.ResponseWriter, r *http.Request) {
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
	})).Methods(http.MethodPut, http.MethodOptions)

	// PATCH
	a.Handle(joinPath(k.Path, "{key}"), serve(func(w http.ResponseWriter, r *http.Request) {
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
			k.PatchHandler(ctx, key)
		} else {
			ctx.PrintError(http.StatusText(http.StatusForbidden), http.StatusForbidden)
		}
	})).Methods(http.MethodPatch, http.MethodOptions)

	// DELETE
	a.Handle(joinPath(k.Path, "{key}"), serve(func(w http.ResponseWriter, r *http.Request) {
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
	})).Methods(http.MethodDelete, http.MethodOptions)

	// DELETE with path
	a.Handle(joinPath(k.Path, "{key}", `{path:[a-zA-Z0-9=\-\/]+}`), serve(func(w http.ResponseWriter, r *http.Request) {
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
	})).Methods(http.MethodDelete, http.MethodOptions)

	// COLLECTIONS

	// QUERY
	a.collectionRouter.Handle(joinPath(k.Path), serve(func(w http.ResponseWriter, r *http.Request) {
		ctx, err := a.NewContext(w, r)
		if err != nil {
			ctx.PrintError(err.Error(), http.StatusForbidden)
			return
		}

		var collectionKey *datastore.Key
		vars := mux.Vars(r)
		if encodedKey, ok := vars["collection"]; ok {
			if collectionKey, err = datastore.DecodeKey(encodedKey); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		}

		if ctx, ok := CheckCollectionAccess(ctx, collectionKey, ReadOnly, ReadWrite, FullControl); ok {
			k.QueryHandler(ctx)
		} else {
			ctx.PrintError(http.StatusText(http.StatusForbidden), http.StatusForbidden)
		}
	})).Methods(http.MethodGet, http.MethodOptions)

	// POST
	a.collectionRouter.Handle(joinPath(k.Path), serve(func(w http.ResponseWriter, r *http.Request) {
		ctx, err := a.NewContext(w, r)
		if err != nil {
			ctx.PrintError(err.Error(), http.StatusForbidden)
			return
		}
		var collectionKey *datastore.Key
		vars := mux.Vars(r)
		if encodedKey, ok := vars["collection"]; ok {
			if collectionKey, err = datastore.DecodeKey(encodedKey); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		}
		if ctx, ok := CheckCollectionAccess(ctx, collectionKey, ReadOnly, ReadWrite, FullControl); ok {
			k.PostHandler(ctx, nil)
		} else {
			ctx.PrintError(http.StatusText(http.StatusForbidden), http.StatusForbidden)
		}
	})).Methods(http.MethodPost, http.MethodOptions)

	// GET
	a.collectionRouter.Handle(joinPath(k.Path, "{key}"), serve(func(w http.ResponseWriter, r *http.Request) {
		ctx, err := a.NewContext(w, r)
		if err != nil {
			ctx.PrintError(err.Error(), http.StatusForbidden)
			return
		}
		var collectionKey *datastore.Key
		vars := mux.Vars(r)
		if encodedKey, ok := vars["collection"]; ok {
			if collectionKey, err = datastore.DecodeKey(encodedKey); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		}
		if ctx, ok := CheckCollectionAccess(ctx, collectionKey, ReadOnly, ReadWrite, FullControl); ok {
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
	})).Methods(http.MethodGet, http.MethodOptions)

	// GET with path
	a.collectionRouter.Handle(joinPath(k.Path, "{key}", `{path:[a-zA-Z0-9=\-\/]+}`), serve(func(w http.ResponseWriter, r *http.Request) {
		ctx, err := a.NewContext(w, r)
		if err != nil {
			ctx.PrintError(err.Error(), http.StatusForbidden)
			return
		}
		var collectionKey *datastore.Key
		vars := mux.Vars(r)
		if encodedKey, ok := vars["collection"]; ok {
			if collectionKey, err = datastore.DecodeKey(encodedKey); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		}
		if ctx, ok := CheckCollectionAccess(ctx, collectionKey, ReadOnly, ReadWrite, FullControl); ok {
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
	})).Methods(http.MethodGet, http.MethodOptions)

	// PUT
	a.collectionRouter.Handle(joinPath(k.Path, "{key}"), serve(func(w http.ResponseWriter, r *http.Request) {
		ctx, err := a.NewContext(w, r)
		if err != nil {
			ctx.PrintError(err.Error(), http.StatusForbidden)
			return
		}
		var collectionKey *datastore.Key
		vars := mux.Vars(r)
		if encodedKey, ok := vars["collection"]; ok {
			if collectionKey, err = datastore.DecodeKey(encodedKey); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		}
		if ctx, ok := CheckCollectionAccess(ctx, collectionKey, ReadWrite, FullControl); ok {
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
	})).Methods(http.MethodPut, http.MethodOptions)

	// PUT with path
	a.collectionRouter.Handle(joinPath(k.Path, "{key}", `{path:[a-zA-Z0-9=\-\/]+}`), serve(func(w http.ResponseWriter, r *http.Request) {
		ctx, err := a.NewContext(w, r)
		if err != nil {
			ctx.PrintError(err.Error(), http.StatusForbidden)
			return
		}
		var collectionKey *datastore.Key
		vars := mux.Vars(r)
		if encodedKey, ok := vars["collection"]; ok {
			if collectionKey, err = datastore.DecodeKey(encodedKey); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		}
		if ctx, ok := CheckCollectionAccess(ctx, collectionKey, ReadWrite, FullControl); ok {
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
	})).Methods(http.MethodPut, http.MethodOptions)

	// DELETE
	a.collectionRouter.Handle(joinPath(k.Path, "{key}"), serve(func(w http.ResponseWriter, r *http.Request) {
		ctx, err := a.NewContext(w, r)
		if err != nil {
			ctx.PrintError(err.Error(), http.StatusForbidden)
			return
		}
		var collectionKey *datastore.Key
		vars := mux.Vars(r)
		if encodedKey, ok := vars["collection"]; ok {
			if collectionKey, err = datastore.DecodeKey(encodedKey); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		}
		if ctx, ok := CheckCollectionAccess(ctx, collectionKey, Delete, FullControl); ok {
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
	})).Methods(http.MethodDelete, http.MethodOptions)

	// DELETE with path
	a.collectionRouter.Handle(joinPath(k.Path, "{key}", `{path:[a-zA-Z0-9=\-\/]+}`), serve(func(w http.ResponseWriter, r *http.Request) {
		ctx, err := a.NewContext(w, r)
		if err != nil {
			ctx.PrintError(err.Error(), http.StatusForbidden)
			return
		}
		var collectionKey *datastore.Key
		vars := mux.Vars(r)
		if encodedKey, ok := vars["collection"]; ok {
			if collectionKey, err = datastore.DecodeKey(encodedKey); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		}
		if ctx, ok := CheckCollectionAccess(ctx, collectionKey, Delete, FullControl); ok {
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
	})).Methods(http.MethodDelete, http.MethodOptions)
}
