package auth

import (
	"golang.org/x/net/context"
	"google.golang.org/appengine/datastore"
)

type Provider struct {
	Name          string
	DefaultScopes []string
}

// db entry
type User struct {
	FirstName  string   `json:"firstName"`
	LastName   string   `json:"lastName"`
	Email      string   `json:"email"`
	IsVerified bool     `json:"isVerified"`
	Avatar     string   `json:"avatar"`
	Scopes     []string `json:"scopes"`
}

// db entry
type connection struct {
}

// Connects provider identity with user account
func (p *Provider) ConnectUser(ctx context.Context, providerIdentityKey *datastore.Key, userName string) (*User, error) {
	//
}
