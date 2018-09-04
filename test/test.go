package test

import (
	"github.com/ales6164/apis"
	"github.com/ales6164/apis/errors"
	"github.com/ales6164/apis/middleware"
	"github.com/ales6164/apis/providers"
	"google.golang.org/appengine/datastore"
	"io/ioutil"
	"net/http"
	"reflect"
)

var ObjectKind = apis.NewKind(reflect.TypeOf(Object{}))

var (
	subscriberRole = "subscriber"
)

func init() {
	// read private key
	signingKey, err := ioutil.ReadFile("key.txt")
	if err != nil {
		panic(err)
	}

	authMiddleware := middleware.AuthMiddleware(signingKey)
	emailPasswordProvider := providers.WithEmailPasswordProvider(12, signingKey)

	api, _ := apis.New(&apis.Options{})
	api.RegisterRole(subscriberRole, ObjectKind.ScopeFullControl)

	api.Handle("/signin", emailPasswordProvider.SignInHandler(api)).Methods(http.MethodPost)
	api.Handle("/signup", emailPasswordProvider.SignUpHandler(api, subscriberRole)).Methods(http.MethodPost)

	api.Handle("/objects", authMiddleware.Handler(ObjectKind)).Methods(http.MethodPost)
	api.Handle("/objects/{id}", authMiddleware.Handler(ObjectKind)).Methods(http.MethodGet)

	api.Handle("/search", authMiddleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := apis.NewContext(r)
		if ok := ctx.HasScope(ObjectKind.ScopeFullControl); !ok {
			ctx.Print(w, ctx.Session.Scopes)
			return

			http.Error(w, errors.ErrForbidden.Error(), http.StatusForbidden)
			return
		}

		ctx.Print(w, "ok")
	}))).Methods(http.MethodGet)

	http.Handle("/", api)
}

// todo: map kind fields with "apis" tag and make those available to use in api handle path -> /items/{id}/{someFieldTag}

type Object struct {
	Id   *datastore.Key `datastore:"-" apis:"id"`
	Name string
}
