package emailpassword

import (
	"errors"
	"github.com/ales6164/apis"
	"github.com/asaskevich/govalidator"
	"github.com/buger/jsonparser"
	"golang.org/x/net/context"
	"google.golang.org/appengine/datastore"
	"net/http"
	"github.com/ales6164/apis/kind"
)

var (
	ErrEmailUndefined    = errors.New("email undefined")
	ErrPasswordUndefined = errors.New("password undefined")
	ErrInvalidCallback   = errors.New("callback is not a valid URL")
	ErrInvalidEmail      = errors.New("email is not valid")
	ErrPasswordTooLong   = errors.New("password must be exactly or less than 256 characters long")
	ErrPasswordTooShort  = errors.New("password must be at least 6 characters long")
)

type Provider struct {
	*Config
	*apis.Auth
	apis.Provider
}

type Config struct {
}

func New(config *Config) *Provider {
	if config == nil {
		config = &Config{}
	}
	return &Provider{
		Config: config,
	}
}

func (p *Provider) GetName() string {
	return "emailpassword"
}

func (p *Provider) ConfigAuth(a *apis.Auth) {
	p.Auth = a
}

func (p *Provider) Login(ctx apis.Context) {
	body := ctx.Body()

	email, _ := jsonparser.GetString(body, "email")
	password, _ := jsonparser.GetString(body, "password")

	if len(email) == 0 {
		ctx.PrintError(ErrEmailUndefined.Error(), http.StatusBadRequest)
		return
	} else if !govalidator.IsEmail(email) || len(email) > 128 || len(email) < 5 {
		ctx.PrintError(ErrInvalidEmail.Error(), http.StatusBadRequest)
		return
	}

	if len(password) > 256 {
		ctx.PrintError(ErrPasswordTooLong.Error(), http.StatusBadRequest)
		return
	} else if len(password) < 6 {
		ctx.PrintError(ErrPasswordTooShort.Error(), http.StatusBadRequest)
		return
	}

	identity, err := p.GetIdentity(ctx, p, email, password)
	if err != nil {
		ctx.PrintError(err.Error(), http.StatusInternalServerError)
		return
	}

	user, err := identity.GetUser(ctx)
	if err != nil {
		ctx.PrintError(err.Error(), http.StatusInternalServerError)
		return
	}

	// create session
	session, err := p.NewSession(ctx, identity.Id, identity.User, user.Roles...)

	signedToken, err := p.Auth.SignedToken(session)
	if err != nil {
		ctx.PrintError(err.Error(), http.StatusInternalServerError)
		return
	}

	ctx.PrintJSON(apis.AuthResponse{
		User: user,
		Token: apis.Token{
			Id:        signedToken,
			ExpiresAt: session.ExpiresAt.Unix(),
		},
	}, http.StatusOK)
}

func (p *Provider) Logout(ctx apis.Context) {

}

func (p *Provider) Register(ctx apis.Context) {
	body := ctx.Body()

	email, _ := jsonparser.GetString(body, "email")
	password, _ := jsonparser.GetString(body, "password")

	if len(email) == 0 {
		ctx.PrintError(ErrEmailUndefined.Error(), http.StatusBadRequest)
		return
	} else if !govalidator.IsEmail(email) || len(email) > 128 || len(email) < 5 {
		ctx.PrintError(ErrInvalidEmail.Error(), http.StatusBadRequest)
		return
	}

	if len(password) > 256 {
		ctx.PrintError(ErrPasswordTooLong.Error(), http.StatusBadRequest)
		return
	} else if len(password) < 6 {
		ctx.PrintError(ErrPasswordTooShort.Error(), http.StatusBadRequest)
		return
	}

	var userDoc kind.Doc
	var user *apis.User
	var session *apis.Session
	err := datastore.RunInTransaction(ctx, func(tc context.Context) error {
		// connect identity to account
		var err error
		userDoc, err = p.CreateUser(tc, email, false)
		if err != nil {
			return err
		}
		user = apis.UserKind.Data(userDoc).(*apis.User)
		identity, err := p.CreateIdentity(ctx, p, userDoc.Key(), password)
		if err != nil {
			return err
		}
		// create session
		session, err = p.NewSession(ctx, identity.Id, userDoc.Key(), user.Roles...)
		return err
	}, &datastore.TransactionOptions{XG: true})
	if err != nil {
		ctx.PrintError(err.Error(), http.StatusConflict)
		return
	}

	signedToken, err := p.Auth.SignedToken(session)
	if err != nil {
		ctx.PrintError(err.Error(), http.StatusInternalServerError)
		return
	}

	ctx.PrintJSON(apis.AuthResponse{
		User: user,
		Token: apis.Token{
			Id:        signedToken,
			ExpiresAt: session.ExpiresAt.Unix(),
		},
	}, http.StatusOK)
}

func (p *Provider) Callback(ctx apis.Context) {

}

func (p *Provider) ServeHTTP(ctx apis.Context) {

}
