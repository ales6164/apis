package kind

import (
	"cloud.google.com/go/datastore"
	"github.com/ales6164/apis/errors"
	"golang.org/x/net/context"
	"google.golang.org/api/iterator"
)

type Filter struct {
	FilterStr string
	Value     interface{}
}

func (k *Kind) Query(ctx context.Context, order string, limit int, offset int, filters []Filter, ancestor *datastore.Key) ([]*Holder, error) {
	var hs []*Holder
	c, err := datastore.NewClient(ctx, k.ProjectID)
	if err != nil {
		return hs, err
	}
	q := datastore.NewQuery(k.Name)
	if len(order) > 0 {
		q = q.Order(order)
	}
	if limit > 0 {
		q = q.Limit(limit)
	}
	if offset > 0 {
		q = q.Offset(offset)
	}
	if len(filters) > 0 {
		for _, f := range filters {
			q = q.Filter(f.FilterStr, f.Value)
		}
	}
	if ancestor != nil {
		q = q.Ancestor(ancestor)
	}
	t := c.Run(ctx, q)
	for {
		var h = k.NewHolder(k.ProjectID, nil)
		h.key, err = t.Next(h)
		if err == iterator.Done {
			break
		}
		if err != nil {
			return hs, err
		}
		hs = append(hs, h)
	}
	return hs, nil
}

func (h *Holder) Get(ctx context.Context, key *datastore.Key) error {
	h.key = key
	c, err := datastore.NewClient(ctx, h.ProjectID)
	if err != nil {
		return err
	}
	return c.Get(ctx, key, h)
}

// key id must be a string otherwise it creates incomplete key
func (h *Holder) Add(ctx context.Context) error {
	if !h.hasKey || h.key == nil {
		h.key = h.Kind.NewIncompleteKey(ctx, h.user)
	}
	c, err := datastore.NewClient(ctx, h.ProjectID)
	if err != nil {
		return err
	}
	if h.key.Incomplete() {
		var err error
		h.key, err = c.Put(ctx, h.key, h)
		return err
	} else {
		var pendingKey *datastore.PendingKey
		commit, err := c.RunInTransaction(ctx, func(tx *datastore.Transaction) error {
			err := tx.Get(h.key, h)
			if err != nil {
				if err == datastore.ErrNoSuchEntity {
					pendingKey, err = tx.Put(h.key, h)
					return err
				}
				return err
			}
			return errors.ErrEntityExists
		}, nil)
		if err != nil {
			return err
		}
		h.key = commit.Key(pendingKey)
		return err
	}
}

func (h *Holder) Update(ctx context.Context, key *datastore.Key) error {
	h.key = key
	c, err := datastore.NewClient(ctx, h.ProjectID)
	if err != nil {
		return err
	}
	var pendingKey *datastore.PendingKey
	commit, err := c.RunInTransaction(ctx, func(tx *datastore.Transaction) error {
		err := tx.Get(h.key, h)
		if err != nil {
			return err
		}
		pendingKey, err = tx.Put(h.key, h)
		return err
	})
	if err != nil {
		return err
	}
	h.key = commit.Key(pendingKey)
	return err
}

func (h *Holder) Delete(ctx context.Context, key *datastore.Key) error {
	h.key = key
	c, err := datastore.NewClient(ctx, h.ProjectID)
	if err != nil {
		return err
	}
	return c.Delete(ctx, h.key)
}
