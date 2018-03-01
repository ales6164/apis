package kind

import (
	"fmt"
	"reflect"
	"google.golang.org/appengine/datastore"
	"golang.org/x/net/context"
)

type Worker interface {
	Init() error
	Parse(value interface{}) ([]datastore.Property, error)
	Output(ctx context.Context, value interface{}) interface{}
}


func (x *Field) Parse(value interface{}) ([]datastore.Property, error) {
	var list []datastore.Property
	if x.Multiple {
		if multiArray, ok := value.([]interface{}); ok {
			for _, value := range multiArray {
				value, err := x.Check(value)
				if err != nil {
					return list, err
				}
				list = append(list, x.Property(value))
			}
		} else if value == nil {
			value, err := x.Check(value)
			if err != nil {
				return list, err
			}
			list = append(list, x.Property(value))
		} else {
			return list, fmt.Errorf("field '%s' value type '%s' is not valid", x.Name, reflect.TypeOf(value).String())
		}
	} else {
		value, err := x.Check(value)
		if err != nil {
			return list, err
		}
		list = append(list, x.Property(value))
	}
	return list, nil
}

func (x *Field) Property(value interface{}) datastore.Property {
	return datastore.Property{
		Name:     x.Name,
		Multiple: x.Multiple,
		NoIndex:  x.NoIndex,
		Value:    value,
	}
}

func (x *Field) Check(value interface{}) (interface{}, error) {
	var err error
	if value == nil {
		if x.IsRequired {
			return value, fmt.Errorf("field '%s' value is required", x.Name)
		}
	} else {
		err = x.Validate(value)
		if err != nil {
			return value, err
		}
		value, err = x.Transform(value)
	}
	return value, err
}

func (x *Field) Validate(value interface{}) error {
	return nil
}

func (x *Field) Transform(value interface{}) (interface{}, error) {
	return value, nil
}

func (x *Field) Output(ctx context.Context, value interface{}) interface{} {
	return value
}
