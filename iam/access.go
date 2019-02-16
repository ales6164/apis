package iam

import (
	"github.com/ales6164/apis/collection"
	"golang.org/x/net/context"
	"google.golang.org/appengine/datastore"
	"strconv"
)

func SetAccess(ctx Context, doc collection.Doc, member *datastore.Key, scopes ...string) error {
	err := datastore.RunInTransaction(ctx.Default(), func(ctx context.Context) error {
		// get group
		var groupKey = datastore.NewKey(ctx, "_group", doc.Key().Encode(), 0, nil)
		var group = new(collection.Group)
		err := datastore.Get(ctx, groupKey, group)
		if err != nil {
			if err == datastore.ErrNoSuchEntity {
				// create group
				ns, _, err := datastore.AllocateIDs(ctx, "_group", nil, 1)
				if err != nil {
					return err
				}

				group.Namespace = "g-" + strconv.Itoa(int(ns))
				group.Document = doc.Key()
				_, err = datastore.Put(ctx, groupKey, group)
				if err != nil {
					return err
				}
			} else {
				return err
			}
		}

		var relKey = datastore.NewKey(ctx, "_rel", member.Encode(), 0, doc.Key())
		var rel = new(collection.DocUserRelationship)

		rel.Scopes = scopes

		_, err = datastore.Put(ctx, relKey, rel)
		return err
	}, &datastore.TransactionOptions{XG: true})
	return err
}