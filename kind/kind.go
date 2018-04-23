package kind

import (
	"golang.org/x/net/context"
	"google.golang.org/appengine/datastore"
	"reflect"
	"strings"
	"github.com/ales6164/apis/errors"
)

type Kind struct {
	Type   reflect.Type
	*Options
	fields []*Field
}

type Options struct {
	Name                 string
	EnableSearch         bool
	RetrieveByIDOnSearch bool
	SearchType           reflect.Type
}

type Field struct {
	Name       string
	DoStore    bool
	IsRequired bool // moving this somewhere else?
	Multiple   bool
	NoIndex    bool

	SearchName    string
	SearchDoStore bool
	SearchType    string
	SearchField   bool
	SearchFacet   bool

	Kind *Kind
}

func New(t reflect.Type, opt *Options) *Kind {
	if opt == nil {
		opt = new(Options)
	}

	if t.Kind() != reflect.Struct {
		panic(errors.New("type not of kind struct"))
	}

	k := &Kind{
		Type:    t,
		Options: opt,
	}

	for i := 0; i < t.NumField(); i++ {
		structField := t.Field(i)
		field := new(Field)

		// defaults
		field.SearchField = true
		field.SearchDoStore = true

		if val, ok := structField.Tag.Lookup("datastore"); ok {
			for n, v := range strings.Split(val, ",") {
				v = strings.ToLower(strings.TrimSpace(v))
				switch n {
				case 0:
					if v == "-" {
						field.DoStore = false
					} else {
						field.DoStore = true
					}
					field.Name = v
				case 1:
					field.NoIndex = v == "noindex"
				}
			}
		}

		if val, ok := structField.Tag.Lookup("search"); ok {
			for n, v := range strings.Split(val, ",") {
				v = strings.ToLower(strings.TrimSpace(v))
				switch n {
				case 0:
					if v == "-" {
						field.SearchDoStore = false
					} else {
						field.SearchDoStore = true
					}
					field.SearchName = v
				case 1:
					field.SearchType = v // define this, if need type conversion: string|[]byte -> atom, string|[]byte -> html
				case 2:
					if v == "nofilter" {
						field.SearchFacet = false
					} else if v == "onlyfilter" {
						field.SearchField = false
						field.SearchField = true
					}
				}
			}
		}

		if len(field.Name) == 0 {
			field.Name = structField.Name
		}

		if len(field.SearchName) == 0 {
			field.SearchName = structField.Name
		}

		if structField.Type.Kind() == reflect.Slice {
			field.Multiple = true
		}
	}

	if len(k.Name) == 0 {
		k.Name = t.Name()
	}

	if k.SearchType == nil {
		k.SearchType = t
	}

	return k
}

func (k *Kind) New() interface{} {
	return reflect.New(k.Type).Interface()
}

func (k *Kind) NewHolder(user *datastore.Key) *Holder {
	return &Holder{
		Kind:  k,
		user:  user,
		value: k.New(),
	}
}

func (k *Kind) NewIncompleteKey(c context.Context, parent *datastore.Key) *datastore.Key {
	return datastore.NewIncompleteKey(c, k.Name, parent)
}

func (k *Kind) NewKey(c context.Context, nameId string, parent *datastore.Key) *datastore.Key {
	return datastore.NewKey(c, k.Name, nameId, 0, parent)
}
