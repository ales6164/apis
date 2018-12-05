package apis

import (
	"encoding/json"
	"github.com/ales6164/apis/errors"
	"google.golang.org/appengine/datastore"
	"reflect"
	"strconv"
)

type Holder struct {
	Kind               *Kind
	key                *datastore.Key
	hasInputData       bool // when updating
	hasLoadedData      bool
	rollbackProperties []datastore.Property

	reflectValue reflect.Value
}

var (
	keyType = reflect.TypeOf(&datastore.Key{})
)

func (h *Holder) Id() string {
	if h.key != nil {
		return h.key.Encode()
	}
	return ""
}

func (h *Holder) ReflectValue() {
	if h.Kind != nil && h.Kind.hasIdFieldName && h.key != nil {
		v := h.reflectValue.Elem()
		idField := v.FieldByName(h.Kind.idFieldName)
		if idField.IsValid() && idField.CanSet() {
			if h.Kind.dsUseName {
				idField.Set(reflect.ValueOf(h.key.StringID()))
			} else {
				idField.Set(reflect.ValueOf(h.key))
			}
		}
	}
}

func (h *Holder) GetValue() interface{} {
	h.ReflectValue()
	return h.reflectValue.Interface()
}

func (h *Holder) SetValue(v interface{}) {
	h.reflectValue = reflect.ValueOf(v)
}

func (h *Holder) Parse(body []byte) error {
	h.hasInputData = true
	var value = h.Kind.New().Interface()
	err := json.Unmarshal(body, &value)
	h.reflectValue = reflect.ValueOf(value)
	return err
}

func (h *Holder) get(ctx Context, fields ...string) (*Holder, reflect.Value, error) {
	var valueHolder = h.reflectValue
	for i, field := range fields {
		var f *Field
		// get real field name (in case json field has different name)
		if h.Kind != nil {
			var ok bool
			if f, ok = h.Kind.fields[field]; ok {
				field = f.Name
			}
		}

		// do stuff before switching to new value

		// is it array?
		switch valueHolder.Kind() {
		case reflect.Slice, reflect.Array:
			if index, err := strconv.Atoi(field); err == nil {
				valueHolder = valueHolder.Index(index)
			} else {
				return h, valueHolder, errors.New("error converting string to slice index")
			}
		default:
			if valueHolder.Kind() == reflect.Ptr {
				valueHolder = valueHolder.Elem().FieldByName(field)
			} else {
				valueHolder = valueHolder.FieldByName(field)
			}
		}

		// do stuff after we have value

		// check valueHolder kind to know if it's of kind *datastore.Key, in that case we get that object from datastore
		switch valueHolder.Type() {
		case keyType:
			if key, ok := valueHolder.Interface().(*datastore.Key); ok {
				if keyKind := h.Kind.GetNameKind(key.Kind()); keyKind != nil {
					if ok := ctx.HasScope(keyKind.ScopeReadOnly, keyKind.ScopeReadWrite, keyKind.ScopeFullControl); !ok {
						return h, valueHolder, errors.New("unauthorized access: value out of scope")
					}
					h2, err := keyKind.Get(ctx, key)
					if err != nil {
						return h, valueHolder, err
					}
					if i == len(field)-1 {
						h2.ReflectValue()
						return h2, h2.reflectValue, nil
					}
					return h2.get(ctx, fields[i+1:]...)
				} else {
					return h, valueHolder, errors.New("field of type *datastore.Key but it's kind is not registered in kind provider")
				}
			} else {
				return h, valueHolder, errors.New("error reflecting field of type *datastore.Key")
			}
		default:

		}
	}

	return h, valueHolder, nil
}

func (h *Holder) Get(ctx Context, fields ...string) (*Holder, interface{}, error) {
	h2, v, err := h.get(ctx, fields...)
	return h2, v.Interface(), err
}

func (h *Holder) Delete(ctx Context) error {
	return datastore.Delete(ctx, h.key)
}

func (h *Holder) Bytes() ([]byte, error) {
	return json.Marshal(h.GetValue())
}

func (h *Holder) SetKey(k *datastore.Key) {
	h.key = k
}

func (h *Holder) GetKey() *datastore.Key {
	return h.key
}

func (h *Holder) Load(ps []datastore.Property) error {
	h.hasLoadedData = true
	h.rollbackProperties = ps
	if h.hasInputData {
		// replace only empty fields
		n := h.Kind.New().Interface()
		if err := datastore.LoadStruct(n, ps); err != nil {
			return err
		}

		h.reflectValue = reflect.ValueOf(n)

		return nil
	}
	err := datastore.LoadStruct(h.reflectValue.Interface(), ps)
	return err
}

func (h *Holder) Save() ([]datastore.Property, error) {
	//var now = reflect.ValueOf(time.Now())
	//v := reflect.ValueOf(h.value).Elem()
	/*for _, meta := range h.Kind.MetaFields {
		field := v.FieldByName(meta.FieldName)
		if field.CanSet() {
			switch meta.Type {
			case "updatedat":
				field.Set(now)
			case "createdat":
				if !h.hasLoadedData {
					field.Set(now)
				}
			}
		}
	}*/
	return datastore.SaveStruct(h.reflectValue.Interface())
}
