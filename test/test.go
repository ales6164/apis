package test

import (
	"github.com/ales6164/apis"
	"google.golang.org/appengine/datastore"
	"net/http"
)

func init() {

	var parentKind = apis.NewKind("parent", Parent{})
	var objectKind = apis.NewKind("object", Object{})

	api := apis.New(&apis.Options{
		NestedKinds: []*apis.Kind{parentKind, objectKind},
	})
	api.Handle("/objects", objectKind)
	api.Handle("/parents", parentKind)

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
