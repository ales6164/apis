package kind

import (
	"fmt"
	"google.golang.org/appengine/datastore"
	"golang.org/x/net/context"
	"github.com/ales6164/apis/errors"
	"reflect"
)

type Worker interface {
	Init() error
	Parse(value interface{}) ([]datastore.Property, error)
	Output(ctx context.Context, value interface{}) interface{}
}

func (x *Field) Parse(value interface{}) ([]datastore.Property, error) {
	var list []datastore.Property
	if x.Multiple {
		if value == nil {
			value, err := x.Check(value)
			if err != nil {
				return list, err
			}
			props, err := x.Property(value)
			if err != nil {
				return list, err
			}
			list = append(list, props...)
		} else {
			if arr, ok := value.([]interface{}); ok {
				for _, value := range arr {
					value, err := x.Check(value)
					if err != nil {
						return list, err
					}
					props, err := x.Property(value)
					if err != nil {
						return list, err
					}
					list = append(list, props...)
				}
			} else if arr, ok := value.([]string); ok {
				for _, value := range arr {
					value, err := x.Check(value)
					if err != nil {
						return list, err
					}
					props, err := x.Property(value)
					if err != nil {
						return list, err
					}
					list = append(list, props...)
				}
			} else {
				return list, fmt.Errorf("field '%s' value type '%s' is not valid", x.Name, reflect.TypeOf(value).String())
			}
		}
	} else {
		value, err := x.Check(value)
		if err != nil {
			return list, err
		}
		props, err := x.Property(value)
		if err != nil {
			return list, err
		}
		list = append(list, props...)
	}
	return list, nil
}

func (x *Field) Property(value interface{}) ([]datastore.Property, error) {
	var props []datastore.Property
	props = append(props, datastore.Property{
		Name:     x.Name,
		Multiple: x.Multiple,
		NoIndex:  x.NoIndex,
		Value:    value,
	})
	return props, nil
}

func (x *Field) Check(value interface{}) (interface{}, error) {
	var err error
	if value == nil {
		if x.IsRequired {
			return value, errors.ErrFieldRequired.With(fmt.Sprintf("field '%s' value is required", x.Name))
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

func (x *Field) Output(value interface{}) interface{} {
	return value
}
