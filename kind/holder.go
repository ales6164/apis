package kind

import (
	"encoding/json"
	"github.com/ales6164/apis/errors"
	"github.com/imdario/mergo"
	"golang.org/x/net/context"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/search"
	"reflect"
	"time"
)

type Holder struct {
	Kind   *Kind
	user   *datastore.Key
	hasKey bool

	key   *datastore.Key
	value interface{}

	hasInputData  bool // when updating
	hasLoadedData bool

	rollbackProperties []datastore.Property
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
	h.hasKey = k != nil
}

func (h *Holder) GetKey() *datastore.Key {
	return h.key
}

func (h *Holder) Document(ctx context.Context) *Document {
	var doc = new(Document)
	val := reflect.ValueOf(h.Value()).Elem()

	for _, searchField := range h.Kind.searchFields {
		// get real value field
		valField := val.FieldByName(searchField.FieldName)
		if !valField.IsValid() {
			continue
		}

		if searchField.Multiple {
			for j := 0; j < valField.Len(); j++ {
				convertedValue := searchField.Converter.Convert(valField.Index(j))
				if searchField.IsFacet {
					doc.facets = append(doc.facets, search.Facet{Name: searchField.SearchFieldName, Value: convertedValue})
				} else {
					doc.fields = append(doc.fields, search.Field{Name: searchField.SearchFieldName, Value: convertedValue})
				}
			}
		} else {
			convertedValue := searchField.Converter.Convert(valField)
			if searchField.IsFacet {
				doc.facets = append(doc.facets, search.Facet{Name: searchField.SearchFieldName, Value: convertedValue})
			} else {
				doc.fields = append(doc.fields, search.Field{Name: searchField.SearchFieldName, Value: convertedValue})
			}
		}
	}

	return doc
}

func (h *Holder) SaveToIndex(ctx context.Context) error {
	if !h.Kind.EnableSearch {
		return nil
	}
	index, err := search.Open(h.Kind.Name)
	if err != nil {
		return err
	}

	if !h.hasKey {
		return errors.New("undefined key for index storage")
	}

	doc := h.Document(ctx)

	_, err = index.Put(ctx, h.Id(), doc)
	return err
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

		if err := mergo.Merge(n, h.value, mergo.WithOverride, mergo.WithTransformers(timeTransformer{}), mergo.WithTransformers(boolTransformer{}), mergo.WithTransformers(stringTransfomer{})); err != nil {
			return err
		}

		h.value = n

		return nil
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
			case "updatedat":
				field.Set(now)
			case "createdat":
				if !h.hasLoadedData {
					field.Set(now)
				}
			case "createdby":
				if !h.hasLoadedData {
					field.Set(reflect.ValueOf(h.user))
				}
			}
		}
	}
	return datastore.SaveStruct(h.value)
}

/*func (h *Holder) Rollback() error {

}*/

// mergo transformer
type timeTransformer struct {
}

func (t timeTransformer) Transformer(typ reflect.Type) func(dst, src reflect.Value) error {
	if typ == reflect.TypeOf(time.Time{}) {
		return func(dst, src reflect.Value) error {
			if dst.CanSet() {
				isZero := dst.MethodByName("IsZero")
				result := isZero.Call([]reflect.Value{})
				if result[0].Bool() {
					dst.Set(src)
				}
			}
			return nil
		}
	}
	return nil
}

type boolTransformer struct {
}

func (t boolTransformer) Transformer(typ reflect.Type) func(dst, src reflect.Value) error {
	if typ == reflect.TypeOf(true) {
		return func(dst, src reflect.Value) error {
			if dst.CanSet() {
				dst.Set(src)
			}
			return nil
		}
	}
	return nil
}

type stringTransfomer struct {
}

func (t stringTransfomer) Transformer(typ reflect.Type) func(dst, src reflect.Value) error {
	if typ == reflect.TypeOf("") {
		return func(dst, src reflect.Value) error {
			if dst.CanSet() {
				dst.Set(src)
			}
			return nil
		}
	}
	return nil
}
