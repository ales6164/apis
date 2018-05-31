package apis

import (
	"github.com/asaskevich/govalidator"
	"net/http"
	"github.com/ales6164/apis/errors"
	"google.golang.org/appengine/datastore"
	"strings"
	"encoding/json"
	"golang.org/x/net/context"
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

func getUserHandler(R *Route) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := R.NewContext(r)
		if !ctx.HasRole(AdminRole) {
			ctx.PrintError(w, errors.ErrUnathorized)
			return
		}

		id := r.FormValue("id")
		key, err := datastore.DecodeKey(id)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		// get user
		u, err := getUser(ctx, key)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		ctx.Print(w, u)
	}
}

/*func getUsersHandler(R *Route) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ok, ctx := R.NewContext(r).Authenticate()
		if !ok {
			ctx.PrintError(w, errors.ErrUnathorized)
			return
		}
		if ctx.Role != AdminRole {
			ctx.PrintError(w, errors.ErrUnathorized)
			return
		}

		var hs []*user.User
		var err error

		q := datastore.NewQuery("_user")

		t := q.Run(ctx)
		for {
			var h = new(User)
			h.Id, err = t.Next(h)
			if err == datastore.Done {
				break
			}
			if err != nil {
				ctx.PrintError(w, err)
				return
			}
			hs = append(hs, h)
		}

		ctx.Print(w, hs)
	}
}*/

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

		//dont create a token on registration

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

// update locale nad user profile
func updateUserHandler(R *Route) http.HandlerFunc {
	type InputUser struct {
		Locale  string  `json:"locale,omitempty"` // locale
		Profile Profile `json:"profile,omitempty"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := R.NewContext(r)
		if !ctx.IsAuthenticated {
			ctx.PrintError(w, errors.ErrUnathorized)
			return
		}

		var inputUser InputUser
		err := json.Unmarshal(ctx.Body(), &inputUser)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		// do everything in a transaction
		var acc Account
		if err = datastore.RunInTransaction(ctx, func(tc context.Context) error {
			// get user
			err := datastore.Get(ctx, ctx.UserKey(), &acc)
			if err != nil {
				return err
			}

			acc.User.UpdatedAt = time.Now()
			acc.User.Profile = inputUser.Profile
			acc.User.Locale = inputUser.Locale

			_, err = datastore.Put(ctx, ctx.UserKey(), &acc)
			return err
		}, nil); err != nil {
			ctx.PrintError(w, err)
			return
		}
		ctx.Print(w, &acc.User)
	}
}
