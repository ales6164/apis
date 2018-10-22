package apis

import (
	"encoding/json"
	"github.com/ales6164/client"
	"github.com/gorilla/mux"
	"net/http"
)

type Apis struct {
	router *mux.Router
	kinds  map[string]*Kind
	roles  map[string][]string
	client.RoleProvider
}

type Route struct {
	pathPrefix string
	router     *mux.Router
}

func New() *Apis {
	a := &Apis{
		router: mux.NewRouter(),
		roles:  map[string][]string{},
		kinds:  map[string]*Kind{},
	}

	return a
}

func (a *Apis) Handle(path string, kind *Kind) *Route {
	a.kinds[kind.name] = kind

	r := &Route{
		pathPrefix: path,
		router:     a.router.PathPrefix(path).Subrouter(),
	}

	r.router.HandleFunc("", func(writer http.ResponseWriter, request *http.Request) {
		//writer.Write([]byte(r.pathPrefix))
		json.NewEncoder(writer).Encode(kind.fields)
	})

	r.router.HandleFunc("/{id}", func(writer http.ResponseWriter, request *http.Request) {
		writer.Write([]byte(mux.Vars(request)["id"]))
	})

	return r
}

func (a *Apis) Handler() http.Handler {
	return &Server{a.router}
}

func (a *Apis) RegisterRole(name string, scopes ...string) {
	a.roles[name] = append(a.roles[name], scopes...)
}

func (a *Apis) Roles() map[string][]string {
	return a.roles
}
