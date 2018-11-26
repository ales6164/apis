package test

import (
	"github.com/ales6164/apis"
	"github.com/ales6164/apis/middleware"
	"google.golang.org/appengine/datastore"
	"io/ioutil"
	"net/http"
)

func init() {
	// auth
	signingKey, err := ioutil.ReadFile("key.txt")
	if err != nil {
		panic(err)
	}
	_ = middleware.AuthMiddleware(signingKey)

	// kind provider
	kindProvider := apis.NewKindProvider()

	var parentKind = apis.NewKind("parent", Parent{}, kindProvider)
	var objectKind = apis.NewKind("object", Object{}, kindProvider)

	api := apis.New(&apis.Options{
	})

	api.HandleKind("/objects", objectKind)
	api.HandleKind("/parents", parentKind)

	http.Handle("/", api.Handler())
}

// TODO: check scope on every handler operation (get, put, delete, post) - best to put checks inside handler functions

type Parent struct {
	Id     *datastore.Key `datastore:"-" auto:"id" json:"id,omitempty"`
	Child  *datastore.Key `json:"child"`
	Object Object         `json:"object"`
}

type Object struct {
	Id   *datastore.Key `datastore:"-" auto:"id" json:"id,omitempty"`
	Name string         `json:"name"`
}
