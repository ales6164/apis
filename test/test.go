package test

import (
	"github.com/ales6164/apis"
	"github.com/ales6164/apis/collection"
	"github.com/ales6164/apis/group"
	"github.com/ales6164/apis/providers/emailpassword"
	"github.com/dgrijalva/jwt-go"
	"io/ioutil"
	"net/http"
	"time"
)

const (
	// roles
	subscriber = "subscriber"
)

func init() {
	// Signing key for JWT token issuing and authorization process.
	signingKey, err := ioutil.ReadFile("key.txt")
	if err != nil {
		panic(err)
	}

	// Built-in auth library
	auth := apis.NewAuth(&apis.AuthOptions{
		SigningKey:          signingKey,
		Extractors:          []apis.TokenExtractor{apis.FromAuthHeader},
		CredentialsOptional: false,
		DefaultRoles:        []string{subscriber},
		SigningMethod:       jwt.SigningMethodHS256,
		TokenExpiresIn:      60 * 60 * 24 * 7,
	})
	// Login/registration flow provider
	auth.RegisterProvider(emailpassword.New(&emailpassword.Config{}))

	// Set-up API, define user roles and permissions
	api := apis.New(&apis.Options{
		Rules: apis.Rules{
			Match: apis.Match{
				projects: apis.Rules{
					FullControl: []string{apis.AllUsers},
					Match: apis.Match{
						objects: apis.Rules{
							FullControl: []string{apis.AllUsers},
						},
					},
				},
				objects: apis.Rules{
					FullControl: []string{apis.AllUsers},
				},
			},
		},
	})
	//api.SetAuth(auth)

	// Expose collections
	api.HandleKind(projects)
	api.HandleKind(objects)
	//api.HandleKind(projects)

	// Custom handlers
	// Prints datastore info
	/*api.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
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
	})*/

	// Serve
	http.Handle("/", api)
}

// Collections

var (
	projects = group.New("projects", Project{})
	objects  = collection.New("objects", Object{})
)

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
