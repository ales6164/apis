package iam

import (
	"errors"
	"github.com/dgrijalva/jwt-go"
	"google.golang.org/appengine/datastore"
	"time"
)

const SessionKind = "_session"
const NotBeforeCorrection = -10 // seconds

type session struct {
	ctx             Context
	Key             *datastore.Key
	Claims          *claims
	Token           *jwt.Token
	IsValid         bool
	IsAuthenticated bool
	stored          *storedSession

	provider Provider  // only on created session
	identity *Identity // only on created session
}

type claims struct {
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
func startSession(ctx Context, token *jwt.Token) (*session, error) {
	var err error
	var s = &session{
		ctx:    ctx,
		stored: new(storedSession),
	}

	if token != nil {
		if claims, ok := token.Claims.(*claims); ok && token.Valid {
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
func (s *session) Extend(seconds int64) (*session, error) {
	s.stored.ExpiresAt = time.Now().Add(time.Second * time.Duration(seconds))
	_, err := datastore.Put(s.ctx.Default(), s.Key, s)
	return s, err
}
