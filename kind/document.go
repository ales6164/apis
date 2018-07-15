package kind

import (
	"google.golang.org/appengine/search"
)

type Document struct {
	fields []search.Field
	facets []search.Facet
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
