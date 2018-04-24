package kind

import (
	"google.golang.org/appengine/datastore"
	"time"
	"encoding/json"
	"github.com/imdario/mergo"
	"reflect"
)

type Holder struct {
	Kind   *Kind
	user   *datastore.Key
	hasKey bool

	key   *datastore.Key
	value interface{}

	hasInputData  bool // when updating
	hasLoadedData bool
}

type Meta struct {
	Id        *datastore.Key `json:"id"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	CreatedBy *datastore.Key `json:"createdBy"`
	Value     interface{}    `json:"value"`
}

func (h *Holder) Id() string {
	if h.hasKey {
		return h.key.Encode()
	}
	return ""
}

func (h *Holder) Value() interface{} {
	if h.hasKey {
		v := reflect.ValueOf(h.value).Elem()
		idField := v.FieldByName(h.Kind.MetaIdField.FieldName)
		if idField.IsValid() && idField.CanSet() {
			idField.Set(reflect.ValueOf(h.key))
		}
	}
	return h.value
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
	if k != nil {
		h.key = k
		h.hasKey = true
	}
}

func (h *Holder) GetKey() *datastore.Key {
	return h.key
}

func (h *Holder) Load(ps []datastore.Property) error {
	h.hasLoadedData = true

	if h.hasInputData {
		// replace only empty fields
		n := h.Kind.New()
		if err := datastore.LoadStruct(n, ps); err != nil {
			return err
		}
		return mergo.Merge(h.value, n)
	}

	return datastore.LoadStruct(h.value, ps)
}

func (h *Holder) Save() ([]datastore.Property, error) {
	var now = reflect.ValueOf(time.Now())

	v := reflect.ValueOf(h.value).Elem()
	for _, meta := range h.Kind.MetaFields {
		field := v.FieldByName(meta.FieldName)
		if field.CanSet() {
			switch meta.Type {
			case "updatedAt":
				field.Set(now)
			case "createdAt":
				if !h.hasLoadedData {
					field.Set(now)
				}
			case "createdBy":
				if !h.hasLoadedData {
					field.Set(reflect.ValueOf(h.user))
				}
			}
		}
	}
	return datastore.SaveStruct(h.value)
}
