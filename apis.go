package apis

import (
	"net/http"

	"github.com/gorilla/mux"
	"io/ioutil"
	"github.com/ales6164/apis/middleware"
	"github.com/ales6164/apis/kind"
)

type Apis struct {
	options *Options
	routes  []*Route


	middleware          *middleware.JWTMiddleware
	privateKey          []byte
	permissions
	allowedTranslations map[string]bool

	OnUserSignUp func(ctx Context, user User)
	//OnUserSignIn func(ctx context.Context, user User)
	OnUserVerified func(ctx Context, user User)
}

type Options struct {
	AppName                  string
	PrivateKeyPath           string // for password hashing
	AuthorizedOrigins        []string
	AuthorizedRedirectURIs   []string
	AllowUserRegistration    bool
	DefaultRole              Role
	RequireEmailConfirmation bool
	HasTranslationsFor       []string
	DefaultLanguage          string
	/*UserProfileKind          *kind.Kind*/
	RequireTrackingID bool // todo:generated from pages - track users - stored as session cookie
	Permissions
}

func New(opt *Options) (*Apis, error) {
	a := &Apis{
		options:             opt,
		allowedTranslations: map[string]bool{},
		kinds:               map[*kind.Kind]*kind.Kind{},
	}

	// read private key
	var err error
	a.privateKey, err = ioutil.ReadFile(opt.PrivateKeyPath)
	if err != nil {
		return a, err
	}

	// parse permissions
	a.permissions, err = a.options.Permissions.parse()
	if err != nil {
		return a, err
	}

	// set auth middleware
	a.middleware = middleware.AuthMiddleware(a.privateKey)

	// languages
	for _, l := range opt.HasTranslationsFor {
		a.allowedTranslations[l] = true
	}

	return a, nil
}

func (a *Apis) Handle(p string, kind *kind.Kind) *Route {
	r := &Route{
		kind:    kind,
		a:       a,
		path:    p,
		methods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete},
	}

	a.routes = append(a.routes, r)
	return r
}

func (a *Apis) Handler(pathPrefix string) http.Handler {
	r := mux.NewRouter().PathPrefix(pathPrefix).Subrouter()

	for _, route := range a.routes {
		for _, method := range route.methods {
			switch method {
			case http.MethodGet:
				r.Handle(route.path, a.middleware.Handler(route.getHandler())).Methods(http.MethodGet)
			case http.MethodPost:
				r.Handle(route.path, a.middleware.Handler(route.postHandler())).Methods(http.MethodPost)
			case http.MethodPut:
				r.Handle(route.path, a.middleware.Handler(route.putHandler())).Methods(http.MethodPut)
			case http.MethodDelete:
				r.Handle(route.path, a.middleware.Handler(route.deleteHandler())).Methods(http.MethodDelete)
			}
		}
	}

	authRoute := &Route{
		a:       a,
		methods: []string{},
	}
	r.Handle("/auth/login", loginHandler(authRoute)).Methods(http.MethodPost)
	if a.options.AllowUserRegistration {
		r.Handle("/auth/register", registrationHandler(authRoute, a.options.DefaultRole)).Methods(http.MethodPost)
	}
	r.Handle("/auth/confirm", a.middleware.Handler(confirmEmailHandler(authRoute)))
	r.Handle("/auth/password", a.middleware.Handler(changePasswordHandler(authRoute))).Methods(http.MethodPost)
	r.Handle("/user", a.middleware.Handler(getUserHandler(authRoute))).Methods(http.MethodGet)

	return &Server{r}
}
