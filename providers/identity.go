package providers

import (
	"google.golang.org/appengine/datastore"
	"golang.org/x/net/context"
	"gopkg.in/ales6164/apis.v2/errors"
)

const identityKind = "_identity"

type identity struct {
	provider   IdentityProvider `datastore:"-"`
	Provider   string
	AccountKey *datastore.Key
	Secret     []byte `datastore:",noindex" json:"-"`
}

func NewIdentity(p IdentityProvider, secret []byte) (*identity) {
	return &identity{
		provider: p,
		Provider: p.Name(),
		Secret:   secret,
	}
}

func GetIdentity(ctx context.Context, p IdentityProvider, id string) (*identity, error) {
	var i = new(identity)
	err := datastore.Get(ctx, datastore.NewKey(ctx, identityKind, id, 0, nil), i)
	if err != nil {
		return nil, err
	}
	if i.Provider != p.Name() {
		return nil, errors.New("invalid identity provider")
	}

	return i, err
}

func (i *identity) Save(ctx context.Context, id string, role string) (*Account, error) {
	var accountKey *datastore.Key
	var account *Account
	err := datastore.RunInTransaction(ctx, func(tc context.Context) error {
		key := datastore.NewKey(tc, identityKind, id, 0, nil)
		var dst datastore.PropertyList
		err := datastore.Get(tc, key, &dst)
		if err != nil {
			if err == datastore.ErrNoSuchEntity {
				accountKey, account, err = i.provider.Authority().CreateAccount(tc, role)
				if err != nil {
					return err
				}
				i.AccountKey = accountKey
				_, err = datastore.Put(tc, key, i)
				return err
			}
			return err
		}
		return errors.ErrEntityExists
	}, &datastore.TransactionOptions{
		XG: true,
	})
	return account, err
}

func (i *identity) UpdateSecret(ctx context.Context, id string, secret []byte) (error) {
	return datastore.RunInTransaction(ctx, func(tc context.Context) error {
		key := datastore.NewKey(tc, identityKind, id, 0, nil)
		err := datastore.Get(tc, key, i)
		if err != nil {
			return err
		}
		i.Secret = secret
		_, err = datastore.Put(tc, key, i)
		return err
	}, nil)
}