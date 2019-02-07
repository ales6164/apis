package apis

import (
	"errors"
	"github.com/ales6164/apis/collection"
	"github.com/dgrijalva/jwt-go"
	"golang.org/x/net/context"
	"google.golang.org/appengine/datastore"
)

type Auth struct {
	*Apis
	*AuthOptions
	middleware *JWTMiddleware
}

type AuthOptions struct {
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
	HashingCost int
	providers   []Provider
}

type Token struct {
	Id        string `json:"id"`
	ExpiresAt int64  `json:"expiresAt"`
}

type AuthResponse struct {
	User  *collection.User `json:"user"`
	Token Token `json:"token"`
}

func NewAuth(opt *AuthOptions) *Auth {
	if opt == nil {
		opt = &AuthOptions{}
	}
	if opt.HashingCost <= 0 {
		opt.HashingCost = 12
	}
	if opt.TokenExpiresIn <= 0 {
		opt.TokenExpiresIn = 60 * 60 * 24 * 7
	}
	auth := &Auth{AuthOptions: opt}
	auth.middleware = middleware(auth, MiddlewareOptions{
		Extractor: FromFirst(auth.Extractors...),
		ValidationKeyGetter: func(token *jwt.Token) (interface{}, error) {
			return auth.SigningKey, nil
		},
		SigningMethod:       auth.SigningMethod,
		CredentialsOptional: auth.CredentialsOptional,
	})
	collection.UserCollection.KeyGen = func(ctx context.Context, str string, member *datastore.Key) *datastore.Key {
		if str == "me" {
			if member.Kind() == collection.UserCollection.Name() {
				return member
			}
		} else {
			key, err := datastore.DecodeKey(str)
			if err != nil {
				key = datastore.NewKey(ctx, collection.UserCollection.Name(), str, 0, nil)
			}
			return key
		}
		return nil
	}
	return auth
}

func (a *Auth) NewSession(ctx context.Context, provider string, providerIdentity *datastore.Key, subject *datastore.Key, scopes ...string) (*Session, error) {
	return newSession(a, ctx, provider, providerIdentity, subject, scopes...)
}

func (a *Auth) SignedToken(s *Session) (string, error) {
	return s.Token.SignedString(a.SigningKey)
}

func (a *Auth) User(ctx context.Context, member *datastore.Key) (*collection.User, error) {
	if member == nil || member.Kind() != collection.UserCollection.Name() {
		return nil, errors.New("member key not of user")
	}
	var user = new(collection.User)
	err := datastore.Get(ctx, member, user)
	return user, err
}

// when project selected - switch namespace
// when on some project endpoint check for project namespace??

// group/user entity access
// kind datastore is protected by general scope rules - anyone with kind.ReadWrite scope can read/write to that kind
// collection is a datastore data container separated with a namespace
// collections are dynamic, it's rules are stored with auth
// collections can be created dynamically - for a kind entry; creator get's collection.FullControl scope
// collection is a wrapper and is handled with middleware - sets context namespace if user has access

// kinds inside collections can be created, edited, deleted
// keys contain namespace (collection id) so API must check user has scope to edit kind within the collection
// creating entries inside collection?
// how to identify collection access?

// /{collection}/{kind}/... ?
// this would work for collections as is user data, projects, groups
