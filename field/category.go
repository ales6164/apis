package field

import (
	"fmt"
	"reflect"
	"google.golang.org/appengine/datastore"
	"golang.org/x/net/context"
	"github.com/ales6164/go-cms/kind"
)

type Category struct {
	Name     string
	Required bool
	Multiple bool
	NoIndex  bool
	kind     *kind.Kind
}

func (x *Category) Init() error {
	return nil
}

func (x *Category) RegisterSubKind() *kind.Kind {
	x.kind = kind.New("categories")
	return nil
}

func (x *Category) GetName() string {
	return x.Name
}

func (x *Category) GetRequired() bool {
	return x.Required
}

func (x *Category) GetMultiple() bool {
	return x.Multiple
}

func (x *Category) GetNoIndex() bool {
	return x.NoIndex
}

func (x *Category) GetNested() bool {
	return true
}

func (x *Category) Parse(value interface{}) ([]datastore.Property, error) {
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

func (x *Category) Property(value interface{}) datastore.Property {
	return datastore.Property{
		Name:     x.Name,
		Multiple: x.Multiple,
		NoIndex:  x.NoIndex,
		Value:    value,
	}
}

func (x *Category) Check(value interface{}) (interface{}, error) {
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

func (x *Category) Validate(value interface{}) error {
	if _, ok := value.(string); ok {
		return nil
	}
	return fmt.Errorf("field '%s' value type '%s' is not valid", x.Name, reflect.TypeOf(value).String())
}

func (x *Category) Transform(value interface{}) (*datastore.Key, error) {
	return datastore.DecodeKey(value.(string))
}

func (x *Category) Output(ctx context.Context, value interface{}) interface{} {
	if key, ok := value.(*datastore.Key); ok {
		var category category
		datastore.Get(ctx, key, &category)
		value = category
	}
	return value
}

type category struct {
}
