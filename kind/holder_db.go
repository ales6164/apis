package kind

import (
	"golang.org/x/net/context"
	"google.golang.org/appengine/datastore"
)

func (k *Kind) Get(ctx context.Context, key *datastore.Key) (*Holder, error) {
	var h = k.NewHolder(ctx, nil)
	h.key = key

	err := datastore.Get(ctx, key, h)
	return h, err
}

func (h *Holder) Add() error {
	var err error

	h.key = h.Kind.NewIncompleteKey(h.context, nil)
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