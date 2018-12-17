package auth

import (
	"github.com/dgrijalva/jwt-go"
	"golang.org/x/net/context"
	"google.golang.org/appengine/datastore"
	"time"
)

const SessionKind = "_session"

// db entry
type Session struct {
	ProviderIdentity *datastore.Key `json:"-"`
	Subject          *datastore.Key `json:"-"`
	CreatedAt        time.Time
	ExpiresAt        time.Time
	IsBlocked        bool
	token            *jwt.Token `datastore:"-"`
}

func newSession(a *Auth, ctx context.Context, providerIdentity *datastore.Key, subject *datastore.Key, scopes ...string) (*Session, error) {
	now := time.Now()
	s := &Session{
		ProviderIdentity: providerIdentity,
		Subject:          subject,
		CreatedAt:        now,
		ExpiresAt:        now.Add(time.Second * time.Duration(a.TokenExpiresIn)),
		IsBlocked:        false,
	}

	sKey := datastore.NewIncompleteKey(ctx, SessionKind, nil)

	sKey, err := datastore.Put(ctx, sKey, s)
	if err != nil {
		return s, err
	}

	claims := Claims{
		scopes,
		jwt.StandardClaims{
			Issuer:    a.TokenIssuer,
			NotBefore: now.Unix(),
			Id:        sessionId,
			Subject:   subject.Encode(),
			Audience:  a.TokenAudience,
			IssuedAt:  now.Unix(),
		},
	}

	if !a.AutoExtendToken {
		claims.ExpiresAt = s.ExpiresAt.Unix()
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token.SignedString(a.SigningKey)

}

func (s *Session) Extend() {

}
