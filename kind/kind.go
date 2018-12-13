package kind

import (
	"errors"
	"github.com/asaskevich/govalidator"
	"golang.org/x/net/context"
	"google.golang.org/appengine/datastore"
	"net/http"
	"reflect"
	"strings"
)

type Kind struct {
	t                     reflect.Type
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

	ScopeFullControl string
	ScopeReadOnly    string
	ScopeReadWrite   string
	ScopeDelete      string

	dsUseName       bool // default: false; if true, changes the way datastore Keys are generated
	dsNameGenerator func(ctx context.Context, holder *Holder) string

	fields map[string]*Field // map key is json representation for field name

	http.Handler
	*Options
}

type Options struct {
	// control entry access
	// even if field is false gotta store who created entry? so that is switched to true, creators still have access - if no owner is stored nobody has access
	EnableEntryScope bool
	Name             string
	KindProvider     *Provider
	IsCollection     bool
	Type             interface{}
	t                reflect.Type
}

type Field struct {
	Name     string                                                 // real field name
	Fields   map[string]*Field                                      // map key is json representation for field name
	retrieve func(value reflect.Value, path []string) reflect.Value // if *datastore.Key, fetches and returns resource; if array, returns item at index; otherwise returns the value
	Is       string

	IsAutoId bool
}

func New(opt *Options) *Kind {
	if opt == nil {
		panic(errors.New("kind options can't be nil"))
	}

	k := &Kind{
		Options: opt,
		t:       reflect.TypeOf(opt.Type),
		dsNameGenerator: func(ctx context.Context, holder *Holder) string {
			return ""
		},
	}

	if len(opt.Name) == 0 || !govalidator.IsAlphanumeric(opt.Name) {
		panic(errors.New("name must be at least one character and can contain only a-Z0-9"))
	}

	k.ScopeFullControl = opt.Name + ".fullcontrol"
	k.ScopeReadOnly = opt.Name + ".readonly"
	k.ScopeReadWrite = opt.Name + ".readwrite"
	k.ScopeDelete = opt.Name + ".delete"

	if k.t == nil || k.t.Kind() != reflect.Struct {
		panic(errors.New("type not of kind struct"))
	}

	k.fields = Lookup(k, k.t, map[string]*Field{})

	return k
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

func (k *Kind) Get(ctx context.Context, key *datastore.Key) (h *Holder, err error) {
	h = k.NewHolder(key)
	err = datastore.Get(ctx, key, h)
	return h, err
}

func Lookup(kind *Kind, typ reflect.Type, fields map[string]*Field) map[string]*Field {

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
			childFields = Lookup(nil, structField.Type, map[string]*Field{})
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

func (k *Kind) Type() reflect.Type {
	return k.t
}

func (k *Kind) New() reflect.Value {
	return reflect.New(k.t)
}

func (k *Kind) NewHolder(key *datastore.Key) *Holder {
	return &Holder{
		Kind:         k,
		reflectValue: k.New(),
		key:          key,
	}
}
