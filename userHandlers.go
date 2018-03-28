package apis

import (
	"net/http"
	"github.com/asaskevich/govalidator"
	"google.golang.org/appengine/datastore"
	"strings"
	"golang.org/x/net/context"
	"errors"
	"encoding/json"
	"github.com/ales6164/apis/kind"
)

var (
	ErrCallbackUndefined = errors.New("callback undefined")
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

// TODO: check auth origins and callback
func (a *Apis) AuthLoginHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := a.NewContext(r)

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
		userKey := datastore.NewKey(ctx, "_user", email, 0, nil)
		user := new(User)
		err = datastore.Get(ctx, userKey, user)
		if err != nil {
			if err == datastore.ErrNoSuchEntity {
				ctx.PrintError(w, ErrUserDoesNotExist)
				return
			}
			ctx.PrintError(w, err)
			return
		}

		// decrypt hash
		err = decrypt(user.hash, []byte(password))
		if err != nil {
			ctx.PrintError(w, ErrUserPasswordIncorrect)
			// todo: log and report
			return
		}

		// get profile
		user.LoadProfile(ctx, a.options.UserProfileKind)

		// create a token
		token := NewToken(user)

		// sign the new token
		signedToken, err := a.SignToken(token)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		ctx.PrintAuth(w, signedToken, user)
	}
}

// Allows user registration and assigns provided user groups
func (a *Apis) AuthRegistrationHandler(role Role) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := a.NewContext(r)

		email, password, meta := r.FormValue("email"), r.FormValue("password"), r.FormValue("meta")

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

		// create password hash
		hash, err := crypt([]byte(password))
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		// create User
		user := &User{
			Email: email,
			hash:  hash,
			Role:  string(role),
		}

		if len(meta) > 0 {
			json.Unmarshal([]byte(meta), &user.Meta)
		}

		err = datastore.RunInTransaction(ctx, func(tc context.Context) error {
			userKey := datastore.NewKey(tc, "_user", user.Email, 0, nil)
			err := datastore.Get(tc, userKey, &datastore.PropertyList{})
			if err != nil {
				if err == datastore.ErrNoSuchEntity {
					// register
					_, err := datastore.Put(tc, userKey, user)
					return err
				}
				return err
			}
			return ErrUserAlreadyExists
		}, nil)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		// create a token
		token := NewToken(user)

		// sign the new token
		signedToken, err := a.SignToken(token)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		ctx.PrintAuth(w, signedToken, user)
	}
}

func (a *Apis) AuthUpdatePasswordHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ok, ctx := a.NewContext(r).Authenticate()

		if !ok {
			ctx.PrintError(w, ErrUnathorized)
			return
		}

		password, newPassword := r.FormValue("password"), r.FormValue("newPassword")

		err := checkPassword(newPassword)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		// check if password is ok
		// get user
		user := new(User)
		err = datastore.Get(ctx, ctx.UserKey, user)
		if err != nil {
			if err == datastore.ErrNoSuchEntity {
				ctx.PrintError(w, ErrUserDoesNotExist)
				return
			}
			ctx.PrintError(w, err)
			return
		}

		// decrypt hash
		err = decrypt(user.hash, []byte(password))
		if err != nil {
			ctx.PrintError(w, ErrUserPasswordIncorrect)
			// todo: log and report
			return
		}

		// create new password hash
		user.hash , err = crypt([]byte(newPassword))
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		// create User
		err = datastore.RunInTransaction(ctx, func(tc context.Context) error {
			userKey := datastore.NewKey(tc, "_user", user.Email, 0, nil)
			err := datastore.Get(tc, userKey, &datastore.PropertyList{})
			if err != nil {
				if err == datastore.ErrNoSuchEntity {
					// register
					_, err := datastore.Put(tc, userKey, user)
					return err
				}
				return err
			}
			return ErrUserAlreadyExists
		}, nil)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}
/*
		// create a token
		token := NewToken(user)

		// sign the new token
		signedToken, err := a.SignToken(token)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}*/

		ctx.Print(w, "success")

		//ctx.PrintAuth(w, signedToken, user)
	}
}

func (a *Apis) AuthUpdateMeta() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ok, ctx := a.NewContext(r).Authenticate()

		if !ok {
			ctx.PrintError(w, ErrUnathorized)
			return
		}

		meta := ctx.Body()

		var m map[string]interface{}
		if len(meta) > 0 {
			json.Unmarshal(meta, &m)
		} else {
			ctx.PrintError(w, errors.New("meta field empty"))
			return
		}

		// do everything in a transaction
		user := new(User)

		err := datastore.RunInTransaction(ctx, func(tc context.Context) error {
			// get user
			err := datastore.Get(ctx, ctx.UserKey, user)
			if err != nil {
				return err
			}

			for k, v := range m {
				user.SetMeta(k, v)
			}

			_, err = datastore.Put(ctx, ctx.UserKey, user)
			return err
		}, &datastore.TransactionOptions{XG: true})
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		ctx.Print(w, user)
	}
}

/**
Profile handlers
 */

func (a *Apis) AuthGetProfile(k *kind.Kind) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ok, ctx := a.NewContext(r).Authenticate()

		if !ok {
			ctx.PrintError(w, ErrUnathorized)
			return
		}

		// get user
		user := new(User)
		err := datastore.Get(ctx, ctx.UserKey, user)
		if err != nil {
			ctx.PrintError(w, ErrForbidden)
			return
		}

		if user.profile == nil {
			ctx.PrintError(w, ErrUserProfileDoesNotExist)
			return
		}

		h, err := user.LoadProfile(ctx, a.options.UserProfileKind)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		ctx.PrintResult(w, h)
	}
}

func (a *Apis) AuthUpdateProfile(k *kind.Kind) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ok, ctx := a.NewContext(r).Authenticate()

		if !ok {
			ctx.PrintError(w, ErrUnathorized)
			return
		}

		// do everything in a transaction

		profile := k.NewHolder(ctx, ctx.UserKey)
		err := profile.ParseInput(ctx.Body())
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		var m map[string]interface{}
		if meta, ok := profile.ParsedInput["meta"].(map[string]interface{}); ok {
			m = meta
		}

		err = datastore.RunInTransaction(ctx, func(tc context.Context) error {
			// get user
			user := new(User)
			err := datastore.Get(ctx, ctx.UserKey, user)
			if err != nil {
				return err
			}

			if user.profile != nil {
				err = profile.Update(user.profile)
				if err != nil {
					return err
				}
			} else {
				key, err := profile.Add(ctx.UserKey)
				if err != nil {
					return err
				}
				user.profile = key
			}

			user.Meta = m

			_, err = datastore.Put(ctx, ctx.UserKey, user)
			if err != nil {
				return err
			}

			return nil
		}, &datastore.TransactionOptions{XG: true})
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		ctx.PrintResult(w, profile.Output())
	}
}
