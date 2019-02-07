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

		// confirm email
		a.router.Handle("/auth/confirm/{code}", Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := a.NewContext(w, r)

			_, err := a.Auth.ConfirmEmail(ctx, mux.Vars(r)["code"])
			if err != nil {
				ctx.PrintError(err.Error(), http.StatusInternalServerError)
				return
			}

			// TODO: redirect user

			ctx.PrintStatus(http.StatusText(http.StatusOK), http.StatusOK)
		}))).Methods(http.MethodGet)

		for _, p := range a.Auth.providers {
			a.router.Handle(`/auth/`+p.Name()+`/{path:[a-zA-Z0-9=\-\/]+}`, Middleware(p))
		}
	}

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
	a.router.Handle(`/{path:[a-zA-Z0-9=\-\_\/]+}`, Middleware(a))
	return a.router
}

func (a *Apis) Handle(path string, handler http.Handler) *mux.Route {
	return a.router.Handle(path, handler)
}

func (a *Apis) HandleFunc(path string, f func(w http.ResponseWriter, r *http.Request)) *mux.Route {
	return a.router.HandleFunc(path, f)
}
