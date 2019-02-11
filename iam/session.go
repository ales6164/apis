package iam

import (
	"errors"
	"github.com/ales6164/apis/collection"
	"github.com/dgrijalva/jwt-go"
	"google.golang.org/appengine/datastore"
	"time"
)

const SessionKind = "_session"
const NotBeforeCorrection = -10 // seconds

type Session struct {
	Key             *datastore.Key
	Claims          *Claims
	Token           *jwt.Token
	IsValid         bool
	IsAuthenticated bool
	stored          *storedSession

	provider Provider  // only on created session
	identity *Identity // only on created session
}

type Claims struct {
	Id     *datastore.Key `json:"id"`
	Scopes []string       `json:"scopes"`
	jwt.StandardClaims
}

type storedSession struct {
	CreatedAt time.Time
	Identity  *datastore.Key
	Subject   *datastore.Key // user or user group, API client ...
	ExpiresAt time.Time
	Provider  string
	IsBlocked bool
	Scopes    []string
}

// todo: add anonymous user session store
func startSession(ctx Context, token *jwt.Token) (*Session, error) {
	var err error
	var s = &Session{
		stored: new(storedSession),
		Token:  token,
	}

	if token != nil {
		if claims, ok := token.Claims.(*Claims); ok && token.Valid {
			err = datastore.Get(ctx.Default(), claims.Id, s.stored)
			if err != nil {
				return s, err
			}
			if s.stored.IsBlocked {
				return s, errors.New("session is blocked")
			}

			s.Claims = claims
			s.Key = claims.Id
			s.IsAuthenticated = true
			s.Token = token
		} /*else {
			return s, errors.New("token is invalid")
		}*/
	} /*else {
		return s, errors.New("token is not present")
	}*/

	if !s.IsAuthenticated {
		// anonymous
		s.stored.Subject = datastore.NewKey(ctx, "Group", AllUsers, 0, nil)
	}

	s.IsValid = true

	return s, nil
}

// extend by seconds from now
func (s *Session) LoadIdentity(ctx Context) (*Session, error) {
	if s.identity == nil {
		s.identity = new(Identity)
		err := datastore.Get(ctx, s.stored.Identity, s.identity)
		if err != nil {
			return s, err
		}

		// get user
		var userDocument = collection.UserCollection.Doc(s.stored.Subject, nil)
		userDocument, err = userDocument.Get(ctx)
		if err != nil {
			return s, err
		}

		s.identity.User = collection.UserCollection.Data(userDocument, false, false).(*collection.User)
		s.identity.IdentityKey = s.stored.Identity
		s.identity.isOk = true
	}
	return s, nil
}

// extend by seconds from now
func (s *Session) Extend(ctx Context, seconds int64) (*Session, error) {
	s.stored.ExpiresAt = time.Now().Add(time.Second * time.Duration(seconds))
	_, err := datastore.Put(ctx.Default(), s.Key, s.stored)
	return s, err
}
