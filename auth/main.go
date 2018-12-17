package auth

import (
	"github.com/dgrijalva/jwt-go"
	"google.golang.org/appengine/datastore"
	"time"
)

type Auth struct {
	*Options
}

const (
	AllUsers              = "allUsers"              // given to all requests
	AllAuthenticatedUsers = "allAuthenticatedUsers" // giver to all authenticated requests
)

type Options struct {
	SigningKey          []byte
	Extractors          []TokenExtractor
	CredentialsOptional bool
	SigningMethod       jwt.SigningMethod
	// How long until it expires in seconds
	TokenExpiresIn int64
	TokenIssuer    string
	TokenAudience  string
	// Doesn't set token exp; Expiration is managed through sessions
	AutoExtendToken     bool

	Roles               map[string][]string
}

func New(opt *Options) *Auth {
	auth := &Auth{Options: opt}
	return auth
}

func (a *Auth) Middleware() *JWTMiddleware {
	return middleware(MiddlewareOptions{
		Extractor: FromFirst(a.Extractors...),
		ValidationKeyGetter: func(token *jwt.Token) (interface{}, error) {
			return a.SigningKey, nil
		},
		SigningMethod:       a.SigningMethod,
		CredentialsOptional: a.CredentialsOptional,
	})
}

type Claims struct {
	Scopes []string `json:"scopes"`
	jwt.StandardClaims
}

func (a *Auth) NewSession(subject *datastore.Key, scopes ...string) (*Session, error) {
	now := time.Now()

	// new session

	claims := Claims{
		scopes,
		jwt.StandardClaims{
			ExpiresAt: now.Add(time.Second * time.Duration(a.TokenExpiresIn)).Unix(),
			Issuer:    a.TokenIssuer,
			NotBefore: now.Unix(),
			Id:        sessionId,
			Subject:   subject.Encode(),
			Audience:  a.TokenAudience,
			IssuedAt:  now.Unix(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token.SignedString(a.SigningKey)

	// Sign and get the complete encoded token as a string using the secret
	return newSession()
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
