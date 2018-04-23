package kind

import (
	"golang.org/x/net/context"
	"google.golang.org/appengine/datastore"
	"github.com/ales6164/apis/errors"
)

type Filter struct {
	FilterStr string
	Value     interface{}
}

func (k *Kind) Query(ctx context.Context, order string, limit int, offset int, filters []Filter, ancestor *datastore.Key) ([]*Holder, error) {
	var hs []*Holder
	var err error
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
	t := q.Run(ctx)
	for {
		var h = k.NewHolder(nil)
		h.meta.Id, err = t.Next(h)
		if err == datastore.Done {
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
	h.meta.Id = key
	return datastore.Get(ctx, key, h)
}

// key id must be a string otherwise it creates incomplete key
func (h *Holder) Add(ctx context.Context) error {
	if !h.hasKey || h.meta.Id == nil {
		h.meta.Id = h.Kind.NewIncompleteKey(ctx, h.user)
	}
	if h.meta.Id.Incomplete() {
		var err error
		h.meta.Id, err = datastore.Put(ctx, h.meta.Id, h)
		return err
	} else {
		return datastore.RunInTransaction(ctx, func(tc context.Context) error {
			err := datastore.Get(tc, h.meta.Id, h)
			if err != nil {
				if err == datastore.ErrNoSuchEntity {
					h.meta.Id, err = datastore.Put(tc, h.meta.Id, h)
					return err
				}
				return err
			}
			return errors.ErrEntityExists
		}, nil)
	}
}

func (h *Holder) Update(ctx context.Context, key *datastore.Key) error {
	h.meta.Id = key
	err := datastore.RunInTransaction(ctx, func(tc context.Context) error {
		err := datastore.Get(tc, h.meta.Id, h)
		if err != nil {
			return err
		}
		h.meta.Id, err = datastore.Put(ctx, h.meta.Id, h)
		return err
	}, &datastore.TransactionOptions{XG: true})
	return err
}

func (h *Holder) Delete(ctx context.Context, key *datastore.Key) error {
	h.meta.Id = key
	err := datastore.Delete(ctx, h.meta.Id)
	if err != nil {
		return err
	}
	return nil
}
