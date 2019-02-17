package apis

import (
	"encoding/json"
	"github.com/ales6164/apis/collection"
	"github.com/ales6164/apis/iam"
	gctx "github.com/gorilla/context"
	"github.com/gorilla/mux"
	"google.golang.org/appengine/datastore"
	"net/http"
)

const (
	BREAKING_VERSION = 1
	ADMIN_HOST       = "api-v2-dot-admin-si.appspot.com"
)

type Apis struct {
	//router *mux.Router
	*Options
	hasAuth bool
	kinds   map[string]collection.Kind
	//roles      map[string][]string
	//http.Handler
	router *mux.Router
}

type Options struct {
	IAM   *iam.IAM
	Rules *Rules
}

type Match map[collection.Kind]*Rules

type Rules struct {
	AccessControl bool
	Permissions   Permissions
	Match         Match `json:"-"`
}

// map[iam.Role][]iam.Scope
type Permissions map[string][]string

func New(options *Options) *Apis {
	if options == nil {
		options = &Options{}
	}

	a := &Apis{
		Options: options,
		/*Router:  mux.NewRouter(),*/
		kinds: map[string]collection.Kind{},
	}

	a.router = mux.NewRouter()

	if a.IAM != nil {
		a.hasAuth = true

		// renew token
		a.router.Handle("/auth/renew", a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := a.IAM.NewContext(w, r)

			session, err := a.IAM.RenewSession(ctx)
			if err != nil {
				ctx.PrintError(err.Error(), http.StatusInternalServerError)
				return
			}

			session, err = session.LoadIdentity(ctx)
			if err != nil {
				ctx.PrintError(err.Error(), http.StatusInternalServerError)
				return
			}

			a.IAM.PrintResponse(ctx, session)
		}))).Methods(http.MethodOptions, http.MethodPost)

		// confirm email
		a.router.Handle("/auth/confirm/{code}", a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := a.IAM.NewContext(w, r)

			_, err := a.IAM.ConfirmEmail(ctx, mux.Vars(r)["code"])
			if err != nil {
				ctx.PrintError(err.Error(), http.StatusInternalServerError)
				return
			}

			// TODO: redirect user

			ctx.PrintStatus(http.StatusText(http.StatusOK), http.StatusOK)
		}))).Methods(http.MethodGet)

		for _, p := range a.IAM.GetProviders() {
			a.router.Handle(`/auth/`+p.Name()+`/{path:[a-zA-Z0-9=\-\_\/]+}`, a.Middleware(p))
		}
	}

	a.router.HandleFunc("/_info", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Appengine-Inbound-Appid") != "admin-si" {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}

		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"version": BREAKING_VERSION,
		})
	})

	return a
}



func (a *Apis) Middleware(h http.Handler) http.Handler {
	return http.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if origin := r.Header.Get("Origin"); origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
			w.Header().Set("Access-Control-Allow-Headers",
				"Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, Cache-Control, "+
					"X-Requested-With, X-Include-Meta, X-Resolve-Meta-Ref")
		}

		if r.Method == http.MethodOptions {
			return
		}

		tkn, _ := a.IAM.Middleware().CheckJWT(w, r)
		gctx.Set(r, "token", tkn)

		h.ServeHTTP(w, r)
	}))
}

func defaultMiddleware(h http.Handler) http.Handler {
	return http.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if origin := r.Header.Get("Origin"); origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
			w.Header().Set("Access-Control-Allow-Headers",
				"Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, Cache-Control, "+
					"X-Requested-With, X-Include-Meta, X-Resolve-Meta-Ref")
		}

		if r.Method == http.MethodOptions {
			return
		}

		h.ServeHTTP(w, r)
	}))
}

func (a *Apis) HandleKind(k collection.Kind) {
	a.kinds[k.Name()] = k
	//a.handleKind(k.Name(), k)
}

type PathPair struct {
	CollectionName string          `json:"collectionName"`
	CollectionId   string          `json:"collectionId"`
	HasKey         bool            `json:"hasKey"`
	CollectionKey  *datastore.Key  `json:"collectionKey"`
	CollectionKind collection.Kind `json:"collectionKind"`
	IsGroup        bool            `json:"isGroup"`
	GroupKey       *datastore.Key  `json:"groupKey"`
	Rules          Rules           `json:"rules"`
}

func (a *Apis) Handler() *mux.Router {
	if a.hasAuth {
		a.router.Handle(`/{path:[a-zA-Z0-9=\-\_\/]+}`, a.Middleware(a))
	} else {
		a.router.Handle(`/{path:[a-zA-Z0-9=\-\_\/]+}`, defaultMiddleware(a))
	}
	return a.router
}

func (a *Apis) Handle(path string, handler http.Handler) *mux.Route {
	return a.router.Handle(path, handler)
}

func (a *Apis) HandleFunc(path string, f func(w http.ResponseWriter, r *http.Request)) *mux.Route {
	return a.router.HandleFunc(path, f)
}
