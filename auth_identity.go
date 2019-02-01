package apis

import (
	"errors"
	"github.com/ales6164/apis/collection"
	"github.com/ales6164/apis/kind"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/net/context"
	"google.golang.org/appengine/datastore"
	"time"
)

const IdentityKind = "_identity"

type Identity struct {
	User           *User          `datastore:"-" json:"-"`
	IdentityKey    *datastore.Key `datastore:"-" json:"-"`
	UserKey        *datastore.Key `json:"-"`
	CreatedAt      time.Time      `json:"createdAt"`
	UpdatedAt      time.Time      `json:"createdAt"`
	Provider       string         `json:"provider"`
	EmailConfirmed bool           `json:"emailConfirmed"` // TODO: this should be stored with identity provider -- email has to be confirmed for each provider seperately... If user exists, it has at least one provider with confirmed email
	Secret         []byte         `datastore:",noindex" json:"-"`
	isOk           bool           `datastore:"-"` // this should always be true
}

type User struct {
	Email string   `json:"email"`
	Roles []string `json:"roles"`
}

var (
	UserCollection = collection.New("users", User{})

	ErrEmailIsWaitingConfirmation = errors.New("email is waiting confirmation")
	ErrUserConnectionFailure      = errors.New("user connection failure")
	ErrUnlockingIdentity          = errors.New("error unlocking identity")
	ErrEncryptingIdentity          = errors.New("error encrypting identity")
	ErrDatabaseConnection         = errors.New("database connection error")
)

// Creates identity - should not exist; creates user if doesn't exist, otherwise connects user to the new identity if trustEmail is true
func (a *Auth) CreateUser(ctx context.Context, provider Provider, userEmail string, unlockKey string) (*Identity, error) {
	// 1. Create identity and user
	userKey := datastore.NewKey(ctx, UserCollection.Name(), userEmail, 0, nil)
	var userDocument kind.Doc

	identityKey := datastore.NewKey(ctx, IdentityKind, provider.Name()+":"+userEmail, 0, userKey)
	var user = new(User)
	var identity = new(Identity)

	err := datastore.RunInTransaction(ctx, func(ctx context.Context) error {
		var err error
		userDocument, err = UserCollection.Doc(ctx, userKey, nil)
		if err != nil {
			return err
		}

		err = datastore.Get(ctx, identityKey, identity)
		if err != nil {
			if err == datastore.ErrNoSuchEntity {
				// ok

				if !userDocument.Exists() {
					// user doesn't exist -- VERY SAFE
					// create user
					user.Email = userEmail
					user.EmailConfirmed = trustUserEmail
					user.Roles = a.DefaultRoles

					userDocument, err = userDocument.Set(user)
					if err != nil {
						return err
					}

					err = userDocument.SetRole(userDocument.Key(), FullControl)
					if err != nil {
						return err
					}
				} else {
					// user exists -- OOPS!!!

					// load user
					userDocument, err = userDocument.Get()
					if err != nil {
						return err
					}

					user = UserCollection.Data(userDocument, false).(*User)

					if trustUserEmail {
						// this means that we trust provided email and can add any identity to the existing user account -- UNSAFE!!!!

						// since we trust this email, we can update the field
						user.EmailConfirmed = trustUserEmail

						// add default roles to the existing user
						var toAppend []string
						for _, r := range a.DefaultRoles {
							var ok bool
							for _, r2 := range user.Roles {
								if r == r2 {
									ok = true
								}
							}
							if !ok {
								toAppend = append(toAppend, r)
							}
						}
						user.Roles = append(user.Roles, toAppend...)

						userDocument, err = userDocument.Set(user)
						if err != nil {
							return err
						}
					} else {
						// we don't trust the user email and since user already exists we return error!
						return kind.ErrEntityAlreadyExists
					}
				}

				// after checks above and having the user created, let's create identity
				identity.User = user
				identity.IdentityKey = identityKey
				identity.isOk = true

				identity.UserKey = userKey
				identity.CreatedAt = time.Now()
				identity.UpdatedAt = identity.CreatedAt
				identity.Provider = provider.Name()
				// create unlockKey hash
				identity.Secret, err = crypt(a.HashingCost, []byte(unlockKey))
				if err != nil {
					return err
				}

				// save identity
				_, err = datastore.Put(ctx, identityKey, identity)
			}
			return err
		}
		return kind.ErrEntityAlreadyExists
	}, &datastore.TransactionOptions{XG: true})
	if err != nil {
		return nil, err
	}
	return identity, nil
}

var (
	ErrUserDoesNotExist = errors.New("user doesn't exist")
)

