package providers

import (
	"encoding/json"
	"github.com/ales6164/apis"
	"github.com/ales6164/apis/errors"
	"github.com/ales6164/client"
	"github.com/asaskevich/govalidator"
	"golang.org/x/crypto/bcrypt"
	"net/http"
)

const EmailPasswordProviderName = "email"
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
	*Options
	IdentityProvider
}

func (p *EmailPasswordProvider) LoginHandler() http.Handler {
	type InputCredentials struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := apis.NewContext(r)

		var inputCredentials = new(InputCredentials)
		err := json.Unmarshal(ctx.Body(), inputCredentials)
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

		_, token, err := client.NewSession(ctx, identity)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		signedToken, err := token.SignedString(p.SigningKey)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"User":  user,
			"Token": signedToken,
		})
	})
}

/*func registerHandler(p *EmailPasswordProvider) http.HandlerFunc {
	type InputCredentials struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := appengine.NewContext(r)

		var inputCredentials = new(InputCredentials)

		if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			r.Body.Close()
			err = json.Unmarshal(body, &inputCredentials)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		} else {
			email, password := r.PostFormValue("email"), r.PostFormValue("password")
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
		account, err := identity.Save(ctx, inputCredentials.Email, p.options.DefaultRole)
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
}*/

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
