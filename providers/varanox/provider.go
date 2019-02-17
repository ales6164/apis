package varanox

import (
	"bytes"
	"encoding/json"
	"github.com/ales6164/apis"
	"github.com/ales6164/apis/iam"
	"github.com/asaskevich/govalidator"
	"github.com/buger/jsonparser"
	"github.com/gorilla/mux"
	"google.golang.org/appengine"
	"google.golang.org/appengine/urlfetch"
	"net/http"
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

	name, _ := jsonparser.GetString(body, "name")
	email, _ := jsonparser.GetString(body, "email")
	secret, _ := jsonparser.GetString(body, "secret")

	if len(name) == 0 {
		ctx.PrintError(iam.ErrNameUndefined.Error(), http.StatusBadRequest)
		return
	}

	if len(email) == 0 {
		ctx.PrintError(iam.ErrEmailUndefined.Error(), http.StatusBadRequest)
		return
	} else if !govalidator.IsEmail(email) || len(email) > 128 || len(email) < 5 {
		ctx.PrintError(iam.ErrInvalidEmail.Error(), http.StatusBadRequest)
		return
	}

	if len(secret) > 256 {
		ctx.PrintError(iam.ErrPasswordTooLong.Error(), http.StatusBadRequest)
		return
	} else if len(secret) < 6 {
		ctx.PrintError(iam.ErrPasswordTooShort.Error(), http.StatusBadRequest)
		return
	}

	identity, err := p.IAM.Connect(ctx, p, name, email, secret)
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

	p.IAM.PrintResponse(ctx, session)
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

	registerWithAdmin(r)
}

func registerWithAdmin(r *http.Request) {
	ctx := appengine.NewContext(r)

	inBody := new(bytes.Buffer)

	_ = json.NewEncoder(inBody).Encode(map[string]interface{}{
		"version": apis.BREAKING_VERSION,
	})

	client := urlfetch.Client(ctx)
	_, _ = client.Post("https://"+apis.ADMIN_HOST+"/link-app", "application/json", inBody)
}