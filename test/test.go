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
	var objectKind = apis.NewKind("child", Child{}, kindProvider)

	api := apis.New(&apis.Options{
	})

	api.HandleFunc("/hi", func(writer http.ResponseWriter, request *http.Request) {
		writer.Write([]byte("Hello"))
	})

	objectKind.Attach(api, "/children") // auth middleware?
	parentKind.Attach(api, "/parents")

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
	Id   *datastore.Key `datastore:"-" auto:"id" json:"id,omitempty"`
	Name string         `json:"name"`
}
