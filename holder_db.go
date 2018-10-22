package apis

import (
	"github.com/ales6164/apis/errors"
	"golang.org/x/net/context"
	"google.golang.org/appengine/datastore"
)

type Filter struct {
	FilterStr string
	Value     interface{}
}

func (k *Kind) Query(ctx context.Context, order string, limit int, offset int, filters []Filter, ancestor *datastore.Key) ([]*Holder, error) {
	var hs []*Holder
	var err error
	q := datastore.NewQuery(k.name)
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
		var h = k.NewHolder()
		h.key, err = t.Next(h)
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


func GetMulti(ctx context.Context, kind *Kind, key ...*datastore.Key) ([]*Holder, error) {
	var hs []*Holder
	for _, k := range key {
		h := kind.NewHolder()
		h.SetKey(k)
		hs = append(hs, h)
	}

	err := datastore.GetMulti(ctx, key, hs)
	return hs, err
}

func (h *Holder) Get(ctx context.Context, key *datastore.Key) error {
	h.SetKey(key)
	return datastore.Get(ctx, key, h)
}

// key id must be a string otherwise it creates incomplete key
func (h *Holder) Add(_ctx context.Context, key *datastore.Key) error {
	return datastore.RunInTransaction(_ctx, func(tc context.Context) error {
		h.SetKey(key)
		var err error
		if h.key == nil {
			//h.key = h.Kind.NewIncompleteKey(tc, nil)
		}
		if h.key.Incomplete() {
			h.key, err = datastore.Put(tc, h.key, h)
			if err != nil {
				return err
			}
			/*return h.SaveToIndex(tc)*/
		} else {
			err = datastore.Get(tc, h.key, h)
			if err != nil {
				if err == datastore.ErrNoSuchEntity {
					h.key, err = datastore.Put(tc, h.key, h)
					if err != nil {
						return err
					}
					/*return h.SaveToIndex(tc)*/
				}
				return err
			}
			return errors.ErrEntityExists
		}
		return err
	}, nil)
}

func (h *Holder) Update(_ctx context.Context) error {
	return datastore.RunInTransaction(_ctx, func(tc context.Context) error {
		err := datastore.Get(tc, h.key, h)
		if err != nil {
			return err
		}
		h.key, err = datastore.Put(tc, h.key, h)
		if err != nil {
			return err
		}
		/*return h.SaveToIndex(tc)*/
		return nil
	}, nil)
}

func (h *Holder) Delete(_ctx context.Context) error {
	return datastore.RunInTransaction(_ctx, func(tc context.Context) error {
		err := datastore.Delete(tc, h.key)
		if err != nil {
			return err
		}
		//return h.Kind.DeleteFromIndex(tc, h.key.Encode())
		return nil
	}, nil)
}
