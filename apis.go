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
	"github.com/ales6164/apis/providers"
)

type Apis struct {
	permissions
	options             *Options
	routes              []*Route
	router              *mux.Router
	middleware          *middleware.JWTMiddleware
	privateKey          []byte
	allowedTranslations map[string]bool
	kinds               map[string]*kind.Kind
	modules             []module.Module
}

type Options struct {
	Permissions
	AppName                string
	StorageBucket          string // required for file upload and media library
	PrivateKeyPath         string // for password hashing
	IdentityProviders      []providers.IdentityProvider
	DefaultRole            Role   // default role for registered account
	DefaultLanguage        string // fallback language
	HasTranslationsFor     []string
	AuthorizedOrigins      []string // not implemented
	AuthorizedRedirectURIs []string // not implemented
	RequireTrackingID      bool     // not implemented
}

func New(opt *Options) (*Apis, error) {
	a := &Apis{
		router:              mux.NewRouter(),
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
	p := "/" + path.Join("kind", kind.Name)
	m := []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete}
	r := &Route{
		kind:    kind,
		a:       a,
		path:    p,
		methods: m,
	}
	kind.AddRouteSettings(p, m)
	a.kinds[kind.Name] = kind
	a.routes = append(a.routes, r)
	return r
}

// deprecated
func (a *Apis) HandleWPath(p string, kind *kind.Kind) *Route {
	m := []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete}
	r := &Route{
		kind:    kind,
		a:       a,
		path:    p,
		methods: m,
	}
	kind.AddRouteSettings(p, m)
	a.kinds[kind.Name] = kind
	a.routes = append(a.routes, r)
	return r
}

func (a *Apis) Router() *mux.Router {
	return a.router
}

func (a *Apis) Module(module module.Module) {
	if err := module.Init(); err != nil {
		panic(module.Name() + ": " + err.Error())
	}
	a.modules = append(a.modules, module)
}

func (a *Apis) Handler(pathPrefix string) http.Handler {
	r := a.router.PathPrefix(pathPrefix).Subrouter()

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
				r.Handle(route.path+"/{ancestor}", a.middleware.Handler(route.postHandler())).Methods(http.MethodPost)
				r.Handle(route.path, a.middleware.Handler(route.postHandler())).Methods(http.MethodPost)
				if hasLang {
					r.Handle(lang+route.path+"/{ancestor}", a.middleware.Handler(route.postHandler())).Methods(http.MethodPost)
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

	initInfo(a, r)
	initAuth(a, r)
	initUser(a, r)
	initMedia(a, r)
	initChat(a, r)
	initSearch(a, r)

	// modules
	for _, m := range a.modules {
		modulePath := path.Join(pathPrefix, "module", m.Name())
		r.PathPrefix(modulePath).Handler(m.Router(modulePath))
	}

	return &Server{a.router}
}
