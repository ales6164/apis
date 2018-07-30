package providers

import (
	"google.golang.org/appengine/datastore"
	"golang.org/x/net/context"
	"time"
)

type Account struct {
	Id        *datastore.Key `datastore:"-" apis:"id" json:"-"`
	CreatedAt time.Time      `apis:"createdAt"`
	UserId    *datastore.Key `json:"-"`
	User      interface{}    `datastore:"-" json:"user,omitempty"`
	Roles     []string       `json:"roles"`
}

type Output struct {
	Token Token       `json:"token"`
	User  interface{} `json:"user"`
}

type Token struct {
	Id        string `json:"id"`
	ExpiresAt int64  `json:"expiresAt"`
}

type Authority interface {
	GetAccount(ctx context.Context, accountKey *datastore.Key) (account *Account, err error)
	CreateAccount(ctx context.Context, role string) (accountKey *datastore.Key, account *Account, err error)
	SignToken(ctx context.Context, account *Account) (signedToken Token, err error)
}
