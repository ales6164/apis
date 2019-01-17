package kind

import (
	"errors"
	"golang.org/x/net/context"
	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"reflect"
	"time"
)

var (
	ErrEntityAlreadyExists = errors.New("entity already exists") // on doc.Add() if entity already exists
)

type Field interface {
	Name() string
	Fields() map[string]Field
	Type() string
}

type Doc interface {
	Type() reflect.Type
	/*RichData() interface{}*/
	Parse(body []byte) error
	Get() (Doc, error)
	Ancestor() Doc
	Add(data interface{}) (Doc, error) // transaction function in 1/2 case
	Set(data interface{}) (Doc, error)
	Patch(data []byte) error // transaction function
	Delete() error
	Kind() Kind
	Value() reflect.Value
	Key() *datastore.Key
	/*SetParent(doc Doc) (Doc, error)*/
}

type Kind interface {
	Name() string
	Data(doc Doc) interface{}
	ValueAt(value reflect.Value, path []string) (reflect.Value, error)
	Fields() map[string]Field
	Scopes(scopes ...string) []string
	Type() reflect.Type
	Doc(ctx context.Context, key *datastore.Key, ancestor Doc) Doc
}

type meta struct {
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	CreatedBy *datastore.Key `json:"createdBy"`
	UpdatedBy *datastore.Key `json:"updatedBy"`
	GroupID   string         `json:"-"`
	ID        string         `json:"-"` // every entry should have unique namespace --- or maybe auto generated if needed
}

func metaKey(ctx context.Context, d Doc, ancestor *datastore.Key) *datastore.Key {
	k := d.Key()
	return datastore.NewKey(ctx, "_meta_"+d.Kind().Name(), k.StringID(), k.IntID(), ancestor)
}

func getMeta(ctx context.Context, d Doc, ancestor *datastore.Key) (*datastore.Key, *meta, error) {
	m := new(meta)
	k := metaKey(ctx, d, ancestor)
	err := datastore.Get(ctx, k, m)
	return k, m, err
}

func setMeta(ctx context.Context, d Doc, m *meta, ancestor *datastore.Key) error {
	k := metaKey(ctx, d, ancestor)
	_, err := datastore.Put(ctx, k, m)
	return err
}

// Loads relationship table and checks if user has access to the specified namespace.
// Then adds the parent and rewrites document key and context.
func Meta(ctx context.Context, d Doc) (*datastore.Key, *meta, func() error, error) {
	var ancestorMeta *meta
	var ancestorMetaKey *datastore.Key
	var err error
	if d.Ancestor() != nil {
		ancestorMetaKey, ancestorMeta, _, err = Meta(ctx, d.Ancestor())
		if err != nil {
			return nil, nil, nil, err
		}
	}

	var mKey *datastore.Key
	var m *meta

	if d.Key() == nil || d.Key().Incomplete() {
		m = new(meta)
		if ancestorMeta != nil {
			m.GroupID = ancestorMeta.ID
		}
		m.ID = RandStringBytesMaskImprSrc(LetterNumberBytes, 6)
		m.UpdatedAt = time.Now()
		m.CreatedAt = m.UpdatedAt
		return mKey, m, func() error {
			return setMeta(ctx, d, m, ancestorMetaKey)
		}, nil
	}

	err = datastore.RunInTransaction(ctx, func(tc context.Context) error {
		mKey, m, err = getMeta(ctx, d, ancestorMetaKey)
		if err != nil {
			return err
		}
		if len(m.ID) == 0 {
			m.ID = RandStringBytesMaskImprSrc(LetterNumberBytes, 6)
		}
		if ancestorMeta != nil {
			if m.GroupID != ancestorMeta.ID {
				if len(m.GroupID) == 0 {
					m.UpdatedAt = time.Now()
					m.GroupID = ancestorMeta.ID
					err = setMeta(ctx, d, m, ancestorMetaKey)
				} else {
					return errors.New("hierarchy error")
				}
			}
		}

		return err
	}, nil)
	if err != nil {
		return mKey, m, nil, err
	}

	return mKey, m, nil, err
}

func SetNamespace(ctx context.Context, key *datastore.Key, namespace string) (context.Context, *datastore.Key, error) {
	var err error
	ctx, err = appengine.Namespace(ctx, namespace)
	key = datastore.NewKey(ctx, key.Kind(), key.StringID(), key.IntID(), key.Parent())
	return ctx, key, err
}
