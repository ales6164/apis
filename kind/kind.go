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
	permissions
}

type Options struct {
	Name string
	Permissions
	Fields []*Field
}

type Field struct {
	Name       string
	IsRequired bool
	Multiple   bool
	NoIndex    bool

	isNested bool
	Worker
}

var (
	ErrInvalidKindName = errors.New("kind name must contain a-zA-Z characters only")
	ErrEmptyFieldName = errors.New("field name can't be empty")
	ErrFieldNameNotAllowed = errors.New("field name can't begin with an underscore, 'meta' or 'id'")
)

func New(opt *Options) (*Kind, error) {
	var err error
	k := &Kind{
		Options: opt,
	}
	if !govalidator.IsAlpha(k.Name) {
		return k, ErrInvalidKindName
	}
	k.permissions, err = k.parse()
	if err != nil {
		return k, err
	}
	for _, f := range k.Fields {
		if len(f.Name) == 0 {
			return k, ErrEmptyFieldName
		}
		if f.Name == "meta" || f.Name == "id" || f.Name[:1] == "_" {
			return k, ErrFieldNameNotAllowed
		}
		if split := strings.Split(f.Name, "."); len(split) > 1 {
			if split[0] == "meta" || split[0] == "id" || split[0][:1] == "_" {
				return k, ErrFieldNameNotAllowed
			}
			f.isNested = true
		}
		if k.fields == nil {
			k.fields = map[string]*Field{}
		}
		k.fields[f.Name] = f
	}
	return k, nil
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

func (k *Kind) NewIncompleteKey(c context.Context, parent *datastore.Key) *datastore.Key {
	return datastore.NewIncompleteKey(c, k.Name, parent)
}

func (k *Kind) NewKey(c context.Context, nameId string) *datastore.Key {
	return datastore.NewKey(c, k.Name, nameId, 0, nil)
}
