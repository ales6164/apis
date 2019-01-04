package test

import (
	"fmt"
	"github.com/ales6164/apis"
	"github.com/ales6164/apis/collection"
	"github.com/ales6164/apis/providers/emailpassword"
	"github.com/dgrijalva/jwt-go"
	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

var (
	projects = collection.New("projects", Project{}).Group()
	objects = collection.New("objects", Object{})
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
		DefaultRoles:        []string{subscriber},
		SigningMethod:       jwt.SigningMethodHS256,
		TokenExpiresIn:      60 * 60 * 24 * 7,
	})
	auth.RegisterProvider(emailpassword.New(&emailpassword.Config{}))

	api := apis.New(&apis.Options{
		Roles: map[string][][]string{
			apis.AllUsers: {
				objects.Scopes(apis.FullControl),
				projects.Scopes(apis.FullControl),
				projects.Scopes(objects.Scopes(apis.FullControl)...),
			},
		},
	})
	api.SetAuth(auth)


	api.HandleCollection(projects)


	api.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Print all the kinds in the datastore, with all the indexed
		// properties (and their representations) for each.
		ctx := appengine.NewContext(r)

		kinds, err := datastore.Kinds(ctx)
		if err != nil {

			return
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		for _, kind := range kinds {
			fmt.Fprintf(w, "%s:\n", kind)
			props, err := datastore.KindProperties(ctx, kind)
			if err != nil {
				fmt.Fprintln(w, "\t(unable to retrieve properties)")
				continue
			}
			for p, rep := range props {
				fmt.Fprintf(w, "\t-%s (%s)\n", p, strings.Join(rep, ", "))
			}
		}
	})

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
	Id   string `datastore:"-" auto:"id" json:"id,omitempty"`
	Name string `json:"name"`
}

type Object struct {
	Id        string    `datastore:"-" auto:"id" json:"id,omitempty"`
	CreatedAt time.Time `datastore:"-" auto:"createdAt" json:"createdAt,omitempty"`
	UpdatedAt time.Time `datastore:"-" auto:"updatedAt" json:"updatedAt,omitempty"`
	Name      string    `json:"name"`
	Stuff     []string  `json:"stuff"`
}
