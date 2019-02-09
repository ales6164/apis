package collection

import (
	"github.com/ales6164/apis/kind"
	"golang.org/x/net/context"
	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"strconv"
)

// parent is entry key, id is user key
type DocUserRelationship struct {
	Roles []string // fullControl, ...
}

// doc is parent; name is doc kind
type GroupNamespace struct {
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

func GetGroupNamespace(ctx context.Context, _doc kind.Doc) (context.Context, error) {
	var group = new(GroupNamespace)
	err := datastore.Get(ctx, datastore.NewKey(ctx, "_group", _doc.Key().Kind(), 0, _doc.Key()), group)
	if err != nil {
		return ctx, err
	}
	return appengine.Namespace(ctx, group.Namespace)
}

// TODO: add something to load group namespace and namespace after access check?
// TODO: remove ctx as first param fomr collection.Doc? Move it to doc.Get/Set/Add....
func CheckAccess(ctx context.Context, _doc kind.Doc, member *datastore.Key, permission ...string) (context.Context, bool) {
	accessController := _doc.AccessController()
	if accessController != nil {
		var groupRel = new(DocUserRelationship)
		err := datastore.Get(ctx, datastore.NewKey(ctx, "_rel", accessController.Key().Encode(), 0, member), groupRel)
		if err == nil && ContainsScope(groupRel.Roles, permission...) {
			ctx, err = GetGroupNamespace(ctx, accessController)
			if err != nil {
				return ctx, false
			}
			return ctx, true
		}
		return ctx, false
	}
	return ctx, true
}

func SetAccess(ctx context.Context, doc kind.Doc, member *datastore.Key, permission ...string) error {
	err := datastore.RunInTransaction(ctx, func(ctx context.Context) error {
		var group = new(GroupNamespace)
		var key = datastore.NewKey(ctx, "_group", doc.Key().StringID(), doc.Key().IntID(), nil)
		err := datastore.Get(ctx, key, group)
		if err != nil {
			if err == datastore.ErrNoSuchEntity {
				ns, _, err := datastore.AllocateIDs(ctx, "_group", nil, 1)
				if err != nil {
					return err
				}
				group.Namespace = strconv.Itoa(int(ns))
				_, err = datastore.Put(ctx, key, group)
				if err != nil {
					return err
				}
			}
			return err
		}
		
		// TODO: get and update/store DocUserRelationship

		return nil
	}, &datastore.TransactionOptions{
		XG: true,
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
