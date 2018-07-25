package providers

import (
	"github.com/gorilla/mux"
)

type IdentityProvider interface {
	Apply(r *mux.Router, auth Authority)
	Name() string
	Options() *Options
	Authority() Authority
}

type Options struct {
	DefaultRole string
}

func WithEmailPasswordProvider(options *Options) *EmailPasswordProvider {
	p := new(EmailPasswordProvider)
	p.options = options
	return p
}
