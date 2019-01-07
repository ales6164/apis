package kind

import (
	"reflect"
	"google.golang.org/appengine/datastore"
	"golang.org/x/net/context"
	"errors"
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
	Add(data interface{}) (Doc, error) // transaction function in 1/2 case
	Set(data interface{}) (Doc, error)
	Patch(data []byte) error // transaction function
	Delete() error
	Kind() Kind
	Value() reflect.Value
	Key() *datastore.Key
}

type Kind interface {
	Name() string
	Data(doc Doc) interface{}
	ValueAt(value reflect.Value, path []string) (reflect.Value, error)
	Fields() map[string]Field
	IsNamespace() bool
	Scopes(scopes ...string) []string
	Type() reflect.Type
	Doc(ctx context.Context, key *datastore.Key) Doc
}
