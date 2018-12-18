package test

import (
	"github.com/ales6164/apis"
	"github.com/ales6164/apis/providers/emailPassword"
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

	var userKind = apis.NewKind(&apis.KindOptions{
		Name: "user",
		Type: User{},
	})
	var parentKind = apis.NewKind(&apis.KindOptions{
		Name: "parent",
		Type: Parent{},
	})
	var childKind = apis.NewKind(&apis.KindOptions{
		Name: "child",
		Type: Child{},
	})
	var projectKind = apis.NewKind(&apis.KindOptions{
		Name: "project",
		Type: Project{},
		/*IsCollection: true,*/
		/*KindProvider: kind.NewProvider(childKind, parentKind),*/
	})

	a := apis.NewAuth(&apis.AuthOptions{
		SigningKey:          signingKey,
		Extractors:          []apis.TokenExtractor{apis.FromAuthHeader},
		CredentialsOptional: false,
		DefaultScopes:       []string{parentKind.ScopeFullControl},
		SigningMethod:       jwt.SigningMethodHS256,
	})
	middleware := a.Middleware()

	provider := emailpassword.New(a, &emailpassword.Options{
		UserKind: userKind,
		Cost:     12,
	})



	api := apis.New(&apis.Options{
	})

	api.Handle("/auth/signup", provider.SignUpHandler())
	api.HandleKind("/children", middleware.Handler(childKind))
	api.HandleKind("/parents", middleware.Handler(parentKind))
	api.HandleKind("/projects", middleware.Handler(projectKind))

	// kater collection je nas zanima samo ob POST metodi
	// postanje v collection bi lahko bilo urejeno tako:
	// api.Handle(`/projects`, projectKind)                                                  	// ustvarjanje projektov
	// api.Handle(`/projects/{key}`, projectKind)                                             	// urejanje projektov
	// api.Handle(`/projects/{collection}/{kind}`, projectKind)                            		// postanje endpointov v projekt
	// api.Handle(`/projects/{collection}/{kind}/{key}`, projectKind)                        	// urejanje entrijev v endpointu v projektu
	// api.Handle(`/projects/{collection}/{kind}/{key}/{path:[a-zA-Z0-9=\-\/]+}`, projectKind)	// ...

	http.Handle("/", api.Handler())
}

// TODO: check scope on every handler operation (get, put, delete, post) - best to put checks inside handler functions

type User struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
}

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
