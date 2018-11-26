package apis

import (
	"encoding/json"
	"github.com/ales6164/apis/errors"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"
	"reflect"
)

type Holder struct {
	Kind               *Kind
	key                *datastore.Key
	value              interface{} // is pointer to struct -- normally
	hasInputData       bool        // when updating
	hasLoadedData      bool
	rollbackProperties []datastore.Property

	parent       *Holder // holder value reference
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
		v := reflect.ValueOf(h.value).Elem()
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
	return h.value
}

func (h *Holder) SetValue(v interface{}) {
	h.value = v
	h.reflectValue = reflect.ValueOf(h.value)
}

func (h *Holder) Parse(body []byte) error {
	h.hasInputData = true
	h.value = h.Kind.New()
	err := json.Unmarshal(body, &h.value)
	h.reflectValue = reflect.ValueOf(h.value)
	return err
}

// TODO: Check scope before GET
func (h *Holder) Get(ctx Context, field string) (*Holder, error) {
	var holder = new(Holder)
	if h.key != nil || h.parent == nil {
		holder.parent = h
	} else {
		holder.parent = h.parent
	}

	var f *Field
	// get real field name (in case json field has different name)
	if h.Kind != nil {
		var ok bool
		if f, ok = h.Kind.fields[field]; ok {
			field = f.Name
		}
	}

	// get field value
	h.ReflectValue()

	if h.reflectValue.Kind() == reflect.Ptr {
		holder.reflectValue = h.reflectValue.Elem().FieldByName(field)
	} else {
		holder.reflectValue = h.reflectValue.FieldByName(field)

		log.Debugf(ctx, "this is non pointer field: %s %s", field, holder.reflectValue.String())
		if !holder.reflectValue.CanAddr() {
			log.Debugf(ctx, "this value is unaddressable")
		}
	}

	log.Debugf(ctx, "last")

	// check value type
	var holderType = holder.reflectValue.Type()
	switch holderType {
	case keyType:
		// check if it's auto=id field
		if f.IsAutoId {
			holder.value = holder.reflectValue.Interface()
		} else {
			// fetch from datastore
			key := holder.reflectValue.Interface().(*datastore.Key)
			kind := h.Kind.GetNameKind(key.Kind())
			if kind != nil {
				holder = kind.NewHolder(key)
				if err := datastore.Get(ctx, key, holder); err != nil {
					return holder, err
				}
				holderType = holder.reflectValue.Type()
			} else {
				return holder, errors.New("unregistered kind " + key.Kind())
			}
		}
	default:
		holder.value = holder.reflectValue.Interface()
	}

	kind := h.Kind.GetTypeKind(holderType)
	if kind != nil {
		holder.Kind = kind
	}

	return holder, nil
}

func (h *Holder) Delete(ctx Context) error {
	if h.key != nil {
		return datastore.Delete(ctx, h.key)
	} else if h.parent != nil && h.parent.key != nil {

		// delete field
		/*switch h.reflectValue.Kind() {
		case reflect.Bool:
			h.reflectValue.SetBool(false)
		case reflect.String:
			reflect.Indirect(h.reflectValue).SetString("")
			//h.reflectValue.SetString("")
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			h.reflectValue.SetInt(0)
		case reflect.Float32, reflect.Float64:
			h.reflectValue.SetFloat(0)
		case reflect.Struct:
			h.reflectValue.Set(reflect.Zero(h.reflectValue.Type()))

		default:
			return errors.New("unregistered kind " + h.reflectValue.Kind().String())
		//todo: need more: https://cloud.google.com/appengine/docs/standard/go/datastore/reference
		}*/

		log.Debugf(ctx, "kind %s", h.reflectValue.Kind().String())
		log.Debugf(ctx, "type %s", h.reflectValue.Type().String())
		log.Debugf(ctx, "value interface %s", h.reflectValue.Interface())
		log.Debugf(ctx, "this is value: %s", h.reflectValue.String())
		if !h.reflectValue.CanAddr() {
			log.Debugf(ctx, "this value is unaddressable")
		}

		h.reflectValue.Set(reflect.Zero(h.reflectValue.Type()))

		_, err := datastore.Put(ctx, h.parent.key, h.parent)
		return err
	}
	return errors.New("can't resolve path")
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
		n := h.Kind.New()
		if err := datastore.LoadStruct(n, ps); err != nil {
			return err
		}

		h.value = n

		return nil
	}
	err := datastore.LoadStruct(h.value, ps)
	h.reflectValue = reflect.ValueOf(h.value)
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
	return datastore.SaveStruct(h.value)
}
