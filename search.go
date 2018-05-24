package apis

import (
	"golang.org/x/net/context"
	"google.golang.org/appengine/search"
	"reflect"
	"strings"
	"github.com/ales6164/apis/errors"
	"github.com/ales6164/apis/kind"
)

func saveToIndex(ctx context.Context, kind *kind.Kind, id string, value interface{}) error {
	index, err := OpenIndex(kind.Name)
	if err != nil {
		return err
	}

	v := reflect.ValueOf(value).Elem()

	var searchType reflect.Type
	if kind.SearchType == nil {
		searchType = kind.Type
	} else {
		searchType = kind.SearchType
	}

	doc := reflect.New(searchType)

	for i := 0; i < searchType.NumField(); i++ {
		typeField := searchType.Field(i)
		docFieldName := typeField.Name

		var convType reflect.Type
		var hasConvType bool
		if v, ok := typeField.Tag.Lookup("search"); ok {
			vspl := strings.Split(v, ",")
			if len(vspl) >= 3 {
				switch vspl[3] {
				case "atom":
					convType = reflect.TypeOf(search.Atom(""))
					hasConvType = true
				case "string":
					convType = reflect.TypeOf("")
					hasConvType = true
				}
			}
		}

		valField := v.FieldByName(docFieldName)
		if !valField.IsValid() {
			continue
		}

		docField := doc.Elem().FieldByName(docFieldName)
		if docField.CanSet() {
			if docField.Kind() == reflect.Slice {
				// make slice to get value type
				sliceValTyp := reflect.MakeSlice(docField.Type(), 1, 1).Index(0).Type()
				if valField.Kind() == reflect.Slice {
					for j := 0; j < valField.Len(); j++ {
						if hasConvType {
							docField.Set(reflect.Append(docField, valField.Index(j).Convert(convType)))
						} else {
							docField.Set(reflect.Append(docField, valField.Index(j).Convert(sliceValTyp)))
						}

					}
				}
			} else {
				if hasConvType {
					docField.Set(valField.Convert(convType))
				} else {
					docField.Set(valField.Convert(docField.Type()))
				}
			}
		}
	}

	if _, err := index.Put(ctx, id, doc.Interface()); err != nil {
		return err
	}

	return nil
}

func Load(d interface{}, fields []search.Field, meta *search.DocumentMetadata) error {
	ps := reflect.ValueOf(d).Elem()

	// search fields can have field names defined differently
	// todo: move that to initialization to save time
	var mOfFields = map[string]reflect.Value{}
	for i := 0; i < ps.Type().NumField(); i++ {
		f := ps.Type().Field(i)
		var name = f.Name
		if tag, ok := f.Tag.Lookup("search"); ok {
			opts := strings.Split(tag, ",")
			opt := strings.TrimSpace(opts[0])
			if len(opt) > 0 {
				name = opt
			}
		}
		mOfFields[name] = ps.FieldByName(f.Name)
	}

	for _, field := range fields {

		if f, ok := mOfFields[field.Name]; ok {
			if f.IsValid() && f.CanSet() {
				if f.Kind() == reflect.Slice {
					f.Set(reflect.Append(f, reflect.ValueOf(field.Value)))
				} else {
					f.Set(reflect.ValueOf(field.Value))
				}
			} else {
				return errors.New("no valid field " + field.Name)
			}
		}
	}
	for _, facet := range meta.Facets {
		if f, ok := mOfFields[facet.Name]; ok {
			if f.IsValid() && f.CanSet() {
				if f.Kind() == reflect.Slice {
					f.Set(reflect.Append(f, reflect.ValueOf(facet.Value)))
				} else {
					f.Set(reflect.ValueOf(facet.Value))
				}
			}
		}
	}
	return nil
}

func Save(d interface{}) ([]search.Field, *search.DocumentMetadata, error) {
	var fields []search.Field
	var facets []search.Facet

	ps := reflect.ValueOf(d).Elem()

	for i := 0; i < ps.NumField(); i++ {
		f := ps.Field(i)
		t := ps.Type().Field(i)

		if f.IsValid() {
			if f.Kind() == reflect.Slice {
			sl:
				for j := 0; j < f.Len(); j++ {
					var name = t.Name

					// do facet
					if srch, ok := t.Tag.Lookup("search"); ok {
						srchs := strings.Split(srch, ",")
						if len(srchs[0]) > 0 {
							name = srchs[0]
						}

						if len(srchs) >= 2 && strings.TrimSpace(srchs[1]) == "facet" {
							facets = append(facets, search.Facet{Name: name, Value: f.Index(j).Interface()})
							continue sl
						}
					}

					// do field instead
					fields = append(fields, search.Field{Name: name, Value: f.Index(j).Interface()})
				}
			} else {
				var name = t.Name

				// do facet
				if srch, ok := t.Tag.Lookup("search"); ok {
					srchs := strings.Split(srch, ",")
					if len(srchs[0]) > 0 {
						name = srchs[0]
					}

					if len(srchs) >= 2 && strings.TrimSpace(srchs[1]) == "facet" {
						facets = append(facets, search.Facet{Name: name, Value: f.Interface()})
						continue
					}
				}

				// do field instead
				fields = append(fields, search.Field{Name: name, Value: f.Interface()})
			}
		}
	}

	return fields, &search.DocumentMetadata{
		Facets: facets,
	}, nil
}

func PutToIndex(ctx context.Context, indexName string, documentId string, value interface{}) error {
	index, err := search.Open(indexName)
	if err != nil {
		return err
	}
	_, err = index.Put(ctx, documentId, value)
	return err
}

func GetFromIndex(ctx context.Context, indexName string, documentId string, dst interface{}) error {
	index, err := search.Open(indexName)
	if err != nil {
		return err
	}
	return index.Get(ctx, documentId, dst)
}

func OpenIndex(name string) (*search.Index, error) {
	return search.Open(name)
}

func ClearIndex(ctx context.Context, indexName string) error {
	index, err := search.Open(indexName)
	if err != nil {
		return err
	}

	var ids []string
	for t := index.List(ctx, &search.ListOptions{IDsOnly: true}); ; {
		var doc interface{}
		id, err := t.Next(&doc)
		if err == search.Done {
			break
		}
		if err != nil {
			return err
		}
		ids = append(ids, id)
	}

	return index.DeleteMulti(ctx, ids)
}
