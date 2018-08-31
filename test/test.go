package test

import (
	"github.com/ales6164/apis"
	"github.com/ales6164/apis/providers"
	"google.golang.org/appengine/datastore"
	"net/http"
	"reflect"
)

var ObjectKind = apis.NewKind(reflect.TypeOf(Object{}))

func init() {
	api, _ := apis.New(&apis.Options{
		PrivateKeyPath: "key.txt",
		IdentityProviders: []providers.IdentityProvider{
			providers.WithEmailPasswordProvider(&providers.Options{Scopes: []string{ObjectKind.ScopeFullControl}}),
		},
	})

	//api.Handle("/signin", emailPasswordProvider.LoginHandler()).Methods(http.MethodPost)

	api.Handle("/objects", ObjectKind).Methods(http.MethodPost)
	api.Handle("/objects/{id}", ObjectKind).Methods(http.MethodGet)

	http.Handle("/", api)
}

type Object struct {
	Id   *datastore.Key `datastore:"-" apis:"id"`
	Name string
}
