package collection

import (
	"encoding/json"
	"errors"
	"github.com/buger/jsonparser"
	"google.golang.org/appengine/datastore"
	"reflect"
	"strconv"
	"strings"
	"time"
)

type Document struct {
	key    *datastore.Key

	hasInputData       bool // when updating
	hasLoadedData      bool
	rollbackProperties []datastore.Property

	reflectValue reflect.Value
}

var (
	keyType = reflect.TypeOf(&datastore.Key{})
)

func (h *Document) Id() string {
	if d.Key != nil {
		return d.Key.Encode()
	}
	return ""
}

func (h *Document) ReflectValue() {
	if d.Kind != nil && d.Kind.hasIdFieldName && d.Key != nil {
		v := d.reflectValue.Elem()
		idField := v.FieldByName(d.Kind.idFieldName)
		if idField.IsValid() && idField.CanSet() {
			if d.Kind.dsUseName {
				idField.Set(reflect.ValueOf(d.Key.StringID()))
			} else {
				idField.Set(reflect.ValueOf(d.Key))
			}
		}
	}
}

func (h *Document) Data() interface{} {
	d.ReflectValue()
	return d.reflectValue.Interface()
}

type Rich struct {
	ID        string      `json:"id"`
	CreatedAt time.Time   `json:"createdAt"`
	UpdatedAt time.Time   `json:"updatedAt"`
	Value     interface{} `json:"value"`
	//Version   int64     `json:"version"`
}

// encapsulates value inside Rich struct
func (d *Document) RichData() interface{} {
	d.ReflectValue()
	return d.reflectValue.Interface()
}

func (d *Document) SetValue(v interface{}) {
	d.reflectValue = reflect.ValueOf(v)
}

func (d *Document) Parse(body []byte) error {
	d.hasInputData = true
	var value = d.Kind.New().Interface()
	err := json.Unmarshal(body, &value)
	d.reflectValue = reflect.ValueOf(value)
	return err
}

func (d *Document) get(ctx Context, fields []string) (*Document, reflect.Value, error) {
	var valueHolder = d.reflectValue
	for _, field := range fields {
		var f *Field
		// get real field name (in case json field has different name)
		if d.Kind != nil {
			var ok bool
			if f, ok = d.Kind.fields[field]; ok {
				field = f.Name
			}
		}
		switch valueHolder.Kind() {
		case reflect.Slice, reflect.Array:
			if index, err := strconv.Atoi(field); err == nil {
				valueHolder = valueHolder.Index(index)
			} else {
				return d, valueHolder, errors.New("error converting string to slice index")
			}
		default:
			if valueHolder.Kind() == reflect.Ptr {
				valueHolder = valueHolder.Elem().FieldByName(field)
			} else {
				valueHolder = valueHolder.FieldByName(field)
			}
		}
	}
	return d, valueHolder, nil
}

