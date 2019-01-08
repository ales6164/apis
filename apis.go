package apis

import (
	"github.com/ales6164/apis/kind"
	"github.com/gorilla/mux"
	"google.golang.org/appengine/datastore"
	"net/http"
)

type Apis struct {
	*mux.Router
	*Options
	auth       *Auth
	hasAuth    bool
	authRouter *mux.Router
	kinds      map[string]kind.Kind
	//roles      map[string][]string
}

type Options struct {
	Rules Rules
}

type Match map[kind.Kind]Rules
type Roles []string

type Rules struct {
	FullControl Roles
	ReadOnly    Roles
	ReadWrite   Roles
	Delete      Roles
	Match       Match
}

func New(options *Options) *Apis {
	if options == nil {
		options = &Options{}
	}

	a := &Apis{
		Options: options,
		Router:  mux.NewRouter(),
		kinds:   map[string]kind.Kind{},
		//roles:   map[string][]string{},
	}

	//a.authRouter = a.Router.PathPrefix(joinPath("auth")).Subrouter()
	//a.collectionRouter = a.Router.PathPrefix(joinPath("{collection}")).Subrouter()

	/*for role, rules := range a.Roles {
		for _, scopes := range rules {
			a.roles[role] = append(a.roles[role], scopes...)
		}
	}*/

	return a
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

/*func (a *Apis) SetRoles(roles map[string][][]string) {
	a.Roles = roles
	for role, rules := range a.Roles {
		for _, scopes := range rules {
			a.roles[role] = append(a.roles[role], scopes...)
		}
	}
}*/

func (a *Apis) HandleKind(k kind.Kind) {
	a.kinds[k.Name()] = k
	a.handleKind(k.Name(), k)
}

// TODO: before finishing all methods figure out how to handle group keys and keys containing namespace
func (a *Apis) handleKind(rootPath string, k kind.Kind) {
	pathWId := joinPath("/", rootPath, "{id}")

	// QUERY
	/*a.Handle(rootPath, serve(func(w http.ResponseWriter, r *http.Request) {
		if ctx, ok := a.NewContext(w, r, k.Scopes(ReadOnly, ReadWrite, FullControl)...); ok {

		}
	})).Methods(http.MethodGet, http.MethodOptions)*/

	// GET
	a.Handle(pathWId, serve(func(w http.ResponseWriter, r *http.Request) {
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
	})).Methods(http.MethodPut, http.MethodOptions)

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
