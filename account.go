package apis

import (
	"net/http"
	"strings"
	"google.golang.org/appengine/datastore"
	"github.com/asaskevich/govalidator"
	"golang.org/x/net/context"

	"time"
	"github.com/ales6164/apis/kind"
	"reflect"
	"github.com/ales6164/apis/errors"
)

type account struct {
	Id             *datastore.Key `datastore:"-" apis:"id" json:"-"`
	CreatedAt      time.Time      `apis:"createdAt" json:"-"`
	UpdatedAt      time.Time      `apis:"updatedAt" json:"-"`
	UserId         *datastore.Key `json:"-"`
	Email          string         `json:"email"`
	EmailConfirmed bool           `json:"emailConfirmed"`
	User           interface{}    `datastore:"-" json:"user,omitempty"`
	Roles          []string       `json:"roles"`
	Password       string         `datastore:"-" json:"password,omitempty"`
	NewPassword    string         `datastore:"-" json:"newPassword,omitempty"`
	Token          Token          `datastore:"-" json:"token,omitempty"`
	Hash           []byte         `datastore:",noindex" json:"-"`
}

var accountKind = kind.New(reflect.TypeOf(account{}), &kind.Options{
	Name: "_acc",
})

func loginHandler(R *Route) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := R.NewContext(r)

		var input = accountKind.NewHolder(nil, nil)

		if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
			if err := input.Parse(ctx.Body()); err != nil {
				ctx.PrintError(w, err)
				return
			}
		} else {
			email, password := r.FormValue("email"), r.FormValue("password")
			input.SetValue(&account{
				Email:    email,
				Password: password,
			})
		}

		var inputAcc = input.Value().(*account)

		err := checkEmail(inputAcc.Email)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		err = checkPassword(inputAcc.Password)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		accKey := accountKind.NewKey(ctx, inputAcc.Email, nil)
		accHolder := accountKind.NewHolder(nil, nil)
		err = accHolder.Get(ctx, accKey)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		var acc = accHolder.Value().(*account)

		if err := decrypt(acc.Hash, []byte(inputAcc.Password)); err != nil {
			ctx.PrintError(w, errors.ErrUserPasswordIncorrect)
			return
		}

		usrHolder := UserKind.NewHolder(nil, nil)
		err = usrHolder.Get(ctx, acc.UserId)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		acc.User = usrHolder.Value()

		signedToken, err := createSession(ctx, acc.UserId, acc.Email, acc.Roles)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		acc.Token = signedToken

		ctx.Print(w, acc)
	}
}

func registrationHandler(R *Route) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := R.NewContext(r)

		var input = accountKind.NewHolder(nil, nil)

		if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
			if err := input.Parse(ctx.Body()); err != nil {
				ctx.PrintError(w, err)
				return
			}
		} else {
			email, password := r.FormValue("email"), r.FormValue("password")
			input.SetValue(&account{
				Email:    email,
				Password: password,
			})
		}

		var inputAcc = input.Value().(*account)
		inputAcc.Email = strings.ToLower(inputAcc.Email)

		if err := checkEmail(inputAcc.Email); err != nil {
			ctx.PrintError(w, err, "check email")
			return
		}

		if err := checkPassword(inputAcc.Password); err != nil {
			ctx.PrintError(w, err, "check password")
			return
		}

		// create password hash
		var err error
		inputAcc.Hash, err = crypt([]byte(inputAcc.Password))
		if err != nil {
			ctx.PrintError(w, err, "crypt")
			return
		}

		usrId, _, err := datastore.AllocateIDs(ctx, UserKind.Name, nil, 1)
		if err != nil {
			ctx.PrintError(w, err, "id usr gen")
			return
		}

		accKey := accountKind.NewKey(ctx, inputAcc.Email, nil)
		usrKey := datastore.NewKey(ctx, UserKind.Name, "", usrId, accKey)

		usrHolder := UserKind.NewHolder(nil, nil)
		usrHolder.SetKey(usrKey)

		accHolder := accountKind.NewHolder(nil, nil)
		accHolder.SetKey(accKey)
		accHolder.SetValue(&account{
			Email:  inputAcc.Email,
			Hash:   inputAcc.Hash,
			Roles:  []string{string(R.a.options.DefaultRole)},
			UserId: usrKey,
			User:   usrHolder.Value(),
		})

		if err := R.trigger(BeforeCreate, ctx, accHolder); err != nil {
			ctx.PrintError(w, err)
			return
		}

		if err = datastore.RunInTransaction(ctx, func(tc context.Context) error {
			var dst datastore.PropertyList
			err := datastore.Get(tc, accKey, &dst)
			if err != nil {
				if err == datastore.ErrNoSuchEntity {
					// register
					_, err = datastore.Put(tc, accKey, accHolder)
					if err != nil {
						return err
					}
					_, err = datastore.Put(tc, usrKey, usrHolder)
					if err != nil {
						return err
					}

					return nil
				}
				return err
			}
			return errors.ErrUserAlreadyExists
		}, nil); err != nil {
			ctx.PrintError(w, err, "reg put err")
			return
		}

		var acc = accHolder.Value().(*account)

		signedToken, err := createSession(ctx, usrKey, acc.Email, acc.Roles)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		acc.Token = signedToken

		if err := R.trigger(AfterCreate, ctx, accHolder); err != nil {
			ctx.PrintError(w, err)
			return
		}

		ctx.Print(w, acc)
	}
}

/*func confirmEmailHandler(R *Route) http.HandlerFunc {
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
}*/

func changePasswordHandler(R *Route) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := R.NewContext(r)
		if !ctx.IsAuthenticated {
			ctx.PrintError(w, errors.ErrForbidden)
			return
		}

		var h = accountKind.NewHolder(nil, nil)

		if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
			if err := h.Parse(ctx.Body()); err != nil {
				ctx.PrintError(w, err)
				return
			}
		} else {
			password, newPassword := r.FormValue("password"), r.FormValue("newPassword")
			h.SetValue(&account{
				Password:    password,
				NewPassword: newPassword,
			})
		}

		var acc = h.Value().(*account)

		err := checkPassword(acc.NewPassword)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		if err = datastore.RunInTransaction(ctx, func(tc context.Context) error {
			var hn = accountKind.NewHolder(nil, nil)
			err := hn.Get(tc, ctx.UserKey())
			if err != nil {
				if err == datastore.ErrNoSuchEntity {
					return errors.ErrUserDoesNotExist
				}
				return err
			}

			curAcc := hn.Value().(*account)

			// check old password
			if err = decrypt(curAcc.Hash, []byte(acc.Password)); err != nil {
				return errors.ErrUserPasswordIncorrect
			}

			// create new password hash
			if curAcc.Hash, err = crypt([]byte(acc.NewPassword)); err != nil {
				return err
			}

			_, err = datastore.Put(tc, hn.GetKey(ctx), curAcc)
			return err
		}, nil); err != nil {
			ctx.PrintError(w, err)
			return
		}

		ctx.Print(w, "ok")
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
