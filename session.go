package apis

import (
	"github.com/dgrijalva/jwt-go"
	"golang.org/x/net/context"
	"google.golang.org/appengine/datastore"
	"gopkg.in/ales6164/apis.v1/errors"
	"time"
)

const SessionKind = "_session"
const NotBeforeCorrection = -10 // seconds

// db entry
type Session struct {
	isValid          bool           `datastore:"-"`
	isAuthenticated  bool           `datastore:"-"`
	ProviderIdentity *datastore.Key `json:"-"`
	Subject          *datastore.Key `json:"-"`
	CreatedAt        time.Time
	ExpiresAt        time.Time
	IsBlocked        bool
	Scopes           []string
	token            *jwt.Token `datastore:"-"`
}

type Claims struct {
	Id     *datastore.Key `json:"id"`
	Scopes []string       `json:"scopes"`
	jwt.StandardClaims
}

func newSession(a *Auth, ctx context.Context, providerIdentity *datastore.Key, subject *datastore.Key, scopes ...string) (*Session, error) {
	var noRolesScopes []string
	for _, s := range scopes {
		if roleScopes, ok := a.a.Roles[s]; ok {
			noRolesScopes = append(noRolesScopes, roleScopes...)
		} else {
			noRolesScopes = append(noRolesScopes, s)
		}
	}

	now := time.Now()
	s := &Session{
		ProviderIdentity: providerIdentity,
		Subject:          subject,
		CreatedAt:        now,
		ExpiresAt:        now.Add(time.Second * time.Duration(a.TokenExpiresIn)),
		IsBlocked:        false,
		Scopes:           scopes,
	}

	sKey := datastore.NewIncompleteKey(ctx, SessionKind, nil)

	sKey, err := datastore.Put(ctx, sKey, s)
	if err != nil {
		return s, err
	}

	claims := Claims{
		sKey,
		scopes,
		jwt.StandardClaims{
			Issuer:    a.TokenIssuer,
			NotBefore: now.Add(time.Second * time.Duration(NotBeforeCorrection)).Unix(),
			Id:        sKey.Encode(),
			Subject:   subject.Encode(),
			Audience:  a.TokenAudience,
			IssuedAt:  now.Unix(),
		},
	}

	if !a.AutoExtendToken {
		claims.ExpiresAt = s.ExpiresAt.Unix()
	}

	s.token = jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return s, nil
}

func StartSession(ctx Context, token *jwt.Token) (*Session, error) {
	var err error
	var s = new(Session)
	if token != nil {
		if claims, ok := token.Claims.(*Claims); ok && token.Valid {
			err = datastore.Get(ctx, claims.Id, s)
			if err != nil {
				return s, err
			}
			if s.IsBlocked {
				return s, errors.New("session expired")
			}

			s.isAuthenticated = true
			s.Scopes = append(s.Scopes, ctx.a.Roles[AllAuthenticatedUsers]...)
			s.token = token
		} else {
			return s, errors.New("invalid claims type")
		}
	}

	s.isValid = true
	s.Scopes = append(s.Scopes, ctx.a.Roles[AllUsers]...)

	return s, nil
}

func (s *Session) HasScope(scopes ...string) bool {
	for _, scp := range scopes {
		for _, r := range s.Scopes {
			if r == scp {
				return true
			}
		}
	}
	return false
}

// extend by seconds from now
func (s *Session) Extend(ctx context.Context, seconds int64) error {
	s.ExpiresAt = time.Now().Add(time.Second * time.Duration(seconds))
	_, err := datastore.Put(ctx, s.token.Claims.(Claims).Id, s)
	return err
}
