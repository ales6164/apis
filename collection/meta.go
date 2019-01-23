package collection

import (
	"errors"
	"github.com/ales6164/apis/kind"
	"golang.org/x/net/context"
	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"time"
)

type meta struct {
	key      *datastore.Key `datastore:"-" json:"-"`
	groupKey *datastore.Key `datastore:"-" json:"-"`
	group    kind.Meta      `datastore:"-" json:"-"`
	value    metaValue
	exists   bool
	kind.Meta
}

type metaValue struct {
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	GroupId   string    `json:"-"`
	Id        string    `json:"-"` // every entry should have unique namespace --- or maybe auto generated if needed
}

func metaKey(ctx context.Context, d kind.Doc, groupKey *datastore.Key) *datastore.Key {
	k := d.Key()
	return datastore.NewKey(ctx, "_meta_"+d.Kind().Name(), k.StringID(), k.IntID(), groupKey)
}

func getMeta(ctx context.Context, d kind.Doc, groupMeta kind.Meta) (*meta, error) {
	m := new(meta)
	var err error
	var groupKey *datastore.Key
	var groupId string
	if groupMeta != nil {
		groupKey = groupMeta.Key()
		groupId = groupMeta.ID()
	}
	if d.Key() == nil || d.Key().Incomplete() {
		m.value.CreatedAt = time.Now()
		m.value.UpdatedAt = m.value.CreatedAt
		m.value.GroupId = groupId
		m.value.Id = RandStringBytesMaskImprSrc(LetterNumberBytes, 6)
	} else {
		k := metaKey(ctx, d, groupKey)
		err = datastore.Get(ctx, k, &m.value)
		if err != nil {
			if err == datastore.ErrNoSuchEntity {
				m.value.CreatedAt = time.Now()
				m.value.UpdatedAt = m.value.CreatedAt
				m.value.GroupId = groupId
				m.value.Id = RandStringBytesMaskImprSrc(LetterNumberBytes, 6)
			} else {
				return m, err
			}
		} else {
			m.exists = true
		}
		m.key = k
	}
	m.group = groupMeta
	m.groupKey = groupKey
	return m, nil
}

/*func setMeta(ctx context.Context, d kind.Doc, m *metaValue, ancestor *datastore.Key) error {
	k := metaKey(ctx, d, ancestor)
	_, err := datastore.Put(ctx, k, m)
	return err
}*/

func (m *meta) ID() string {
	return m.value.Id
}

type OutputMeta struct {
	Id        string      `json:"id"`
	CreatedAt time.Time   `json:"createdAt"`
	UpdatedAt time.Time   `json:"updatedAt"`
	Value     interface{} `json:"value"`
}

func (m *meta) Print(d kind.Doc, value interface{}) interface{} {
	var id string
	if d.Key().IntID() > 0 {
		id = d.Key().Encode()
	} else {
		id = d.Key().StringID()
	}
	return &OutputMeta{
		Id:        id,
		CreatedAt: m.value.CreatedAt,
		UpdatedAt: m.value.UpdatedAt,
		Value:     value,
	}
}

func (m *meta) Key() *datastore.Key {
	return m.key
}

func (m *meta) Exists() bool {
	return m.exists
}

func (m *meta) Save(ctx context.Context, d kind.Doc, groupMeta kind.Meta) error {
	var err error
	if d.Key() == nil || d.Key().Incomplete() {
		return errors.New("incomplete key")
	}
	if m.key == nil {
		var groupMetaKey *datastore.Key
		if groupMeta != nil {
			groupMetaKey = groupMeta.Key()
		}
		m.key = metaKey(ctx, d, groupMetaKey)
	} else {
		m.value.UpdatedAt = time.Now()
	}
	m.key, err = datastore.Put(ctx, m.key, &m.value)
	m.exists = err == nil
	return err
}

// Loads relationship table and checks if user has access to the specified namespace.
// Then adds the parent and rewrites document key and context.
/*func Meta(ctx context.Context, d kind.Doc) (*meta, func() error, error) {
	var ancestorMeta *meta
	var err error
	if d.Ancestor() != nil {
		ancestorMeta, _, err = Meta(ctx, d.Ancestor())
		if err != nil {
			return nil, nil, err
		}
	}

	var entryMeta = new(meta)
	entryMeta.AncestorKey = ancestorMeta.Key

	if d.Key() == nil || d.Key().Incomplete() {
		entryMeta.metaValue = new(metaValue)
		if ancestorMeta != nil {
			entryMeta.metaValue.GroupID = ancestorMeta.ID
		}
		entryMeta.metaValue.ID = RandStringBytesMaskImprSrc(LetterNumberBytes, 6)
		entryMeta.metaValue.UpdatedAt = time.Now()
		entryMeta.metaValue.CreatedAt = entryMeta.metaValue.UpdatedAt
		return entryMeta, func() error {
			return setMeta(ctx, d, entryMeta.metaValue, ancestorMeta.Key)
		}, nil
	}

	err = datastore.RunInTransaction(ctx, func(tc context.Context) error {
		entryMeta.Key, entryMeta.metaValue, err = getMeta(ctx, d, ancestorMeta.Key)
		if err != nil {
			return err
		}
		if len(entryMeta.metaValue.ID) == 0 {
			entryMeta.metaValue.ID = RandStringBytesMaskImprSrc(LetterNumberBytes, 6)
		}
		if ancestorMeta != nil {
			if entryMeta.metaValue.GroupID != ancestorMeta.ID {
				if len(entryMeta.metaValue.GroupID) == 0 {
					entryMeta.metaValue.UpdatedAt = time.Now()
					entryMeta.metaValue.GroupID = ancestorMeta.ID
					err = setMeta(ctx, d, entryMeta.metaValue, ancestorMeta.Key)
				} else {
					return errors.New("hierarchy error")
				}
			}
		}

		return err
	}, nil)
	if err != nil {
		return entryMeta, nil, err
	}

	return entryMeta, nil, err
}*/

func SetNamespace(ctx context.Context, key *datastore.Key, namespace string) (context.Context, *datastore.Key, error) {
	var err error
	ctx, err = appengine.Namespace(ctx, namespace)
	key = datastore.NewKey(ctx, key.Kind(), key.StringID(), key.IntID(), key.Parent())
	return ctx, key, err
}
