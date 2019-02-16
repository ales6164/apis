package iam

import (
	"cloud.google.com/go/iam"
	"github.com/ales6164/apis/collection"
	"golang.org/x/net/context"
	"google.golang.org/appengine/datastore"
	"strconv"
)

func GetScopes(ctx Context, accessController collection.Doc, rules map[string][]string) ([]string, error) {
	var userScopes []string
	var scopesDefined bool
	if accessController != nil && accessController.Key() != nil && !accessController.Key().Incomplete() {
		var rel = new(collection.DocUserRelationship)
		err := datastore.Get(ctx.Default(), datastore.NewKey(ctx.Default(), "_rel", ctx.Member().Encode(), 0, accessController.Key()), rel)
		userScopes = rel.Scopes
		if err != nil {
			if err != datastore.ErrNoSuchEntity {
				return nil, err
			}
		} else {
			scopesDefined = true
		}
	}

	if !scopesDefined && rules != nil {
		userScopes = rules[iam.AllUsers]
		if ctx.IsAuthenticated() {
			userScopes = append(userScopes, rules[iam.AllAuthenticatedUsers]...)
			for _, r := range ctx.Roles() {
				userScopes = append(userScopes, rules[r]...)
			}
		}
	}

	return userScopes, nil
}

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
