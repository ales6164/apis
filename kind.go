package apis

import (
	"github.com/ales6164/apis/errors"
	gContext "github.com/gorilla/context"
	"github.com/gorilla/mux"
	"golang.org/x/net/context"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/search"
	"net/http"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

type Kind struct {
	t           reflect.Type
	MetaFields  []MetaField
	MetaIdField MetaField
	fields      []*Field
	Name        string

	searchFields map[string]*SearchField // map of all fields

	//todo: above searchFields - add additional fields and below mentioned functions for use in global search
	// map[fieldName] or array of fields with appropriate convertion functions (pointer to a function)
	// do we need to convert from search value back to original value when outputing??? - probably should? - this could mean trouble later on... maybe just fetch original datastore entries
	// for output uses field json string

	// add some meta tag to db entry to now when it was last synced with search?
	ScopeFullControl string
	ScopeReadOnly    string
	ScopeReadWrite   string

	/*router *mux.Router*/
	http.Handler
}

type MetaField struct {
	Type      string
	FieldName string
}

type Field struct {
	Name       string
	DoStore    bool
	IsRequired bool // moving this somewhere else?
	Multiple   bool
	NoIndex    bool

	// search tag
	SearchField *SearchField

	MetaField   string
	Label       string // json field name
	Json        string // json field name
	Type        string
	StructField reflect.StructField

	Kind *Kind
}

type SearchField struct {
	Field           *Field
	Enabled         bool
	Multiple        bool
	FieldName       string
	SearchFieldName string
	IsFacet         bool
	ConvertType     reflect.Type
	Converter       Converter
}

func NewKind(t reflect.Type) *Kind {
	k := &Kind{
		searchFields: map[string]*SearchField{},
	}

	err := k.SetType(t)
	if err != nil {
		panic(err)
	}

	k.ScopeFullControl = k.Name + ".fullcontrol"
	k.ScopeReadOnly = k.Name + ".readonly"
	k.ScopeReadWrite = k.Name + ".readwrite"

	return k
}

func (k *Kind) SetType(t reflect.Type) error {
	k.t = t

	if t == nil {
		return errors.New("type not of kind struct")
	}

	if t.Kind() != reflect.Struct {
		return errors.New("type not of kind struct")
	}

	err := k.checkFields()
	if err != nil {
		return err
	}

	if len(k.Name) == 0 {
		k.Name = t.Name()
	}

	// build search parser
	/*if k.EnableSearch {
	searchField:
		for _, field := range k.fields {
			if !field.SearchField.Enabled {
				continue searchField
			}

			// 1. Set convert type from search tag or try to find a good converter
			if field.SearchField.ConvertType == nil {
				var fieldType reflect.Type
				if field.Multiple {
					// get slice type
					fieldType = reflect.MakeSlice(field.StructField.Type, 1, 1).Index(0).Type()
				} else {
					fieldType = field.StructField.Type
				}

				if fieldType == nil {
					return errors.New("error reflecting type for " + field.Name)
				}

				// convert type differs for search.Field and search.Facet
				if field.SearchField.IsFacet {
					switch fieldType {
					case stringType:
						field.SearchField.ConvertType = atomType
					case intType, int8Type, int16Type, int32Type, int64Type, float32Type:
						field.SearchField.ConvertType = float64Type
					case keyType:
						field.SearchField.ConvertType = keyType
					case boolType:
						field.SearchField.ConvertType = boolType
					default:
						if fieldType.ConvertibleTo(atomType) {
							field.SearchField.ConvertType = atomType
						} else if fieldType.ConvertibleTo(float64Type) {
							field.SearchField.ConvertType = float64Type
						} else {
							return errors.New("no convert type specified for searchable facet " + field.Name)
						}
					}
				} else {
					switch fieldType {
					case stringType, atomType, htmlType, float64Type, timeType, geoPointType:
						// ok
					case intType, int8Type, int16Type, int32Type, int64Type, float32Type:
						field.SearchField.ConvertType = float64Type
					case keyType:
						field.SearchField.ConvertType = keyType
					case boolType:
						field.SearchField.ConvertType = boolType
					default:
						return errors.New("no convert type specified for searchable field " + field.Name)
					}
				}
			}

			var converter Converter
			if field.SearchField.ConvertType == nil {
				converter = new(EmptyConverter)
			} else if field.SearchField.IsFacet {
				switch field.SearchField.ConvertType {
				case atomType:
					converter = new(AtomConverter)
				case float64Type:
					converter = new(Float64Converter)
				case keyType:
					converter = new(KeyConverter)
				case boolType:
					converter = new(BoolConverter)
				default:
					return errors.New("invalid convert type for searchable facet " + field.Name)
				}
			} else {
				switch field.SearchField.ConvertType {
				case atomType:
					converter = new(AtomConverter)
				case float64Type:
					converter = new(Float64Converter)
				case stringType:
					converter = new(StringConverter)
				case htmlType:
					converter = new(HTMLConverter)
				case keyType:
					converter = new(KeyConverter)
				case boolType:
					converter = new(BoolConverter)
				default:
					return errors.New("invalid convert type for searchable field " + field.Name)
				}
			}

			field.SearchField.Converter = converter

			k.searchFields[field.SearchField.SearchFieldName] = field.SearchField
		}
	}*/

	return nil
}

func (k *Kind) Type() reflect.Type {
	return k.t
}

func (k *Kind) SearchFields() map[string]*SearchField {
	return k.searchFields
}

func (k *Kind) checkFields() error {
	k.fields = []*Field{}
	//var hasId, hasCreatedAt bool
	for i := 0; i < k.t.NumField(); i++ {
		structField := k.t.Field(i)
		field := new(Field)
		field.SearchField = &SearchField{
			Field:           field,
			FieldName:       structField.Name,
			SearchFieldName: structField.Name,
		}
		field.StructField = structField
		field.Type = structField.Type.String()
		if val, ok := structField.Tag.Lookup("datastore"); ok {
			for n, v := range strings.Split(val, ",") {
				v = strings.TrimSpace(v)
				switch n {
				case 0:
					if v == "-" {
						field.DoStore = false
					} else {
						field.DoStore = true
					}
					field.Name = v
				case 1:
					field.NoIndex = v == "noindex"
				}
			}
		} else {
			field.DoStore = true
			field.Name = structField.Name
		}
		if val, ok := structField.Tag.Lookup("json"); ok {
			for n, v := range strings.Split(val, ",") {
				v = strings.TrimSpace(v)
				switch n {
				case 0:
					if v != "-" {
						field.Json = v
					}
				}
			}
		} else {
			field.Json = field.Name
		}
		if val, ok := structField.Tag.Lookup("label"); ok {
			for n, v := range strings.Split(val, ",") {
				v = strings.TrimSpace(v)
				switch n {
				case 0:
					field.Label = v
				}
			}
		} else {
			field.Label = structField.Name
		}
		if val, ok := structField.Tag.Lookup("apis"); ok {
			for n, v := range strings.Split(val, ",") {
				v = strings.TrimSpace(v)
				v = strings.ToLower(v)
				switch n {
				case 0:
					if v == "id" {
						//hasId = true
						k.MetaIdField = MetaField{
							Type:      v,
							FieldName: structField.Name,
						}
					} else {
						if v == "createdat" {
							//hasCreatedAt = true
						}
						k.MetaFields = append(k.MetaFields, MetaField{
							Type:      v,
							FieldName: structField.Name,
						})
					}
					field.MetaField = v

				}
			}
		}
		if val, ok := structField.Tag.Lookup("search"); ok {
			for n, v := range strings.Split(val, ",") {
				v = strings.TrimSpace(v)
				switch n {
				case 0:
					if len(v) > 0 && v != "-" {
						field.SearchField.Enabled = true
						field.SearchField.SearchFieldName = v
					}
				case 1:
					field.SearchField.IsFacet = v == "facet"
				case 2:
					switch strings.TrimSpace(v) {
					case "search.Atom":
						field.SearchField.ConvertType = atomType
					case "search.HTML":
						field.SearchField.ConvertType = htmlType
					case "float64":
						field.SearchField.ConvertType = float64Type
					case "string":
						field.SearchField.ConvertType = stringType
					case "time.Time":
						field.SearchField.ConvertType = timeType
					}
				}
			}
		} else {
			field.SearchField.Enabled = field.DoStore
		}

		if structField.Type.Kind() == reflect.Slice {
			field.Multiple = true
			field.SearchField.Multiple = true
		}
		k.fields = append(k.fields, field)
	}
	/*if !(hasId && hasCreatedAt) {
		return errors.New("kind " + k.Name + " requires id and createdAt fields")
	}*/
	return nil
}

func (k *Kind) New() interface{} {
	return reflect.New(k.t).Interface()
}

func (k *Kind) DeleteFromIndex(ctx context.Context, id string) error {
	index, err := search.Open(k.Name)
	if err != nil {
		return err
	}
	return index.Delete(ctx, id)
}

func (k *Kind) NewHolder() *Holder {
	return &Holder{
		Kind:  k,
		value: k.New(),
	}
}

func (k *Kind) NewIncompleteKey(c context.Context, parent *datastore.Key) *datastore.Key {
	return datastore.NewIncompleteKey(c, k.Name, parent)
}

func (k *Kind) NewKey(c context.Context, nameId string, parent *datastore.Key) *datastore.Key {
	return datastore.NewKey(c, k.Name, nameId, 0, parent)
}

func (k *Kind) RetrieveSearchParameter(parameterName string, value string, fields []search.Field, facets []search.Facet) ([]search.Field, []search.Facet) {
	if f, ok := k.searchFields[parameterName]; ok {
		if f.IsFacet {
			// todo: currently only supports facet type search.Atom
			facets = append(facets, search.Facet{Name: parameterName, Value: search.Atom(value)})
		} else {
			fields = append(fields, search.Field{Name: parameterName, Value: value})
		}
	}
	return fields, facets
}

// todo: implement {userId} for all methods
func (k *Kind) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		k.get(w, r)
	case http.MethodPost:
		k.post(w, r)
	}
}

