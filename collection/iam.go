package collection

import (
	"golang.org/x/net/context"
	"google.golang.org/appengine/datastore"
)

// parent is entry key, id is user key
type GroupRelationship struct {
	Roles []string // fullControl, ...
}

type Role string

const (
	AllUsers              = "allUsers"              // given to all requests
	AllAuthenticatedUsers = "allAuthenticatedUsers" // giver to all authenticated requests

	FullControl = "fullcontrol"
	ReadOnly    = "readonly"
	ReadWrite   = "readwrite"
	Delete      = "delete"
)

func OwnerIAM(ctx context.Context, memberKey *datastore.Key, ancestorMetaKey *datastore.Key) error {
	_, err := datastore.Put(ctx, datastore.NewKey(ctx, "_groupRelationship", memberKey.StringID(), memberKey.IntID(), ancestorMetaKey), &GroupRelationship{
		Roles: []string{FullControl},
	})
	return err
}

// should check if got scope groupName.collectionName.scope
func CheckCollectionAccess(ctx context.Context, memberKey *datastore.Key, isAuthenticated bool, ancestorMetaKey *datastore.Key, roles ...string) bool {
	var iam = new(GroupRelationship)
	// AllUsers
	err := datastore.Get(ctx, datastore.NewKey(ctx, "_groupRelationship", AllUsers, 0, ancestorMetaKey), iam)
	if err == nil && ContainsScope(iam.Roles, roles...) {
		return true
	}
	if isAuthenticated {
		// AllAuthenticatedUsers
		err = datastore.Get(ctx, datastore.NewKey(ctx, "_groupRelationship", AllAuthenticatedUsers, 0, ancestorMetaKey), iam)
		if err == nil && ContainsScope(iam.Roles, roles...) {
			return true
		}
		// User
		err = datastore.Get(ctx, datastore.NewKey(ctx, "_groupRelationship", memberKey.StringID(), memberKey.IntID(), ancestorMetaKey), iam)
		if err == nil && ContainsScope(iam.Roles, roles...) {
			return true
		}
	}
	return false
}

func ContainsScope(arr []string, scopes ...string) bool {
	for _, scp := range scopes {
		for _, r := range arr {
			if r == scp {
				return true
			}
		}
	}
	return false
}