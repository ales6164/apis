package iam

import (
	"github.com/ales6164/apis/collection"
	"golang.org/x/net/context"
	"google.golang.org/appengine/datastore"
	"strconv"
)

// TODO: add something to load group namespace and namespace after access check?
// TODO: remove ctx as first param from collection.Doc? Move it to doc.Get/Set/Add....
func CheckAccess(ctx Context, doc collection.Doc, member *datastore.Key, permission ...Scope) (Context, bool) {
	accessController := doc.AccessController()
	if accessController != nil && accessController.Key() != nil && !accessController.Key().Incomplete() {
		var groupRel = new(DocUserRelationship)
		err := datastore.Get(ctx.Default(), datastore.NewKey(ctx.Default(), "_rel", member.StringID(), 0, accessController.Key()), groupRel)
		if err == nil && ContainsScope(groupRel.Roles, permission...) {
			ctx, err = GetGroupNamespace(ctx, accessController)
			if err != nil {
				return ctx, false
			}
			k := doc.Key()
			if k != nil {
				doc.SetKey(datastore.NewKey(ctx, k.Kind(), k.StringID(), k.IntID(), k.Parent()))
			}
			return ctx, true
		} else if err == datastore.ErrNoSuchEntity {
			return ctx, true
		}

		return ctx, false
	}
	return ctx, true
}

func GetGroupNamespace(ctx Context, doc collection.Doc) (Context, error) {
	var group = new(GroupNamespace)
	err := datastore.Get(ctx.Default(), datastore.NewKey(ctx.Default(), "_group", doc.Key().Encode(), 0, nil), group)
	if err != nil {
		return ctx, err
	}
	return ctx.SetNamespace(group.Namespace)
}

func SetAccess(ctx Context, doc collection.Doc, member *datastore.Key, permission ...string) error {
	docKey := doc.Key()
	docDefaultNsKey := datastore.NewKey(ctx.Default(), docKey.Kind(), docKey.StringID(), docKey.IntID(), docKey.Parent())

	err := datastore.RunInTransaction(ctx.Default(), func(ctx context.Context) error {
		var group = new(GroupNamespace)
		var key = datastore.NewKey(ctx, "_group", docDefaultNsKey.Encode(), 0, nil)
		err := datastore.Get(ctx, key, group)
		if err != nil {
			if err == datastore.ErrNoSuchEntity {
				ns, _, err := datastore.AllocateIDs(ctx, "_group", nil, 1)
				if err != nil {
					return err
				}
				group.Document = doc.Key()
				group.Namespace = "g-" + strconv.Itoa(int(ns))
				_, err = datastore.Put(ctx, key, group)
				if err != nil {
					return err
				}
				// continue
			} else {
				return err
			}
		}

		// TODO: get and update/store DocUserRelationship
		var groupRel = new(DocUserRelationship)
		var relKey = datastore.NewKey(ctx, "_rel", member.StringID(), 0, docDefaultNsKey)
		err = datastore.Get(ctx, relKey, groupRel)
		if err != nil {
			if err == datastore.ErrNoSuchEntity {
				groupRel.Roles = permission
				_, err = datastore.Put(ctx, relKey, groupRel)
				if err != nil {
					return err
				}
				return nil
			}
			return err
		}

		// add default roles to the existing user
		var toAppend []string
		for _, r := range permission {
			var ok bool
			for _, r2 := range groupRel.Roles {
				if r == r2 {
					ok = true
				}
			}
			if !ok {
				toAppend = append(toAppend, r)
			}
		}
		groupRel.Roles = append(groupRel.Roles, toAppend...)
		_, err = datastore.Put(ctx, relKey, groupRel)
		return err
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
