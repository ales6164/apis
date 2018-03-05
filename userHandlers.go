package apis

import (
	"net/http"
	"github.com/asaskevich/govalidator"
	"google.golang.org/appengine/datastore"
	"strings"
	"golang.org/x/net/context"
	"errors"
	"net/url"
	"strconv"
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

type UserGroup string

const (
	public UserGroup = "public"
	admin  UserGroup = "admin"
)

type User struct {
	Email string `json:"email"`
	Group string `json:"group"`
}

type user struct {
	Hash  []byte `datastore:"hash,noindex" json:"-"`
	Email string `datastore:"email" json:"-"`
	Group string `datastore:"group" json:"-"`
}

func checkCallback(v string) (*url.URL, error) {
	if len(v) == 0 {
		return nil, ErrCallbackUndefined
	}

	if !govalidator.IsURL(v) {
		return nil, ErrInvalidCallback
	}

	return url.ParseRequestURI(v)
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
// Allows user login for provided user group
func (a *Apis) AuthLoginHandler(userGroup ...UserGroup) http.HandlerFunc {
	var allowedGroups = map[string]bool{}
	for _, group := range userGroup {
		allowedGroups[string(group)] = true
	}
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := a.NewContext(r)
		if r.Method != http.MethodPost {
			ctx.PrintError(w, ErrPageNotFound)
			return
		}

		email, password, callback := r.FormValue("email"), r.FormValue("password"), r.FormValue("callback")

		callbackURL, err := checkCallback(callback)
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

		// check if user has allowed user group
		if _, hasAllowedUserGroup := allowedGroups[user.Group]; !hasAllowedUserGroup {
			ctx.PrintError(w, ErrForbidden)
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

		query := url.Values{}
		query.Set("key", signedToken.Id)
		query.Set("expiresAt", strconv.Itoa(int(signedToken.ExpiresAt)))

		callbackURL.Fragment = ""
		callbackURL.RawQuery = query.Encode()

		http.Redirect(w, r, callbackURL.String(), http.StatusTemporaryRedirect)

		//ctx.PrintAuth(w, user, signedToken)
	}
}

// Allows user registration and assigns provided user groups
func (a *Apis) AuthRegistrationHandler(userGroup UserGroup) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := a.NewContext(r)
		if r.Method != http.MethodPost {
			ctx.PrintError(w, ErrPageNotFound)
			return
		}

		email, password, callback := r.FormValue("email"), r.FormValue("password"), r.FormValue("callback")

		_, err := checkCallback(callback)
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
			Group: string(userGroup),
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