// Connects identity with user
func (a *Auth) Connect(ctx context.Context, provider Provider, userEmail string, unlockKey string) (*Identity, error) {
	var userKey = datastore.NewKey(ctx, UserCollection.Name(), userEmail, 0, nil)
	var identityKey = datastore.NewKey(ctx, IdentityKind, provider.Name()+":"+userEmail, 0, userKey)

	var userDocument kind.Doc
	var user = new(User)
	var identity = new(Identity)

	err := datastore.RunInTransaction(ctx, func(ctx context.Context) error {
		var err error
		userDocument, err = UserCollection.Doc(ctx, userKey, nil)
		if err != nil {
			return ErrDatabaseConnection
		}

		err = datastore.Get(ctx, identityKey, identity)
		if err != nil {
			if err == datastore.ErrNoSuchEntity {
				// identity doesn't exist

				if provider.TrustProvidedEmail() {
					// create identity and connect with user

					if !userDocument.Exists() {
						// create user

						user.Email = userEmail
						user.Roles = a.DefaultRoles

						userDocument, err = userDocument.Set(user)
						if err != nil {
							return ErrDatabaseConnection
						}

						err = userDocument.SetRole(userDocument.Key(), FullControl)
						if err != nil {
							return ErrDatabaseConnection
						}
					} else {
						// load user and save changes

						userDocument, err = userDocument.Get()
						if err != nil {
							return ErrDatabaseConnection
						}

						user = UserCollection.Data(userDocument, false).(*User)

						// add default roles to the existing user
						var toAppend []string
						for _, r := range a.DefaultRoles {
							var ok bool
							for _, r2 := range user.Roles {
								if r == r2 {
									ok = true
								}
							}
							if !ok {
								toAppend = append(toAppend, r)
							}
						}
						user.Roles = append(user.Roles, toAppend...)

						// save user
						userDocument, err = userDocument.Set(user)
						if err != nil {
							return ErrDatabaseConnection
						}

					}

					// connect identity to user and save

					identity.User = user
					identity.IdentityKey = identityKey
					identity.isOk = true
					identity.EmailConfirmed = true
					identity.UserKey = userKey
					identity.CreatedAt = time.Now()
					identity.UpdatedAt = identity.CreatedAt
					identity.Provider = provider.Name()

					// encrypt and save unlock key
					identity.Secret, err = crypt(a.HashingCost, []byte(unlockKey))
					if err != nil {
						return ErrEncryptingIdentity
					}

					// save identity
					_, err = datastore.Put(ctx, identityKey, identity)
					if err != nil {
						return ErrDatabaseConnection
					}

					return nil
				} else {
					// create identity and send confirmation email (postpones user connection)

					identity.IdentityKey = identityKey
					identity.isOk = true

					identity.EmailConfirmed = false
					identity.CreatedAt = time.Now()
					identity.UpdatedAt = identity.CreatedAt
					identity.Provider = provider.Name()

					// encrypt and save unlock key
					identity.Secret, err = crypt(a.HashingCost, []byte(unlockKey))
					if err != nil {
						return ErrEncryptingIdentity
					}

					// save identity
					_, err = datastore.Put(ctx, identityKey, identity)
					if err != nil {
						return ErrDatabaseConnection
					}

					// send email for email confirmation and to continue connecting identity to user
					// TODO:
					
					return errors.New("error sending confirmation email")
				}
			}
			return ErrDatabaseConnection
		}

		// identity exists

		if !identity.EmailConfirmed {
			return ErrEmailIsWaitingConfirmation
		}

		if !userDocument.Exists() {
			return ErrUserConnectionFailure
		}

		// check unlock key
		err = decrypt(identity.Secret, []byte(unlockKey))
		if err != nil {
			return ErrUnlockingIdentity
		}

		// get user
		userDocument, err = userDocument.Get()
		if err != nil {
			return ErrDatabaseConnection
		}

		identity.User = UserCollection.Data(userDocument, false).(*User)
		identity.IdentityKey = identityKey
		identity.isOk = true

		return nil
	}, &datastore.TransactionOptions{XG: true})
	if err != nil {
		return nil, err
	}

	return identity, nil
}

func (a *Auth) GetIdentity(ctx context.Context, provider Provider, userEmail string, unlockKey string) (*Identity, error) {
	userKey := datastore.NewKey(ctx, UserCollection.Name(), userEmail, 0, nil)
	userDocument, err := UserCollection.Doc(ctx, userKey, nil)
	if err != nil {
		return nil, err
	}
	if !userDocument.Exists() {
		return nil, ErrUserDoesNotExist
	}

	identityKey := datastore.NewKey(ctx, IdentityKind, provider.Name()+":"+userEmail, 0, userKey)
	var identity = new(Identity)

	err = datastore.Get(ctx, identityKey, identity)
	if err != nil {
		return nil, err
	}

	// check unlockKey
	err = decrypt(identity.Secret, []byte(unlockKey))
	if err != nil {
		return nil, err
	}

	userDocument, err = userDocument.Get()
	if err != nil {
		return nil, err
	}

	identity.User = UserCollection.Data(userDocument, false).(*User)
	identity.IdentityKey = identityKey
	identity.isOk = true

	return identity, nil
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
