package apis

import (
	"encoding/json"
	"github.com/ales6164/apis-v1/errors"
	"github.com/asaskevich/govalidator"
	"golang.org/x/net/context"
	"google.golang.org/appengine/datastore"
	"net/http"
	"strings"
	"time"
)

var (
	ErrEmailUndefined    = errors.New("email undefined")
	ErrPasswordUndefined = errors.New("password undefined")
	ErrInvalidCallback   = errors.New("callback is not a valid URL")
	ErrInvalidEmail      = errors.New("email is not valid")
	ErrPasswordTooLong   = errors.New("password must be exactly or less than 128 characters long")
	ErrPasswordTooShort  = errors.New("password must be at least 6 characters long")
)

func checkEmail(v string) error {
	if len(v) == 0 {
		return ErrEmailUndefined
	}
	if !govalidator.IsEmail(v) {
		return ErrInvalidEmail
	}
	if len(v) > 128 || len(v) < 5 {
		return ErrInvalidEmail
	}

	return nil
}

func checkPassword(v string) error {
	if len(v) == 0 {
		return ErrPasswordUndefined
	}
	if len(v) > 128 {
		return ErrPasswordTooLong
	}
	if len(v) < 6 {
		return ErrPasswordTooShort
	}

	return nil
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

func loginHandler(R *Route) http.HandlerFunc {
	type Login struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	type AuthOut struct {
		TokenID string `json:"token_id"`
		User    User   `json:"user"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := R.NewContext(r)

		var email, password string

		ct := r.Header.Get("content-type")
		if strings.Contains(ct, "application/json") {
			var login Login
			if err := json.Unmarshal(ctx.Body(), &login); err != nil {
				ctx.PrintError(w, err)
				return
			}
		} else {
			email, password = r.FormValue("email"), r.FormValue("password")
		}

		err := checkEmail(email)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}
		err = checkPassword(password)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		email = strings.ToLower(email)

		signedToken, u, err := login(ctx, email, password)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		ctx.Print(w, AuthOut{
			TokenID: signedToken,
			User:    *u,
		})
	}
}

func registrationHandler(R *Route, role Role) http.HandlerFunc {
	type InputUser struct {
		Password string `json:"password"`
		Email    string `json:"email"`

		Locale string `json:"locale,omitempty"` // locale

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
	}
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := R.NewContext(r)

		var inputUser InputUser
		err := json.Unmarshal(ctx.Body(), &inputUser)
		if err != nil {
			ctx.PrintError(w, err, "unmarshal")
			return
		}

		if err = checkEmail(inputUser.Email); err != nil {
			ctx.PrintError(w, err, "check email")
			return
		}

		if err = checkPassword(inputUser.Password); err != nil {
			ctx.PrintError(w, err, "check password")
			return
		}

		inputUser.Email = strings.ToLower(inputUser.Email)

		// create password hash
		hash, err := crypt([]byte(inputUser.Password))
		if err != nil {
			ctx.PrintError(w, err, "crypt")
			return
		}

		userKey := datastore.NewKey(ctx, "_user", inputUser.Email, 0, nil)

		// create User
		var acc = Account{
			Hash:  hash,
			Email: inputUser.Email,
			User: User{
				UserID:              userKey,
				Roles:               []string{string(role)},
				Email:               inputUser.Email,
				EmailVerified:       false,
				PhoneNumber:         "",
				PhoneNumberVerified: false,
				CreatedAt:           ctx.Time,
				UpdatedAt:           ctx.Time,
				Locale:              inputUser.Locale,
				Profile: Profile{
					Name:       inputUser.Name,
					GivenName:  inputUser.GivenName,
					FamilyName: inputUser.FamilyName,
					MiddleName: inputUser.MiddleName,
					Nickname:   inputUser.Nickname,
					Picture:    inputUser.Picture,
					Website:    inputUser.Website,

					DeliveryAddresses: inputUser.DeliveryAddresses,
					DateOfBirth:       inputUser.DateOfBirth,
					PlaceOfBirth:      inputUser.PlaceOfBirth,
					Title:             inputUser.Title,
					Address:           inputUser.Address,
					Address2:          inputUser.Address2,
					Company:           inputUser.Company,
					Contact:           inputUser.Contact,
					SocialProfiles:    inputUser.SocialProfiles,
				},
				IsPublic: false,
			},
		}

		if err = datastore.RunInTransaction(ctx, func(tc context.Context) error {
			err := datastore.Get(tc, userKey, &datastore.PropertyList{})
			if err != nil {
				if err == datastore.ErrNoSuchEntity {
					// register
					_, err = datastore.Put(tc, userKey, &acc)
					return err
				}
				return err
			}
			return errors.ErrUserAlreadyExists
		}, nil); err != nil {
			ctx.PrintError(w, err, "reg put err")
			return
		}

		// todo: user search
		/*if UserKind.EnableSearch {
			var visibility string
			if acc.User.IsPublic {
				visibility = "public"
			} else {
				visibility = "private"
			}
			if err := saveToIndex(ctx, UserKind, userKey.Encode(), &UserDoc{
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

		if R.a.OnUserSignUp != nil {
			signedToken, err := createSession(ctx, userKey, &acc.User)
			if err != nil {
				ctx.PrintError(w, err)
				return
			}
			R.a.OnUserSignUp(ctx, acc.User, signedToken)
		}
	}
}

func confirmEmailHandler(R *Route) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := R.NewContext(r)
		if !ctx.IsAuthenticated {
			ctx.PrintError(w, errors.ErrUnathorized)
			return
		}

		callback := r.FormValue("callback")
		if !govalidator.IsURL(callback) {
			ctx.PrintError(w, ErrInvalidCallback)
			return
		}

		var match bool
		if len(R.a.options.AuthorizedRedirectURIs) == 0 {
			match = true
		} else {
			for _, uri := range R.a.options.AuthorizedRedirectURIs {
				if uri == callback {
					match = true
					break
				}
			}
		}
		if !match {
			ctx.PrintError(w, ErrInvalidCallback)
			return
		}

		var acc Account
		err := datastore.RunInTransaction(ctx, func(tc context.Context) error {
			err := datastore.Get(tc, ctx.UserKey(), &acc)
			if err != nil {
				if err == datastore.ErrNoSuchEntity {
					return errors.ErrUserDoesNotExist
				}
				return err
			}
			acc.User.EmailVerified = true
			_, err = datastore.Put(tc, ctx.UserKey(), &acc)
			return err
		}, nil)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		if R.a.OnUserVerified != nil {
			signedToken, err := createSession(ctx, ctx.UserKey(), &acc.User)
			if err != nil {
				ctx.PrintError(w, err)
				return
			}
			R.a.OnUserVerified(ctx, acc.User, signedToken)
		}

		http.Redirect(w, r, callback, http.StatusTemporaryRedirect)
	}
}

func changePasswordHandler(R *Route) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := R.NewContext(r)
		if !ctx.IsAuthenticated {
			ctx.PrintError(w, errors.ErrUnathorized)
			return
		}

		password, newPassword := r.FormValue("password"), r.FormValue("new_password")

		err := checkPassword(newPassword)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		var acc Account
		if err = datastore.RunInTransaction(ctx, func(tc context.Context) error {
			err := datastore.Get(tc, ctx.UserKey(), &acc)
			if err != nil {
				if err == datastore.ErrNoSuchEntity {
					return errors.ErrUserDoesNotExist
				}
				return err
			}

			// check old password
			if err = decrypt(acc.Hash, []byte(password)); err != nil {
				return errors.ErrUserPasswordIncorrect
			}

			// create new password hash
			if acc.Hash, err = crypt([]byte(newPassword)); err != nil {
				return err
			}

			_, err = datastore.Put(tc, ctx.UserKey(), &acc)
			return err
		}, nil); err != nil {
			ctx.PrintError(w, err)
			return
		}

		ctx.Print(w, "ok")
	}
}
