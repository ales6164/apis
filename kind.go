package apis

import (
	"fmt"
	"github.com/ales6164/apis/errors"
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

func (k *Kind) Handler() *Handler {
	return &Handler{
		Kind: k,
	}
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

// todo: implement {userId}
func (k *Kind) get(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	var err error
	ctx := NewContext(r)

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

// za post je ancestor in userId sign, da v key vključim ancestor
func (k *Kind) post(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ctx := NewContext(r)
	_, ancestor, err := getKeysFromRequest(ctx, vars)
	if err != nil {
		ctx.PrintError(w, err, "error retrieving keys")
		return
	}

	h := k.NewHolder()

	if ancestor != nil {
		h.SetKey(k.NewIncompleteKey(ctx, ancestor))
	}

	err = h.Parse(ctx.Body())
	if err != nil {
		ctx.PrintError(w, err, "error parsing")
		return
	}

	err = h.Add(ctx, h.key)
	if err != nil {
		ctx.PrintError(w, err, "error adding")
		return
	}

	ctx.Print(w, h.Value())
}

func (k *Kind) put(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	var err error

	ctx := NewContext(r)

	var ancestorKey *datastore.Key
	if ancestor, ok := vars["ancestor"]; ok {
		ancestorKey, err = datastore.DecodeKey(ancestor)
		if err != nil {
			ctx.PrintError(w, errors.ErrDecodingKey)
			return
		}
	}
	if uid, ok := vars["userId"]; ok {
		ancestorKey, _ = datastore.DecodeKey(uid)
		if !ctx.Session.User.Equal(ancestorKey) {
			http.Error(w, "forbidden", http.StatusForbidden)
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

		h := k.NewHolder()
		err = h.Parse(ctx.Body())
		if err != nil {
			ctx.PrintError(w, err)
			return
		}
		h.SetKey(idKey)

		err = h.Update(ctx)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		ctx.Print(w, h.Value())
	} else {
		ctx.PrintError(w, errors.ErrIdRequired)
	}
}

func (k *Kind) delete(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	var err error

	ctx := NewContext(r)

	id, ancestor, err := getKeysFromRequest(ctx, vars)
	if err != nil {
		ctx.PrintError(w, errors.ErrDecodingKey)
		return
	}

	// if both id and ancestor are present, check their hierarchy
	if ancestor != nil && id != nil && !id.Parent().Equal(ancestor) {
		ctx.PrintError(w, errors.ErrNoHierarchy)
		return
	}

	var count int

	// delete ancestor children
	if ancestor != nil {
		// delete all or nothing (if error occurs)
		err = datastore.RunInTransaction(ctx, func(tc context.Context) error {
			q := datastore.NewQuery(k.Name).Ancestor(ancestor).KeysOnly()
			t := q.Run(ctx)
			var keys []*datastore.Key
			for {
				key, err := t.Next(nil)
				if err == datastore.Done {
					if len(keys) > 0 {
						err := datastore.DeleteMulti(ctx, keys)
						if err != nil {
							return err
						}
						count += len(keys)
					}
					break
				}
				keys = append(keys, key)
				if len(keys) >= 1000 {
					err := datastore.DeleteMulti(ctx, keys)
					if err != nil {
						return err
					}
					count += len(keys)
					keys = []*datastore.Key{}
				}
			}
			return nil
		}, &datastore.TransactionOptions{XG: true})
		if err != nil {
			ctx.PrintError(w, err)
			return
		}
	}

	// delete self
	if id != nil {
		err = k.Delete(ctx, id)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}
		count++
	}

	ctx.Print(w, map[string]interface{}{
		"count":   count,
		"message": fmt.Sprintf("deleted %d entries", count),
	})
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
			if err == datastore.Done {
				break
			}
			return h, err
		}
		return h, datastore.ErrNoSuchEntity
	}
	return h, err
}

func getKeysFromRequest(ctx Context, vars map[string]string) (id *datastore.Key, ancestor *datastore.Key, err error) {
	if encodedAncestorKey, ok := vars["ancestor"]; ok {
		ancestor, err = datastore.DecodeKey(encodedAncestorKey)
		if err != nil {
			return id, ancestor, errors.ErrDecodingKey
		}
	}
	if encodedUserIdKey, ok := vars["userId"]; ok {
		ancestor, err = datastore.DecodeKey(encodedUserIdKey)
		if err != nil {
			return id, ancestor, errors.ErrDecodingKey
		}
		if !ctx.Session.User.Equal(ancestor) {
			return id, ancestor, errors.ErrForbidden
		}
	}
	if encodedIdKey, ok := vars["id"]; ok {
		// got encoded key
		id, err = datastore.DecodeKey(encodedIdKey)
		if err != nil {
			return id, ancestor, errors.ErrDecodingKey
		}
	}

	return id, ancestor, nil
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
