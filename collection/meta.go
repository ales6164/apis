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
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	CreatedBy *datastore.Key `json:"createdBy"`
	UpdatedBy *datastore.Key `json:"updatedBy"`
	GroupId   string         `json:"-"`
	Id        string         `json:"-"` // every entry should have unique namespace --- or maybe auto generated if needed
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
		m.value.CreatedBy = d.GetAuthor()
		m.value.UpdatedBy = d.GetAuthor()
		m.value.CreatedAt = time.Now()
		m.value.UpdatedAt = m.value.CreatedAt
		m.value.GroupId = groupId
		m.value.Id = RandStringBytesMaskImprSrc(LetterNumberBytes, 6)
	} else {
		k := metaKey(ctx, d, groupKey)
		err = datastore.Get(ctx, k, &m.value)
		if err != nil {
			if err == datastore.ErrNoSuchEntity {
				m.value.CreatedBy = d.GetAuthor()
				m.value.UpdatedBy = d.GetAuthor()
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

func (m *meta) ID() string {
	return m.value.Id
}

func (m *meta) UpdatedAt() time.Time {
	return m.value.UpdatedAt
}

func (m *meta) CreatedAt() time.Time {
	return m.value.CreatedAt
}

func (m *meta) CreatedBy() *datastore.Key {
	return m.value.CreatedBy
}

func (m *meta) UpdatedBy() *datastore.Key {
	return m.value.UpdatedBy
}

type OutputMeta struct {
	Id        string      `json:"id"`
	CreatedAt time.Time   `json:"createdAt"`
	UpdatedAt time.Time   `json:"updatedAt"`
	CreatedBy interface{} `json:"createdBy,omitempty"`
	UpdatedBy interface{} `json:"updatedBy,omitempty"`
	Value     interface{} `json:"value"`
}

func (m *meta) Print(d kind.Doc, value interface{}) interface{} {
	var id string
	if d.Key().IntID() > 0 {
		id = d.Key().Encode()
	} else {
		id = d.Key().StringID()
	}

	createdBy, _ := PublicUserCollection.Doc(d.DefaultContext(), m.CreatedBy(), nil)
	updatedBy, _ := PublicUserCollection.Doc(d.DefaultContext(), m.UpdatedBy(), nil)

	createdBy, _ = createdBy.Get()
	updatedBy, _ = updatedBy.Get()

	return &OutputMeta{
		Id:        id,
		CreatedAt: m.value.CreatedAt,
		UpdatedAt: m.value.UpdatedAt,
		CreatedBy: PublicUserCollection.Data(createdBy, false),
		UpdatedBy: PublicUserCollection.Data(updatedBy, false),
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
	}
	if m.value.CreatedBy == nil {
		m.value.CreatedBy = d.GetAuthor()
	}
	m.value.UpdatedBy = d.GetAuthor()
	m.value.UpdatedAt = time.Now()

	m.key, err = datastore.Put(ctx, m.key, &m.value)
	m.exists = err == nil
	return err
}

func SetNamespace(ctx context.Context, key *datastore.Key, namespace string) (context.Context, *datastore.Key, error) {
	var err error
	ctx, err = appengine.Namespace(ctx, namespace)
	key = datastore.NewKey(ctx, key.Kind(), key.StringID(), key.IntID(), key.Parent())
	return ctx, key, err
}
