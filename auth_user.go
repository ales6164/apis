package apis

import (
	"errors"
	"golang.org/x/net/context"
	"google.golang.org/appengine/datastore"
)

type User struct {
	Id             *datastore.Key `datastore:"-" auto:"id" json:"id"`
	Email          string         `json:"email"`
	EmailConfirmed bool           `json:"emailConfirmed"`
	Scopes         []string       `json:"scopes"`
}

var UserKind = NewKind(&KindOptions{
	Path:         "users",
	Type:         User{},
	IsCollection: true,
})

// Connects provider identity with user account. Creates account if it doesn't exist. Should be run inside a transaction.
// TrustUserEmail should be always false.
func (a *Auth) GetUser(ctx context.Context, userEmail string, trustUserEmail bool) (*User, error) {
	userKey := datastore.NewKey(ctx, UserKind.Path, userEmail, 0, nil)
	userHolder, err := UserKind.Get(ctx, userKey)
	if err != nil {
		if err == datastore.ErrNoSuchEntity {
			// Create user
			userHolder.SetValue(&User{
				Id:             userKey,
				EmailConfirmed: trustUserEmail,
				Email:          userEmail,
				Scopes:         a.DefaultScopes,
			})
			err = UserKind.Put(ctx, userHolder)
			return userHolder.GetValue().(*User), err
		}
		return nil, err
	} else if trustUserEmail {
		user := userHolder.GetValue().(*User)
		// account already exists and userEmail is trusted and confirmed
		if user.EmailConfirmed != trustUserEmail {
			// since userEmail is trusted and confirmed, update db entry
			user.EmailConfirmed = trustUserEmail
			userHolder.SetValue(user)
			err = UserKind.Put(ctx, userHolder)
			return userHolder.GetValue().(*User), err
		}
		return user, nil
	}
	return nil, errors.New("user already exists")
}
