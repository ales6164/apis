package test

import (
	"github.com/ales6164/apis"
	"github.com/ales6164/apis/providers/emailpassword"
	"github.com/dgrijalva/jwt-go"
	"google.golang.org/appengine/datastore"
	"io/ioutil"
	"net/http"
)

var (
	parentKind = apis.NewKind(&apis.KindOptions{
		Path: "parents",
		Type: Parent{},
	})
	childKind = apis.NewKind(&apis.KindOptions{
		Path: "children",
		Type: Child{},
	})
	projectKind = apis.NewKind(&apis.KindOptions{
		Path: "projects",
		Type: Project{},
		/*IsCollection: true,*/
		/*KindProvider: kind.NewProvider(childKind, parentKind),*/
	})
)



const (
	subscriber = "subscriber"
)

/*
TODO: 1. auth handler -- apis needs to know what roles have what scopes.. maybe move roles to apis? And GetSession (in context) must always receive a session...
TODO: 2. collection creator is by default it's owner (scope is owner)
TODO: 3. collections
 */
func init() {
	// auth
	signingKey, err := ioutil.ReadFile("key.txt")
	if err != nil {
		panic(err)
	}

	auth := apis.NewAuth(&apis.AuthOptions{
		SigningKey:          signingKey,
		Extractors:          []apis.TokenExtractor{apis.FromAuthHeader},
		CredentialsOptional: false,
		DefaultScopes:       []string{subscriber},

		SigningMethod:  jwt.SigningMethodHS256,
		TokenExpiresIn: 60 * 60 * 24 * 7,
	})
	auth.RegisterProvider(emailpassword.New(&emailpassword.Config{}))

	api := apis.New(&apis.Options{
		Roles: map[string][]string{
			subscriber:    {parentKind.ScopeFullControl},
			apis.AllUsers: {parentKind.ScopeFullControl},
		},
	})
	api.SetAuth(auth)

	//api.Handle("/auth/signup", provider.SignUpHandler())
	api.RegisterKind(childKind)
	api.RegisterKind(parentKind)
	api.RegisterKind(projectKind)

	// kater collection je nas zanima samo ob POST metodi
	// postanje v collection bi lahko bilo urejeno tako:
	// api.Handle(`/projects`, projectKind)                                                  	// ustvarjanje projektov
	// api.Handle(`/projects/{key}`, projectKind)                                             	// urejanje projektov
	// api.Handle(`/projects/{collection}/{kind}`, projectKind)                            		// postanje endpointov v projekt
	// api.Handle(`/projects/{collection}/{kind}/{key}`, projectKind)                        	// urejanje entrijev v endpointu v projektu
	// api.Handle(`/projects/{collection}/{kind}/{key}/{path:[a-zA-Z0-9=\-\/]+}`, projectKind)	// ...

	http.Handle("/", api)
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
