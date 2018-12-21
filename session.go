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
	Member           *datastore.Key `json:"-"`
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

func newSession(a *Auth, ctx context.Context, providerIdentity *datastore.Key, member *datastore.Key, roles ...string) (*Session, error) {
	var noRolesScopes []string
	for _, s := range roles {
		if roleScopes, ok := a.a.Roles[s]; ok {
			noRolesScopes = append(noRolesScopes, roleScopes...)
		}
	}

	now := time.Now()
	s := &Session{
		ProviderIdentity: providerIdentity,
		Member:           member,
		CreatedAt:        now,
		ExpiresAt:        now.Add(time.Second * time.Duration(a.TokenExpiresIn)),
		IsBlocked:        false,
		Scopes:           noRolesScopes,
	}

	sKey := datastore.NewIncompleteKey(ctx, SessionKind, nil)

	sKey, err := datastore.Put(ctx, sKey, s)
	if err != nil {
		return s, err
	}

	claims := Claims{
		sKey,
		noRolesScopes,
		jwt.StandardClaims{
			Issuer:    a.TokenIssuer,
			NotBefore: now.Add(time.Second * time.Duration(NotBeforeCorrection)).Unix(),
			Id:        sKey.Encode(),
			Subject:   member.Encode(),
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
	if !s.isAuthenticated {
		s.Member = datastore.NewKey(ctx, "Group", AllUsers, 0, nil)
	}

	return s, nil
}

func (s *Session) HasScope(scopes ...string) bool {
	return ContainsScope(s.Scopes, scopes...)
}

// extend by seconds from now
func (s *Session) Extend(ctx context.Context, seconds int64) error {
	s.ExpiresAt = time.Now().Add(time.Second * time.Duration(seconds))
	_, err := datastore.Put(ctx, s.token.Claims.(Claims).Id, s)
	return err
}
