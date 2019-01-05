package apis

import (
	"golang.org/x/net/context"
	"google.golang.org/appengine/datastore"
	"github.com/ales6164/apis/group"
	"github.com/ales6164/apis/kind"
)

type User struct {
	Id             string   `datastore:"-" auto:"id" json:"id"`
	Email          string   `json:"email"`
	EmailConfirmed bool     `json:"emailConfirmed"`
	Roles          []string `json:"roles"`
}

var UserKind = group.New("users", User{})

// Connects provider identity with user account. Creates account if it doesn't exist. Should be run inside a transaction.
// TrustUserEmail should be always false.
func (a *Auth) CreateUser(ctx context.Context, userEmail string, trustUserEmail bool) (kind.Doc, error) {
	userKey := datastore.NewKey(ctx, UserKind.Name(), userEmail, 0, nil)
	userHolder, err := UserKind.Doc(ctx, userKey).Add(&User{
		Id:             userEmail,
		EmailConfirmed: trustUserEmail,
		Email:          userEmail,
		Roles:          a.DefaultRoles,
	})
	if err != nil && err == kind.ErrEntityAlreadyExists && trustUserEmail {
		user := UserKind.Data(userHolder).(*User)
		// account already exists and userEmail is trusted and confirmed
		if user.EmailConfirmed != trustUserEmail {
			// since userEmail is trusted and confirmed, update db entry
			user.EmailConfirmed = trustUserEmail
			userHolder, err = userHolder.Set(user)
			return userHolder, err
		}
		return userHolder, nil
	}
	return userHolder, kind.ErrEntityAlreadyExists
}
