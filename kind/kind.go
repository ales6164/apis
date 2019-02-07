package kind

import (
	"errors"
	"golang.org/x/net/context"
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
	SetAuthor(key *datastore.Key)
	GetAuthor() *datastore.Key
	Key() *datastore.Key
	SetKey(key *datastore.Key)
	Copy() Doc
	// TODO: SetRole and HasRole rename into SetAccess and HasAccess ... and only check for these if Rule.EnableAccessControl is true
	SetRole(member *datastore.Key, role ...string) error
	HasRole(member *datastore.Key, role ...string) bool
	HasAncestor() bool
	Context() context.Context
	DefaultContext() context.Context
	Meta() (Meta, error)
	Exists() bool
	/*SetParent(doc Doc) (Doc, error)*/
}

type Meta interface {
	Save(ctx context.Context, doc Doc, groupMeta Meta) error
	Key() *datastore.Key
	ID() string
	Exists() bool
	UpdatedAt() time.Time
	CreatedAt() time.Time
	CreatedBy() *datastore.Key
	UpdatedBy() *datastore.Key
	Print(doc Doc, value interface{}) interface{}
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
	Doc(ctx context.Context, key *datastore.Key, ancestor Doc) (Doc, error)
}
