package apis

import (
	"google.golang.org/appengine/search"
	"reflect"
)

type Document struct {
	fields []search.Field
	facets []search.Facet
}

func (x *Document) Parse(kind *Kind) interface{} {
	val := reflect.ValueOf(kind.New()).Elem()

	for _, field := range x.fields {
		searchField := kind.searchFields[field.Name]

		valField := val.FieldByName(field.Name)
		if !valField.IsValid() || !valField.CanSet() {
			continue
		}

		if searchField.Multiple {
			convertedValue := searchField.Converter.ConvertBack(valField, field.Value)
			valField.Set(reflect.Append(valField, convertedValue))
		} else {
			convertedValue := searchField.Converter.ConvertBack(valField, field.Value)
			valField.Set(convertedValue)
		}
	}

	return nil
}

func (x *Document) Load(fields []search.Field, meta *search.DocumentMetadata) error {
	x.fields = fields
	x.facets = meta.Facets
	return nil
}

func (x *Document) Save() ([]search.Field, *search.DocumentMetadata, error) {
	meta := &search.DocumentMetadata{
		Facets: x.facets,
	}
	return x.fields, meta, nil
}
