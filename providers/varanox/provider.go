package varanox

import (
	"errors"
	"github.com/ales6164/apis/iam"
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
	*iam.IAM
	iam.Provider
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

func (p *Provider) ConfigAuth(a *iam.IAM) {
	p.IAM = a
}

func (p *Provider) TrustProvidedEmail() bool {
	return true
}

func (p *Provider) Connect(ctx iam.Context) {
	body := ctx.Body()

	email, _ := jsonparser.GetString(body, "email")
	secret, _ := jsonparser.GetString(body, "secret")

	if len(email) == 0 {
		ctx.PrintError(ErrEmailUndefined.Error(), http.StatusBadRequest)
		return
	} else if !govalidator.IsEmail(email) || len(email) > 128 || len(email) < 5 {
		ctx.PrintError(ErrInvalidEmail.Error(), http.StatusBadRequest)
		return
	}

	if len(secret) > 256 {
		ctx.PrintError(ErrPasswordTooLong.Error(), http.StatusBadRequest)
		return
	} else if len(secret) < 6 {
		ctx.PrintError(ErrPasswordTooShort.Error(), http.StatusBadRequest)
		return
	}

	identity, err := p.IAM.Connect(ctx, p, email, secret)
	if err != nil {
		ctx.PrintError(err.Error(), http.StatusInternalServerError)
		return
	}

	// create session
	session, err := p.IAM.NewSession(ctx, p, identity)
	if err != nil {
		ctx.PrintError(err.Error(), http.StatusConflict)
		return
	}

	p.IAM.PrintResponse(session)
}

func (p *Provider) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := mux.Vars(r)["path"]
	ctx := p.IAM.NewContext(w, r)

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
