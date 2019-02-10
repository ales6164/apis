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
	"time"
)

type document struct {
	kind                kind.Kind
	owner               *datastore.Key
	defaultCtx          context.Context
	key                 *datastore.Key
	value               reflect.Value
	hasInputData        bool // when updating
	hasLoadedData       bool
	rollbackProperties  []datastore.Property
	accessControllerDoc kind.Doc
	parent              kind.Doc
	metaWrapper         *metaWrapper

	kind.Doc
}

type Meta struct {
	CreatedAt       time.Time      `json:"createdAt"`
	UpdatedAt       time.Time      `json:"updatedAt"`
	CreatedBy       *datastore.Key `json:"createdBy"`
	UpdatedBy       *datastore.Key `json:"updatedBy"`
	Version         int64          `json:"version"`
	Namespace       string         `json:"-"`
	ParentNamespace string         `json:"-"`
}

type metaWrapper struct {
	Meta *Meta `datastore:"_"`
}

type DocumentCollectionRelationship struct {
	Roles []string // fullControl, ...
}

func NewDoc(kind kind.Kind, key *datastore.Key, parent kind.Doc) *document {
	if key != nil && key.Kind() != kind.Name() {
		key = nil
	}
	doc := &document{
		kind: kind,
		/*defaultCtx:  ctx,*/
		value:               reflect.New(kind.Type()),
		key:                 key,
		parent:              parent,
		metaWrapper:         &metaWrapper{new(Meta)},
	}

	if parent != nil && parent.AccessController() != nil {
		doc.accessControllerDoc = parent.AccessController()
	}

	return doc
}

func (d *document) SetOwner(key *datastore.Key) {
	d.owner = key
}

func (d *document) GetOwner() *datastore.Key {
	return d.owner
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

const (
	op_test    = "test"
	op_remove  = "remove"
	op_add     = "add"
	op_replace = "replace"
	op_move    = "move"
	op_copy    = "copy"
)

func (d *document) SetAccessControl(enable bool) {
	if enable {
		d.accessControllerDoc = d
	}
}

func (d *document) Get(ctx context.Context) (kind.Doc, error) {
	return d, datastore.Get(ctx, d.key, d)
}

// not implemented!
func (d *document) Patch(ctx context.Context, data []byte) error {
	return errors.New("not implemented")

	// todo: add access control and meta

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

func (d *document) Delete(ctx context.Context) error {
	return datastore.RunInTransaction(ctx, func(tc context.Context) error {
		err := datastore.Delete(tc, d.key)
		if err != nil {
			return err
		}
		return d.kind.Decrement(tc)
	}, &datastore.TransactionOptions{XG: true})
}

func (d *document) Set(ctx context.Context ,data interface{}) (kind.Doc, error) {
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

	now := time.Now()
	err = datastore.RunInTransaction(ctx, func(tc context.Context) error {
		err = datastore.Get(tc, d.key, d)
		if err != nil {
			if err == datastore.ErrNoSuchEntity {
				// creating new

				d.metaWrapper.Meta.CreatedAt = now
				d.metaWrapper.Meta.CreatedBy = d.owner
				d.metaWrapper.Meta.Version = 0

				d.key, err = datastore.Put(tc, d.key, d)
				if err != nil {
					return err
				}
				return d.kind.Increment(tc)
			}
			return err
		}
		// overwriting existing

		d.metaWrapper.Meta.UpdatedAt = now
		d.metaWrapper.Meta.UpdatedBy = d.owner
		d.metaWrapper.Meta.Version += 1

		d.key, err = datastore.Put(tc, d.key, d)
		return err
	}, &datastore.TransactionOptions{XG: true})

	d.key, err = datastore.Put(ctx, d.key, d)
	return d, err
}

// todo: some function for giving access to this document
// Run from inside a transaction.
func (d *document) Add(ctx context.Context, data interface{}) (kind.Doc, error) {
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

		d.value.Elem().Set(value)
	} else {
		return d, errors.New("field value can't be set" + d.value.String())
	}

	// 2. Set key
	if d.key == nil {
		d.key = datastore.NewIncompleteKey(ctx, d.Kind().Name(), nil)
	}

	now := time.Now()

	// 4. Store value
	if d.key.Incomplete() {
		err = datastore.RunInTransaction(ctx, func(tc context.Context) error {

			d.metaWrapper.Meta.CreatedAt = now
			d.metaWrapper.Meta.CreatedBy = d.owner
			d.metaWrapper.Meta.Version = 0

			d.key, err = datastore.Put(tc, d.key, d)
			if err != nil {
				return err
			}
			return d.kind.Increment(tc)
		}, &datastore.TransactionOptions{XG: true})
	} else {
		err = datastore.RunInTransaction(ctx, func(tc context.Context) error {
			err = datastore.Get(tc, d.key, d)
			if err != nil {
				if err == datastore.ErrNoSuchEntity {
					// ok

					d.metaWrapper.Meta.CreatedAt = now
					d.metaWrapper.Meta.CreatedBy = d.owner
					d.metaWrapper.Meta.Version = 0

					d.key, err = datastore.Put(tc, d.key, d)
					if err != nil {
						return err
					}
					return d.kind.Increment(tc)
				}
				return err
			}
			return kind.ErrEntityAlreadyExists
		}, &datastore.TransactionOptions{XG: true})
	}
	return d, err
}

/*func (d *document) SetRole(member *datastore.Key, role ...string) error {
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
}*/

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

	// just remove meta for now
	var metaPs []datastore.Property
	noMetaPs := ps[:0]
	for _, p := range ps {
		if string(p.Name[0]) == "_" {
			metaPs = append(metaPs, p)
		} else {
			noMetaPs = append(noMetaPs, p)
		}
	}

	err := datastore.LoadStruct(d.metaWrapper, metaPs)
	if err != nil {
		return err
	}

	return datastore.LoadStruct(d.value.Interface(), noMetaPs)
}

func (d *document) Save() ([]datastore.Property, error) {

	// meta
	metaPs, err := datastore.SaveStruct(d.metaWrapper)
	if err != nil {
		return metaPs, err
	}

	ps, err := datastore.SaveStruct(d.value.Interface())
	return append(metaPs, ps...), err
}
