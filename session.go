package apis

import (
	"errors"
	"github.com/dgrijalva/jwt-go"
	"golang.org/x/net/context"
	"google.golang.org/appengine/datastore"
	"time"
)

const SessionKind = "_session"
const NotBeforeCorrection = -10 // seconds

// db entry
type Session struct {
	Key              *datastore.Key `datastore:"-"`
	Claims           *Claims        `datastore:"-"`
	IsValid          bool           `datastore:"-"`
	IsAuthenticated  bool           `datastore:"-"`
	ProviderIdentity *datastore.Key
	Member           *datastore.Key
	CreatedAt        time.Time
	ExpiresAt        time.Time
	Provider         string
	IsBlocked        bool
	Roles            []string
	Token            *jwt.Token `datastore:"-"`
}

type Claims struct {
	Id     *datastore.Key `json:"id"`
	Scopes []string       `json:"scopes"`
	jwt.StandardClaims
}

func newSession(a *Auth, ctx context.Context, provider string, providerIdentity *datastore.Key, member *datastore.Key, roles ...string) (*Session, error) {
	now := time.Now()
	s := &Session{
		ProviderIdentity: providerIdentity,
		Member:           member,
		Provider:         provider,
		CreatedAt:        now,
		ExpiresAt:        now.Add(time.Second * time.Duration(a.TokenExpiresIn)),
		IsBlocked:        false,
		Roles:            roles,
		Key:              datastore.NewIncompleteKey(ctx, SessionKind, nil),
	}

	var err error
	s.Key, err = datastore.Put(ctx, s.Key, s)
	if err != nil {
		return s, err
	}

	s.Claims = &Claims{
		s.Key,
		roles,
		jwt.StandardClaims{
			Issuer:    a.TokenIssuer,
			NotBefore: now.Add(time.Second * time.Duration(NotBeforeCorrection)).Unix(),
			Id:        s.Key.Encode(),
			Subject:   member.Encode(),
			Audience:  a.TokenAudience,
			IssuedAt:  now.Unix(),
		},
	}

	if !a.AutoExtendToken {
		s.Claims.ExpiresAt = s.ExpiresAt.Unix()
	}

	s.Token = jwt.NewWithClaims(jwt.SigningMethodHS256, *s.Claims)

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
				return s, errors.New("session is blocked")
			}

			s.Claims = claims
			s.Key = claims.Id
			s.IsAuthenticated = true
			s.Token = token
			s.IsValid = true
		} else {
			return s, errors.New("token is invalid")
		}
	} else {
		return s, errors.New("token is not present")
	}

	if !s.IsAuthenticated {
		// anonymous
		s.Member = datastore.NewKey(ctx, "Group", AllUsers, 0, nil)
	}
	return s, nil
}

var (
	// User groups
	AllUsers              = "allUsers"              // given to all requests
	AllAuthenticatedUsers = "allAuthenticatedUsers" // giver to all authenticated requests

	// Scopes
	FullControl = "fullcontrol"
	ReadOnly    = "readonly"
	ReadWrite   = "readwrite"
	Delete      = "delete"
)



// extend by seconds from now
func (s *Session) Extend(ctx context.Context, seconds int64) error {
	s.ExpiresAt = time.Now().Add(time.Second * time.Duration(seconds))
	_, err := datastore.Put(ctx, s.Key, s)
	return err
}
