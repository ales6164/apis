package test

import (
	"github.com/ales6164/apis"
	"google.golang.org/appengine/datastore"
	"net/http"
	"reflect"
)

var ObjectKind = apis.NewKind(reflect.TypeOf(Object{}))

func init() {
	api, _ := apis.New(&apis.Options{PrivateKeyPath: "key.txt"})

	api.Handle("/objects", ObjectKind).Methods(http.MethodPost)
	api.Handle("/objects/{id}", ObjectKind).Methods(http.MethodGet)

	http.Handle("/", api)
}

type Object struct {
	Id   *datastore.Key `datastore:"-" apis:"id"`
	Name string
}
