package apis

import (
	"google.golang.org/appengine/datastore"
	"time"
	"github.com/ales6164/apis/errors"
	"github.com/ales6164/apis/kind"
	"reflect"
	"github.com/gorilla/mux"
	"net/http"
	"google.golang.org/appengine/search"
	"encoding/json"
	"github.com/imdario/mergo"
	"golang.org/x/net/context"
	"strings"
)

type Account struct {
	Email string `json:"-"`
	Hash  []byte `json:"-"`
	User  User
}

type User struct {
	// these cant be updated by user
	UserID              *datastore.Key `datastore:"-" json:"user_id,omitempty"`
	Roles               []string       `json:"roles,omitempty"`
	Email               string         `json:"email,omitempty"`                 // login email
	EmailVerified       bool           `json:"email_verified,omitempty"`        // true if email verified
	PhoneNumber         string         `json:"phone_number,omitempty"`          // login phone number
	PhoneNumberVerified bool           `json:"phone_number_verified,omitempty"` // true if phone number verified
	CreatedAt           time.Time      `json:"created_at,omitempty"`
	UpdatedAt           time.Time      `json:"updated_at,omitempty"`
	IsPublic            bool           `json:"is_public,omitempty"` // this is only relevant for chat atm - public profiles can be contacted
	Locale              string         `json:"locale,omitempty"`    // locale
	Profile             Profile        `json:"profile,omitempty"`
}

type UserDoc struct {
	UserID    search.Atom `json:"user_id,omitempty"`
	Roles     string      `json:"roles,omitempty"`
	Email     search.Atom `json:"email,omitempty"`
	CreatedAt time.Time   `json:"created_at,omitempty"`
	UpdatedAt time.Time   `json:"updated_at,omitempty"`
	Locale    search.Atom `json:"locale,omitempty"`
}

type Profile struct {
	// can be changed by user
	Name       string `json:"name,omitempty"`
	GivenName  string `json:"given_name,omitempty"`
	FamilyName string `json:"family_name,omitempty"`
	MiddleName string `json:"middle_name,omitempty"`
	Nickname   string `json:"nickname,omitempty"`
	Picture    string `json:"picture,omitempty"` // profile picture URL
	Website    string `json:"website,omitempty"` // website URL

	// is not added to JWT and is private to user
	DeliveryAddresses []DeliveryAddress `json:"delivery_addresses,omitempty"`
	DateOfBirth       time.Time         `json:"date_of_birth,omitempty"`
	PlaceOfBirth      Address           `json:"place_of_birth,omitempty"`
	Title             string            `json:"title,omitempty"`
	Address           Address           `json:"address,omitempty"`
	Address2          Address           `json:"address_2,omitempty"`
	Company           Company           `json:"company,omitempty"`
	Contact           Contact           `json:"contact,omitempty"`
	SocialProfiles    []SocialProfile   `json:"social_profiles,omitempty"`
	Slogan            string            `json:"slogan,omitempty"`
}

type Identity struct {
	Provider   string `json:"provider,omitempty"` // our app name, google-auth2
	UserId     int64  `json:"user_id,omitempty"`
	Connection string `json:"connection,omitempty"` // client-defined-connection-name?, google-auth2, ...
	IsSocial   bool   `json:"is_social,omitempty"`  // true when from social network
}

type SocialProfile struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type DeliveryAddress struct {
	Name       string `json:"name,omitempty"`
	GivenName  string `json:"given_name,omitempty"`
	FamilyName string `json:"family_name,omitempty"`
	MiddleName string `json:"middle_name,omitempty"`
	Address    string `json:"address,omitempty"`
	PostCode   string `json:"post_code,omitempty"`
	City       string `json:"city,omitempty"`
	State      string `json:"state,omitempty"`
	Country    string `json:"country,omitempty"`
}

type Address struct {
	Name      string  `json:"name,omitempty"`
	Company   string  `json:"company,omitempty"`
	VatNumber string  `json:"vat_number,omitempty"`
	Address   string  `json:"address,omitempty"`
	PostCode  string  `json:"post_code,omitempty"`
	City      string  `json:"city,omitempty"`
	State     string  `json:"state,omitempty"`
	Country   string  `json:"country,omitempty"`
	Lat       float64 `json:"lat,omitempty"`
	Lng       float64 `json:"lng,omitempty"`
}

type Company struct {
	Name      string  `json:"name,omitempty"`
	VatNumber string  `json:"vat_number,omitempty"`
	Address   Address `json:"address,omitempty"`
	Contact   Contact `json:"contact,omitempty"`
}

type Contact struct {
	Email        string `json:"email,omitempty"`
	Email2       string `json:"email_2,omitempty"`
	PhoneNumber  string `json:"phone_number,omitempty"`
	PhoneNumber2 string `json:"phone_number_2,omitempty"`
}

type ClientSession struct {
	CreatedAt time.Time
	ExpiresAt time.Time
	JwtID     string
	IsBlocked bool
	User      *datastore.Key
}

