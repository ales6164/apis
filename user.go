package apis

import (
	"encoding/json"
	"gopkg.in/ales6164/apis.v1/errors"
	"gopkg.in/ales6164/apis.v1/kind"
	"github.com/gorilla/mux"
	"github.com/imdario/mergo"
	"golang.org/x/net/context"
	"google.golang.org/appengine/datastore"
	"net/http"
	"reflect"
	"time"
)

type Account struct {
	Email string `search:"-" json:"-"`
	Hash  []byte `search:"-" json:"-"`
	User  User   `search:"-"`
}

type User struct {
	// these cant be updated by user
	Id                  *datastore.Key `search:"-" datastore:"-" apis:"id" json:"id"`
	UserID              *datastore.Key `datastore:"-" json:"user_id,omitempty"`
	Roles               []string       `json:"roles,omitempty"`
	Email               string         `json:"email,omitempty"`                 // login email
	EmailVerified       bool           `json:"email_verified,omitempty"`        // true if email verified
	PhoneNumber         string         `json:"phone_number,omitempty"`          // login phone number
	PhoneNumberVerified bool           `json:"phone_number_verified,omitempty"` // true if phone number verified

	CreatedAt time.Time `apis:"createdAt" json:"created_at,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
	IsPublic  bool      `json:"is_public,omitempty"` // this is only relevant for chat atm - public profiles can be contacted
	Locale    string    `json:"locale,omitempty"`    // locale

	// za združljivost s staro različico
	Profile Profile `json:"profile,omitempty"`

	Name              string            `json:"name,omitempty"`
	GivenName         string            `json:"given_name,omitempty"`
	FamilyName        string            `json:"family_name,omitempty"`
	MiddleName        string            `json:"middle_name,omitempty"`
	Nickname          string            `json:"nickname,omitempty"`
	Picture           string            `json:"picture,omitempty"` // profile picture URL
	Website           string            `json:"website,omitempty"` // website URL
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
	Name: "_user",
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
		methods: []string{http.MethodGet, http.MethodPut /*, http.MethodDelete*/},
	}

	userRoute.Get(func(w http.ResponseWriter, r *http.Request) {
		ctx := userRoute.NewContext(r)
		if !ctx.IsAuthenticated {
			ctx.PrintError(w, errors.ErrUnathorized)
			return
		}
		_, isPrivateOnly := ctx.HasPermission(userRoute.kind, GET)
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

		_, isPrivateOnly := ctx.HasPermission(userRoute.kind, GET)

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

		/*if userRoute.kind.EnableSearch {
			var visibility string
			if acc.User.IsPublic {
				visibility = "public"
			} else {
				visibility = "private"
			}
			if err := saveToIndex(ctx, userRoute.kind, userKey.Encode(), &UserDoc{
				UserID:     search.Atom(userKey.Encode()),
				Roles:      strings.Join(acc.User.Roles, ","),
				Locale:     search.Atom(acc.User.Locale),
				Email:      search.Atom(acc.User.Email),
				CreatedAt:  acc.User.CreatedAt,
				UpdatedAt:  acc.User.UpdatedAt,
				Visibility: search.Atom(visibility),
			}); err != nil {
				ctx.PrintError(w, err)
				return
			}
		}*/

		ctx.Print(w, acc.User)
	})

	r.Handle("/user", a.middleware.Handler(userRoute.getHandler())).Methods(http.MethodGet)
	r.Handle("/user/{id}", a.middleware.Handler(userRoute.getHandler())).Methods(http.MethodGet)

	r.Handle("/user", a.middleware.Handler(userRoute.putHandler())).Methods(http.MethodPut)
	r.Handle("/user/{id}", a.middleware.Handler(userRoute.putHandler())).Methods(http.MethodPut)

	r.Handle("/fix-users", a.middleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := userRoute.NewContext(r)
		if ok := ctx.HasRole(AdminRole); !ok {
			ctx.PrintError(w, errors.ErrForbidden)
			return
		}

		var hs []*Account
		q := datastore.NewQuery(UserKind.Name)
		t := q.Run(ctx)
		for {
			var h = new(Account)
			key, err := t.Next(h)
			h.User.UserID = key
			if err == datastore.Done {
				break
			}
			if err != nil {
				ctx.PrintError(w, err)
				return
			}
			hs = append(hs, h)
		}

		/*for _, acc := range hs {
			var visibility string
			if acc.User.IsPublic {
				visibility = "public"
			} else {
				visibility = "private"
			}
			err := saveToIndex(ctx, UserKind, acc.User.UserID.Encode(), &UserDoc{
				UserID:     search.Atom(acc.User.UserID.Encode()),
				Roles:      strings.Join(acc.User.Roles, ","),
				Keywords:   strings.Join([]string{acc.User.Profile.Name, acc.User.Profile.MiddleName, acc.User.Profile.FamilyName, acc.User.Profile.GivenName, acc.User.Profile.Nickname}, ","),
				Locale:     search.Atom(acc.User.Locale),
				Email:      search.Atom(acc.User.Email),
				CreatedAt:  acc.User.CreatedAt,
				UpdatedAt:  acc.User.UpdatedAt,
				Visibility: search.Atom(visibility),
			})
			if err != nil {
				ctx.PrintError(w, err)
				return
			}
		}*/
		ctx.Print(w, "success")
	}))).Methods(http.MethodGet)

	// SEARCH
	/*r.Handle("/users", a.middleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := userRoute.NewContext(r)
		var ok, isPrivate bool
		if ok, isPrivate = ctx.HasPermission(UserKind, QUERY); !ok {
			ctx.PrintError(w, errors.ErrForbidden)
			return
		}

		q, next, sort, limit, offset := r.FormValue("q"), r.FormValue("next"), r.FormValue("sort"), r.FormValue("limit"), r.FormValue("offset")

		index, err := OpenIndex(UserKind.Name)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		// build facets and retrieve filters from query parameters
		var fields []search.Field
		var facets []search.Facet
		for key, val := range r.URL.Query() {
			if key == "filter" {
				for _, v := range val {
					filter := strings.Split(v, ":")
					if len(filter) == 2 {
						// todo: currently only supports facet type search.Atom
						facets = append(facets, search.Facet{Name: filter[0], Value: search.Atom(filter[1])})
					}
				}
			} else if key == "range" {
				for _, v := range val {
					filter := strings.Split(v, ":")
					if len(filter) == 2 {

						rangeStr := strings.Split(filter[1], "-")
						if len(rangeStr) == 2 {
							rangeStart, _ := strconv.ParseFloat(rangeStr[0], 64)
							rangeEnd, _ := strconv.ParseFloat(rangeStr[1], 64)

							facets = append(facets, search.Facet{Name: filter[0], Value: search.Range{
								Start: rangeStart,
								End:   rangeEnd,
							}})
						}
					}
				}
			} else if key == "sort" {
				//skip
			} else if key == "key" {
				// used for auth
				//skip
			} else {
				for _, v := range val {
					fields, facets = UserKind.RetrieveSearchParameter(key, v, fields, facets)
				}
			}
		}

		if isPrivate {
			fields = append(fields, search.Field{Name: "Visibility", Value: "public"})
		}

		// build []search.Field to a query string and append
		if len(fields) > 0 {
			for _, f := range fields {
				if len(q) > 0 {
					q += " AND " + f.Name + ":" + f.Value.(string)
				} else {
					q += f.Name + ":" + f.Value.(string)
				}
			}
		}

		// limit
		var intLimit int
		if len(limit) > 0 {
			intLimit, _ = strconv.Atoi(limit)
		}
		// offset
		var intOffset int
		if len(offset) > 0 {
			intOffset, _ = strconv.Atoi(offset)
		}

		// sorting
		var sortExpr []search.SortExpression
		if len(sort) > 0 {
			var desc bool
			if sort[:1] == "-" {
				sort = sort[1:]
				desc = true
			}
			sortExpr = append(sortExpr, search.SortExpression{Expr: sort, Reverse: !desc})
		}

		// real search
		var results []interface{}
		var docKeys []*datastore.Key
		var t *search.Iterator
		for t = index.Search(ctx, q, &search.SearchOptions{
			IDsOnly:       UserKind.RetrieveByIDOnSearch,
			Cursor:        search.Cursor(next),
			CountAccuracy: 1000,
			Offset:        intOffset,
			Limit:         intLimit,
			Sort: &search.SortOptions{
				Expressions: sortExpr,
			}}); ; {
			var doc = reflect.New(UserKind.SearchType).Interface()
			docKey, err := t.Next(doc)
			if err == search.Done {
				break
			}
			if err != nil {
				ctx.PrintError(w, err)
				return
			}

			if key, err := UserKind.DecodeKey(docKey); err == nil {
				docKeys = append(docKeys, key)
				results = append(results, doc)
			}
		}

		// fetch real entries from datastore
		if len(docKeys) == len(results) {
			hs, err := kind.GetMulti(ctx, accountKind, docKeys...)
			if err != nil {
				ctx.PrintError(w, err)
				return
			}
			for k, h := range hs {
				acc := h.Value().(*Account)
				acc.User.UserID = h.GetKey()
				results[k] = &acc.User
			}
		} else {
			ctx.PrintError(w, errors.New("results mismatch"))
			return
		}

		var cursor *Cursor
		if len(t.Cursor()) > 0 || len(next) > 0 {
			cursor = &Cursor{
				Next: string(t.Cursor()),
				Prev: next,
			}
		}

		ctx.Print(w, &SearchOutput{
			Count:   len(results),
			Total:   t.Count(),
			Results: results,
			Cursor:  cursor,
		})
	}))).Methods(http.MethodGet)*/
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
