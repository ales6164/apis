package iam

import (
	"errors"
	"github.com/ales6164/apis/collection"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/net/context"
	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"time"
)

const IdentityKind = "_identity"

type Identity struct {
	User           *collection.User `datastore:"-" json:"-"`
	IdentityKey    *datastore.Key   `datastore:"-" json:"-"`
	UserKey        *datastore.Key   `json:"-"`
	CreatedAt      time.Time        `json:"createdAt"`
	UpdatedAt      time.Time        `json:"createdAt"`
	Provider       string           `json:"provider"`
	EmailConfirmed bool             `json:"emailConfirmed"` // TODO: this should be stored with identity provider -- email has to be confirmed for each provider seperately... If user exists, it has at least one provider with confirmed email
	Secret         []byte           `datastore:",noindex" json:"-"`
	isOk           bool             `datastore:"-"` // this should always be true
}

var (
	ErrEmailIsWaitingConfirmation = errors.New("email is waiting confirmation")
	ErrUserConnectionFailure      = errors.New("user connection failure")
	ErrUnlockingIdentity          = errors.New("error unlocking identity")
	ErrEncryptingIdentity         = errors.New("error encrypting identity")
	ErrDatabaseConnection         = errors.New("database connection error")
	ErrSendingConfirmationEmail   = errors.New("error sending confirmation email")
	ErrUserDoesNotExist           = errors.New("user doesn't exist")
)

