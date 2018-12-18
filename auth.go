package apis

import (
	"github.com/dgrijalva/jwt-go"
	"golang.org/x/net/context"
	"google.golang.org/appengine/datastore"
)

type Auth struct {
	*AuthOptions
}

const (
	AllUsers              = "allUsers"              // given to all requests
	AllAuthenticatedUsers = "allAuthenticatedUsers" // giver to all authenticated requests
)

type AuthOptions struct {
	SigningKey          []byte
	Extractors          []TokenExtractor
	CredentialsOptional bool
	// These scopes are assigned to new users
	DefaultScopes []string
	SigningMethod jwt.SigningMethod
	// How long until it expires in seconds. Default is 7 days.
	TokenExpiresIn int64
	TokenIssuer    string
	TokenAudience  string
	// Doesn't set token exp; Expiration is managed through sessions
	AutoExtendToken bool
	// Default is 12
	HashingCost int

	Roles map[string][]string

	providers []Provider
}

type Token struct {
	Id        string `json:"id"`
	ExpiresAt int64  `json:"expiresAt"`
}

type AuthResponse struct {
	User  interface{} `json:"user"`
	Token Token       `json:"token"`
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
	return auth
}

func (a *Auth) Middleware() *JWTMiddleware {
	return middleware(a, MiddlewareOptions{
		Extractor: FromFirst(a.Extractors...),
		ValidationKeyGetter: func(token *jwt.Token) (interface{}, error) {
			return a.SigningKey, nil
		},
		SigningMethod:       a.SigningMethod,
		CredentialsOptional: a.CredentialsOptional,
	})
}

func (a *Auth) NewSession(ctx context.Context, providerIdentity *datastore.Key, subject *datastore.Key, scopes ...string) (*Session, error) {
	return newSession(a, ctx, providerIdentity, subject, scopes...)
}

func (a *Auth) SignedToken(s *Session) (string, error) {
	return s.token.SignedString(a.SigningKey)
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
