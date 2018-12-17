package kind

import (
	"encoding/json"
	"errors"
	"google.golang.org/appengine/datastore"
	"reflect"
	"strconv"
)

type Holder struct {
	Kind               *Kind
	Key                *datastore.Key
	hasInputData       bool // when updating
	hasLoadedData      bool
	rollbackProperties []datastore.Property

	reflectValue reflect.Value
}

var (
	keyType = reflect.TypeOf(&datastore.Key{})
)

func (h *Holder) Id() string {
	if h.Key != nil {
		return h.Key.Encode()
	}
	return ""
}

func (h *Holder) ReflectValue() {
	if h.Kind != nil && h.Kind.hasIdFieldName && h.Key != nil {
		v := h.reflectValue.Elem()
		idField := v.FieldByName(h.Kind.idFieldName)
		if idField.IsValid() && idField.CanSet() {
			if h.Kind.dsUseName {
				idField.Set(reflect.ValueOf(h.Key.StringID()))
			} else {
				idField.Set(reflect.ValueOf(h.Key))
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

func (h *Holder) get(ctx Context, fields []string) (*Holder, reflect.Value, error) {
	var valueHolder = h.reflectValue
	for _, field := range fields {
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
		/*switch valueHolder.Type() {
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
					return h2.get(ctx, fields[i+1:])
				} else {
					return h, valueHolder, errors.New("field of type *datastore.Key but it's kind is not registered in kind provider")
				}
			} else {
				return h, valueHolder, errors.New("error reflecting field of type *datastore.Key")
			}
		default:

		}*/
	}

	return h, valueHolder, nil
}

func (h *Holder) Set(ctx Context, fields []string, value []byte) (*Holder, error) {
	h2, v, err := h.get(ctx, fields)
	if err != nil {
		return h2, err
	}

	if v.CanSet() {
		inputValue := reflect.New(v.Type()).Interface()
		err = json.Unmarshal(value, &inputValue)
		if err != nil {
			return h2, err
		}
		v.Set(reflect.ValueOf(inputValue).Elem())
	} else {
		return h2, errors.New("field value can't be set")
	}

	return h2, nil
}

func (h *Holder) Delete(ctx Context, fields []string) (*Holder, error) {
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
				if i == len(fields)-1 {
					if valueHolder.CanSet() {
						// todo: check for index out of bounds
						valueHolder.Set(reflect.AppendSlice(valueHolder.Slice(0, index), valueHolder.Slice(index+1, valueHolder.Len())))
					} else {
						return h, errors.New("field value can't be set")
					}
					return h, nil
				} else {
					valueHolder = valueHolder.Index(index)
				}
			} else {
				return h, errors.New("error converting string to slice index")
			}
		default:
			if valueHolder.Kind() == reflect.Ptr {
				valueHolder = valueHolder.Elem().FieldByName(field)
			} else {
				valueHolder = valueHolder.FieldByName(field)
			}
		}
		// do stuff after we have value
	}

	if valueHolder.CanSet() {
		valueHolder.Set(reflect.Zero(valueHolder.Type()))
	} else {
		return h, errors.New("field value can't be set")
	}

	return h, nil
}

func (h *Holder) Get(ctx Context, fields []string) (*Holder, interface{}, error) {
	h2, v, err := h.get(ctx, fields)
	return h2, v.Interface(), err
}

func (h *Holder) Bytes() ([]byte, error) {
	return json.Marshal(h.GetValue())
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