var UserKind = kind.New(reflect.TypeOf(User{}), &kind.Options{
	Name:         "_user",
	EnableSearch: true,
	SearchType:   reflect.TypeOf(UserDoc{}),
})

type InputUser struct {
	Locale  string   `json:"locale,omitempty"` // locale
	Roles   []string `json:"roles,omitempty"`  // locale
	Profile Profile  `json:"profile,omitempty"`
}

func getUser(ctx context.Context, key *datastore.Key) (*User, error) {
	var acc = new(Account)
	if err := datastore.Get(ctx, key, acc); err != nil {
		return nil, err
	}
	acc.User.UserID = key
	return &acc.User, nil
}

func initUser(a *Apis, r *mux.Router) {
	userRoute := &Route{
		kind:    UserKind,
		a:       a,
		path:    "/user",
		methods: []string{http.MethodGet, http.MethodPut /*, http.MethodDelete*/ },
	}

	userRoute.Get(func(w http.ResponseWriter, r *http.Request) {
		ctx := userRoute.NewContext(r)
		if !ctx.IsAuthenticated {
			ctx.PrintError(w, errors.ErrUnathorized)
			return
		}
		_, isPrivateOnly := ctx.HasPermission(userRoute.kind, READ)
		var userKey *datastore.Key
		var userId = mux.Vars(r)["id"]
		if len(userId) > 0 {
			var err error
			if isPrivateOnly {
				ctx.PrintError(w, errors.ErrForbidden)
				return
			}
			userKey, err = UserKind.DecodeKey(userId)
			if err != nil {
				ctx.PrintError(w, err)
				return
			}
		} else {
			userKey = ctx.UserKey()
		}
		user, err := getUser(ctx, userKey)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}
		ctx.Print(w, user)
	})
	userRoute.Put(func(w http.ResponseWriter, r *http.Request) {
		ctx := userRoute.NewContext(r)
		if !ctx.IsAuthenticated {
			ctx.PrintError(w, errors.ErrUnathorized)
			return
		}

		_, isPrivateOnly := ctx.HasPermission(userRoute.kind, READ)

		var userKey *datastore.Key
		var userId = mux.Vars(r)["id"]
		if len(userId) > 0 {
			var err error
			if isPrivateOnly {
				ctx.PrintError(w, errors.ErrForbidden)
				return
			}
			userKey, err = UserKind.DecodeKey(userId)
			if err != nil {
				ctx.PrintError(w, err)
				return
			}
		} else {
			userKey = ctx.UserKey()
		}
		var inputUser = new(InputUser)
		if err := json.Unmarshal(ctx.Body(), inputUser); err != nil {
			ctx.PrintError(w, err)
			return
		}
		var inputAccount = &Account{
			User: User{
				Locale:  inputUser.Locale,
				Roles:   inputUser.Roles,
				Profile: inputUser.Profile,
			},
		}
		var acc = new(Account)
		if err := datastore.RunInTransaction(ctx, func(tc context.Context) error {
			err := datastore.Get(ctx, userKey, acc)
			if err != nil {
				return err
			}
			if err := mergo.Merge(acc, inputAccount, mergo.WithOverride, mergo.WithTransformers(timeTransformer{})); err != nil {
				return err
			}
			acc.User.UpdatedAt = time.Now()
			_, err = datastore.Put(ctx, userKey, acc)
			return err
		}, nil); err != nil {
			ctx.PrintError(w, err)
			return
		}

		acc.User.UserID = userKey

		if userRoute.kind.EnableSearch {
			if err := saveToIndex(ctx, userRoute.kind, userKey.Encode(), &UserDoc{
				UserID:    search.Atom(userKey.Encode()),
				Roles:     strings.Join(acc.User.Roles, ","),
				Locale:    search.Atom(acc.User.Locale),
				Email:     search.Atom(acc.User.Email),
				CreatedAt: acc.User.CreatedAt,
				UpdatedAt: acc.User.UpdatedAt,
			}); err != nil {
				ctx.PrintError(w, err)
				return
			}
		}

		ctx.Print(w, acc.User)
	})

	r.Handle("/user", a.middleware.Handler(userRoute.getHandler())).Methods(http.MethodGet)
	r.Handle("/user/{id}", a.middleware.Handler(userRoute.getHandler())).Methods(http.MethodGet)

	r.Handle("/user", a.middleware.Handler(userRoute.putHandler())).Methods(http.MethodPut)
	r.Handle("/user/{id}", a.middleware.Handler(userRoute.putHandler())).Methods(http.MethodPut)

	// SEARCH
	a.kinds[UserKind.Name] = UserKind
}

type timeTransformer struct{}

func (t timeTransformer) Transformer(typ reflect.Type) func(dst, src reflect.Value) error {
	if typ == reflect.TypeOf(time.Time{}) {
		return func(dst, src reflect.Value) error {
			if dst.CanSet() {
				isZero := dst.MethodByName("IsZero")
				result := isZero.Call([]reflect.Value{})
				if result[0].Bool() {
					dst.Set(src)
				}
			}
			return nil
		}
	}
	return nil
}