func (d *Document) Set(ctx Context, fields []string, value []byte) (*Document, error) {
	h2, v, err := d.get(ctx, fields)
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

const (
	op_test    = "test"
	op_remove  = "remove"
	op_add     = "add"
	op_replace = "replace"
	op_move    = "move"
	op_copy    = "copy"
)

func (d *Document) Patch(ctx Context, dataArray []byte) error {
	var endErr error
	var cb = func(err error) {
		endErr = err
	}
	_, err := jsonparser.ArrayEach(dataArray, func(patch []byte, dataType jsonparser.ValueType, offset int, err error) {
		operation, err := jsonparser.GetString(patcd, "op")
		if err != nil {
			cb(errors.New("invalid operation"))
			return
		}
		value, _, _, _ := jsonparser.Get(patcd, "value")
		patd, err := jsonparser.GetString(patcd, "path")
		if err != nil {
			cb(errors.New("invalid path"))
			return
		}
		pathArray := strings.Split(patd, "/")
		if len(pathArray) > 0 && len(pathArray[0]) == 0 {
			pathArray = pathArray[1:]
		}
		_, v, err := d.get(ctx, pathArray)
		if err != nil {
			cb(err)
			return
		}
		switch operation {
		/*case op_test:*/
		case op_remove:
			if v.CanSet() {
				inputValue := reflect.New(v.Type()).Interface()
				v.Set(reflect.ValueOf(inputValue).Elem())
			} else {
				cb(errors.New("field value can't be set"))
				return
			}
		case op_add:
			if v.CanSet() {
				_, err := jsonparser.ArrayEach(value, func(valueItem []byte, dataType jsonparser.ValueType, offset int, err error) {
					inputValue := reflect.New(v.Type().Elem()).Interface()
					err = json.Unmarshal(valueItem, &inputValue)
					if err != nil {
						cb(err)
						return
					}
					v.Set(reflect.Append(v, reflect.ValueOf(inputValue).Elem()))
				})
				if err != nil {
					cb(err)
					return
				}
			} else {
				cb(errors.New("field value can't be set"))
				return
			}
		case op_replace:
			if v.CanSet() {
				if v.Kind() == reflect.String {
					v.SetString(string(value))
				} else {
					inputValue := reflect.New(v.Type()).Interface()
					err = json.Unmarshal(value, &inputValue)
					if err != nil {
						cb(err)
						return
					}
					v.Set(reflect.ValueOf(inputValue).Elem())
				}
			} else {
				cb(errors.New("field value can't be set"))
				return
			}
		case op_move:
			from, err := jsonparser.GetString(patcd, "from")
			if err != nil {
				cb(errors.New("invalid from"))
				return
			}

			fromPath := strings.Split(from, "/")
			if len(fromPath) > 0 && len(fromPath[0]) == 0 {
				fromPath = fromPath[1:]
			}

			_, fromV, err := d.get(ctx, fromPath)
			if err != nil {
				cb(err)
				return
			}

			if v.CanSet() {
				v.Set(fromV)
			} else {
				cb(errors.New("field value can't be set"))
				return
			}

			if fromV.CanSet() {
				fromValue := reflect.New(fromV.Type()).Interface()
				fromV.Set(reflect.ValueOf(fromValue).Elem())
			} else {
				cb(errors.New("field value can't be set"))
				return
			}
		case op_copy:
			from, err := jsonparser.GetString(patcd, "from")
			if err != nil {
				cb(errors.New("invalid from"))
				return
			}

			fromPath := strings.Split(from, "/")
			if len(fromPath) > 0 && len(fromPath[0]) == 0 {
				fromPath = fromPath[1:]
			}

			_, fromV, err := d.get(ctx, fromPath)
			if err != nil {
				cb(err)
				return
			}

			if v.CanSet() {
				v.Set(fromV)
			} else {
				cb(errors.New("field value can't be set"))
				return
			}
		default:
			cb(errors.New("invalid operation"))
			return
		}
	})
	if err != nil {
		return err
	}
	return endErr
}

func (d *Document) Delete(ctx Context) (*Document, error) {
	var valueHolder = d.reflectValue
	for i, field := range fields {
		var f *Field
		// get real field name (in case json field has different name)
		if d.Kind != nil {
			var ok bool
			if f, ok = d.Kind.fields[field]; ok {
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
						return d, errors.New("field value can't be set")
					}
					return d, nil
				} else {
					valueHolder = valueHolder.Index(index)
				}
			} else {
				return d, errors.New("error converting string to slice index")
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
		return d, errors.New("field value can't be set")
	}

	return d, nil
}

func (d *Document) Get(ctx Context, fields []string) (*Document, interface{}, error) {
	h2, v, err := d.get(ctx, fields)
	return h2, v.Interface(), err
}

func (d *Document) Bytes() ([]byte, error) {
	return json.Marshal(d.GetValue())
}

func (d *Document) Load(ps []datastore.Property) error {
	d.hasLoadedData = true
	d.rollbackProperties = ps
	if d.hasInputData {
		// replace only empty fields
		n := d.Kind.New().Interface()
		if err := datastore.LoadStruct(n, ps); err != nil {
			return err
		}

		d.reflectValue = reflect.ValueOf(n)

		return nil
	}
	err := datastore.LoadStruct(d.reflectValue.Interface(), ps)
	return err
}

func (d *Document) Save() ([]datastore.Property, error) {
	//var now = reflect.ValueOf(time.Now())
	//v := reflect.ValueOf(d.value).Elem()
	/*for _, meta := range d.Kind.MetaFields {
		field := v.FieldByName(meta.FieldName)
		if field.CanSet() {
			switch meta.Type {
			case "updatedat":
				field.Set(now)
			case "createdat":
				if !d.hasLoadedData {
					field.Set(now)
				}
			}
		}
	}*/
	return datastore.SaveStruct(d.reflectValue.Interface())
}
