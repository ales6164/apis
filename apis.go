package apis

import (
	"github.com/gorilla/mux"
	"net/http"
)

type Apis struct {
	router *mux.Router
	*Options
}

type Options struct {
	Roles map[string][]string
}

func New(options *Options) *Apis {
	a := &Apis{
		Options: options,
		router:  mux.NewRouter(),
	}

	if a.Roles == nil {
		a.Roles = map[string][]string{}
	}

	return a
}

func (a *Apis) SetRole(name string, scopes ...string) {
	a.Roles[name] = append(a.Roles[name], scopes...)
}

func (a *Apis) Handle(path string, h http.Handler) {
	a.router.Handle(path, h)
	a.router.Handle(path+`/{key}/{path:[a-zA-Z0-9=\-\/]+}`, h)
	a.router.Handle(path+`/{path:[a-zA-Z0-9=\-\/]+}`, h)
}

func (a *Apis) HandleFunc(path string, f func(http.ResponseWriter, *http.Request)) {
	a.router.HandleFunc(path, f)
	a.router.HandleFunc(path+`/{key}/{path:[a-zA-Z0-9=\-\/]+}`, f)
	a.router.HandleFunc(path+`/{path:[a-zA-Z0-9=\-\/]+}`, f)
}

func (a *Apis) Handler() http.Handler {
	return &Server{a.router}
}
