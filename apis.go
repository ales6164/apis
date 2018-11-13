package apis

import (
	"github.com/gorilla/mux"
	"net/http"
	"reflect"
)

type Apis struct {
	router *mux.Router
	types  map[reflect.Type]*Kind
	kinds  map[string]*Kind
	routes map[string]*Route
	*Options
}

type Options struct {
	NestedKinds []*Kind
	Roles       map[string][]string
}

func New(options *Options) *Apis {
	a := &Apis{
		Options: options,
		router:  mux.NewRouter(),
		routes:  map[string]*Route{},
		types:   map[reflect.Type]*Kind{},
		kinds:   map[string]*Kind{},
	}

	if a.Roles == nil {
		a.Roles = map[string][]string{}
	}

	for _, k := range options.NestedKinds {
		a.types[k.t] = k
		a.kinds[k.name] = k
	}

	return a
}

func (a *Apis) SetRole(name string, scopes ...string) {
	a.Roles[name] = append(a.Roles[name], scopes...)
}

func (a *Apis) Handle(path string, kind *Kind) *Route {
	a.types[kind.t] = kind
	a.kinds[kind.name] = kind
	a.routes[path] = &Route{
		a:          a,
		pathPrefix: path,
		kind:       kind,
		router:     a.router.PathPrefix(path).Subrouter(),
	}
	return a.routes[path]
}

func (a *Apis) Handler() http.Handler {
	for _, r := range a.routes {
		r.init()
	}
	return &Server{a.router}
}
