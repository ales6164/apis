package varanox

import (
	"errors"
	"github.com/ales6164/apis"
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
	*apis.Auth
	apis.Provider
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
	p.Auth = a
}

func (p *Provider) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := mux.Vars(r)["path"]
	ctx := p.Auth.NewContext(w, r)
	if len(path) > 0 {
		switch path {
		case "connect":
			p.Connect(ctx)
		}
	}
}

