package apis

import (
	"net/http"
	"github.com/asaskevich/govalidator"
	"google.golang.org/appengine/datastore"
	"strings"
	"golang.org/x/net/context"
	"errors"
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

type User struct {
	Email   string                 `json:"email"`
	Group   string                 `json:"group"`
}

type user struct {
	Hash    []byte         `datastore:"hash,noindex" json:"-"`
	Email   string         `datastore:"email" json:"email"`
	Role    string         `datastore:"role" json:"role"`
	Profile *datastore.Key `datastore:"profile" json:"profile"`
}

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
		user := new(user)
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
		err = decrypt(user.Hash, []byte(password))
		if err != nil {
			ctx.PrintError(w, ErrUserPasswordIncorrect)
			// todo: log and report
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

		ctx.PrintAuth(w, user, signedToken)
	}
}

// Allows user registration and assigns provided user groups
func (a *Apis) AuthRegistrationHandler(role Role) http.HandlerFunc {
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

		// create password hash
		hash, err := crypt([]byte(password))
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		// create User
		user := &user{
			Email: email,
			Hash:  hash,
			Role:  string(role),
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

		ctx.PrintAuth(w, user, signedToken)
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
		user := new(user)
		err := datastore.Get(ctx, ctx.UserKey, user)
		if err != nil {
			ctx.PrintError(w, ErrForbidden)
			return
		}

		if user.Profile == nil {
			ctx.PrintResult(w, ErrUserProfileDoesNotExist)
			return
		}

		h, err := k.Get(ctx, user.Profile)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		ctx.PrintResult(w, h.Output())
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

		datastore.RunInTransaction(ctx, func(tc context.Context) error {
			// get user
			user := new(user)
			err := datastore.Get(ctx, ctx.UserKey, user)
			if err != nil {
				return err
			}

			if user.Profile != nil {
				err = profile.Get(user.Profile)
				if err != nil {
					return err
				}
				err = profile.Update(user.Profile)
				if err != nil {
					return err
				}
			} else {
				key, err := profile.Add(ctx.UserKey)
				if err != nil {
					return err
				}

				user.Profile = key

				_, err = datastore.Put(ctx, ctx.UserKey, user)
				if err != nil {
					return err
				}
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
