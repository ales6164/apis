package varanox

import (
	"errors"
	"github.com/ales6164/apis"
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
	return "varanox"
}

func (p *Provider) ConfigAuth(a *apis.Auth) {
	p.Auth = a
}

func (p *Provider) TrustProvidedEmail() bool {
	return true
}

func (p *Provider) Connect(ctx apis.Context) {
	body := ctx.Body()

	email, _ := jsonparser.GetString(body, "email")
	secret, _ := jsonparser.GetString(body, "secret")

	identity, err := p.Auth.Connect(ctx, p, email, secret)
	if err != nil {
		ctx.PrintError(err.Error(), http.StatusInternalServerError)
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

	if r.Header.Get("X-Appengine-Inbound-Appid") != "admin-si" {
		ctx.PrintError(http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}

	if len(path) > 0 {
		switch path {
		case "connect":
			p.Connect(ctx)
		}
	}
}
