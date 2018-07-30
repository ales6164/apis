package apis

import (
	"reflect"
	"github.com/ales6164/apis/kind"
	"github.com/ales6164/apis/providers"
	"github.com/gorilla/mux"
	"time"
	"net/http"
	"google.golang.org/appengine/datastore"
	"github.com/ales6164/apis/errors"
)

var accountKind = kind.New(reflect.TypeOf(providers.Account{}), &kind.Options{
	Name: "_account",
})

// default User object
type User struct {
	Id        *datastore.Key `datastore:"-" apis:"id" json:"id"`
	CreatedAt time.Time      `apis:"createdAt" json:"createdAt"`
	UpdatedAt time.Time      `apis:"updatedAt" json:"updatedAt"`
	Email     string         `json:"email,omitempty"`
	Phone     string         `json:"phone,omitempty"`
	FirstName string         `json:"firstName,omitempty"`
	LastName  string         `json:"lastName,omitempty"`
	Picture   string         `json:"picture,omitempty"` // profile picture URL
	Website   string         `json:"website,omitempty"` // website URL
	Address   Address        `json:"address,omitempty"`
	Company   Company        `json:"company,omitempty"`
	Slogan    string         `json:"slogan,omitempty"`
	Locale    string         `json:"locale,omitempty"`
}

type Address struct {
	Name        string      `json:"name,omitempty"`
	Address     string      `json:"address,omitempty"`
	PostCode    string      `json:"postCode,omitempty"`
	City        string      `json:"city,omitempty"`
	State       string      `json:"state,omitempty"`
	Country     string      `json:"country,omitempty"`
	Coordinates Coordinates `json:"coordinates,omitempty"`
}

type Coordinates struct {
	Lat float64 `json:"lat,omitempty"`
	Lng float64 `json:"lng,omitempty"`
}

type Company struct {
	Name      string  `json:"name,omitempty"`
	VatNumber string  `json:"vatNumber,omitempty"`
	Address   Address `json:"address,omitempty"`
	Email     string  `json:"email,omitempty"`
	Phone     string  `json:"phone,omitempty"`
}

var UserKind = kind.New(reflect.TypeOf(User{}), &kind.Options{
	Name: "_user",
})

func initUser(a *Apis, r *mux.Router) {
	userRoute := &Route{
		a:       a,
		path:    "/user",
		methods: []string{},
	}

	r.Handle("/user/me", a.middleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := userRoute.NewContext(r)
		if ok := ctx.IsAuthenticated; !ok {
			ctx.PrintError(w, errors.ErrUnathorized)
			return
		}

		h := UserKind.NewHolder(nil)
		if err := h.Get(ctx, ctx.UserKey()); err != nil {
			ctx.PrintError(w, err)
			return
		}

		ctx.Print(w, h.Value())
	}))).Methods(http.MethodGet)
	r.Handle("/user/{id}", a.middleware.Handler(userRoute.getHandler())).Methods(http.MethodGet)
	r.Handle("/user", a.middleware.Handler(userRoute.queryHandler())).Methods(http.MethodGet)
	r.Handle("/user/me", a.middleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := userRoute.NewContext(r)
		if ok := ctx.IsAuthenticated; !ok {
			ctx.PrintError(w, errors.ErrUnathorized)
			return
		}

		h := UserKind.NewHolder(nil)
		if err := h.Parse(ctx.Body()); err != nil {
			ctx.PrintError(w, err)
			return
		}
		h.SetKey(ctx.UserKey())
		if err := h.Update(ctx); err != nil {
			ctx.PrintError(w, err)
			return
		}
		ctx.Print(w, h.Value())
	}))).Methods(http.MethodPut)
	r.Handle("/user/{id}", a.middleware.Handler(userRoute.putHandler())).Methods(http.MethodPut)
}
