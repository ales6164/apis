package apis

import (
	"github.com/gorilla/mux"
	"net/http"
)

type Apis struct {
	*mux.Router
	*Options
	collectionRouter *mux.Router
	kinds            map[string]*Kind
}

type Options struct {
	Roles map[string][]string
}

func New(options *Options) *Apis {
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

func (a *Apis) SetRole(name string, scopes ...string) {
	a.Roles[name] = append(a.Roles[name], scopes...)
}

func (a *Apis) RegisterKind(k *Kind) {
	a.kinds[k.Path] = k

	a.HandleFunc(joinPath(k.Path), func(w http.ResponseWriter, r *http.Request) {
		ctx := NewContext(w, r)
		if ok := ctx.HasScope(k.ScopeReadOnly, k.ScopeReadWrite, k.ScopeFullControl); ok {
			k.QueryHandler(ctx)
		} else {
			ctx.PrintError(http.StatusText(http.StatusForbidden), http.StatusForbidden)
		}
	}).Methods(http.MethodGet)

	/*a.Handle(path, h)
	a.Handle(path+`/{key}`, h)
	a.Handle(path+`/{key}/{path:[a-zA-Z0-9=\-\/]+}`, h)*/

	// TODO: HANDLE collection permission checking right inside handlefunc
	/*a.collectionRouter.HandleFunc(joinPath(k.Path), func(w http.ResponseWriter, r *http.Request) {
		ctx := NewContext(w, r)
		k.QueryHandler(ctx)
	}).Methods(http.MethodGet)*/
	/*a.Handle("{collection}"+path, h)
	a.Handle("{collection}"+path+`/{key}`, h)
	a.Handle("{collection}"+path+`/{key}/{path:[a-zA-Z0-9=\-\/]+}`, h)*/
}
