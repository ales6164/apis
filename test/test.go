package test

import (
	"github.com/ales6164/apis"
	"google.golang.org/appengine/datastore"
	"net/http"
)

func init() {
	// auth
	/*signingKey, err := ioutil.ReadFile("key.txt")
	if err != nil {
		panic(err)
	}
	_ = middleware.AuthMiddleware(signingKey)*/

	// custom auth middleware + login/registration/session library
	// user profile? private entities with scope access? like projects and user profile

	// kind provider
	kindProvider := apis.NewKindProvider()

	var parentKind = apis.NewKind("parent", Parent{}, kindProvider)
	var childKind = apis.NewKind("child", Child{}, kindProvider)

	api := apis.New(&apis.Options{
	})

	api.HandleFunc("/hi", func(writer http.ResponseWriter, request *http.Request) {
		writer.Write([]byte("Hello"))
	})

	api.Handle(`/children`, childKind)
	api.Handle(`/children/{key}`, childKind)
	api.Handle(`/children/{key}/{path:[a-zA-Z0-9=\-\/]+}`, childKind)
	api.Handle(`/parents`, parentKind)
	api.Handle(`/parents/{key}`, parentKind)
	api.Handle(`/parents/{key}/{path:[a-zA-Z0-9=\-\/]+}`, parentKind)

	http.Handle("/", api.Handler())
}

// TODO: check scope on every handler operation (get, put, delete, post) - best to put checks inside handler functions

type Parent struct {
	Id        *datastore.Key   `datastore:"-" auto:"id" json:"id,omitempty"`
	Name      string           `json:"name"`
	ChildRef  *datastore.Key   `json:"childRef"`
	ChildRefs []*datastore.Key `json:"childRefs"`
	Child     Child            `json:"child"`
	Children  []Child          `json:"children"`
}

type Child struct {
	Id     *datastore.Key `datastore:"-" auto:"id" json:"id,omitempty"`
	Name   string         `json:"name"`
	Reason string         `json:"reason"`
}
