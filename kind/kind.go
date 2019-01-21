package kind

import (
	"errors"
	"golang.org/x/net/context"
	"google.golang.org/appengine/datastore"
	"reflect"
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
	SetKey(key *datastore.Key)
	Copy() Doc
	SetRole(member *datastore.Key, role ...string) error
	HasRole(member *datastore.Key, role ...string) bool
	HasAncestor() bool
	Context() context.Context
	Meta() (Meta, error)
	/*SetParent(doc Doc) (Doc, error)*/
}

type Meta interface {
	Save(ctx context.Context, doc Doc, groupMeta Meta) error
	Key() *datastore.Key
	ID() string
}

type Kind interface {
	Name() string
	Data(doc Doc) interface{}
	ValueAt(value reflect.Value, path []string) (reflect.Value, error)
	Fields() map[string]Field
	Scopes(scopes ...string) []string
	Type() reflect.Type
	Count(ctx context.Context) (int, error)
	Increment(ctx context.Context) error
	Decrement(ctx context.Context) error
	Doc(ctx context.Context, key *datastore.Key, ancestor Doc) (Doc, error)
}
