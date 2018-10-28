package test

import (
	"github.com/ales6164/apis"
	"net/http"
	"google.golang.org/appengine/datastore"
)

func init() {

	var parentKind = apis.NewKind("parent", Parent{})
	var objectKind = apis.NewKind("object", Object{})

	api := apis.New(parentKind, objectKind)
	api.Handle("/objects", objectKind)
	api.Handle("/parents", parentKind)

	http.Handle("/", api.Handler())
}

type Parent struct {
	Id    *datastore.Key `auto:"id" json:"id"`
	Child *datastore.Key `json:"child"`
}

type Object struct {
	Name string `json:"name"`
}
