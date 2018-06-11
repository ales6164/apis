package apis

import (
	"io/ioutil"
	"github.com/ales6164/apis/middleware"
	"net/http"
	"github.com/ales6164/apis/kind"
	"github.com/gorilla/mux"
	"path"
	"strings"
	"github.com/ales6164/apis/module"
)

type Apis struct {
	options *Options
	routes  []*Route

	middleware          *middleware.JWTMiddleware
	privateKey          []byte
	permissions
	allowedTranslations map[string]bool
	kinds               map[string]*kind.Kind
	modules             []module.Module

	OnUserSignUp func(ctx Context, user User, token string)
	//OnUserSignIn func(ctx context.Context, user User)
	OnUserVerified func(ctx Context, user User, token string)
}

type Options struct {
	AppName       string
	StorageBucket string // required for file upload and media library
	/*Chat                     ChatOptions*/ // required for built in chat service
	PrivateKeyPath           string          // for password hashing
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
		kinds:               map[string]*kind.Kind{},
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

func (a *Apis) Handle(kind *kind.Kind) *Route {
	r := &Route{
		kind:    kind,
		a:       a,
		path:    "/" + path.Join("kind", kind.Name),
		methods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete},
	}
	a.kinds[kind.Name] = kind
	a.routes = append(a.routes, r)
	return r
}

func (a *Apis) Module(module module.Module) {
	if err := module.Init(); err != nil {
		panic(module.Name() + ": " + err.Error())
	}
	a.modules = append(a.modules, module)
}

// /kind/:order?name=some-key-string-id GET - query
// /kind/:order/:id GET - get single
// /kind/:order/:id PUT - put single
// /kind/:order/:id
/// ...
// /search/order GET - search

func (a *Apis) Handler(pathPrefix string) http.Handler {
	r := mux.NewRouter().PathPrefix(pathPrefix).Subrouter()

	// {sort:(?:asc|desc|new)}
	// lang path
	var lang string
	var hasLang bool
	if len(a.options.HasTranslationsFor) > 0 {
		lang = "/{lang:(?:" + strings.Join(a.options.HasTranslationsFor, "|") + ")}" // /{lang:(?:sl|en|gb)}
		hasLang = true
	}

	for _, route := range a.routes {
		for _, method := range route.methods {
			switch method {
			case http.MethodGet:
				r.Handle(route.path+"/{id}", a.middleware.Handler(route.getHandler())).Methods(http.MethodGet)
				r.Handle(route.path, a.middleware.Handler(route.queryHandler())).Methods(http.MethodGet)
				if hasLang {
					r.Handle(lang+route.path+"/{id}", a.middleware.Handler(route.getHandler())).Methods(http.MethodGet)
					r.Handle(lang+route.path, a.middleware.Handler(route.queryHandler())).Methods(http.MethodGet)
				}
			case http.MethodPost:
				r.Handle(route.path, a.middleware.Handler(route.postHandler())).Methods(http.MethodPost)
				if hasLang {
					r.Handle(lang+route.path, a.middleware.Handler(route.postHandler())).Methods(http.MethodPost)
				}
			case http.MethodPut:
				r.Handle(route.path+"/{id}", a.middleware.Handler(route.putHandler())).Methods(http.MethodPut)
				if hasLang {
					r.Handle(lang+route.path+"/{id}", a.middleware.Handler(route.putHandler())).Methods(http.MethodPut)
				}
			case http.MethodDelete:
				r.Handle(route.path+"/{id}", a.middleware.Handler(route.deleteHandler())).Methods(http.MethodDelete)
				if hasLang {
					r.Handle(lang+route.path+"/{id}", a.middleware.Handler(route.deleteHandler())).Methods(http.MethodDelete)
				}
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

	// USER
	r.Handle("/users", a.middleware.Handler(getPublicUsersHandler(authRoute))).Methods(http.MethodGet)
	r.Handle("/user", a.middleware.Handler(getUserHandler(authRoute))).Methods(http.MethodGet)
	r.Handle("/user", a.middleware.Handler(updateUserHandler(authRoute))).Methods(http.MethodPut)

	r.Handle("/apis", a.middleware.Handler(infoHandler(authRoute))).Methods(http.MethodGet)

	// SEARCH
	r.Handle("/search/{kind}", a.middleware.Handler(a.searchHandler())).Methods(http.MethodGet)

	initMedia(a, r)
	initAgreement(a, r)
	initChat(a, r)

	// modules
	for _, m := range a.modules {
		modulePath := path.Join(pathPrefix, "module", m.Name())
		r.PathPrefix(modulePath).Handler(m.Router(modulePath))
	}

	return &Server{r}
}
