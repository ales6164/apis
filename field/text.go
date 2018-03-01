package field

import (
	"fmt"
	"reflect"
	"google.golang.org/appengine/datastore"
	"golang.org/x/net/context"
	"github.com/ales6164/go-cms/kind"
)

type Text struct {
	Name     string
	Required bool
	Multiple bool
	NoIndex  bool
	Nested   bool
}

func (x *Text) Init() error {
	return nil
}

func (x *Text) RegisterSubKind() *kind.Kind {
	return nil
}

func (x *Text) GetName() string {
	return x.Name
}

func (x *Text) GetRequired() bool {
	return x.Required
}

func (x *Text) GetMultiple() bool {
	return x.Multiple
}

func (x *Text) GetNoIndex() bool {
	return x.NoIndex
}

func (x *Text) GetNested() bool {
	return x.Nested
}

func (x *Text) Parse(value interface{}) ([]datastore.Property, error) {
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


func (x *Text) Property(value interface{}) datastore.Property {
	return datastore.Property{
		Name:     x.Name,
		Multiple: x.Multiple,
		NoIndex:  x.NoIndex,
		Value:    value,
	}
}

func (x *Text) Check(value interface{}) (interface{}, error) {
	var err error
	if value == nil {
		if x.Required {
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

func (x *Text) Validate(value interface{}) error {
	if _, ok := value.(string); ok {
		return nil
	}
	return fmt.Errorf("field '%s' value type '%s' is not valid", x.Name, reflect.TypeOf(value).String())
}

func (x *Text) Transform(value interface{}) (interface{}, error) {
	return value, nil
}


func (x *Text) Output(ctx context.Context, value interface{}) interface{} {
	return value
}
