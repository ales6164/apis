package apis

import (
	"encoding/json"
	"github.com/ales6164/client"
	"github.com/gorilla/mux"
	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"net/http"
	"reflect"
	"strings"
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

	route := &Route{
		pathPrefix: path,
		router:     a.router.PathPrefix(path).Subrouter(),
	}

	/*
	TODO:
	1. Use current handlers
	2. Upgrade to use REST API standard flow
	3. Add scopes

	 */

	// query
	route.router.HandleFunc("", func(writer http.ResponseWriter, request *http.Request) {
		//writer.Write([]byte(r.pathPrefix))
		json.NewEncoder(writer).Encode(kind.fields)
	})

	// get
	route.router.HandleFunc("/{id}", func(writer http.ResponseWriter, request *http.Request) {
		writer.Write([]byte(mux.Vars(request)["id"]))
	})

	route.router.HandleFunc(`/{id}/{rest:[a-zA-Z0-9=\-\/]+}`, func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		ctx := appengine.NewContext(r)

		key, err := datastore.DecodeKey(vars["id"])
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		holder := kind.NewHolder()

		err = datastore.Get(ctx, key, holder)
		if err != nil {
			if err == datastore.ErrNoSuchEntity {
				http.NotFound(w, r)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		value := reflect.ValueOf(holder)

		for _, name := range strings.Split(vars["rest"], "/") {
			if value.Kind() == reflect.Ptr {
				value = value.Elem().FieldByName(name)
			} else {
				value = value.FieldByName(name)
			}
		}

		json.NewEncoder(w).Encode(value.Interface())
	})

	return route
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
