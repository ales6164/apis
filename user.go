package apis

import (
	"google.golang.org/appengine/datastore"
	"time"
	"github.com/ales6164/apis/errors"
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

func getUser(ctx Context, key *datastore.Key) (*User, error) {
	var acc Account
	if err := datastore.Get(ctx, key, &acc); err != nil {
		if err == datastore.ErrNoSuchEntity {
			return nil, errors.ErrUserDoesNotExist
		}
		return nil, err
	}
	acc.User.UserID = key
	return &acc.User, nil
}


func getUsers(ctx Context) ([]*User, error) {
	var hs []*User
	q := datastore.NewQuery("_user")
	t := q.Run(ctx)
	for {
		var h = new(Account)
		key, err := t.Next(h)
		if err == datastore.Done {
			break
		}
		if err != nil {
			return hs, err
		}
		h.User.UserID = key
		hs = append(hs, &h.User)
	}
	return hs, nil
}

func getPublicUsers(ctx Context) ([]*User, error) {
	var hs []*User
	q := datastore.NewQuery("_user").Filter("IsPublic =", true)
	t := q.Run(ctx)
	for {
		var h = new(Account)
		key, err := t.Next(h)
		if err == datastore.Done {
			break
		}
		if err != nil {
			return hs, err
		}
		h.User.UserID = key
		hs = append(hs, &h.User)
	}
	return hs, nil
}

func login(ctx Context, email, password string) (string, *User, error) {
	var signedToken string
	acc := new(Account)
	key := datastore.NewKey(ctx, "_user", email, 0, nil)
	if err := datastore.Get(ctx, key, acc); err != nil {
		if err == datastore.ErrNoSuchEntity {
			return signedToken, nil, errors.ErrUserDoesNotExist
		}
		return signedToken, nil, err
	}
	acc.User.UserID = key

	if err := decrypt(acc.Hash, []byte(password)); err != nil {
		return signedToken, nil, errors.ErrUserPasswordIncorrect
	}

	signedToken, err := createSession(ctx, key, &acc.User)

	return signedToken, &acc.User, err
}

func createSession(ctx Context, key *datastore.Key, user *User) (string, error) {
	var signedToken string
	now := time.Now()
	expiresAt := now.Add(time.Hour * time.Duration(72))
	// create a new session
	sess := new(ClientSession)
	sess.CreatedAt = now
	sess.ExpiresAt = expiresAt
	sess.User = key
	sess.JwtID = RandStringBytesMaskImprSrc(LetterBytes, 16)

	sessKey := datastore.NewIncompleteKey(ctx, "_clientSession", nil)
	sessKey, err := datastore.Put(ctx, sessKey, sess)
	if err != nil {
		return signedToken, err
	}

	// create a JWT token
	return ctx.authenticate(sess, sessKey.Encode(), user, expiresAt.Unix())
}
