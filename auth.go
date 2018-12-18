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
	// How long until it expires in seconds
	TokenExpiresIn int64
	TokenIssuer    string
	TokenAudience  string
	// Doesn't set token exp; Expiration is managed through sessions
	AutoExtendToken bool

	Roles map[string][]string
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

const AccountKind = "_account"

type Account struct {
	Id   *datastore.Key `datastore:"-"`
	User *datastore.Key
	// Email is always required
	Email  string   `json:"email"`
	Scopes []string `json:"scopes"`
}

// Connects provider identity with user account. Creates account if it doesn't exist. Should be run inside a transaction.
func (a *Auth) ConnectUser(ctx context.Context, providerIdentityKey *datastore.Key, userEmail string, userHolder *Holder) (*Account, error) {
	var account = new(Account)

	err := userHolder.Kind.Put(ctx, userHolder)
	if err != nil {
		return account, err
	}

	accountKey := datastore.NewKey(ctx, AccountKind, userEmail, 0, nil)
	err = datastore.Get(ctx, accountKey, account)
	if err != nil {
		if err == datastore.ErrNoSuchEntity {
			account.User = userHolder.Key
			account.Email = userEmail
			account.Scopes = a.DefaultScopes
			_, err = datastore.Put(ctx, accountKey, account)
			if err != nil {
				return account, err
			}
		} else {
			return account, err
		}
	}
	account.Id = accountKey

	return account, err
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
