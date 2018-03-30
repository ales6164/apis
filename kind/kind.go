package kind

import (
	"errors"
	"strings"

	"golang.org/x/net/context"
	"google.golang.org/appengine/datastore"
	"github.com/asaskevich/govalidator"
)

type Kind struct {
	*Options
	fields map[string]*Field
}

type Options struct {
	Name   string
	Fields []*Field

	OnBeforeCreate func(ctx context.Context, h *Holder) error `json:"-"`
	OnAfterCreate  func(ctx context.Context, h *Holder) error `json:"-"`

	OnBeforeUpdate func(ctx context.Context, h *Holder) error `json:"-"`
	OnAfterUpdate  func(ctx context.Context, h *Holder) error `json:"-"`
}

type Field struct {
	Name       string
	IsRequired bool
	Multiple   bool
	NoIndex    bool
	Kind       *Kind

	isKind   bool
	isNested bool
}

var (
	ErrInvalidKindName     = errors.New("kind name must contain a-zA-Z characters only")
	ErrEmptyFieldName      = errors.New("field name can't be empty")
	ErrFieldNameNotAllowed = errors.New("field name can't begin with an underscore, 'meta' or 'id'")
)

func New(opt *Options) *Kind {
	k := &Kind{
		Options: opt,
	}
	return k
}

func (k *Kind) Init() error {
	var err error
	if !govalidator.IsAlpha(k.Name) {
		return ErrInvalidKindName
	}
	if err != nil {
		return err
	}
	for _, f := range k.Fields {
		if len(f.Name) == 0 {
			return ErrEmptyFieldName
		}
		if f.Name == "meta" || f.Name == "id" || f.Name[:1] == "_" {
			return ErrFieldNameNotAllowed
		}
		if split := strings.Split(f.Name, "."); len(split) > 1 {
			if split[0] == "meta" || split[0] == "id" || split[0][:1] == "_" {
				return ErrFieldNameNotAllowed
			}
			f.isNested = true
		}
		if f.Kind != nil {
			f.isKind = true
		}
		if k.fields == nil {
			k.fields = map[string]*Field{}
		}
		k.fields[f.Name] = f
	}
	return nil
}

func (k *Kind) NewHolder(ctx context.Context, user *datastore.Key) *Holder {
	return &Holder{
		Kind:              k,
		context:           ctx,
		user:              user,
		preparedInputData: map[*Field][]datastore.Property{},
		loadedStoredData:  map[string][]datastore.Property{},
	}
}

func (k *Kind) NewEmptyHolder() *Holder {
	return &Holder{
		Kind:              k,
		preparedInputData: map[*Field][]datastore.Property{},
		loadedStoredData:  map[string][]datastore.Property{},
	}
}

func (k *Kind) NewIncompleteKey(c context.Context, parent *datastore.Key) *datastore.Key {
	return datastore.NewIncompleteKey(c, k.Name, parent)
}

func (k *Kind) NewKey(c context.Context, nameId string) *datastore.Key {
	return datastore.NewKey(c, k.Name, nameId, 0, nil)
}
