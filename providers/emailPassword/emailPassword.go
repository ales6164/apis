package emailpassword

import (
	"encoding/json"
	"errors"
	"github.com/ales6164/apis"
	"github.com/asaskevich/govalidator"
	"github.com/buger/jsonparser"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/net/context"
	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"io/ioutil"
	"net/http"
)

var (
	EmailPasswordProviderName     = "emailpassword"
	EmailPasswordProviderKindName = "_provider_emailpassword"
)

var (
	ErrEmailUndefined    = errors.New("email undefined")
	ErrPasswordUndefined = errors.New("password undefined")
	ErrInvalidCallback   = errors.New("callback is not a valid URL")
	ErrInvalidEmail      = errors.New("email is not valid")
	ErrPasswordTooLong   = errors.New("password must be exactly or less than 128 characters long")
	ErrPasswordTooShort  = errors.New("password must be at least 6 characters long")
)

type emailPassword struct {
	*Options
	*apis.Auth
}

type Options struct {
	UserKind *apis.Kind
	Cost     int
}

type Entry struct {
	Account *datastore.Key
	Email   string `datastore:",noindex"`
	Hash    []byte `datastore:",noindex"`
}

func New(a *apis.Auth, options *Options) *emailPassword {
	return &emailPassword{
		Options: options,
		Auth:    a,
	}
}

/*func (p *emailPassword) SignInHandler(rp client.RoleProvider) http.Handler {
	type InputCredentials struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := appengine.NewContext(r)

		body, _ := ioutil.ReadAll(r.Body)
		r.Body.Close()

		var inputCredentials = new(InputCredentials)
		err := json.Unmarshal(body, inputCredentials)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		err = checkEmail(inputCredentials.Email)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		identity, err := client.GetIdentity(ctx, EmailPasswordProviderName, inputCredentials.Email)
		if err != nil {
			http.Error(w, errors.ErrUserDoesNotExist.Error(), http.StatusBadRequest)
			return
		}

		err = decrypt(identity.Secret, []byte(inputCredentials.Password))
		if err != nil {
			http.Error(w, errors.ErrUserPasswordIncorrect.Error(), http.StatusBadRequest)
			return
		}

		user, err := identity.GetUser(ctx)
		if err != nil {
			http.Error(w, errors.ErrUserDoesNotExist.Error(), http.StatusInternalServerError)
			return
		}

		ss, token, err := client.NewSession(ctx, identity, rp, user.Scopes...)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		signedToken, err := token.SignedString(p.SigningKey)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(&Response{
			User: user,
			Token: &Token{
				Id:        signedToken,
				ExpiresAt: ss.ExpiresAt,
			},
		})
	})
}*/

func (p *emailPassword) SignUpHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := appengine.NewContext(r)

		body, _ := ioutil.ReadAll(r.Body)
		r.Body.Close()

		email, _ := jsonparser.GetString(body, "email")
		password, _ := jsonparser.GetString(body, "password")
		userData, _, _, _ := jsonparser.Get(body, "user")

		err := p.checkEmail(email)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		err = p.checkPassword(password)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		userHolder := p.UserKind.NewHolder(nil)
		err = userHolder.Parse(userData)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		var session *apis.Session
		err = datastore.RunInTransaction(ctx, func(tc context.Context) error {
			// check if user exists
			providerKey := datastore.NewKey(tc, EmailPasswordProviderKindName, email, 0, nil)
			emailPasswordEntry := new(Entry)
			err = datastore.Get(tc, providerKey, emailPasswordEntry)
			if err == nil {
				return errors.New("user already exists")
			}
			if err != nil && err != datastore.ErrNoSuchEntity {
				return err
			}

			// create password hash
			emailPasswordEntry.Hash, err = p.crypt([]byte(password))
			if err != nil {
				return err
			}

			// connect identity to account
			account, err := p.ConnectUser(tc, providerKey, email, userHolder)
			if err != nil {
				return err
			}

			emailPasswordEntry.Account = account.Id

			// create emailPassword entry
			providerKey, err = datastore.Put(tc, providerKey, emailPasswordEntry)
			if err != nil {
				return err
			}

			// create session
			session, err = p.NewSession(ctx, providerKey, account.Id, account.Scopes...)
			return err
		}, &datastore.TransactionOptions{XG: true})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		signedToken, err := p.Auth.SignedToken(session)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(apis.AuthResponse{
			User: userHolder.GetValue(),
			Token: apis.Token{
				Id:        signedToken,
				ExpiresAt: session.ExpiresAt.Unix(),
			},
		})
	})
}

func (p *emailPassword) checkEmail(v string) error {
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

func (p *emailPassword) checkPassword(v string) error {
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

func decrypt(hash []byte, password []byte) error {
	defer clear(password)
	return bcrypt.CompareHashAndPassword(hash, password)
}

func (p *emailPassword) crypt(password []byte) ([]byte, error) {
	defer clear(password)
	return bcrypt.GenerateFromPassword(password, p.Cost)
}

func clear(b []byte) {
	for i := 0; i < len(b); i++ {
		b[i] = 0
	}
}
