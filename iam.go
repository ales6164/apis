package apis

import (
	"golang.org/x/net/context"
	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
)

// parent is entry key, id is user key
type IAM struct {
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

func OwnerIAM(ctx context.Context, memberKey *datastore.Key, collectionKey *datastore.Key) error {
	ctx, _ = appengine.Namespace(ctx, "default")
	_, err := datastore.Put(ctx, datastore.NewKey(ctx, "_iam", memberKey.StringID(), memberKey.IntID(), collectionKey), &IAM{
		Roles: []string{FullControl},
	})
	return err
}

// should check if got scope groupName.collectionName.scope
func CheckCollectionAccess(ctx Context, collectionKey *datastore.Key, roles ...string) (Context, bool) {
	var iam = new(IAM)
	// AllUsers
	err := datastore.Get(ctx, datastore.NewKey(ctx, "_iam", AllUsers, 0, collectionKey), iam)
	if err == nil && ContainsScope(iam.Roles, roles...) {
		ctx.Context, _ = appengine.Namespace(ctx, collectionKey.Encode())
		return ctx, true
	}
	if ctx.session.isAuthenticated {
		// AllAuthenticatedUsers
		err = datastore.Get(ctx, datastore.NewKey(ctx, "_iam", AllAuthenticatedUsers, 0, collectionKey), iam)
		if err == nil && ContainsScope(iam.Roles, roles...) {
			ctx.Context, _ = appengine.Namespace(ctx, collectionKey.Encode())
			return ctx, true
		}

		// User
		member := ctx.Member()
		err = datastore.Get(ctx, datastore.NewKey(ctx, "_iam", member.StringID(), member.IntID(), collectionKey), iam)
		if err == nil && ContainsScope(iam.Roles, roles...) {
			ctx.Context, _ = appengine.Namespace(ctx, collectionKey.Encode())
			return ctx, true
		}
	}

	return ctx, false
}

// key is made of group entity key and parent is parent group id
type Group struct {
	GroupID string
}

func createGroup() {

}

func getGroupId(ctx context.Context, parentGroupKey *datastore.Key, entityKey *datastore.Key) (*datastore.Key, string, error) {
	groupKey := datastore.NewKey(ctx, "_group", entityKey.StringID(), entityKey.IntID(), parentGroupKey)
	var group = new(Group)
	err := datastore.Get(ctx, groupKey, group)
	return groupKey, group.GroupID, err
}
