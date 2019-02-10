package apis

import (
	"github.com/ales6164/apis/iam"
	"github.com/ales6164/apis/kind"
	gctx "github.com/gorilla/context"
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
	IAM   *iam.IAM
	Rules *Rules
}

type Match map[kind.Kind]*Rules
type Roles []string

type Rules struct {
	AccessControl bool
	Permissions   Permissions
	Match         Match `json:"-"`
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

	if a.IAM != nil {
		a.hasAuth = true

		// renew token
		a.router.Handle("/auth/renew", authMiddleware(a, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := a.IAM.NewContext(w, r)

			session, err := a.IAM.RenewSession(ctx)
			if err != nil {
				ctx.PrintError(err.Error(), http.StatusInternalServerError)
				return
			}

			a.IAM.PrintResponse(session)
		}))).Methods(http.MethodOptions, http.MethodPost)

		// confirm email
		a.router.Handle("/auth/confirm/{code}", authMiddleware(a, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
			a.router.Handle(`/auth/`+p.Name()+`/{path:[a-zA-Z0-9=\-\/]+}`, authMiddleware(a, p))
		}
	}

	return a
}

func authMiddleware(a *Apis, h http.Handler) http.Handler {
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
					"X-Requested-With, X-Include-Meta")
		}

		if r.Method == http.MethodOptions {
			return
		}

		h.ServeHTTP(w, r)
	}))
}

func (a *Apis) HandleKind(k kind.Kind) {
	a.kinds[k.Name()] = k
	//a.handleKind(k.Name(), k)
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

func (a *Apis) Handler() *mux.Router {
	if a.hasAuth {
		a.router.Handle(`/{path:[a-zA-Z0-9=\-\_\/]+}`, authMiddleware(a, a))
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
