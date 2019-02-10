package iam

import (
	"fmt"
	"net/http"
)

type Provider interface {
	Name() string
	TrustProvidedEmail() bool
	ConfigAuth(*IAM)
	http.Handler
}

func (iam *IAM) RegisterProvider(provider Provider) {
	name := provider.Name()
	for _, p := range iam.providers {
		if p.Name() == name {
			fmt.Printf("warning: auth provider %v already registered", name)
			return
		}
	}

	provider.ConfigAuth(iam)
	iam.providers = append(iam.providers, provider)
}

// GetProvider get provider with name
func (iam *IAM) GetProvider(name string) Provider {
	for _, provider := range iam.providers {
		if provider.Name() == name {
			return provider
		}
	}
	return nil
}

// GetProviders return registered providers
func (iam *IAM) GetProviders() (providers []Provider) {
	for _, provider := range iam.providers {
		providers = append(providers, provider)
	}
	return
}
