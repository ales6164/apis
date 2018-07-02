package apis

import (
	"golang.org/x/net/context"
	"google.golang.org/appengine/search"
	"reflect"
	"strings"
	"github.com/ales6164/apis/errors"
	"github.com/ales6164/apis/kind"
	"net/http"
	"strconv"
	"math"
	"github.com/gorilla/mux"
	"google.golang.org/appengine/datastore"
)

func (a *Apis) searchHandler() http.HandlerFunc {
	R := &Route{
		a:       a,
		methods: []string{},
	}
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := a.newContext(r, R)

		kindName := mux.Vars(r)["kind"]
		k := a.kinds[kindName]

		if k == nil || !k.EnableSearch {
			ctx.PrintError(w, errors.ErrPageNotFound)
			return
		}

		var ok, isPrivate bool
		if ok, isPrivate = ctx.HasPermission(k, QUERY); !ok {
			ctx.PrintError(w, errors.ErrForbidden)
			return
		}

		q, next, autoFilterDiscovery, sort, limit, offset := r.FormValue("q"), r.FormValue("next"), r.FormValue("autoFilterDiscovery"), r.FormValue("sort"), r.FormValue("limit"), r.FormValue("offset")

		index, err := OpenIndex(kindName)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		// build facets and retrieve filters from query parameters
		var fields []search.Field
		var facets []search.Facet
		for key, val := range r.URL.Query() {
			if key == "filter" {
				for _, v := range val {
					filter := strings.Split(v, ":")
					if len(filter) == 2 {
						// todo: currently only supports facet type search.Atom
						facets = append(facets, search.Facet{Name: filter[0], Value: search.Atom(filter[1])})
					}
				}
			} else if key == "range" {
				for _, v := range val {
					filter := strings.Split(v, ":")
					if len(filter) == 2 {

						rangeStr := strings.Split(filter[1], "-")
						if len(rangeStr) == 2 {
							rangeStart, _ := strconv.ParseFloat(rangeStr[0], 64)
							rangeEnd, _ := strconv.ParseFloat(rangeStr[1], 64)

							facets = append(facets, search.Facet{Name: filter[0], Value: search.Range{
								Start: rangeStart,
								End:   rangeEnd,
							}})
						}
					}
				}
			} else if key == "sort" {
				//skip
			} else if key == "key" {
				// used for auth
				//skip
			} else {
				for _, v := range val {
					fields, facets = k.RetrieveSearchParameter(key, v, fields, facets)
				}
			}
		}

		// build []search.Field to a query string and append
		if len(fields) > 0 {
			for _, f := range fields {
				if len(q) > 0 {
					q += " AND " + f.Name + ":" + f.Value.(string)
				} else {
					q += f.Name + ":" + f.Value.(string)
				}
			}
		}

		// we need this to retrieve possible facets/filters
		var facsOutput = map[string][]FacetOutput{}
		if len(autoFilterDiscovery) > 0 {
			var itDiscovery = index.Search(ctx, q, &search.SearchOptions{
				IDsOnly: k.RetrieveByIDOnSearch,
				Facets: []search.FacetSearchOption{
					search.AutoFacetDiscovery(0, 0),
				},
			})

			facetsResult, _ := itDiscovery.Facets()
			for _, f := range facetsResult {
				for _, v := range f {
					if _, ok := facsOutput[v.Name]; !ok {
						facsOutput[v.Name] = []FacetOutput{}
					}
					if rang, ok := v.Value.(search.Range); ok {
						var value interface{}
						if rang.Start == math.Inf(-1) || rang.End == math.Inf(1) {
							value = "Inf"
						} else {
							value = map[string]interface{}{
								"start": rang.Start,
								"end":   rang.End,
							}
						}
						facsOutput[v.Name] = append(facsOutput[v.Name], FacetOutput{
							Count: v.Count,
							Value: value,
							Name:  v.Name,
						})
					} else if rang, ok := v.Value.(search.Atom); ok {
						facsOutput[v.Name] = append(facsOutput[v.Name], FacetOutput{
							Count: v.Count,
							Value: string(rang),
							Name:  v.Name,
						})
					}

				}
			}
		}

		// limit
		var intLimit int
		if len(limit) > 0 {
			intLimit, _ = strconv.Atoi(limit)
		}
		// offset
		var intOffset int
		if len(offset) > 0 {
			intOffset, _ = strconv.Atoi(offset)
		}

		// sorting
		var sortExpr []search.SortExpression
		if len(sort) > 0 {
			var desc bool
			if sort[:1] == "-" {
				sort = sort[1:]
				desc = true
			}
			sortExpr = append(sortExpr, search.SortExpression{Expr: sort, Reverse: !desc})
		}

		// real search
		var results []interface{}
		var docKeys []*datastore.Key
		var t *search.Iterator
		for t = index.Search(ctx, q, &search.SearchOptions{
			IDsOnly:       k.RetrieveByIDOnSearch,
			Refinements:   facets,
			Cursor:        search.Cursor(next),
			CountAccuracy: 1000,
			Offset:        intOffset,
			Limit:         intLimit,
			Sort: &search.SortOptions{
				Expressions: sortExpr,
			}}); ; {
			var doc = reflect.New(k.SearchType).Interface()
			docKey, err := t.Next(doc)
			if err == search.Done {
				break
			}
			if err != nil {
				ctx.PrintError(w, err)
				return
			}

			if key, err := k.DecodeKey(docKey); err == nil {
				if (isPrivate && key.Parent().Equal(ctx.UserKey())) || !isPrivate {
					docKeys = append(docKeys, key)
					results = append(results, doc)
				}
			}
		}

		// fetch real entries from datastore
		if k.RetrieveByIDOnSearch {
			if len(docKeys) == len(results) {
				hs, err := kind.GetMulti(ctx, k, docKeys...)
				if err != nil {
					ctx.PrintError(w, err)
					return
				}
				for k, h := range hs {
					results[k] = h.Value()
				}
			} else {
				ctx.PrintError(w, errors.New("results mismatch"))
				return
			}
		}

		var cursor *Cursor
		if len(t.Cursor()) > 0 || len(next) > 0 {
			cursor = &Cursor{
				Next: string(t.Cursor()),
				Prev: next,
			}
		}

		ctx.Print(w, &SearchOutput{
			Count:   len(results),
			Total:   t.Count(),
			Results: results,
			Filters: facsOutput,
			Cursor:  cursor,
		})
	}
}

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
			if len(vspl) > 2 {
				switch vspl[2] {
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
						} else if docField.Type() == valField.Type() {
							docField.Set(reflect.Append(docField, valField.Index(j)))
						} else {
							docField.Set(reflect.Append(docField, valField.Index(j).Convert(sliceValTyp)))
						}

					}
				}
			} else {
				if hasConvType {
					docField.Set(valField.Convert(convType))
				} else if docField.Type() == valField.Type() {
					docField.Set(valField)
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
