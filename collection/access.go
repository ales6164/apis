package collection

import (
	"github.com/ales6164/apis/kind"
	"golang.org/x/net/context"
	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
)

// parent is entry key, id is user key
type DocUserRelationship struct {
	Roles     []string // fullControl, ...
	Namespace string
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

// TODO: add something to load group namespace and namespace after access check?
// TODO: remove ctx as first param fomr collection.Doc? Move it to doc.Get/Set/Add....
func CheckAccess(ctx context.Context, _doc kind.Doc, member *datastore.Key, permission ...string) (context.Context, bool) {
	accessController := _doc.AccessController()
	if accessController != nil {
		var groupRel = new(DocUserRelationship)
		err := datastore.Get(ctx, datastore.NewKey(ctx, "_rel", accessController.Key().Encode(), 0, member), groupRel)
		if err == nil && ContainsScope(groupRel.Roles, permission...) {
			appengine.Namespace(ctx, )
			return true,
		}
		return false
	}
	return true
}

func SetAccess(ctx context.Context, doc kind.Doc, member *datastore.Key, permission ...string) error {
	_, err := datastore.Put(ctx, datastore.NewKey(ctx, "_rel", doc.Key().Encode(), 0, member), &DocUserRelationship{
		Roles: permission,
	})
	return err
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
