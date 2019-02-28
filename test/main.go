package main

import (
	"fmt"
	"github.com/ales6164/apis"
	"github.com/ales6164/apis/collection"
	"github.com/ales6164/apis/iam"
	"github.com/ales6164/apis/providers/emailpassword"
	"github.com/dgrijalva/jwt-go"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

const (
	// roles
	subscriber = "subscriber"
)

func main() {
	// Signing key for JWT token issuing and authorization process.
	signingKey, err := ioutil.ReadFile("key.txt")
	if err != nil {
		panic(err)
	}

	// Built-in auth library
	auth := iam.NewIAM(&iam.Options{
		SigningKey:          signingKey,
		Extractors:          []iam.TokenExtractor{iam.FromAuthHeader},
		CredentialsOptional: false,
		DefaultRoles:        []string{subscriber},
		SigningMethod:       jwt.SigningMethodHS256,
		TokenExpiresIn:      60 * 60 * 24 * 7,
	})
	// Login/registration flow provider
	auth.RegisterProvider(emailpassword.New(&emailpassword.Config{}))

	// Set-up API, define user roles and permissions
	api := apis.New(&apis.Options{
		IAM: auth,
		Rules: &apis.Rules{
			Match: apis.Match{
				projects: &apis.Rules{
					AccessControl: true,
					Permissions: apis.Permissions{
						iam.AllUsers: []string{iam.FullControl},
					},
					Match: apis.Match{
						objects: &apis.Rules{
							Permissions: apis.Permissions{
								iam.AllUsers: []string{iam.FullControl},
							},
						},
					},
				},
				objects: &apis.Rules{
					Permissions: apis.Permissions{
						iam.AllUsers: []string{iam.FullControl},
					},
					Match: apis.Match{
						objects: &apis.Rules{
							Permissions: apis.Permissions{
								iam.AllUsers: []string{iam.FullControl},
							},
							Match: apis.Match{
								objects: &apis.Rules{
									Permissions: apis.Permissions{
										iam.AllUsers: []string{iam.FullControl},
									},
								},
							},
						},
					},
				},
			},
		},
	})

	// Expose collections
	api.HandleKind(projects)
	api.HandleKind(objects)
	//api.HandleKind(projects)

	// Serve
	http.Handle("/", api.Handler())

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		log.Printf("Defaulting to port %s", port)
	}

	log.Printf("Listening on port %s", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}

// Collections

var (
	projects = collection.New("projects", Project{})
	objects  = collection.New("objects", Object{})
)

type Project struct {
	Id   string `datastore:"-" auto:"id" json:"id,omitempty"`
	Name string `json:"name"`
}

type Object struct {
	Name  string   `json:"name"`
	Stuff []string `json:"stuff"`
}
