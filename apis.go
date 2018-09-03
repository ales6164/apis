package apis

import (
	"github.com/ales6164/apis/module"
	"github.com/ales6164/client"
	"github.com/gorilla/mux"
	"net/http"
)

type Apis struct {
	*mux.Router
	options             *Options
	allowedTranslations map[string]bool
	modules             []module.Module
	roles               map[string][]string
	client.RoleProvider
}

type Options struct {
	AppName                string
	StorageBucket          string // required for file upload and media library
	DefaultLanguage        string // fallback language
	HasTranslationsFor     []string
	AuthorizedOrigins      []string // not implemented
	AuthorizedRedirectURIs []string // not implemented
	RequireTrackingID      bool     // not implemented
}

func New(opt *Options) (*Apis, error) {
	a := &Apis{
		Router:              mux.NewRouter(),
		options:             opt,
		allowedTranslations: map[string]bool{},
		roles:               map[string][]string{},
	}

	a.Router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if origin := r.Header.Get("Origin"); origin != "" {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
				w.Header().Set("Access-Control-Allow-Headers",
					"Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, Cache-Control, "+
						"X-Requested-With")
			}
			if r.Method == "OPTIONS" {
				return
			}
			next.ServeHTTP(w, r)
		})
	})

	// languages
	for _, l := range opt.HasTranslationsFor {
		a.allowedTranslations[l] = true
	}

	return a, nil
}

func (a *Apis) RegisterRole(name string, scopes ...string) {
	a.roles[name] = append(a.roles[name], scopes...)
}

func (a *Apis) Roles() map[string][]string {
	return a.roles
}

/*func (a *Apis) Module(module module.Module) {
	if err := module.Init(); err != nil {
		panic(module.Name() + ": " + err.Error())
	}
	a.modules = append(a.modules, module)
}

func (a *Apis) Handler() http.Handler {
	// modules
	for _, m := range a.modules {
		modulePath := path.Join("/", "module", m.Name())
		a.PathPrefix(modulePath).Handler(m.Router(modulePath))
	}
	return &Server{a.Router}
}
*/
