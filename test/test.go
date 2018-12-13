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

	var parentKind = kind.New(&kind.Options{
		Name: "parent",
		Type: Parent{},
	})
	var childKind = kind.New(&kind.Options{
		Name: "child",
		Type: Child{},
	})
	var projectKind = kind.New(&kind.Options{
		Name:         "project",
		Type:         Project{},
		IsCollection: true,
		KindProvider: kind.NewProvider(childKind, parentKind),
	})

	api := apis.New(&apis.Options{
	})

	api.Handle("/children", middleware.Handler(childKind))
	api.Handle("/parents", middleware.Handler(parentKind))
	api.Handle("/projects", middleware.Handler(projectKind))

	// kater collection je nas zanima samo ob POST metodi
	// postanje v collection bi lahko bilo urejeno tako:
	//api.Handle(`/projects`, projectKind)                                                     // ustvarjanje projektov
	//api.Handle(`/projects/{key}`, projectKind)                                               // urejanje projektov
	//api.Handle(`/projects/{collection}/{kind}`, projectKind)                                // postanje endpointov v projekt
	//api.Handle(`/projects/{collection}/{kind}/{key}`, projectKind)                          // urejanje entrijev v endpointu v projektu
	//api.Handle(`/projects/{collection}/{kind}/{key}/{path:[a-zA-Z0-9=\-\/]+}`, projectKind) // ...

	//api.Handle(`/collections/{collection}`)

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
