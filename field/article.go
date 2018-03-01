package field

import (
	"fmt"
	"reflect"
	"google.golang.org/appengine/datastore"
	"github.com/gosimple/slug"
	"golang.org/x/net/context"
	"github.com/ales6164/go-cms/kind"
)

// Transforms text value into a slug string producing { text: originalValue, slug: newSlugValue }
type Article struct {
	Name     string
	Required bool
}

func (x *Article) Init() error {
	return  nil
}

func (x *Article) RegisterSubKind() *kind.Kind {
	return nil
}

func (x *Article) GetName() string {
	return x.Name
}

func (x *Article) GetRequired() bool {
	return x.Required
}

func (x *Article) GetMultiple() bool {
	return false
}

func (x *Article) GetNoIndex() bool {
	return true
}

func (x *Article) GetNested() bool {
	return true
}

func (x *Article) Parse(value interface{}) ([]datastore.Property, error) {
	var list []datastore.Property
	var v map[string]interface{}

	var err error
	if value == nil {
		if x.Required {
			return list, fmt.Errorf("field '%s' value is required", x.Name)
		}
		return list, nil
	} else {
		v, err = x.Transform(value)
	}
	if err != nil {
		return list, err
	}

	valueText := v["text"].(string)
	valueSlug := v["slug"].(string)

	if len(valueText) == 0 {
		return list, fmt.Errorf("field '%s' value[text] is required", x.Name)
	}

	if len(valueSlug) == 0 {
		valueSlug = slug.Make(valueSlug)
	}

	list = append(list, datastore.Property{
		Name:     x.Name + ".text",
		Multiple: false,
		NoIndex:  true,
		Value:    valueText,
	})
	list = append(list, datastore.Property{
		Name:     x.Name + ".slug",
		Multiple: false,
		NoIndex:  true,
		Value:    valueSlug,
	})

	return list, nil
}

func (x *Article) Transform(value interface{}) (map[string]interface{}, error) {
	var v map[string]interface{}
	if v, ok := value.(map[string]interface{}); ok {
		return v, nil
	}
	return v, fmt.Errorf("field '%s' value type '%s' is not valid", x.Name, reflect.TypeOf(value).String())
}

func (x *Article) Output(ctx context.Context, value interface{}) interface{} {
	return value
}
