package apis

import (
	"net/http"
	"encoding/json"
	"github.com/asaskevich/govalidator"
	"google.golang.org/appengine/datastore"
	"strings"
	"golang.org/x/net/context"
	"errors"
	"os/user"
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
	Hash  []byte `datastore:"hash,noindex" json:"-"`
	Email string `datastore:"email,noindex" json:"email"`
}

func checkCallback(v string) error {
	if len(v) == 0 {
		return ErrCallbackUndefined
	}

	if !govalidator.IsURL(v) {
		return ErrInvalidCallback
	}

	return nil
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

func (a *Apis) AuthLoginHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := NewContext(r)

		email, password, callback := r.FormValue("email"), r.FormValue("password"), r.FormValue("callback")

		err := checkCallback(callback)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}
		err = checkEmail(email)
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
		userKey := datastore.NewKey(ctx, "User", email, 0, nil)
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
		err = decrypt(user.Hash, []byte(password))
		if err != nil {
			ctx.PrintError(w, ErrUserPasswordIncorrect)
			// todo: log and report
			return
		}

		// create a token
		token := NewToken(user.Email)

		// sign the new token
		signedToken, err := a.SignToken(token)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		ctx.PrintAuth(w, user, signedToken)
	}
}

func (a *Apis) AuthRegistrationHandler() http.HandlerFunc {
	type Input struct {
		Email     string `json:"email"`
		Password  string `json:"password"`
		FirstName string `json:"firstName"`
		LastName  string `json:"lastName"`
		Photo     string `json:"photo"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		ctx := instance.NewContext(r)

		var input Input
		err := json.Unmarshal(ctx.Body(), &input)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		input.Email = strings.ToLower(input.Email)

		// verify input
		if !govalidator.IsEmail(input.Email) || len(input.Email) < 6 || len(input.Email) > 64 {
			ctx.PrintError(w, instance.ErrInvalidEmail)
			return
		}
		if len(input.Password) < 6 || len(input.Password) > 128 {
			ctx.PrintError(w, instance.ErrPasswordLength)
			return
		}
		if len(input.Photo) > 0 && !govalidator.IsURL(input.Photo) {
			ctx.PrintError(w, instance.ErrPhotoInvalidFormat)
			return
		}

		// create password hash
		hash, err := crypt([]byte(input.Password))
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		// create User
		user := &user.User{
			Email:     input.Email,
			Hash:      hash,
			Photo:     input.Photo,
			FirstName: input.FirstName,
			LastName:  input.LastName,
		}

		err = datastore.RunInTransaction(ctx, func(tc context.Context) error {
			userKey := datastore.NewKey(tc, "User", user.Email, 0, nil)
			err := datastore.Get(tc, userKey, &datastore.PropertyList{})
			if err != nil {
				if err == datastore.ErrNoSuchEntity {
					// register
					_, err := datastore.Put(tc, userKey, user)
					return err
				}
				return err
			}
			return instance.ErrUserAlreadyExists
		}, nil)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		// create a token
		token := instance.NewToken(user.Email, "")

		// sign the new token
		signedToken, err := a.SignToken(token)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		ctx.PrintAuth(w, user, signedToken)
	}
}
