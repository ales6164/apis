package apis

import (
	"fmt"
)

type Provider interface {
	GetName() string

	ConfigAuth(*Auth)
	Login(*Context)
	Logout(*Context)
	Register(*Context)
	Callback(*Context)
	ServeHTTP(*Context)
}

func (a *Auth) RegisterProvider(provider Provider) {
	name := provider.GetName()
	for _, p := range a.providers {
		if p.GetName() == name {
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
		if provider.GetName() == name {
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
