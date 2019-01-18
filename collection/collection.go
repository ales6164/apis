package collection

import (
	"errors"
	"github.com/ales6164/apis/kind"
	"github.com/asaskevich/govalidator"
	"golang.org/x/net/context"
	"google.golang.org/appengine/datastore"
	"reflect"
	"strconv"
	"strings"
)

type Collection struct {
	name    string
	i       interface{}
	t       reflect.Type
	isGroup bool
	member  *datastore.Key

	hasIdFieldName        bool
	hasCreatedAtFieldName bool
	hasUpdatedAtFieldName bool
	hasVersionFieldName   bool
	hasCreatedByFieldName bool
	hasUpdatedByFieldName bool

	idFieldName        string
	createdAtFieldName string
	updatedAtFieldName string
	versionFieldName   string
	createdByFieldName string
	updatedByFieldName string

	fields map[string]*Field // map key is json representation for field name
	kind.Kind
}

type Field struct {
	Name     string                                                 // real field name
	Fields   map[string]*Field                                      // map key is json representation for field name
	retrieve func(value reflect.Value, path []string) reflect.Value // if *datastore.Key, fetches and returns resource; if array, returns item at index; otherwise returns the value
	Is       string
	IsAutoId bool
}

func New(name string, i interface{}) *Collection {
	t := reflect.TypeOf(i)
	c := &Collection{
		name: name,
		t:    t,
	}

	if len(name) == 0 || !govalidator.IsAlphanumeric(name) {
		panic(errors.New("name must be at least one character and can contain only alphanumeric characters"))
	}

	if c.t == nil || c.t.Kind() != reflect.Struct {
		panic(errors.New("type not of kind struct"))
	}

	c.fields = lookup(c, c.t, map[string]*Field{})

	return c
}

var (
	keyKind = reflect.TypeOf(&datastore.Key{}).Kind()
)

const (
	id        = "id"
	createdat = "createdat"
	updatedat = "updatedat"
	version   = "version"
	createdby = "createdby"
	updatedby = "updatedby"
)

func (c *Collection) Name() string {
	return c.name
}

func (c *Collection) Scopes(scopes ...string) []string {
	var r []string
	for _, s := range scopes {
		r = append(r, c.name+"."+s)
	}
	return r
}

// Gets value at path
func (c *Collection) ValueAt(value reflect.Value, path []string) (reflect.Value, error) {
	var valueHolder = value
	for _, pathPart := range path {
		var f *Field
		// get real field name (in case json field has different name)
		var ok bool
		if f, ok = c.fields[pathPart]; ok {
			pathPart = f.Name
		}
		switch valueHolder.Kind() {
		case reflect.Slice, reflect.Array:
			if index, err := strconv.Atoi(pathPart); err == nil {
				valueHolder = valueHolder.Index(index)
			} else {
				return valueHolder, errors.New("error converting string to slice index")
			}
		default:
			if valueHolder.Kind() == reflect.Ptr {
				valueHolder = valueHolder.Elem().FieldByName(pathPart)
			} else {
				valueHolder = valueHolder.FieldByName(pathPart)
			}
		}
	}
	return valueHolder, nil
}

func lookup(kind *Collection, typ reflect.Type, fields map[string]*Field) map[string]*Field {
loop:
	for i := 0; i < typ.NumField(); i++ {
		var isAutoId bool
		structField := typ.Field(i)
		var jsonName = structField.Name

		if kind != nil {
			if autoValue, ok := structField.Tag.Lookup("auto"); ok {
				autoValue = strings.ToLower(autoValue)
				switch autoValue {
				case id:
					kind.idFieldName = structField.Name
					kind.hasIdFieldName = true
					isAutoId = true
				case createdat:
					kind.createdAtFieldName = structField.Name
					kind.hasCreatedAtFieldName = true
				case updatedat:
					kind.updatedAtFieldName = structField.Name
					kind.hasUpdatedAtFieldName = true
				case version:
					kind.versionFieldName = structField.Name
					kind.hasVersionFieldName = true
				case createdby:
					kind.createdByFieldName = structField.Name
					kind.hasCreatedByFieldName = true
				case updatedby:
					kind.updatedByFieldName = structField.Name
					kind.hasUpdatedByFieldName = true
				}
			}
		}

		if val, ok := structField.Tag.Lookup("json"); ok {
			for n, v := range strings.Split(val, ",") {
				v = strings.TrimSpace(v)
				switch n {
				case 0:
					if v == "-" {
						continue loop
					}
					jsonName = v
				}
			}
		}

		var fun func(value reflect.Value, path []string) reflect.Value
		var is string

		switch structField.Type.Kind() {
		case keyKind:
			// receives *datastore.Key as value
			// type struct that is fetched then with this value should be "registered" kind and somehow mapped to the api
			// so that the value query can continue onwards
			fun = func(value reflect.Value, path []string) reflect.Value {
				return value
			}
			is = "*datastore.Key{}"
		case reflect.Slice:
			// receives slice as value
			fun = func(value reflect.Value, path []string) reflect.Value {
				return value
			}
			is = "slice"
		default:
			fun = func(value reflect.Value, path []string) reflect.Value {
				for _, jsonName := range path {
					if field, ok := fields[jsonName]; ok {
						if value.Kind() == reflect.Ptr {
							value = value.Elem().FieldByName(field.Name)
						} else {
							value = value.FieldByName(field.Name)
						}
						return field.retrieve(value, path[1:])
					} else {

					}
					// error
				}
				return value
			}
			is = "default"
		}

		var childFields map[string]*Field

		if structField.Type.Kind() == reflect.Struct {
			childFields = lookup(nil, structField.Type, map[string]*Field{})
		}

		fields[jsonName] = &Field{
			Fields:   childFields,
			Name:     structField.Name,
			retrieve: fun,
			Is:       is,
			IsAutoId: isAutoId,
		}
	}

	return fields
}

func (c *Collection) Type() reflect.Type {
	return c.t
}

func (c *Collection) SetMember(member *datastore.Key) {
	c.member = member
}

func (c *Collection) Data(doc kind.Doc) interface{} {
	reflectValue := doc.Value()
	key := doc.Key()
	if c.hasIdFieldName && key != nil {
		v := reflectValue.Elem()
		idField := v.FieldByName(c.idFieldName)
		if idField.IsValid() && idField.CanSet() {
			if key.IntID() > 0 {
				idField.Set(reflect.ValueOf(key.Encode()))
			} else {
				idField.Set(reflect.ValueOf(key.StringID()))
			}
		}
	}
	return reflectValue.Interface()
}

func (c *Collection) Doc(ctx context.Context, key *datastore.Key, ancestor kind.Doc) kind.Doc {
	if key != nil && key.Kind() != c.name {
		key = nil
	}
	return &Document{
		kind:       c,
		defaultCtx: ctx,
		ctx:        ctx,
		value:      reflect.New(c.t),
		key:        key,
		ancestor:   ancestor,
	}
}
