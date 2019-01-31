package apis

import (
	"github.com/ales6164/apis/kind"
	"github.com/gorilla/mux"
	"google.golang.org/appengine/datastore"
	"net/http"
)

type Apis struct {
	//router *mux.Router
	*Options
	hasAuth bool
	kinds   map[string]kind.Kind
	//roles      map[string][]string
	//http.Handler
	router *mux.Router
}

type Options struct {
	Auth  *Auth
	Rules *Rules
}

type Match map[kind.Kind]*Rules
type Roles []string

type Rules struct {
	Permissions Permissions
	Match       Match `json:"-"`
}

type Permissions map[string]Roles

func New(options *Options) *Apis {
	if options == nil {
		options = &Options{}
	}

	a := &Apis{
		Options: options,
		/*Router:  mux.NewRouter(),*/
		kinds: map[string]kind.Kind{},
	}

	a.router = mux.NewRouter()

	if a.Auth != nil {
		a.hasAuth = true
		a.Auth.Apis = a

		// renew token
		a.router.Handle("/auth/renew", Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := a.NewContext(w, r)

			err := ctx.ExtendSession(a.Auth.TokenExpiresIn)
			if err != nil {
				ctx.PrintError(err.Error(), http.StatusInternalServerError)
				return
			}

			user, err := a.Auth.User(ctx, ctx.Member())
			if err != nil {
				ctx.PrintError(err.Error(), http.StatusInternalServerError)
				return
			}

			signedToken, err := a.Auth.SignedToken(ctx.session)
			if err != nil {
				ctx.PrintError(err.Error(), http.StatusInternalServerError)
				return
			}

			ctx.PrintJSON(AuthResponse{
				User: user,
				Token: Token{
					Id:        signedToken,
					ExpiresAt: ctx.session.ExpiresAt.Unix(),
				},
			}, http.StatusOK)
		}))).Methods(http.MethodOptions, http.MethodPost)

		for _, p := range a.Auth.providers {
			a.router.Handle(`/auth/`+p.Name()+`/{path:[a-zA-Z0-9=\-\/]+}`, Middleware(p))
		}
	}

	a.router.Handle(`/{path:[a-zA-Z0-9=\-\/]+}`, Middleware(a))

	return a
}


func Middleware(h http.Handler) http.Handler {
	return http.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if origin := r.Header.Get("Origin"); origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
			w.Header().Set("Access-Control-Allow-Headers",
				"Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, Cache-Control, "+
					"X-Requested-With, X-Include-Meta")
		}

		if r.Method == http.MethodOptions {
			return
		}

		h.ServeHTTP(w, r)
	}))
}

/*func (a *Apis) SetAuth(auth *Auth) {
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
}*/

func (a *Apis) HandleKind(k kind.Kind) {
	a.kinds[k.Name()] = k
	a.handleKind(k.Name(), k)
}

type PathPair struct {
	CollectionName string         `json:"collectionName"`
	CollectionId   string         `json:"collectionId"`
	HasKey         bool           `json:"hasKey"`
	CollectionKey  *datastore.Key `json:"collectionKey"`
	CollectionKind kind.Kind      `json:"collectionKind"`
	IsGroup        bool           `json:"isGroup"`
	GroupKey       *datastore.Key `json:"groupKey"`
	Rules          Rules          `json:"rules"`
}

func (a *Apis) Handler() http.Handler {
	return a.router
}

func (a *Apis) Handle(path string, handler http.Handler) *mux.Route {
	return a.router.Handle(path, handler)
}

func (a *Apis) HandleFunc(path string, f func(w http.ResponseWriter, r *http.Request)) *mux.Route {
	return a.router.HandleFunc(path, f)
}

