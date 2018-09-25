package apis

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
	if h.key != nil {
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
					if convertedValue != nil && !IsZeroOfUnderlyingType(convertedValue) {
						doc.facets = append(doc.facets, search.Facet{Name: searchField.SearchFieldName, Value: convertedValue})
					}
				} else {
					doc.fields = append(doc.fields, search.Field{Name: searchField.SearchFieldName, Value: convertedValue})
				}
			}
		} else {
			convertedValue := searchField.Converter.Convert(valField)
			if searchField.IsFacet {
				if convertedValue != nil && !IsZeroOfUnderlyingType(convertedValue) {
					doc.facets = append(doc.facets, search.Facet{Name: searchField.SearchFieldName, Value: convertedValue})
				}
			} else {
				doc.fields = append(doc.fields, search.Field{Name: searchField.SearchFieldName, Value: convertedValue})
			}
		}
	}

	return doc
}

func IsZeroOfUnderlyingType(x interface{}) bool {
	return x == reflect.Zero(reflect.TypeOf(x)).Interface()
}

// for non comparable types
func IsZeroOfDeepUnderlyingType(x interface{}) bool {
	return reflect.DeepEqual(x, reflect.Zero(reflect.TypeOf(x)).Interface())
}

func (h *Holder) SaveToIndex(ctx context.Context) error {
	/*if !h.Kind.EnableSearch {
		return nil
	}*/
	index, err := search.Open(h.Kind.Name)
	if err != nil {
		return err
	}

	if h.key == nil {
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

		if err := mergo.Merge(n, h.value, func(config *mergo.Config) {
			config.AppendSlice = false
		}, mergo.WithTransformers(Transformer{})); err != nil {
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
			}
		}
	}
	return datastore.SaveStruct(h.value)
}

/*func (h *Holder) Rollback() error {

}*/

//todo: override arrays
// todo: all transforms into one

// mergo transformer
type Transformer struct {
}

func (t Transformer) Transformer(typ reflect.Type) func(dst, src reflect.Value) error {
	switch typ {
	case timeType:
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
	case boolType:
		return func(dst, src reflect.Value) error {
			if dst.CanSet() {
				dst.Set(src)
			}
			return nil
		}
	case stringType:
		return func(dst, src reflect.Value) error {
			if dst.CanSet() {
				dst.Set(src)
			}
			return nil
		}
	default:
		// override slices
		if typ.Kind() == reflect.Slice || typ.Kind() == reflect.Array {
			return func(dst, src reflect.Value) error {
				if dst.CanSet() {
					dst.Set(src)
				}
				return nil
			}
		}
	}
	return nil
}
