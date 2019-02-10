package iam

import (
	"errors"
	"github.com/ales6164/apis/collection"
	"github.com/dgrijalva/jwt-go"
	"google.golang.org/appengine/datastore"
	"net/http"
	"time"
)

type Options struct {
	SigningKey          []byte
	Extractors          []TokenExtractor
	CredentialsOptional bool
	// These scopes are assigned to new users
	DefaultRoles  []string
	SigningMethod jwt.SigningMethod
	// How long until it expires in seconds. Default is 7 days.
	TokenExpiresIn int64
	TokenIssuer    string
	TokenAudience  string
	// Doesn't set token exp; Expiration is managed through sessions
	AutoExtendToken bool
	// Default is 12
	HashingCost         int
	Debug               bool
	EnableAuthOnOptions bool
	ErrorHandler        errorHandler
	RedirectOnError     string
}

type IAM struct {
	*Options
	providers  []Provider
	middleware *JWTMiddleware
}

const (
	week = 60 * 60 * 24 * 7
)

// Manages users, their relationships, roles, permissions, tokens
func NewIAM(opt *Options) *IAM {
	if opt == nil {
		opt = new(Options)
	}

	iam := &IAM{
		Options: opt,
	}

	if iam.HashingCost <= 0 {
		iam.HashingCost = 12
	}
	if iam.TokenExpiresIn <= 0 {
		iam.TokenExpiresIn = week
	}

	iam.middleware = middleware(MiddlewareOptions{
		Extractor: FromFirst(iam.Extractors...),
		ValidationKeyGetter: func(token *jwt.Token) (interface{}, error) {
			return iam.SigningKey, nil
		},
		SigningMethod:       iam.SigningMethod,
		CredentialsOptional: iam.CredentialsOptional,
		Debug:               iam.Debug,
		EnableAuthOnOptions: iam.EnableAuthOnOptions,
		ErrorHandler:        iam.ErrorHandler,
		RedirectOnError:     iam.RedirectOnError,
	})

	return iam
}

 func (iam *IAM) Middleware() *JWTMiddleware {
 	return iam.middleware
 }

func (iam *IAM) NewSession(ctx Context, provider Provider, identity *Identity) (*session, error) {
	now := time.Now()
	s := &session{
		provider: provider,
		identity: identity,
		stored: &storedSession{
			Identity:  identity.IdentityKey,
			Subject:   identity.UserKey,
			Provider:  provider.Name(),
			CreatedAt: now,
			ExpiresAt: now.Add(time.Second * time.Duration(iam.TokenExpiresIn)),
			IsBlocked: false,
			Scopes:    iam.DefaultRoles,
		},
		Key: datastore.NewIncompleteKey(ctx.Default(), SessionKind, nil),
	}

	var err error
	s.Key, err = datastore.Put(ctx.Default(), s.Key, s)
	if err != nil {
		return s, err
	}

	s.Claims = &claims{
		s.Key,
		iam.DefaultRoles,
		jwt.StandardClaims{
			Issuer:    iam.TokenIssuer,
			NotBefore: now.Add(time.Second * time.Duration(NotBeforeCorrection)).Unix(),
			Id:        s.Key.Encode(),
			Subject:   identity.UserKey.Encode(),
			Audience:  iam.TokenAudience,
			IssuedAt:  now.Unix(),
		},
	}

	if !iam.AutoExtendToken {
		s.Claims.ExpiresAt = s.stored.ExpiresAt.Unix()
	}

	s.Token = jwt.NewWithClaims(jwt.SigningMethodHS256, *s.Claims)

	return s, nil
}

func (iam *IAM) RenewSession(ctx Context) (*session, error) {
	s, err := ctx.session.Extend(iam.TokenExpiresIn)
	return s, err
}

func (iam *IAM) SignedToken(ctx Context, s *session) (string, error) {
	return s.Token.SignedString(ctx.SigningKey)
}

func (iam *IAM) PrintResponse(session *session) {
	signedToken, err := iam.SignedToken(session.ctx, session)
	if err != nil {
		session.ctx.PrintError(err.Error(), http.StatusInternalServerError)
		return
	}

	session.ctx.PrintJSON(map[string]interface{}{
		"user": session.identity.User,
		"token": map[string]interface{}{
			"id":        signedToken,
			"expiredAt": session.stored.ExpiresAt.Unix(),
		},
	}, http.StatusOK)
}

func (iam *IAM) User(ctx Context, member *datastore.Key) (*collection.User, error) {
	if member == nil || member.Kind() != collection.UserCollection.Name() {
		return nil, errors.New("member key not of user")
	}
	var user = new(collection.User)
	err := datastore.Get(ctx.Default(), member, user)
	return user, err
}
