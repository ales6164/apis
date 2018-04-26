package apis

import (
	"github.com/asaskevich/govalidator"
	"net/http"
	"github.com/ales6164/apis/errors"
	"google.golang.org/appengine/datastore"
	"strings"
	"encoding/json"
	"golang.org/x/net/context"
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


func getAnonymousToken(R *Route) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ok, ctx := R.NewContext(r).Authenticate()
		if ok {
			ctx.PrintError(w, errors.New("user already authenticated"))
			return
		}


	}
}

func getUserHandler(R *Route) http.HandlerFunc {
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

		id := r.FormValue("id")
		keyId, err := datastore.DecodeKey(id)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		// get user
		usr := new(User)
		err = datastore.Get(ctx, keyId, usr)
		if err != nil {
			if err == datastore.ErrNoSuchEntity {
				ctx.PrintError(w, errors.ErrUserDoesNotExist)
				return
			}
			ctx.PrintError(w, err)
			return
		}

		ctx.Print(w, usr)
	}
}

func getUsersHandler(R *Route) http.HandlerFunc {
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

		var hs []*User
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
}

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

		// get user
		var users []*User
		keys, err := datastore.NewQuery("_user").Filter("email =", email).Limit(1).GetAll(ctx, &users)
		if err != nil || len(users) == 0 {
			if err == datastore.ErrNoSuchEntity {
				ctx.PrintError(w, errors.ErrUserDoesNotExist)
				return
			}
			ctx.PrintError(w, err)
			return
		}

		users[0].Id = keys[0]

		// decrypt hash
		err = decrypt(users[0].Hash, []byte(password))
		if err != nil {
			ctx.PrintError(w, errors.ErrUserPasswordIncorrect)
			// todo: log and report
			return
		}

		// create a token
		token := NewToken(users[0])

		// sign the new token
		signedToken, err := R.a.SignToken(token)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		ctx.PrintAuth(w, signedToken, users[0])
	}
}

func registrationHandler(R *Route, role Role) http.HandlerFunc {
	type InputUser struct {
		Email    string                 `json:"email"`
		Password string                 `json:"password"`
		Meta     map[string]interface{} `json:"meta"`
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

		// create User
		usr := &User{
			Email: inputUser.Email,
			Hash:  hash,
			Role:  string(role),
			Meta:  inputUser.Meta,
		}

		if usr.Meta == nil {
			usr.Meta = map[string]interface{}{}
		}

		if _, ok := usr.Meta["lang"]; !ok {
			usr.Meta["lang"] = ctx.Language()
		}

		// check if it already exists and try to log in if it does
		var users []*User
		_, err = datastore.NewQuery("_user").Filter("email =", inputUser.Email).Limit(1).GetAll(ctx, &users)
		if err == nil || len(users) > 0 {
			ctx.PrintError(w, errors.ErrUserAlreadyExists)
			return
		}

		// store the new user
		userKey := datastore.NewIncompleteKey(ctx, "_user", nil)
		userKey, err = datastore.Put(ctx, userKey, usr)

		usr.Id = userKey

		// create a token
		token := NewToken(usr)

		// sign the new token
		signedToken, err := R.a.SignToken(token)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		if R.a.options.RequireEmailConfirmation && usr.HasConfirmedEmail {
			ctx.PrintAuth(w, signedToken, usr)
		} else {
			ctx.Print(w, "success")
		}

		if R.a.OnUserSignUp != nil {
			R.a.OnUserSignUp(ctx, *usr, *signedToken)
		}
	}
}

func confirmEmailHandler(R *Route) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ok, ctx := R.NewContext(r).Authenticate()

		if !ok {
			ctx.PrintError(w, errors.ErrUnathorized)
			return
		}

		callback := r.FormValue("callback")
		if !govalidator.IsURL(callback) {
			ctx.PrintError(w, ErrInvalidCallback)
			return
		}

		usr := new(User)
		// update User
		err := datastore.RunInTransaction(ctx, func(tc context.Context) error {
			err := datastore.Get(tc, ctx.UserKey, usr)
			if err != nil {
				if err == datastore.ErrNoSuchEntity {
					return errors.ErrUserDoesNotExist
				}
				return err
			}
			usr.HasConfirmedEmail = true
			_, err = datastore.Put(tc, ctx.UserKey, usr)
			return err
		}, nil)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		usr.Id = ctx.UserKey

		// create a token
		token := NewToken(usr)

		// sign the new token
		signedToken, err := R.a.SignToken(token)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		if R.a.OnUserVerified != nil {
			R.a.OnUserVerified(ctx, *usr, *signedToken)
		}

		http.Redirect(w, r, callback, http.StatusTemporaryRedirect)
	}
}

func changePasswordHandler(R *Route) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ok, ctx := R.NewContext(r).Authenticate()

		if !ok {
			ctx.PrintError(w, errors.ErrUnathorized)
			return
		}

		password, newPassword := r.FormValue("password"), r.FormValue("newPassword")

		err := checkPassword(newPassword)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		usr := new(User)
		// update User
		err = datastore.RunInTransaction(ctx, func(tc context.Context) error {
			err = datastore.Get(tc, ctx.UserKey, usr)
			if err != nil {
				if err == datastore.ErrNoSuchEntity {
					return errors.ErrUserDoesNotExist
				}
				return err
			}

			// decrypt hash
			err = decrypt(usr.Hash, []byte(password))
			if err != nil {
				return errors.ErrUserPasswordIncorrect
			}

			// create new password hash
			usr.Hash, err = crypt([]byte(newPassword))
			if err != nil {
				return err
			}

			_, err := datastore.Put(tc, ctx.UserKey, usr)
			return err
		}, nil)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		usr.Id = ctx.UserKey
		ctx.Print(w, "success")
	}
}

func updateMeta(R *Route) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ok, ctx := R.NewContext(r).Authenticate()
		if !ok {
			ctx.PrintError(w, errors.ErrUnathorized)
			return
		}
		meta := ctx.Body()

		var m map[string]interface{}
		if len(meta) > 0 {
			json.Unmarshal(meta, &m)
		} else {
			ctx.PrintError(w, errors.New("body is empty"))
			return
		}

		// do everything in a transaction
		usr := new(User)
		err := datastore.RunInTransaction(ctx, func(tc context.Context) error {
			// get user
			err := datastore.Get(ctx, ctx.UserKey, usr)
			if err != nil {
				return err
			}
			for k, v := range m {
				usr.SetMeta(k, v)
			}
			_, err = datastore.Put(ctx, ctx.UserKey, usr)
			return err
		}, &datastore.TransactionOptions{XG: true})
		if err != nil {
			ctx.PrintError(w, err)
			return
		}
		usr.Id = ctx.UserKey
		ctx.Print(w, usr)
	}
}
