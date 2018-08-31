package providers

import "github.com/gorilla/mux"

type IdentityProvider interface {
	Apply(r *mux.Router)
}

type Options struct {
	Roles  []string
	Scopes []string
	Cost   int // default is 12
}

func WithEmailPasswordProvider(options *Options) *EmailPasswordProvider {
	p := new(EmailPasswordProvider)
	if options == nil {
		options = new(Options)
	}
	p.Options = options
	return p
}
