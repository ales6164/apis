package apis

import (
	"encoding/json"
	"google.golang.org/appengine/datastore"
)

type Holder struct {
	Kind               *Kind
	key                *datastore.Key
	value              interface{}
	hasInputData       bool // when updating
	hasLoadedData      bool
	rollbackProperties []datastore.Property
}

func (h *Holder) Id() string {
	if h.key != nil {
		return h.key.Encode()
	}
	return ""
}

func (h *Holder) Value() interface{} {
	/*if h.key != nil {
		v := reflect.ValueOf(h.value).Elem()
		idField := v.FieldByName(h.Kind.MetaIdField.FieldName)
		if idField.IsValid() && idField.CanSet() {
			idField.Set(reflect.ValueOf(h.key))
		}
	}*/
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
