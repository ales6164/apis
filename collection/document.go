package collection

import (
	"encoding/json"
	"errors"
	"github.com/ales6164/apis/kind"
	"github.com/buger/jsonparser"
	"golang.org/x/net/context"
	"google.golang.org/appengine/datastore"
	"reflect"
	"strings"
)

type document struct {
	kind               kind.Kind
	member             *datastore.Key
	defaultCtx         context.Context
	ctx                context.Context
	key                *datastore.Key
	value              reflect.Value
	hasInputData       bool // when updating
	hasLoadedData      bool
	rollbackProperties []datastore.Property
	ancestor           kind.Doc
	hasAncestor        bool
	meta               *meta
	kind.Doc
}

type DocumentCollectionRelationship struct {
	Roles []string // fullControl, ...
}

func NewDoc(ctx context.Context, kind kind.Kind, key *datastore.Key, ancestor kind.Doc) (*document, error) {
	if key != nil && key.Kind() != kind.Name() {
		key = nil
	}
	doc := &document{
		kind:        kind,
		defaultCtx:  ctx,
		ctx:         ctx,
		value:       reflect.New(kind.Type()),
		key:         key,
		ancestor:    ancestor,
		hasAncestor: ancestor != nil,
	}

	_, err := doc.Meta()

	return doc, err
}

func (d *document) Exists() bool {
	return d.meta.exists
}

func (d *document) Meta() (kind.Meta, error) {
	var err error
	var ancestorMeta kind.Meta
	if d.hasAncestor {
		ancestorMeta, err = d.ancestor.Meta()
		if err != nil {
			return d.meta, err
		}
	}

	if d.meta != nil {
		return d.meta, nil
	}

	if d.key == nil {
		d.key = datastore.NewIncompleteKey(d.defaultCtx, d.Kind().Name(), nil)
	}

	d.meta, err = getMeta(d.defaultCtx, d, ancestorMeta)
	if err != nil {
		return d.meta, err
	}

	if d.ctx, d.key, err = SetNamespace(d.ctx, d.key, d.meta.value.GroupId); err != nil {
		return d.meta, err
	}

	return d.meta, err
}

func (d *document) Commit() error {
	if d.meta == nil {
		return errors.New("entry doesn't match meta")
	}
	err := d.meta.Save(d.defaultCtx, d, d.meta.group)
	return err
}

func (d *document) Context() context.Context {
	return d.ctx
}

/*

Get - gets entry and value simultaneously
Add - adds entry after operation
Set - runs Get and sees what's going on... if value exists but it has no entry it creates one ... if entry exists and the person has access it updates the value and entry simultaneously
Del - deletes both

Every action should check if operation is permitted


 */

var (
	keyType = reflect.TypeOf(&datastore.Key{})
)

func (d *document) Value() reflect.Value {
	return d.value
}

func (d *document) Key() *datastore.Key {
	return d.key
}

// SetKey(key *datastore.Key)
//	Copy() Doc

func (d *document) SetKey(key *datastore.Key) {
	if key != nil && key.Kind() != d.kind.Name() {
		key = nil
	}
	d.key = key
}

func (d *document) Copy() kind.Doc {
	return &document{
		kind:               d.kind,
		member:             d.member,
		defaultCtx:         d.defaultCtx,
		ctx:                d.ctx,
		key:                d.key,
		value:              reflect.New(d.kind.Type()),
		hasInputData:       d.hasAncestor,
		hasLoadedData:      d.hasLoadedData,
		rollbackProperties: d.rollbackProperties,
		ancestor:           d.ancestor,
		hasAncestor:        d.hasAncestor,
	}
}

func (d *document) Type() reflect.Type {
	return d.kind.Type()
}

func (d *document) Parse(body []byte) error {
	d.hasInputData = true
	var value = reflect.New(d.Type()).Interface()
	err := json.Unmarshal(body, &value)
	d.value = reflect.ValueOf(value)
	return err
}

// Loads relationship table and checks if user has access to the specified namespace.
// Then adds the parent and rewrites document key and context.
/*func (d *document) SetParent(doc kind.Doc) (kind.Doc, error) {
	if doc != nil {
		m, err := kind.GetMeta(d.ctx, doc)
		if err != nil {
			return d, err
		}

		d.parent = doc
	}
	return d, nil
}*/

func (d *document) Ancestor() kind.Doc {
	return d.ancestor
}

func (d *document) HasAncestor() bool {
	return d.hasAncestor
}

const (
	op_test    = "test"
	op_remove  = "remove"
	op_add     = "add"
	op_replace = "replace"
	op_move    = "move"
	op_copy    = "copy"
)

func (d *document) Get() (kind.Doc, error) {
	return d, datastore.Get(d.ctx, d.key, d)
}

func (d *document) Patch(data []byte) error {
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

func (d *document) Delete() error {
	return datastore.RunInTransaction(d.ctx, func(tc context.Context) error {
		err := datastore.Delete(tc, d.key)
		if err != nil {
			return err
		}
		return d.kind.Decrement(tc)
	}, &datastore.TransactionOptions{XG: true})
}

func (d *document) Set(data interface{}) (kind.Doc, error) {
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
	if err != nil {
		return d, err
	}

	err = d.Commit()

	return d, err
}

func (d *document) SetMember(member *datastore.Key) {
	d.member = member
}

// todo: some function for giving access to this document
// Run from inside a transaction.
func (d *document) Add(data interface{}) (kind.Doc, error) {
	// 1. Parse value
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

	// 2. Set key
	if d.key == nil {
		d.key = datastore.NewIncompleteKey(d.ctx, d.Kind().Name(), nil)
	}

	// 4. Store value
	if d.key.Incomplete() {
		d.value.Elem().Set(value)
		err = datastore.RunInTransaction(d.ctx, func(tc context.Context) error {
			d.key, err = datastore.Put(tc, d.key, d)
			if err != nil {
				return err
			}
			err = d.kind.Increment(tc)
			if err != nil {
				return err
			}
			return d.Commit()
		}, &datastore.TransactionOptions{XG: true})
	} else {
		err = datastore.RunInTransaction(d.ctx, func(tc context.Context) error {
			err = datastore.Get(tc, d.key, d)
			if err != nil {
				if err == datastore.ErrNoSuchEntity {
					// ok
					d.value.Elem().Set(value)
					d.key, err = datastore.Put(tc, d.key, d)
					if err != nil {
						return err
					}
					err = d.kind.Increment(tc)
					if err != nil {
						return err
					}
					return d.Commit()
				}
				return err
			}
			return kind.ErrEntityAlreadyExists
		}, &datastore.TransactionOptions{XG: true})
	}
	return d, err
}

func (d *document) SetRole(member *datastore.Key, role ...string) error {
	if d.key == nil || d.key.Incomplete() {
		return errors.New("can't set role if key is incomplete")
	}
	_, err := datastore.Put(d.defaultCtx, datastore.NewKey(d.defaultCtx, "_groupRelationship", d.key.Encode(), 0, member), &GroupRelationship{
		Roles: role,
	})
	return err
}

func (d *document) HasRole(member *datastore.Key, role ...string) bool {
	var iam = new(GroupRelationship)
	err := datastore.Get(d.defaultCtx, datastore.NewKey(d.defaultCtx, "_groupRelationship", d.key.Encode(), 0, member), iam)
	if err == nil && ContainsScope(iam.Roles, role...) {
		return true
	}
	return false
}

func (d *document) Kind() kind.Kind {
	return d.kind
}

func (d *document) Load(ps []datastore.Property) error {
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

func (d *document) Save() ([]datastore.Property, error) {
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
