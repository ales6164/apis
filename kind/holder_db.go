package kind

import (
	"golang.org/x/net/context"
	"google.golang.org/appengine/datastore"
)

func (k *Kind) Query(ctx context.Context, order string, limit int, offset int, ancestor *datastore.Key) ([]*Holder, error) {
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

	if ancestor != nil {
		q = q.Ancestor(ancestor)
	}

	t := q.Run(ctx)
	for {
		var h = k.NewEmptyHolder()
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

func (k *Kind) Get(ctx context.Context, key *datastore.Key) (*Holder, error) {
	var h = k.NewHolder(ctx, nil)
	h.key = key

	err := datastore.Get(ctx, key, h)
	return h, err
}

func (h *Holder) Get(key *datastore.Key) error {
	h.key = key
	return datastore.Get(h.context, key, h)
}

func (h *Holder) Add(userKey *datastore.Key) error {
	var err error

	h.key = h.Kind.NewIncompleteKey(h.context, userKey)
	h.key, err = datastore.Put(h.context, h.key, h)
	if err != nil {
		return err
	}

	//dataHolder.updateSearchIndex()

	return nil
}

func (h *Holder) Update(key *datastore.Key) error {
	h.key = key
	err := datastore.RunInTransaction(h.context, func(tc context.Context) error {
		err := datastore.Get(tc, h.key, h)
		if err != nil {
			return err
		}

		var replacementKey = h.Kind.NewIncompleteKey(tc, h.key)
		var oldHolder = h.OldHolder(replacementKey)

		var keys = []*datastore.Key{replacementKey, h.key}
		var holders = []interface{}{oldHolder, h}

		keys, err = datastore.PutMulti(tc, keys, holders)
		return err
	}, &datastore.TransactionOptions{XG: true})

	//dataHolder.updateSearchIndex()

	return err
}

func (h *Holder) Delete(key *datastore.Key) error {
	h.key = key
	err := datastore.Delete(h.context, h.key)
	if err != nil {
		return err
	}
	//dataHolder.updateSearchIndex()
	return nil
}
