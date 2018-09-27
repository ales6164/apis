package providers

import (
	"github.com/ales6164/client"
	"time"
)

type Response struct {
	User  *client.User `json:"user"`
	Token *Token       `json:"token"`
}

type Token struct {
	Id        string    `json:"id"`
	ExpiresAt time.Time `json:"expiresAt"`
}

func WithEmailPasswordProvider(cost int, signingKey []byte) *EmailPasswordProvider {
	return &EmailPasswordProvider{
		Cost:       cost,
		SigningKey: signingKey,
	}
}
