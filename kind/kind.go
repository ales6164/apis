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
	Parse(body []byte) error
	Get() (Doc, error)
	Add(data interface{}) (Doc, error) // transaction function in 1/2 case
	Set(data interface{}) (Doc, error)
	Patch(data []byte) error // transaction function
	Delete() error
	Kind() Kind
	Value() reflect.Value
	SetOwner(key *datastore.Key)
	GetOwner() *datastore.Key
	Key() *datastore.Key
	SetKey(key *datastore.Key)
	SetAccessControl(b bool)
	AccessController() Doc
}

type Kind interface {
	Name() string
	Key(ctx context.Context, str string, member *datastore.Key) *datastore.Key
	Data(doc Doc, includeMeta bool) interface{}
	ValueAt(value reflect.Value, path []string) (reflect.Value, error)
	Fields() map[string]Field
	Scopes(scopes ...string) []string
	Type() reflect.Type
	Count(ctx context.Context) (int, error)
	Increment(ctx context.Context) error
	Decrement(ctx context.Context) error
	Doc(ctx context.Context, key *datastore.Key, ancestor Doc) Doc
}
