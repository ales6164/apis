package emailpassword

import (
	"errors"
	"github.com/ales6164/apis"
	"github.com/asaskevich/govalidator"
	"github.com/buger/jsonparser"
	"github.com/gorilla/mux"
	"net/http"
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

func (p *Provider) Name() string {
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

	// create user
	identity, err := p.Auth.GetIdentity(ctx, p, email, password)
	if err != nil {
		ctx.PrintError(err.Error(), http.StatusConflict)
		return
	}

	// create session
	session, err := p.NewSession(ctx, p.Name(), identity.IdentityKey, identity.UserKey, identity.User.Roles...)
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
		User: identity.User,
		Token: apis.Token{
			Id:        signedToken,
			ExpiresAt: session.ExpiresAt.Unix(),
		},
	}, http.StatusOK)
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

	// create user
	identity, err := p.Auth.CreateUser(ctx, p, email, false, password)
	if err != nil {
		ctx.PrintError(err.Error(), http.StatusConflict)
		return
	}

	// create session
	session, err := p.NewSession(ctx, p.Name(), identity.IdentityKey, identity.UserKey, identity.User.Roles...)
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
		User: identity.User,
		Token: apis.Token{
			Id:        signedToken,
			ExpiresAt: session.ExpiresAt.Unix(),
		},
	}, http.StatusOK)
}

func (p *Provider) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := mux.Vars(r)["path"]
	ctx := p.Auth.NewContext(w, r)
	if len(path) > 0 {
		switch path {
		case "login":
			p.Login(ctx)
		case "register":
			p.Register(ctx)
		}
	}
}
