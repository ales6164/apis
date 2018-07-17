package kind

import (
	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/search"
	"reflect"
	"time"
)

type Converter interface {
	Convert(value reflect.Value) interface{}
	ConvertBack(field reflect.Value, value interface{}) reflect.Value
}

type EmptyConverter struct{}

type KeyConverter struct{}

type BoolConverter struct{}

type Float64Converter struct{}

type AtomConverter struct{}

type StringConverter struct{}

type HTMLConverter struct{}

var (
	intType      = reflect.TypeOf(int(0))
	int8Type     = reflect.TypeOf(int8(0))
	int16Type    = reflect.TypeOf(int16(0))
	int32Type    = reflect.TypeOf(int32(0))
	int64Type    = reflect.TypeOf(int64(0))
	float32Type  = reflect.TypeOf(float32(0))
	float64Type  = reflect.TypeOf(float64(0))
	boolType     = reflect.TypeOf(false)
	stringType   = reflect.TypeOf("")
	atomType     = reflect.TypeOf(search.Atom(""))
	htmlType     = reflect.TypeOf(search.HTML(""))
	timeType     = reflect.TypeOf(time.Time{})
	keyType      = reflect.TypeOf(datastore.Key{})
	geoPointType = reflect.TypeOf(appengine.GeoPoint{})
)

func (x *EmptyConverter) Convert(value reflect.Value) interface{} {
	return value.Interface()
}

func (x *KeyConverter) Convert(value reflect.Value) interface{} {
	key := value.Interface().(*datastore.Key)
	if key != nil {
		return search.Atom(key.Encode())
	}
	return search.Atom("")
}

func (x *BoolConverter) Convert(value reflect.Value) interface{} {
	val := value.Interface().(bool)
	if val {
		return search.Atom("true")
	}
	return search.Atom("false")
}

func (x *Float64Converter) Convert(value reflect.Value) interface{} {
	return value.Convert(float64Type).Interface()
}

func (x *AtomConverter) Convert(value reflect.Value) interface{} {
	return value.Convert(atomType).Interface()
}

func (x *StringConverter) Convert(value reflect.Value) interface{} {
	return value.Convert(stringType).Interface()
}

func (x *HTMLConverter) Convert(value reflect.Value) interface{} {
	return value.Convert(htmlType).Interface()
}

func (x *EmptyConverter) ConvertBack(field reflect.Value, value interface{}) reflect.Value {
	return reflect.ValueOf(value)
}

func (x *KeyConverter) ConvertBack(field reflect.Value, value interface{}) reflect.Value {
	if value != nil {
		if v, ok := value.(search.Atom); ok {
			return reflect.ValueOf(string(v))
		}
	}
	return reflect.Zero(atomType)
}

func (x *BoolConverter) ConvertBack(field reflect.Value, value interface{}) reflect.Value {
	if value != nil {
		if v, ok := value.(search.Atom); ok {
			if v == "true" {
				return reflect.ValueOf(true)
			}
			return reflect.ValueOf(false)
		}
	}
	return reflect.Zero(boolType)
}

func (x *Float64Converter) ConvertBack(field reflect.Value, value interface{}) reflect.Value {
	return reflect.ValueOf(value).Convert(field.Type())
}

func (x *AtomConverter) ConvertBack(field reflect.Value, value interface{}) reflect.Value {
	return reflect.ValueOf(value).Convert(field.Type())
}

func (x *StringConverter) ConvertBack(field reflect.Value, value interface{}) reflect.Value {
	return reflect.ValueOf(value).Convert(field.Type())
}

func (x *HTMLConverter) ConvertBack(field reflect.Value, value interface{}) reflect.Value {
	return reflect.ValueOf(value).Convert(field.Type())
}
