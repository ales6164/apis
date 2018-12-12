package test

import (
	"github.com/ales6164/apis"
	"github.com/ales6164/apis/kind"
	"github.com/ales6164/auth"
	"github.com/dgrijalva/jwt-go"
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

	// custom auth middleware + login/registration/session library
	// user profile? private entities with scope access? like projects and user profile

	a := auth.New(&auth.Options{
		SigningKey:          signingKey,
		Extractors:          []auth.TokenExtractor{auth.FromAuthHeader},
		CredentialsOptional: false,
		SigningMethod:       jwt.SigningMethodHS256,
	})
	middleware := a.Middleware()

	// kind provider
	kindProvider := kind.NewKindProvider()

	var parentKind = kind.New(&kind.Options{
		Name:         "parent",
		Type:         Parent{},
		KindProvider: kindProvider,
	})
	var childKind = kind.New(&kind.Options{
		Name:         "child",
		Type:         Child{},
		KindProvider: kindProvider,
	})
	var projectKind = kind.New(&kind.Options{
		Name:         "project",
		Type:         Project{},
		KindProvider: kindProvider,
		IsCollection: true, // creates /collections/{key} - can change roles, rules       -- OR -- could change how /projects/{key} ... or both
	})

	api := apis.New(&apis.Options{
	})

	api.HandleFunc("/hi", func(writer http.ResponseWriter, request *http.Request) {
		writer.Write([]byte("Hello"))
	})

	api.Handle(`/children`, middleware.Handler(childKind))
	api.Handle(`/children/{key}`, childKind)
	api.Handle(`/children/{key}/{path:[a-zA-Z0-9=\-\/]+}`, childKind)
	api.Handle(`/parents`, parentKind)
	api.Handle(`/parents/{key}`, parentKind)
	api.Handle(`/parents/{key}/{path:[a-zA-Z0-9=\-\/]+}`, parentKind)

	// create projects and edit project details
	api.Handle(`/projects`, projectKind)
	api.Handle(`/projects/{key}`, projectKind)
	api.Handle(`/projects/{key}/{path:[a-zA-Z0-9=\-\/]+}`, projectKind)
	// project key can be used to manage project collection

	// kater collection je nas zanima samo ob POST metodi
	// postanje v collection bi lahko bilo urejeno tako:
	api.Handle(`/projects`, projectKind)                                                    // ustvarjanje projektov
	api.Handle(`/projects/{key}`, projectKind)                                              // urejanje projektov
	api.Handle(`/projects/{collection}/{kind}`, projectKind)                                // postanje endpointov v projekt
	api.Handle(`/projects/{collection}/{kind}/{key}`, projectKind)                          // urejanje entrijev v endpointu v projektu
	api.Handle(`/projects/{collection}/{kind}/{key}/{path:[a-zA-Z0-9=\-\/]+}`, projectKind) // ...

	api.Handle(`/{collection}/parents`, middleware.Handler(auth.CollectionMiddleware(parentKind)))
	api.Handle(`/{collection}/parents/{key}`, middleware.Handler(auth.CollectionMiddleware(parentKind)))
	api.Handle(`/{collection}/parents/{key}/{path:[a-zA-Z0-9=\-\/]+}`, middleware.Handler(auth.CollectionMiddleware(parentKind)))

	http.Handle("/", api.Handler())
}

// TODO: check scope on every handler operation (get, put, delete, post) - best to put checks inside handler functions

type Project struct {
	Id   *datastore.Key `datastore:"-" auto:"id" json:"id,omitempty"`
	Name string         `json:"name"`
}

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
