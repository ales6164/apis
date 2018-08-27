package apis

import (
	"google.golang.org/appengine/datastore"
	"time"
	"github.com/dgrijalva/jwt-go"
	"gopkg.in/ales6164/apis.v2/middleware"
	"github.com/gorilla/mux"
	"gopkg.in/ales6164/apis.v2/providers"
	"golang.org/x/net/context"
)

type ClientSession struct {
	CreatedAt time.Time
	ExpiresAt time.Time
	JwtID     string
	IsBlocked bool
	Roles     []string
	Account   *datastore.Key
	User      *datastore.Key
}

type Auth struct {
	defaultRole string
	a           *Apis
	providers.Authority
}

func (a *Auth) GetAccount(ctx context.Context, accountKey *datastore.Key) (account *providers.Account, err error) {
	// 1. Get account
	// 2. Get user and insert object into account

	accountHolder := accountKind.NewHolder( nil)
	err = accountHolder.Get(ctx, accountKey)
	if err != nil {
		return account, err
	}

	account = accountHolder.Value().(*providers.Account)

	userHolder := UserKind.NewHolder(nil)
	err = userHolder.Get(ctx, account.UserId)
	if err != nil {
		return account, err
	}

	account.User = userHolder.Value()

	return account, err
}
func (a *Auth) CreateAccount(ctx context.Context, role string) (accountKey *datastore.Key, account *providers.Account, err error) {
	// 1. Create and save user
	// 2. Create and save account
	userHolder := UserKind.NewHolder( nil)
	accountHolder := accountKind.NewHolder( nil)

	userKey, err := datastore.Put(ctx, UserKind.NewIncompleteKey(ctx, nil), userHolder)
	if err != nil {
		return accountKey, account, err
	}
	userHolder.SetKey(userKey)

	accountHolder.SetValue(&providers.Account{
		Roles:  []string{role},
		UserId: userKey,
		User:   userHolder.Value(),
	})

	accountKey, err = datastore.Put(ctx, accountKind.NewIncompleteKey(ctx, nil), accountHolder)
	if err != nil {
		return accountKey, account, err
	}
	accountHolder.SetKey(accountKey)

	return accountHolder.GetKey(ctx), accountHolder.Value().(*providers.Account), err
}

func (a *Auth) SignToken(ctx context.Context, account *providers.Account) (signedToken providers.Token, err error) {
	return a.a.createSession(ctx, account)
}

func initAuth(a *Apis, r *mux.Router) {
	auth := &Auth{
		a: a,
	}

	authRouter := r.PathPrefix("/auth").Subrouter()
	for _, provider := range a.options.IdentityProviders {
		provider.Apply(authRouter, auth)
	}
}

func (a *Apis) createSession(ctx context.Context, account *providers.Account) (signedToken providers.Token, err error) {
	now := time.Now()
	expiresAt := now.Add(time.Hour * time.Duration(72))
	sess := new(ClientSession)
	sess.CreatedAt = now
	sess.ExpiresAt = expiresAt
	sess.Account = account.Id
	sess.User = account.UserId
	sess.Roles = account.Roles
	sess.JwtID = RandStringBytesMaskImprSrc(LetterBytes, 16)
	sessKey := datastore.NewIncompleteKey(ctx, "_clientSession", nil)
	sessKey, err = datastore.Put(ctx, sessKey, sess)
	if err != nil {
		return signedToken, err
	}
	return a.authenticate(account.UserId, sessKey, sess, expiresAt.Unix())
}

func (a *Apis) authenticate(userKey, sessionKey *datastore.Key, session *ClientSession, expiresAt int64) (providers.Token, error) {
	var err error
	now := time.Now()
	claims := middleware.Claims{
		Nonce:  sessionKey.Encode(),
		StandardClaims: jwt.StandardClaims{
			Audience:  "all",
			Id:        session.JwtID,
			ExpiresAt: expiresAt,
			IssuedAt:  now.Unix(),
			Issuer:    a.options.AppName,
			NotBefore: now.Add(-time.Minute).Unix(),
			Subject:   userKey.Encode(),
		},
	}
	token := providers.Token{
		ExpiresAt: expiresAt,
	}
	token.Id, err = jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(a.privateKey)
	return token, err
}
