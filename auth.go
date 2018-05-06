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
	"reflect"
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
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := R.NewContext(r)

		email, password := r.FormValue("email"), r.FormValue("password")

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

		ctx.Print(w, map[string]interface{}{
			"token_id": signedToken,
			"user":     u,
		})
	}
}

func registrationHandler(R *Route, role Role) http.HandlerFunc {
	type InputUser struct {
		Password string `json:"password"`
		Email    string `json:"email"`

		// can be changed by user
		Name       string `json:"name"`
		GivenName  string `json:"given_name"`
		FamilyName string `json:"family_name"`
		MiddleName string `json:"middle_name"`
		Nickname   string `json:"nickname"`
		Picture    string `json:"picture"` // profile picture URL
		Website    string `json:"website"` // website URL
		Locale     string `json:"locale"`  // locale

		// is not added to JWT and is private to user
		DateOfBirth    time.Time       `json:"date_of_birth"`
		PlaceOfBirth   Address         `json:"place_of_birth"`
		Title          string          `json:"title"`
		Address        Address         `json:"address"`
		Address2       Address         `json:"address_2"`
		Company        Company         `json:"company"`
		Contact        Contact         `json:"contact"`
		SocialProfiles []SocialProfile `json:"social_profiles"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := R.NewContext(r)

		var inputUser InputUser
		err := json.Unmarshal(ctx.Body(), &inputUser)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		if err = checkEmail(inputUser.Email); err != nil {
			ctx.PrintError(w, err)
			return
		}

		if err = checkPassword(inputUser.Password); err != nil {
			ctx.PrintError(w, err)
			return
		}

		inputUser.Email = strings.ToLower(inputUser.Email)

		// create password hash
		hash, err := crypt([]byte(inputUser.Password))
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		userKey := datastore.NewKey(ctx, "_user", inputUser.Email, 0, nil)

		// create User
		u := &user{
			Hash:  hash,
			Email: inputUser.Email,
			User: User{
				userKey,
				[]string{string(role)},
				inputUser.Email,
				false,
				"",
				false,
				ctx.Time,
				ctx.Time,
				inputUser.Name,
				inputUser.GivenName,
				inputUser.FamilyName,
				inputUser.MiddleName,
				inputUser.Nickname,
				inputUser.Picture,
				inputUser.Website,
				inputUser.Locale,
				inputUser.DateOfBirth,
				inputUser.PlaceOfBirth,
				inputUser.Title,
				inputUser.Address,
				inputUser.Address2,
				inputUser.Company,
				inputUser.Contact,
				inputUser.SocialProfiles,
			},
		}

		if err = datastore.RunInTransaction(ctx, func(tc context.Context) error {
			err := datastore.Get(tc, userKey, &datastore.PropertyList{})
			if err != nil {
				if err == datastore.ErrNoSuchEntity {
					// register
					_, err = datastore.Put(tc, userKey, u)
					return err
				}
				return err
			}
			return errors.ErrUserAlreadyExists
		}, nil); err != nil {
			ctx.PrintError(w, err)
			return
		}

		//dont create a token on registration

		if R.a.OnUserSignUp != nil {
			R.a.OnUserSignUp(ctx, u.User)
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

		var u user
		err := datastore.RunInTransaction(ctx, func(tc context.Context) error {
			err := datastore.Get(tc, ctx.UserKey(), &u)
			if err != nil {
				if err == datastore.ErrNoSuchEntity {
					return errors.ErrUserDoesNotExist
				}
				return err
			}
			u.User.EmailVerified = true
			_, err = datastore.Put(tc, ctx.UserKey(), &u)
			return err
		}, nil)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		if R.a.OnUserVerified != nil {
			R.a.OnUserVerified(ctx, u.User)
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

		var u user
		if err = datastore.RunInTransaction(ctx, func(tc context.Context) error {
			err := datastore.Get(tc, ctx.UserKey(), &u)
			if err != nil {
				if err == datastore.ErrNoSuchEntity {
					return errors.ErrUserDoesNotExist
				}
				return err
			}

			// check old password
			if err = decrypt(u.Hash, []byte(password)); err != nil {
				return errors.ErrUserPasswordIncorrect
			}

			// create new password hash
			if u.Hash, err = crypt([]byte(newPassword)); err != nil {
				return err
			}

			_, err = datastore.Put(tc, ctx.UserKey(), &u)
			return err
		}, nil); err != nil {
			ctx.PrintError(w, err)
			return
		}

		ctx.Print(w, "ok")
	}
}

// todo:
func updateProfile(R *Route) http.HandlerFunc {
	type InputUser struct {
		Email      string `json:"email"`
		Name       string `json:"name"`
		GivenName  string `json:"given_name"`
		FamilyName string `json:"family_name"`
		MiddleName string `json:"middle_name"`
		Nickname   string `json:"nickname"`
		Picture    string `json:"picture"` // profile picture URL
		Website    string `json:"website"` // website URL
		Locale     string `json:"locale"`  // locale
		// is not added to JWT and is private to user
		DateOfBirth    time.Time       `json:"date_of_birth"`
		PlaceOfBirth   Address         `json:"place_of_birth"`
		Title          string          `json:"title"`
		Address        Address         `json:"address"`
		Address2       Address         `json:"address_2"`
		Company        Company         `json:"company"`
		Contact        Contact         `json:"contact"`
		SocialProfiles []SocialProfile `json:"social_profiles"`
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
		var u user
		if err = datastore.RunInTransaction(ctx, func(tc context.Context) error {
			// get user
			err := datastore.Get(ctx, ctx.UserKey(), &u)
			if err != nil {
				return err
			}

			src := reflect.ValueOf(inputUser)
			dst := reflect.ValueOf(u.User)
			for i := 0; i < src.Type().NumField(); i++ {
				srcFieldType := src.Type().Field(i)
				srcField := src.FieldByName(srcFieldType.Name)
				dstField := dst.FieldByName(srcFieldType.Name)
				if dstField.IsValid() && dstField.CanSet() {
					dstField.Set(srcField)
				}
			}
			u.User = dst.Interface().(User)

			_, err = datastore.Put(ctx, ctx.UserKey(), &u)
			return err
		}, nil); err != nil {
			ctx.PrintError(w, err)
			return
		}
		ctx.Print(w, &u.User)
	}
}
