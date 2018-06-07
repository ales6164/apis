package kind

import (
	"golang.org/x/net/context"
	"google.golang.org/appengine/datastore"
	"reflect"
	"strings"
	"github.com/ales6164/apis/errors"
	"google.golang.org/appengine/search"
)

type Kind struct {
	Type        reflect.Type
	MetaFields  []MetaField
	MetaIdField MetaField
	*Options
	fields      []*Field

	info *KindInfo
	ui   *UI

	searchFields map[string]SearchField // map of all fields
}

type Options struct {
	Name                 string // name used to represent kind on the backend
	EnableSearch         bool
	RetrieveByIDOnSearch bool
	SearchType           reflect.Type
}

type MetaField struct {
	Type      string
	FieldName string
}

type Field struct {
	Name       string
	DoStore    bool
	IsRequired bool // moving this somewhere else?
	Multiple   bool
	NoIndex    bool

	MetaField   string
	Label       string // json field name
	Json        string // json field name
	Type        string
	StructField reflect.StructField

	Kind *Kind
}

type SearchField struct {
	Name    string
	IsFacet bool
}

func New(t reflect.Type, opt *Options) *Kind {
	if opt == nil {
		opt = new(Options)
	}

	if t.Kind() != reflect.Struct {
		panic(errors.New("type not of kind struct"))
	}

	k := &Kind{
		Type:         t,
		Options:      opt,
		searchFields: map[string]SearchField{},
	}

	k.checkFields()

	if len(k.Name) == 0 {
		k.Name = t.Name()
	}

	if k.SearchType == nil {
		k.SearchType = t
	}

	for i := 0; i < k.SearchType.NumField(); i++ {
		searchField := k.SearchType.Field(i)

		var field = SearchField{
			Name:    searchField.Name,
			IsFacet: false,
		}

		if val, ok := searchField.Tag.Lookup("search"); ok {

			for n, v := range strings.Split(val, ",") {
				v = strings.TrimSpace(v)
				switch n {
				case 0:
					if v == "-" {
						field.Name = ""
					} else {
						field.Name = v
					}
				case 1:
					field.IsFacet = v == "facet"
				}
			}
		}

		k.searchFields[field.Name] = field

	}

	return k
}

func (k *Kind) checkFields() {
	k.fields = []*Field{}
	for i := 0; i < k.Type.NumField(); i++ {
		structField := k.Type.Field(i)
		field := new(Field)
		field.StructField = structField
		field.Type = structField.Type.String()
		if val, ok := structField.Tag.Lookup("datastore"); ok {
			for n, v := range strings.Split(val, ",") {
				v = strings.TrimSpace(v)
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
		if val, ok := structField.Tag.Lookup("json"); ok {
			for n, v := range strings.Split(val, ",") {
				v = strings.TrimSpace(v)
				switch n {
				case 0:
					field.Json = v
				}
			}
		}
		if val, ok := structField.Tag.Lookup("label"); ok {
			for n, v := range strings.Split(val, ",") {
				v = strings.TrimSpace(v)
				switch n {
				case 0:
					field.Label = v
				}
			}
		}
		if val, ok := structField.Tag.Lookup("apis"); ok {
			for n, v := range strings.Split(val, ",") {
				v = strings.TrimSpace(v)
				v = strings.ToLower(v)
				switch n {
				case 0:
					if v == "id" {
						k.MetaIdField = MetaField{
							Type:      v,
							FieldName: structField.Name,
						}
					} else {
						k.MetaFields = append(k.MetaFields, MetaField{
							Type:      v,
							FieldName: structField.Name,
						})
					}
					field.MetaField = v

				}
			}
		}

		if len(field.Name) == 0 {
			field.Name = structField.Name
		}

		if len(field.Json) == 0 {
			field.Json = field.Name
		}

		if structField.Type.Kind() == reflect.Slice {
			field.Multiple = true
		}
		k.fields = append(k.fields, field)
	}
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

func (k *Kind) DecodeKey(encoded string) (*datastore.Key, error) {
	key, err := datastore.DecodeKey(encoded)
	if err != nil {
		return nil, err
	}
	if key.Kind() != k.Name {
		return nil, errors.New("key unathorized access")
	}
	return key, nil
}

func (k *Kind) RetrieveSearchParameter(parameterName string, value string, fields []search.Field, facets []search.Facet) ([]search.Field, []search.Facet) {
	if f, ok := k.searchFields[parameterName]; ok {
		if f.IsFacet {
			// todo: currently only supports facet type search.Atom
			facets = append(facets, search.Facet{Name: parameterName, Value: search.Atom(value)})
		} else {
			fields = append(fields, search.Field{Name: parameterName, Value: value})
		}
	}
	return fields, facets
}