// Connects identity with user
func (iam *IAM) Connect(dctx Context, provider Provider, userEmail string, unlockKey string) (*Identity, error) {
	var userKey = datastore.NewKey(dctx, collection.UserCollection.Name(), userEmail, 0, nil)
	var identityKey = datastore.NewKey(dctx, IdentityKind, provider.Name()+":"+userEmail, 0, userKey)

	var userDocument collection.Doc
	var user = new(collection.User)
	var identity = new(Identity)

	err := datastore.RunInTransaction(dctx, func(ctx context.Context) error {
		var err error
		userDocument = collection.UserCollection.Doc(userKey, nil)
		userDocument, err = userDocument.Get(ctx)
		var userDocumentExists = err == nil

		err = datastore.Get(ctx, identityKey, identity)
		if err != nil {
			if err == datastore.ErrNoSuchEntity {
				// identity doesn't exist

				if provider.TrustProvidedEmail() {
					// create identity and connect with user

					if !userDocumentExists {
						// create user

						user.Email = userEmail
						user.Roles = iam.DefaultRoles

						userDocument, err = userDocument.Set(ctx, user)
						if err != nil {
							return errors.New("1" + err.Error())
						}

						userDocument.SetOwner(userKey)

						err = SetAccess(dctx, userDocument, userDocument.Key(), FullControl)
						if err != nil {
							return errors.New("2" + err.Error())
						}
					} else {
						// load user and save changes

						userDocument, err = userDocument.Get(ctx)
						if err != nil {
							return errors.New("3" + err.Error())
						}

						user = collection.UserCollection.Data(userDocument, false, false).(*collection.User)

						// add default roles to the existing user
						var toAppend []string
						for _, r := range iam.DefaultRoles {
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
						userDocument, err = userDocument.Set(ctx, user)
						if err != nil {
							return errors.New("4" + err.Error())
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
					identity.Secret, err = crypt(iam.HashingCost, []byte(unlockKey))
					if err != nil {
						return errors.New("6" + err.Error())
					}

					// save identity
					_, err = datastore.Put(ctx, identityKey, identity)
					if err != nil {
						return errors.New("5" + err.Error())
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
					identity.Secret, err = crypt(iam.HashingCost, []byte(unlockKey))
					if err != nil {
						return errors.New("7" + err.Error())
					}

					// save identity
					_, err = datastore.Put(ctx, identityKey, identity)
					if err != nil {
						return errors.New("8" + err.Error())
					}

					// create db entry and send email for confirmation and to continue connecting identity to user
					return sendEmailConfirmation(ctx, provider, identityKey, userEmail)
				}
			}
			return errors.New("9" + err.Error())
		}

		// identity exists

		if !identity.EmailConfirmed {
			return sendEmailConfirmation(ctx, provider, identityKey, userEmail)
		}

		if !userDocumentExists {
			return errors.New("10" + err.Error())
		}

		// check unlock key
		err = decrypt(identity.Secret, []byte(unlockKey))
		if err != nil {
			return errors.New("11" + err.Error())
		}

		// get user
		userDocument, err = userDocument.Get(ctx)
		if err != nil {
			return errors.New("12" + err.Error())
		}

		identity.User = collection.UserCollection.Data(userDocument, false, false).(*collection.User)
		identity.IdentityKey = identityKey
		identity.isOk = true

		return nil
	}, &datastore.TransactionOptions{XG: true})
	if err != nil {
		return nil, err
	}

	return identity, nil
}

var (
	ErrInvalidKey               = errors.New("invalid confirmation key")
	ErrConfirmationKeyExpired   = errors.New("confirmation key expired")
	ErrInvalidProvider          = errors.New("invalid provider")
	ErrIdentityDoesNotExist     = errors.New("identity does not exist")
	ErrIdentityAlreadyConfirmed = errors.New("identity is already confirmed")
)

func (iam *IAM) ConfirmEmail(dctx Context, code string) (*Identity, error) {

	emailWaitingConfirmationKey, err := datastore.DecodeKey(code)
	if err != nil {
		return nil, ErrInvalidKey
	}

	var identity = new(Identity)
	var userKey *datastore.Key
	var userEmail string

	err = datastore.RunInTransaction(dctx, func(ctx context.Context) error {
		var emailWaitingConfirmation = new(EmailWaitingConfirmation)
		err = datastore.Get(ctx, emailWaitingConfirmationKey, emailWaitingConfirmation)
		if err != nil {
			if err == datastore.ErrNoSuchEntity {
				return ErrInvalidKey
			}
			return errors.New(err.Error()+"1")
		}

		if !time.Now().Before(emailWaitingConfirmation.ValidUntil) {
			return ErrConfirmationKeyExpired
		}

		if emailWaitingConfirmation.Confirmed {
			return ErrConfirmationKeyExpired
		}

		userEmail = emailWaitingConfirmation.Email

		// get provider
		var provider = iam.GetProvider(emailWaitingConfirmation.Provider)
		if provider == nil {
			return ErrInvalidProvider
		}

		// get identity and check if everything okay there
		err = datastore.Get(ctx, emailWaitingConfirmation.Identity, identity)
		if err != nil {
			if err == datastore.ErrNoSuchEntity {
				return ErrIdentityDoesNotExist
			}
			return errors.New(err.Error()+"2")
		}

		if identity.EmailConfirmed {
			return ErrIdentityAlreadyConfirmed
		}

		// get user and check if everything okay there
		userKey = datastore.NewKey(dctx, collection.UserCollection.Name(), userEmail, 0, nil)

		// connect identity to user and save

		identity.IdentityKey = emailWaitingConfirmation.Identity
		identity.isOk = true
		identity.EmailConfirmed = true
		identity.UserKey = userKey
		identity.UpdatedAt = identity.CreatedAt
		identity.Provider = provider.Name()

		// save identity
		_, err = datastore.Put(ctx, emailWaitingConfirmation.Identity, identity)
		if err != nil {
			return errors.New(err.Error()+"7")
		}

		// make confirmation expired and save to db
		emailWaitingConfirmation.Confirmed = true
		_, err = datastore.Put(ctx, emailWaitingConfirmationKey, emailWaitingConfirmation)
		if err != nil {
			return errors.New(err.Error()+"8")
		}

		return nil
	}, &datastore.TransactionOptions{XG: true})
	if err != nil {
		return nil, err
	}

	var userDocument collection.Doc
	var user = new(collection.User)
	userDocument = collection.UserCollection.Doc(userKey, nil)
	userDocument, err = userDocument.Get(dctx)
	var userDocumentExists = err == nil

	// save user
	if !userDocumentExists {
		// create user

		user.Email = userEmail
		user.Roles = iam.DefaultRoles

		userDocument, err = userDocument.Set(dctx, user)
		if err != nil {
			return nil, errors.New(err.Error()+"3")
		}

		userDocument.SetOwner(userKey)

		err = SetAccess(dctx, userDocument, userDocument.Key(), FullControl)
		if err != nil {
			return nil, errors.New(err.Error()+"4")
		}
	} else {
		// load user and save changes

		user = collection.UserCollection.Data(userDocument, false, false).(*collection.User)

		// add default roles to the existing user
		var toAppend []string
		for _, r := range iam.DefaultRoles {
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
		userDocument, err = userDocument.Set(dctx, user)
		if err != nil {
			return nil, errors.New(err.Error()+"6")
		}

	}

	// todo: set user here
	identity.User = user


	return identity, nil
}

func createConfirmationURL(ctx context.Context, key *datastore.Key) string {
	var hostname string
	var module = appengine.ModuleName(ctx)
	var app = appengine.AppID(ctx)

	if len(module) > 0 {
		hostname += module + "-dot-"
	}

	hostname += app

	return "https://" + hostname + ".appspot.com/auth/confirm/" + key.Encode()
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
