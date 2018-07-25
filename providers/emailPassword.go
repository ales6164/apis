package providers

import (
	"net/http"
	"strings"
	"github.com/asaskevich/govalidator"
	"github.com/ales6164/apis/errors"
	"github.com/gorilla/mux"
	"google.golang.org/appengine"
	"encoding/json"
	"io/ioutil"
	"golang.org/x/crypto/bcrypt"
)

const emailPasswordProvider = "email"

const COST = 12

var (
	ErrEmailUndefined    = errors.New("email undefined")
	ErrPasswordUndefined = errors.New("password undefined")
	ErrInvalidCallback   = errors.New("callback is not a valid URL")
	ErrInvalidEmail      = errors.New("email is not valid")
	ErrPasswordTooLong   = errors.New("password must be exactly or less than 128 characters long")
	ErrPasswordTooShort  = errors.New("password must be at least 6 characters long")
)

type EmailPasswordProvider struct {
	authority Authority
	options   *Options
}

func (p *EmailPasswordProvider) Apply(r *mux.Router, a Authority) {
	p.authority = a
	r.HandleFunc("/login", loginHandler(p)).Methods(http.MethodPost)
	r.HandleFunc("/register", registerHandler(p)).Methods(http.MethodPost)
}

func (p *EmailPasswordProvider) Authority() Authority {
	return p.authority
}
func (p *EmailPasswordProvider) Options() *Options {
	return p.options
}

func (p *EmailPasswordProvider) Name() string {
	return emailPasswordProvider
}

func loginHandler(p *EmailPasswordProvider) http.HandlerFunc {
	type InputCredentials struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := appengine.NewContext(r)

		var inputCredentials *InputCredentials

		if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			r.Body.Close()
			err = json.Unmarshal(body, inputCredentials)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		} else {
			email, password := r.FormValue("email"), r.FormValue("password")
			inputCredentials = &InputCredentials{
				Email:    email,
				Password: password,
			}
		}

		err := checkEmail(inputCredentials.Email)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		err = checkPassword(inputCredentials.Password)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		identity, err := GetIdentity(ctx, p, inputCredentials.Email)
		if err != nil {
			http.Error(w, errors.ErrUserDoesNotExist.Error(), http.StatusInternalServerError)
			return
		}

		err = decrypt(identity.Secret, []byte(inputCredentials.Password))
		if err != nil {
			http.Error(w, errors.ErrUserDoesNotExist.Error(), http.StatusInternalServerError)
			return
		}

		account, err := p.Authority().GetAccount(ctx, identity.AccountKey)
		if err != nil {
			http.Error(w, errors.ErrUserDoesNotExist.Error(), http.StatusInternalServerError)
			return
		}

		signedToken, err := p.Authority().SignToken(ctx, account)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(Output{
			Token: signedToken,
			User:  account.User,
		})
	}
}

func registerHandler(p *EmailPasswordProvider) http.HandlerFunc {
	type InputCredentials struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := appengine.NewContext(r)

		var inputCredentials *InputCredentials

		if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			r.Body.Close()
			err = json.Unmarshal(body, inputCredentials)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		} else {
			email, password := r.FormValue("email"), r.FormValue("password")
			inputCredentials = &InputCredentials{
				Email:    email,
				Password: password,
			}
		}

		err := checkEmail(inputCredentials.Email)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		err = checkPassword(inputCredentials.Password)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// create password hash
		hash, err := crypt([]byte(inputCredentials.Password))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		identity := NewIdentity(p, hash)
		account, err := identity.Save(ctx, inputCredentials.Email)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		signedToken, err := p.Authority().SignToken(ctx, account)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(Output{
			Token: signedToken,
			User:  account.User,
		})
	}
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

func decrypt(hash []byte, password []byte) error {
	defer clear(password)
	return bcrypt.CompareHashAndPassword(hash, password)
}

func crypt(password []byte) ([]byte, error) {
	defer clear(password)
	return bcrypt.GenerateFromPassword(password, COST)
}

func clear(b []byte) {
	for i := 0; i < len(b); i++ {
		b[i] = 0
	}
}
