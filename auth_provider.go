package apis

import (
	"fmt"
	"net/http"
)

type Provider interface {
	Name() string
	ConfigAuth(*Auth)
	http.Handler
}

func (a *Auth) RegisterProvider(provider Provider) {
	name := provider.Name()
	for _, p := range a.providers {
		if p.Name() == name {
			fmt.Printf("warning: auth provider %v already registered", name)
			return
		}
	}

	provider.ConfigAuth(a)
	a.providers = append(a.providers, provider)
}

// GetProvider get provider with name
func (a *Auth) GetProvider(name string) Provider {
	for _, provider := range a.providers {
		if provider.Name() == name {
			return provider
		}
	}
	return nil
}

// GetProviders return registered providers
func (a *Auth) GetProviders() (providers []Provider) {
	for _, provider := range a.providers {
		providers = append(providers, provider)
	}
	return
}