// todo: implement {userId}
func (k *Kind) get(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	var err error
	var ctx Context
	var ok bool
	if ctx, ok = gContext.Get(r, "context").(Context); !ok {
		ctx = NewContext(r)
	}

	var ancestorKey *datastore.Key
	if ancestor, ok := vars["ancestor"]; ok {
		ancestorKey, err = datastore.DecodeKey(ancestor)
		if err != nil {
			ctx.PrintError(w, errors.ErrDecodingKey)
			return
		}
	}
	if id, ok := vars["id"]; ok {
		// got encoded key
		idKey, err := datastore.DecodeKey(id)
		if err != nil {
			ctx.PrintError(w, errors.ErrDecodingKey)
			return
		}

		h, err := k.Get(ctx, idKey, ancestorKey)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		ctx.Print(w, h.Value())
	} else {
		// no id - query instead
		var hs []*Holder
		var ids []*datastore.Key
		var order, limit, offset string
		var filters []Filter
		var filterMap = map[string]map[string]string{}

		for name, values := range r.URL.Query() {
			switch name {
			case "id":
				for _, v := range values {
					idKey, err := datastore.DecodeKey(v)
					if err != nil {
						ctx.PrintError(w, errors.ErrDecodingKey)
						return
					}
					ids = append(ids, idKey)
					h := k.NewHolder()
					h.SetKey(idKey)
					hs = append(hs, h)
				}
			case "order":
				order = values[len(values)-1]
			case "limit":
				limit = values[len(values)-1]
			case "offset":
				offset = values[len(values)-1]
			default:
				if strings.Split(name, "[")[0] == "filters" {
					fm := getParams(name)
					if len(fm["num"]) > 0 && len(fm["nam"]) > 0 {
						if m, ok := filterMap[fm["num"]]; ok {
							m[fm["nam"]] = values[len(values)-1]
							var filterStr = m["filterStr"]
							var value = m["value"]
							if len(filterStr) > 0 && len(value) > 0 {
								filters = append(filters, Filter{
									FilterStr: filterStr,
									Value:     value,
								})
							}
						} else {
							filterMap[fm["num"]] = map[string]string{
								fm["nam"]: values[len(values)-1],
							}
						}
					}
				}
			}
		}

		if len(ids) > 0 {
			if len(order) > 0 {
				ctx.PrintError(w, errors.ErrOrderUnavailableWithIdParam)
				return
			} else if len(limit) > 0 {
				ctx.PrintError(w, errors.ErrLimitUnavailableWithIdParam)
				return
			} else if len(offset) > 0 {
				ctx.PrintError(w, errors.ErrOffsetUnavailableWithIdParam)
				return
			} else if len(filters) > 0 {
				ctx.PrintError(w, errors.ErrFiltersUnavailableWithIdParam)
				return
			}

			err := datastore.GetMulti(ctx, ids, hs)
			if err != nil {
				ctx.PrintError(w, err)
				return
			}

			var out []interface{}
			for _, h := range hs {
				out = append(out, h.Value())
			}

			ctx.PrintResult(w, map[string]interface{}{
				"count":   len(out),
				"results": out,
			})
		} else {
			q := datastore.NewQuery(k.Name)

			if ancestorKey != nil {
				q = q.Ancestor(ancestorKey)
			}

			if len(order) > 0 {
				q = q.Order(order)
			}

			if len(limit) > 0 {
				l, err := strconv.Atoi(limit)
				if err != nil {
					ctx.PrintError(w, err)
					return
				}
				q = q.Limit(l)
			}

			if len(offset) > 0 {
				l, err := strconv.Atoi(offset)
				if err != nil {
					ctx.PrintError(w, err)
					return
				}
				q = q.Offset(l)
			}

			for _, f := range filters {
				q = q.Filter(f.FilterStr, f.Value)
			}

			var out []interface{}
			t := q.Run(ctx)
			for {
				var h = k.NewHolder()
				h.key, err = t.Next(h)
				h.hasKey = true
				if err == datastore.Done {
					break
				}
				out = append(out, h.Value())
			}
			ctx.PrintResult(w, map[string]interface{}{
				"count":   len(out),
				"results": out,
			})
		}
	}
}

