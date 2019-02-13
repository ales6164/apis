package collection

import (
	"google.golang.org/appengine/datastore"
	"time"
)

type Meta struct {
	Id              string         `datastore:"-" json:"id,omitempty"`
	CreatedAt       time.Time      `datastore:"createdAt" json:"createdAt,omitempty"`
	UpdatedAt       time.Time      `datastore:"updatedAt" json:"updatedAt,omitempty"`
	CreatedBy       *datastore.Key `datastore:"createdBy" json:"createdBy,omitempty"`
	UpdatedBy       *datastore.Key `datastore:"updatedBy" json:"updatedBy,omitempty"`
	Version         int64          `datastore:"version" json:"version,omitempty"`
	Namespace       string         `datastore:"namespace,noindex" json:"-"`
	ParentNamespace string         `datastore:"parentNamespace,noindex" json:"-"`
	Value           interface{}    `datastore:"-" json:"value"`
	DocMeta         `datastore:"-" json:"-"`
}

func (m Meta) WithValue(key *datastore.Key, v interface{}) DocMeta {
	if len(key.StringID()) > 0 {
		m.Id = key.StringID()
	} else {
		m.Id = key.Encode()
	}
	m.Value = v
	return m
}

func (m Meta) GetUpdatedAt() time.Time {
	return m.UpdatedAt
}
