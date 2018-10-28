package apis

import (
	"encoding/json"
	"google.golang.org/appengine/datastore"
	"reflect"
	"google.golang.org/appengine/log"
	"github.com/ales6164/apis/errors"
)

type Holder struct {
	Kind               *Kind
	key                *datastore.Key
	value              interface{}
	hasInputData       bool // when updating
	hasLoadedData      bool
	rollbackProperties []datastore.Property
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

func (h *Holder) Value() interface{} {
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
	return h.value
}

func (h *Holder) SetValue(v interface{}) {
	h.value = v
}

func (h *Holder) Parse(body []byte) error {
	h.hasInputData = true
	h.value = h.Kind.New()
	return json.Unmarshal(body, &h.value)
}

func (h *Holder) Get(ctx Context, field string) (*Holder, error) {
	var holder = new(Holder)

	// get real field name (in case json field has different name)
	if h.Kind != nil {
		if f, ok := h.Kind.fields[field]; ok {
			field = f.Name
		}
	}

	log.Debugf(ctx, "%s", field)

	// get field value
	value := reflect.ValueOf(h.value)
	if value.Kind() == reflect.Ptr {
		value = value.Elem().FieldByName(field)
	} else {
		value = value.FieldByName(field)
	}

	// check value type
	var holderType = value.Type()
	switch value.Type() {
	case keyType:
		// fetch from datastore
		key := value.Interface().(*datastore.Key)
		if kind, ok := ctx.a.kinds[key.Kind()]; ok {
			holder = kind.NewHolder(key)
			if err := datastore.Get(ctx, value.Interface().(*datastore.Key), holder); err != nil {
				return holder, err
			}
			holderType = reflect.TypeOf(holder.value)
		} else {
			return holder, errors.New("unregistered kind " + key.Kind())
		}
	default:
		holder.value = value.Interface()
	}

	if kind, ok := ctx.a.types[holderType]; ok {
		holder.Kind = kind
	}

	return holder, nil
}

func (h *Holder) Bytes() ([]byte, error) {
	return json.Marshal(h.Value())
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
	return datastore.LoadStruct(h.value, ps)
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
