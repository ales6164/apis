package apis

import (
	"errors"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/net/context"
	"google.golang.org/appengine/datastore"
)

const IdentityKind = "_identity"

type Identity struct {
	Id       *datastore.Key `datastore:"-" json:"id"`
	User     *datastore.Key `json:"user"`
	Provider string         `json:"provider"`
	Secret   []byte         `datastore:",noindex" json:"-"`
}

func (a *Auth) CreateIdentity(ctx context.Context, provider Provider, user *datastore.Key, unlockKey string) (*Identity, error) {
	identityKey := datastore.NewKey(ctx, IdentityKind, provider.GetName()+":"+user.StringID(), 0, nil)
	identity := new(Identity)
	err := datastore.Get(ctx, identityKey, identity)
	if err == nil {
		return nil, errors.New("identity already exists")
	}
	if err != nil && err != datastore.ErrNoSuchEntity {
		return nil, err
	}

	// create unlockKey hash
	identity.Secret, err = crypt(a.HashingCost, []byte(unlockKey))
	if err != nil {
		return nil, err
	}
	identity.Id = identityKey
	identity.User = user
	identity.Provider = provider.GetName()

	_, err = datastore.Put(ctx, identityKey, identity)
	return identity, err
}

func (a *Auth) GetIdentity(ctx context.Context, provider Provider, userEmail string, unlockKey string) (*Identity, error) {
	identityKey := datastore.NewKey(ctx, IdentityKind, provider.GetName()+":"+userEmail, 0, nil)
	identity := new(Identity)
	err := datastore.Get(ctx, identityKey, identity)
	if err != nil {
		return nil, err
	}

	// check unlockKey
	err = decrypt(identity.Secret, []byte(unlockKey))
	if err != nil {
		return nil, err
	}

	identity.Id = identityKey

	return identity, err
}

func (i *Identity) GetUser(ctx context.Context) (*User, error) {
	var user = new(User)
	err := datastore.Get(ctx, i.User, user)
	user.Id = i.User
	return user, err
}

func decrypt(hash []byte, password []byte) error {
	defer clear(password)
	return bcrypt.CompareHashAndPassword(hash, password)
}

func crypt(cost int, password []byte) ([]byte, error) {
	defer clear(password)
	return bcrypt.GenerateFromPassword(password, cost)
}

func clear(b []byte) {
	for i := 0; i < len(b); i++ {
		b[i] = 0
	}
}