func (k *Kind) post(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	var err error
	var ctx Context
	var ok bool
	if ctx, ok = gContext.Get(r, "context").(Context); !ok {
		ctx = NewContext(r)
	}

	var userId *datastore.Key
	var hasUserId bool
	if uid, ok := vars["userId"]; ok {
		userId, _ = datastore.DecodeKey(uid)
		if !ctx.Session.User.Equal(userId) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		hasUserId = true
	}

	h := k.NewHolder()

	if hasUserId {
		h.SetAncestor(userId)
	}

	err = h.Parse(ctx.Body())
	if err != nil {
		ctx.PrintError(w, err, "error parsing")
		return
	}

	err = h.Add(ctx)
	if err != nil {
		ctx.PrintError(w, err, "error adding")
		return
	}

	ctx.Print(w, h.Value())
}

func (k *Kind) Get(ctx context.Context, key *datastore.Key, ancestor *datastore.Key) (h *Holder, err error) {
	h = k.NewHolder()
	if ancestor == nil {
		err = h.Get(ctx, key)
	} else {
		// todo: check if it works
		q := datastore.NewQuery(k.Name).Ancestor(ancestor).Filter("__key__ =", key).Limit(1)
		t := q.Run(ctx)
		for {
			var h = k.NewHolder()
			h.key, err = t.Next(h)
			h.hasKey = true
			if err == datastore.Done {
				break
			}
			return h, err
		}
		return h, datastore.ErrNoSuchEntity
	}
	return h, err
}

var queryFilters = regexp.MustCompile(`(?m)filters\[(?P<num>[^\]]+)\]\[(?P<nam>[^\]]+)\]`)

func getParams(url string) (paramsMap map[string]string) {
	match := queryFilters.FindStringSubmatch(url)
	paramsMap = make(map[string]string)
	for i, name := range queryFilters.SubexpNames() {
		if i > 0 && i <= len(match) {
			paramsMap[name] = match[i]
		}
	}
	return
}
