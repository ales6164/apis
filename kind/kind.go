package kind

import (
	"github.com/ales6164/apis/errors"
	"golang.org/x/net/context"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/search"
	"reflect"
	"strings"
)

type Kind struct {
	Type        reflect.Type
	MetaFields  []MetaField
	MetaIdField MetaField
	*Options
	fields []*Field

	info    *KindInfo
	path    string   // serving path
	methods []string // serving methods

	searchFields map[string]*SearchField // map of all fields

	//todo: above searchFields - add additional fields and below mentioned functions for use in global search
	// map[fieldName] or array of fields with appropriate convertion functions (pointer to a function)
	// do we need to convert from search value back to original value when outputing??? - probably should? - this could mean trouble later on... maybe just fetch original datastore entries
	// for output uses field json string

	// add some meta tag to db entry to now when it was last synced with search?
}

type Options struct {
	Label                string
	Name                 string // name used to represent kind on the backend
	EnableSearch         bool
	RetrieveByIDOnSearch bool
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

	// search tag
	SearchField *SearchField

	MetaField   string
	Label       string // json field name
	Json        string // json field name
	Type        string
	StructField reflect.StructField

	Kind *Kind
}

type SearchField struct {
	Field           *Field
	Enabled         bool
	Multiple        bool
	FieldName       string
	SearchFieldName string
	IsFacet         bool
	ConvertType     reflect.Type
	Converter       Converter
}

func New(t reflect.Type, opt *Options) *Kind {
	if opt == nil {
		opt = new(Options)
	}

	k := &Kind{
		Type:         t,
		Options:      opt,
		searchFields: map[string]*SearchField{},
	}

	if t == nil {
		return k
	}

	if t.Kind() != reflect.Struct {
		panic(errors.New("type not of kind struct"))
	}

	k.checkFields()

	if len(k.Options.Name) == 0 {
		k.Name = t.Name()
	}

	if len(k.Options.Label) == 0 {
		k.Label = t.Name()
	}

	// build search parser
	if k.EnableSearch {
	searchField:
		for _, field := range k.fields {
			if !field.SearchField.Enabled {
				continue searchField
			}

			// 1. Set convert type from search tag or try to find a good converter
			if field.SearchField.ConvertType == nil {
				var fieldType reflect.Type
				if field.Multiple {
					// get slice type
					fieldType = reflect.MakeSlice(field.StructField.Type, 1, 1).Index(0).Type()
				} else {
					fieldType = field.StructField.Type
				}

				if fieldType == nil {
					panic(errors.New("error reflecting type for " + field.Name))
				}

				// convert type differs for search.Field and search.Facet
				if field.SearchField.IsFacet {
					switch fieldType {
					case stringType:
						field.SearchField.ConvertType = atomType
					case intType, int8Type, int16Type, int32Type, int64Type, float32Type:
						field.SearchField.ConvertType = float64Type
					case keyType:
						field.SearchField.ConvertType = keyType
					case boolType:
						field.SearchField.ConvertType = boolType
					default:
						if fieldType.ConvertibleTo(atomType) {
							field.SearchField.ConvertType = atomType
						} else if fieldType.ConvertibleTo(float64Type) {
							field.SearchField.ConvertType = float64Type
						} else {
							panic(errors.New("no convert type specified for searchable facet " + field.Name))
						}
					}
				} else {
					switch fieldType {
					case stringType, atomType, htmlType, float64Type, timeType, geoPointType:
						// ok
					case intType, int8Type, int16Type, int32Type, int64Type, float32Type:
						field.SearchField.ConvertType = float64Type
					case keyType:
						field.SearchField.ConvertType = keyType
					case boolType:
						field.SearchField.ConvertType = boolType
					default:
						panic(errors.New("no convert type specified for searchable field " + field.Name))
					}
				}
			}

			var converter Converter
			if field.SearchField.ConvertType == nil {
				converter = new(EmptyConverter)
			} else if field.SearchField.IsFacet {
				switch field.SearchField.ConvertType {
				case atomType:
					converter = new(AtomConverter)
				case float64Type:
					converter = new(Float64Converter)
				case keyType:
					converter = new(KeyConverter)
				case boolType:
					converter = new(BoolConverter)
				default:
					panic(errors.New("invalid convert type for searchable facet " + field.Name))
				}
			} else {
				switch field.SearchField.ConvertType {
				case atomType:
					converter = new(AtomConverter)
				case float64Type:
					converter = new(Float64Converter)
				case stringType:
					converter = new(StringConverter)
				case htmlType:
					converter = new(HTMLConverter)
				case keyType:
					converter = new(KeyConverter)
				case boolType:
					converter = new(BoolConverter)
				default:
					panic(errors.New("invalid convert type for searchable field " + field.Name))
				}
			}

			field.SearchField.Converter = converter

			k.searchFields[field.SearchField.SearchFieldName] = field.SearchField
		}
	}

	return k
}