// TODO: before finishing all methods figure out how to handle group keys and keys containing namespace
func (a *Apis) handleKind(rootPath string, k kind.Kind) {
	//pathWId := joinPath("/", rootPath, "{id}")

	// QUERY
	/*a.Handle(rootPath, serve(func(w http.ResponseWriter, r *http.Request) {
		if ctx, ok := a.NewContext(w, r, k.Scopes(ReadOnly, ReadWrite, FullControl)...); ok {

		}
	})).Methods(http.MethodGet, http.MethodOptions)*/

	// GET
	/*a.Handle(pathWId, serve(func(w http.ResponseWriter, r *http.Request) {
		if ctx, ok := a.NewContext(w, r, k.Scopes(ReadOnly, ReadWrite, FullControl)...); ok {
			if id, ok := mux.Vars(r)["id"]; ok {
				key, err := datastore.DecodeKey(id)
				if err != nil {
					key = datastore.NewKey(ctx, k.Name(), id, 0, nil)
				}
				doc, err := k.Doc(ctx, key).Get()
				if err != nil {
					ctx.PrintError(err.Error(), http.StatusInternalServerError)
					return
				}
				ctx.PrintJSON(k.Data(doc), http.StatusOK)
			}
		}
	})).Methods(http.MethodGet, http.MethodOptions)

	// POST
	a.Handle(rootPath, serve(func(w http.ResponseWriter, r *http.Request) {
		if ctx, ok := a.NewContext(w, r, k.Scopes(ReadWrite, FullControl)...); ok {
			doc, err := k.Doc(ctx, nil).Add(ctx.Body())
			if err != nil {
				ctx.PrintError(err.Error(), http.StatusInternalServerError)
				return
			}
			ctx.PrintJSON(k.Data(doc), http.StatusOK)
		}
	})).Methods(http.MethodPost, http.MethodOptions)

	// POST TO GROUP
	a.Handle(pathWId, serve(func(w http.ResponseWriter, r *http.Request) {
		if ctx, ok := a.NewContext(w, r, k.Scopes(ReadWrite, FullControl)...); ok {
			doc, err := k.Doc(ctx, nil).Add(ctx.Body())
			if err != nil {
				ctx.PrintError(err.Error(), http.StatusInternalServerError)
				return
			}
			ctx.PrintJSON(k.Data(doc), http.StatusOK)
		}
	})).Methods(http.MethodPost, http.MethodOptions)

	// PUT
	a.Handle(pathWId, serve(func(w http.ResponseWriter, r *http.Request) {
		if ctx, ok := a.NewContext(w, r, k.Scopes(ReadWrite, FullControl)...); ok {
			if id, ok := mux.Vars(r)["id"]; ok {
				key, err := datastore.DecodeKey(id)
				if err != nil {
					key = datastore.NewKey(ctx, k.Name(), id, 0, nil)
				}
				doc, err := k.Doc(ctx, key).Set(ctx.Body())
				if err != nil {
					ctx.PrintError(err.Error(), http.StatusInternalServerError)
					return
				}
				ctx.PrintJSON(k.Data(doc), http.StatusOK)
			}
		}
	})).Methods(http.MethodPut, http.MethodOptions)*/

	/*
		// GET with path
		a.Handle(joinPath(k.Path, "{key}", `{path:[a-zA-Z0-9=\-\/]+}`), serve(func(w http.ResponseWriter, r *http.Request) {
			ctx, err := a.NewContext(w, r)
			if err != nil {
				ctx.PrintError(err.Error(), http.StatusForbidden)
				return
			}
			if ok := ctx.HasScope(k.Rules(ReadOnly, ReadWrite, FullControl)...); ok {
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
			if ok := ctx.HasScope(k.Rules(ReadWrite, FullControl)...); ok {
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
			if ok := ctx.HasScope(k.Rules(ReadWrite, FullControl)...); ok {
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
			if ok := ctx.HasScope(k.Rules(ReadWrite, FullControl)...); ok {
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
			if ok := ctx.HasScope(k.Rules(Delete, FullControl)...); ok {
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
			if ok := ctx.HasScope(k.Rules(Delete, FullControl)...); ok {
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
				groupKind := a.kinds[collectionKey.Kind()]
				if ok := ctx.HasScope(groupKind.Rules(k.Rules(ReadOnly, ReadWrite, FullControl)...)...); ok {
					k.QueryHandler(ctx)
					return
				}
			}
			ctx.PrintError(http.StatusText(http.StatusForbidden), http.StatusForbidden)
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
			if ctx, ok := CheckCollectionAccess(ctx, collectionKey, ReadWrite, FullControl); ok {
				groupKind := a.kinds[collectionKey.Kind()]
				if ok := ctx.HasScope(groupKind.Rules(k.Rules(ReadWrite, FullControl)...)...); ok {
					k.PostHandler(ctx, nil)
					return
				}
			}
			ctx.PrintError(http.StatusText(http.StatusForbidden), http.StatusForbidden)
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
				groupKind := a.kinds[collectionKey.Kind()]
				if ok := ctx.HasScope(groupKind.Rules(k.Rules(ReadOnly, ReadWrite, FullControl)...)...); ok {
					k.GetHandler(ctx, key)
					return
				}
			}
			ctx.PrintError(http.StatusText(http.StatusForbidden), http.StatusForbidden)
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
				groupKind := a.kinds[collectionKey.Kind()]
				if ok := ctx.HasScope(groupKind.Rules(k.Rules(ReadOnly, ReadWrite, FullControl)...)...); ok {
					k.GetHandler(ctx, key, path...)
					return
				}
			}
			ctx.PrintError(http.StatusText(http.StatusForbidden), http.StatusForbidden)
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

				groupKind := a.kinds[collectionKey.Kind()]
				if ok := ctx.HasScope(groupKind.Rules(k.Rules(ReadWrite, FullControl)...)...); ok {
					k.PutHandler(ctx, key)
					return
				}
			}
			ctx.PrintError(http.StatusText(http.StatusForbidden), http.StatusForbidden)
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

				groupKind := a.kinds[collectionKey.Kind()]
				if ok := ctx.HasScope(groupKind.Rules(k.Rules(ReadWrite, FullControl)...)...); ok {
					k.PutHandler(ctx, key, path...)
					return
				}
			}
			ctx.PrintError(http.StatusText(http.StatusForbidden), http.StatusForbidden)
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

				groupKind := a.kinds[collectionKey.Kind()]
				if ok := ctx.HasScope(groupKind.Rules(k.Rules(Delete, FullControl)...)...); ok {
					k.DeleteHandler(ctx, key)
					return
				}
			}
			ctx.PrintError(http.StatusText(http.StatusForbidden), http.StatusForbidden)
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

				groupKind := a.kinds[collectionKey.Kind()]
				if ok := ctx.HasScope(groupKind.Rules(k.Rules(Delete, FullControl)...)...); ok {
					k.DeleteHandler(ctx, key, path...)
					return
				}
			}
			ctx.PrintError(http.StatusText(http.StatusForbidden), http.StatusForbidden)
		})).Methods(http.MethodDelete, http.MethodOptions)*/
}
