package apis

import (
	"github.com/ales6164/apis/kind"
	"github.com/gorilla/mux"
	"google.golang.org/appengine/datastore"
	"net/http"
	"strings"
)

type Apis struct {
	//router *mux.Router
	*Options
	auth       *Auth
	hasAuth    bool
	authRouter *mux.Router
	kinds      map[string]kind.Kind
	//roles      map[string][]string
	http.Handler
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
	Match       Match `json:"-"`
}

func New(options *Options) *Apis {
	if options == nil {
		options = &Options{}
	}

	a := &Apis{
		Options: options,
		/*Router:  mux.NewRouter(),*/
		kinds: map[string]kind.Kind{},
	}

	return a
}

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

func (a *Apis) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if origin := r.Header.Get("Origin"); origin != "" {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers",
			"Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, Cache-Control, "+
				"X-Requested-With")
	}

	if r.Method == http.MethodOptions {
		return
	}

	a.serve(w, r)
	return

	ctx := a.NewContext(w, r)

	rules := a.Rules

	path := r.URL.Path
	if path[:1] == "/" {
		path = path[1:]
	}

	// GET /projects
	// GET /projects/FSDFKosfsfsefssgsdgf
	// GET /projects/FSDFKosfsfsefssgsdgf/objects
	// GET /projects/FSDFKosfsfsefssgsdgf/objects/sdfGdsGDSAFSDfdsgsdd
	// ...

	var err error
	var pairs []*PathPair
	var lastPair *PathPair
	parts := strings.Split(path, "/")
	for i := 0; i < len(parts); i += 2 {
		// get collection name
		pair := &PathPair{
			CollectionName: parts[i],
		}

		// get collection kind and match it to rules
		if k, ok := a.kinds[pair.CollectionName]; ok {
			pair.CollectionKind = k
			if rules, ok = rules.Match[k]; ok {
				// got latest rules
				pair.Rules = rules
			} else {
				//ctx.PrintError(http.StatusText(http.StatusNotFound), http.StatusNotFound)
				ctx.PrintError("asdd", http.StatusNotFound)
				return
			}
		} else {
			//ctx.PrintError(http.StatusText(http.StatusNotFound), http.StatusNotFound)
			ctx.PrintError("da", http.StatusNotFound)
			return
		}

		// get collection id if it exists
		if (i + 1) < len(parts) {
			pair.HasKey = true
			pair.CollectionId = parts[i+1]

			// convert collectionId to *datastore.Key
			pair.CollectionKey, err = datastore.DecodeKey(pair.CollectionId)
			if err != nil {

				// if it has previous key, context should have namespace defined
				/*if i > 0 {
					pair.CollectionKey.Namespace()
				}*/

				pair.CollectionKey = datastore.NewKey(ctx, pair.CollectionKind.Name(), pair.CollectionId, 0, nil)
			}

			// compare this key to the previous one
			/*if i > 0 {
				pair.CollectionKey.Namespace()
			}*/
		}

		pairs = append(pairs, pair)
		lastPair = pair
	}

	// authenticate using r.Method() and also check if user has access to specified namespace ->
	// namespace can get from second to last pair or from key that has namespace specified

	// ALSO! must check if hierarchy of keys is OK
	/*
	project is a group ... create a new project
	we take that project group key and use it for creating an object ... that object key contains namespace information of the project group -----
	when accessing said object the namespace must match project key
	this can go on for unlimited times
	 */

	var ok bool
	if ctx, ok = ctx.WithSession(); !ok {
		return
	}

	if lastPair.HasKey {
		switch r.Method {
		case http.MethodGet:
			if ok := ctx.HasRole(rules.ReadOnly, rules.ReadWrite, rules.FullControl); ok {
				doc, err := lastPair.CollectionKind.Doc(ctx, lastPair.CollectionKey).Get()
				if err != nil {
					ctx.PrintError(err.Error(), http.StatusInternalServerError)
					return
				}
				ctx.PrintJSON(lastPair.CollectionKind.Data(doc), http.StatusOK)
			} else {
				ctx.PrintError(http.StatusText(http.StatusForbidden), http.StatusForbidden)
			}

		case http.MethodPut:
		case http.MethodPatch:
		case http.MethodDelete:
		case http.MethodPost:
			// nothing
		default:
			return
		}
	} else {
		switch r.Method {
		case http.MethodGet:
			ctx.PrintJSON(pairs, 200)
			// QUERY
		case http.MethodPost:
			if ok := ctx.HasRole(rules.ReadWrite, rules.FullControl); ok {
				doc, err := lastPair.CollectionKind.Doc(ctx, nil).Add(ctx.Body())
				if err != nil {
					ctx.PrintError(err.Error(), http.StatusInternalServerError)
					return
				}
				ctx.PrintJSON(lastPair.CollectionKind.Data(doc), http.StatusOK)
			} else {
				ctx.PrintError(http.StatusText(http.StatusForbidden), http.StatusForbidden)
			}

		case http.MethodPut, http.MethodPatch, http.MethodDelete:
			// nothing
		default:
			return
		}
	}
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
