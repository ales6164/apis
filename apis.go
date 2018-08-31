package apis

import (
	"github.com/ales6164/apis/errors"
	"github.com/ales6164/apis/middleware"
	"github.com/ales6164/apis/module"
	"github.com/gorilla/context"
	"github.com/gorilla/mux"
	"io/ioutil"
	"net/http"
)

type Apis struct {
	*mux.Router
	options *Options

	middleware          *middleware.JWTMiddleware
	signingKey          []byte
	allowedTranslations map[string]bool
	modules             []module.Module
}

type Options struct {
	Roles                  map[string][]string
	AppName                string
	StorageBucket          string // required for file upload and media library
	PrivateKeyPath         string // for password hashing
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

	// read private key
	var err error
	a.signingKey, err = ioutil.ReadFile(opt.PrivateKeyPath)
	if err != nil {
		return a, err
	}

	// set auth middleware
	a.middleware = middleware.AuthMiddleware(a.signingKey)

	// languages
	for _, l := range opt.HasTranslationsFor {
		a.allowedTranslations[l] = true
	}

	return a, nil
}

func (a *Apis) SigningKey() []byte {
	return a.signingKey
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
func (a *Apis) AuthMiddleware(h http.Handler, scopes ...string) http.Handler {
	return a.middleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := NewContext(r)
		var ok bool
		if !ctx.IsAuthenticated {
			ctx.PrintError(w, errors.ErrUnathorized)
			return
		}
		if ok = ctx.HasScope(scopes...); !ok {
			ctx.PrintError(w, errors.ErrForbidden)
			return
		}
		context.Set(r, "context", ctx)
		h.ServeHTTP(w, r)
	}))
}