func (k *Kind) SearchFields() map[string]*SearchField {
	return k.searchFields
}

func (k *Kind) checkFields() {
	k.fields = []*Field{}
	var hasId, hasCreatedAt bool
	for i := 0; i < k.Type.NumField(); i++ {
		structField := k.Type.Field(i)
		field := new(Field)
		field.SearchField = &SearchField{
			Field:           field,
			FieldName:       structField.Name,
			SearchFieldName: structField.Name,
		}
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
		} else {
			field.DoStore = true
			field.Name = structField.Name
		}
		if val, ok := structField.Tag.Lookup("json"); ok {
			for n, v := range strings.Split(val, ",") {
				v = strings.TrimSpace(v)
				switch n {
				case 0:
					if v != "-" {
						field.Json = v
					}
				}
			}
		} else {
			field.Json = field.Name
		}
		if val, ok := structField.Tag.Lookup("label"); ok {
			for n, v := range strings.Split(val, ",") {
				v = strings.TrimSpace(v)
				switch n {
				case 0:
					field.Label = v
				}
			}
		} else {
			field.Label = structField.Name
		}
		if val, ok := structField.Tag.Lookup("apis"); ok {
			for n, v := range strings.Split(val, ",") {
				v = strings.TrimSpace(v)
				v = strings.ToLower(v)
				switch n {
				case 0:
					if v == "id" {
						hasId = true
						k.MetaIdField = MetaField{
							Type:      v,
							FieldName: structField.Name,
						}
					} else {
						if v == "createdat" {
							hasCreatedAt = true
						}
						k.MetaFields = append(k.MetaFields, MetaField{
							Type:      v,
							FieldName: structField.Name,
						})
					}
					field.MetaField = v

				}
			}
		}
		if val, ok := structField.Tag.Lookup("search"); ok {
			for n, v := range strings.Split(val, ",") {
				v = strings.TrimSpace(v)
				switch n {
				case 0:
					if len(v) > 0 && v != "-" {
						field.SearchField.Enabled = true
						field.SearchField.SearchFieldName = v
					}
				case 1:
					field.SearchField.IsFacet = v == "facet"
				case 2:
					switch strings.TrimSpace(v) {
					case "search.Atom":
						field.SearchField.ConvertType = atomType
					case "search.HTML":
						field.SearchField.ConvertType = htmlType
					case "float64":
						field.SearchField.ConvertType = float64Type
					case "string":
						field.SearchField.ConvertType = stringType
					case "time.Time":
						field.SearchField.ConvertType = timeType
					}
				}
			}
		} else {
			field.SearchField.Enabled = field.DoStore
		}

		if structField.Type.Kind() == reflect.Slice {
			field.Multiple = true
			field.SearchField.Multiple = true
		}
		k.fields = append(k.fields, field)
	}
	if !(hasId && hasCreatedAt) {
		panic(errors.New("kind " + k.Name + " requires id and createdAt fields"))
	}
}

func (k *Kind) New() interface{} {
	return reflect.New(k.Type).Interface()
}

func (k *Kind) DeleteFromIndex(ctx context.Context, id string) error {
	index, err := search.Open(k.Name)
	if err != nil {
		return err
	}
	return index.Delete(ctx, id)
}

func (k *Kind) NewHolder(user *datastore.Key) *Holder {
	return &Holder{
		Kind:  k,
		user:  user,
		value: k.New(),
	}
}

func (k *Kind) AddRouteSettings(path string, methods []string) {
	k.path = path
	k.methods = methods
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
