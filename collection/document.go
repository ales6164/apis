package collection

import (
	"encoding/json"
	"errors"
	"github.com/buger/jsonparser"
	"google.golang.org/appengine/datastore"
	"reflect"
	"strings"
	"github.com/ales6164/apis/kind"
	"golang.org/x/net/context"
)

type Document struct {
	kind               kind.Kind
	ctx                context.Context
	key                *datastore.Key
	value              reflect.Value
	hasInputData       bool // when updating
	hasLoadedData      bool
	rollbackProperties []datastore.Property
	kind.Doc
}

var (
	keyType = reflect.TypeOf(&datastore.Key{})
)

/*func (h *Document) Id() string {
	if d.Key != nil {
		return d.Key.Encode()
	}
	return ""
}*/

/*func (h *Document) ReflectValue() {
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
}*/

func (d *Document) Value() reflect.Value {
	return d.value
}

func (d *Document) Key() *datastore.Key {
	return d.key
}

func (d *Document) Type() reflect.Type {
	return d.kind.Type()
}

/*type Rich struct {
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
}*/

/*func (d *Document) setData(v interface{}) {
	d.reflectValue = reflect.ValueOf(v)
}*/

func (d *Document) Parse(body []byte) error {
	d.hasInputData = true
	var value = reflect.New(d.Type()).Interface()
	err := json.Unmarshal(body, &value)
	d.value = reflect.ValueOf(value)
	return err
}

const (
	op_test    = "test"
	op_remove  = "remove"
	op_add     = "add"
	op_replace = "replace"
	op_move    = "move"
	op_copy    = "copy"
)

func (d *Document) Patch(data []byte) error {
	var endErr error
	var cb = func(err error) {
		endErr = err
	}
	_, err := jsonparser.ArrayEach(data, func(patch []byte, dataType jsonparser.ValueType, offset int, err error) {
		operation, _ := jsonparser.GetString(patch, "op")
		value, _, _, _ := jsonparser.Get(patch, "value")
		path, err := jsonparser.GetString(patch, "path")
		if err != nil {
			cb(errors.New("invalid path"))
			return
		}
		pathArray := strings.Split(path, "/")
		if len(pathArray) > 0 && len(pathArray[0]) == 0 {
			pathArray = pathArray[1:]
		}
		v, err := d.Kind().ValueAt(d.value, pathArray)
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
			from, err := jsonparser.GetString(patch, "from")
			if err != nil {
				cb(errors.New("invalid from"))
				return
			}

			fromPath := strings.Split(from, "/")
			if len(fromPath) > 0 && len(fromPath[0]) == 0 {
				fromPath = fromPath[1:]
			}

			fromV, err := d.Kind().ValueAt(d.value, fromPath)
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
			from, err := jsonparser.GetString(patch, "from")
			if err != nil {
				cb(errors.New("invalid from"))
				return
			}

			fromPath := strings.Split(from, "/")
			if len(fromPath) > 0 && len(fromPath[0]) == 0 {
				fromPath = fromPath[1:]
			}

			fromV, err := d.Kind().ValueAt(d.value, fromPath)
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

func (d *Document) Delete() error {
	return datastore.Delete(d.ctx, d.key)
}

func (d *Document) Set(data interface{}) (kind.Doc, error) {
	var err error
	if d.key == nil || d.key.Incomplete() {
		return d, errors.New("can't set value for undefined key")
	}
	if d.value.Elem().CanSet() {
		if bytes, ok := data.([]byte); ok {
			inputValue := reflect.New(d.Type()).Interface()
			err := json.Unmarshal(bytes, &inputValue)
			if err != nil {
				return d, err
			}
			d.value.Elem().Set(reflect.ValueOf(inputValue).Elem())
		} else {
			d.value.Elem().Set(reflect.ValueOf(data).Elem())
		}
	} else {
		return d, errors.New("field value can't be set")
	}
	d.key, err = datastore.Put(d.ctx, d.key, d)
	return d, err
}

// todo: some function for giving access to this document
func (d *Document) Add(data interface{}) (kind.Doc, error) {
	var err error
	var value reflect.Value
	if d.value.Elem().CanSet() {
		if bytes, ok := data.([]byte); ok {
			inputValue := reflect.New(d.Type()).Interface()
			err := json.Unmarshal(bytes, &inputValue)
			if err != nil {
				return d, err
			}
			value = reflect.ValueOf(inputValue).Elem()
		} else {
			value = reflect.ValueOf(data).Elem()
		}
	} else {
		return d, errors.New("field value can't be set" + d.value.String())
	}
	if d.key == nil || d.key.Incomplete() {
		d.key = datastore.NewIncompleteKey(d.ctx, d.Kind().Name(), nil)
		d.value.Elem().Set(value)
		d.key, err = datastore.Put(d.ctx, d.key, d)
	} else {
		err = datastore.RunInTransaction(d.ctx, func(ctx context.Context) error {
			err = datastore.Get(ctx, d.key, d)
			if err != nil {
				if err == datastore.ErrNoSuchEntity {
					// ok
					d.value.Elem().Set(value)
					d.key, err = datastore.Put(ctx, d.key, d)
				}
				return err
			}
			return kind.ErrEntityAlreadyExists
		}, nil)
	}
	return d, err
}

func (d *Document) Get() (kind.Doc, error) {
	return d, datastore.Get(d.ctx, d.key, d)
}

func (d *Document) Kind() kind.Kind {
	return d.kind
}

func (d *Document) Load(ps []datastore.Property) error {
	d.hasLoadedData = true
	d.rollbackProperties = ps
	if d.hasInputData {
		// replace only empty fields
		n := reflect.New(d.Type()).Interface()
		if err := datastore.LoadStruct(n, ps); err != nil {
			return err
		}
		d.value = reflect.ValueOf(n)
		return nil
	}
	err := datastore.LoadStruct(d.value.Interface(), ps)
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
	return datastore.SaveStruct(d.value.Interface())
}
