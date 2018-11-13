package apis

import (
	"github.com/ales6164/apis/errors"
	"github.com/asaskevich/govalidator"
	"golang.org/x/net/context"
	"google.golang.org/appengine/datastore"
	"reflect"
	"strings"
)

type Kind struct {
	i    interface{}
	t    reflect.Type
	name string

	hasIdFieldName        bool
	hasCreatedAtFieldName bool
	hasUpdatedAtFieldName bool
	hasVersionFieldName   bool
	hasCreatedByFieldName bool
	hasUpdatedByFieldName bool

	idFieldName        string
	createdAtFieldName string
	updatedAtFieldName string
	versionFieldName   string
	createdByFieldName string
	updatedByFieldName string

	ScopeFullControl string
	ScopeReadOnly    string
	ScopeReadWrite   string
	ScopeDelete      string

	dsUseName       bool // default: false; if true, changes the way datastore Keys are generated
	dsNameGenerator func(ctx context.Context, holder *Holder) string

	fields map[string]*Field // map key is json representation for field name
}

type Field struct {
	Name     string                                                 // real field name
	Fields   map[string]*Field                                      // map key is json representation for field name
	retrieve func(value reflect.Value, path []string) reflect.Value // if *datastore.Key, fetches and returns resource; if array, returns item at index; otherwise returns the value
	Is       string

	IsAutoId bool
}

func NewKind(name string, i interface{}) *Kind {
	k := &Kind{
		i:    i,
		t:    reflect.TypeOf(i),
		name: name,
		dsNameGenerator: func(ctx context.Context, holder *Holder) string {
			return ""
		},
	}

	k.ScopeFullControl = name + ".fullcontrol"
	k.ScopeReadOnly = name + ".readonly"
	k.ScopeReadWrite = name + ".readwrite"
	k.ScopeDelete= name + ".delete"

	if len(name) == 0 || !govalidator.IsAlphanumeric(name) {
		panic(errors.New("name must be at least one character and can contain only a-Z0-9"))
	}

	if k.t == nil || k.t.Kind() != reflect.Struct {
		panic(errors.New("type not of kind struct"))
	}

	k.fields = Lookup(k, k.t, map[string]*Field{})

	return k
}

var (
	keyKind = reflect.TypeOf(&datastore.Key{}).Kind()
)

const (
	id        = "id"
	createdat = "createdat"
	updatedat = "updatedat"
	version   = "version"
	createdby = "createdby"
	updatedby = "updatedby"
)

func Lookup(kind *Kind, typ reflect.Type, fields map[string]*Field) map[string]*Field {

loop:
	for i := 0; i < typ.NumField(); i++ {
		var isAutoId bool
		structField := typ.Field(i)
		var jsonName = structField.Name

		if kind != nil {
			if autoValue, ok := structField.Tag.Lookup("auto"); ok {
				autoValue = strings.ToLower(autoValue)
				switch autoValue {
				case id:
					kind.idFieldName = structField.Name
					kind.hasIdFieldName = true
					isAutoId = true
				case createdat:
					kind.createdAtFieldName = structField.Name
					kind.hasCreatedAtFieldName = true
				case updatedat:
					kind.updatedAtFieldName = structField.Name
					kind.hasUpdatedAtFieldName = true
				case version:
					kind.versionFieldName = structField.Name
					kind.hasVersionFieldName = true
				case createdby:
					kind.createdByFieldName = structField.Name
					kind.hasCreatedByFieldName = true
				case updatedby:
					kind.updatedByFieldName = structField.Name
					kind.hasUpdatedByFieldName = true
				}
			}
		}

		if val, ok := structField.Tag.Lookup("json"); ok {
			for n, v := range strings.Split(val, ",") {
				v = strings.TrimSpace(v)
				switch n {
				case 0:
					if v == "-" {
						continue loop
					}
					jsonName = v
				}
			}
		}

		var fun func(value reflect.Value, path []string) reflect.Value
		var is string

		switch structField.Type.Kind() {
		case keyKind:
			// receives *datastore.Key as value
			// type struct that is fetched then with this value should be "registered" kind and somehow mapped to the api
			// so that the value query can continue onwards
			fun = func(value reflect.Value, path []string) reflect.Value {
				return value
			}
			is = "*datastore.Key{}"
		case reflect.Slice:
			// receives slice as value
			fun = func(value reflect.Value, path []string) reflect.Value {
				return value
			}
			is = "slice"
		default:
			fun = func(value reflect.Value, path []string) reflect.Value {
				for _, jsonName := range path {
					if field, ok := fields[jsonName]; ok {
						if value.Kind() == reflect.Ptr {
							value = value.Elem().FieldByName(field.Name)
						} else {
							value = value.FieldByName(field.Name)
						}
						return field.retrieve(value, path[1:])
					} else {

					}
					// error
				}
				return value
			}
			is = "default"
		}

		var childFields map[string]*Field

		if structField.Type.Kind() == reflect.Struct {
			childFields = Lookup(nil, structField.Type, map[string]*Field{})
		}

		fields[jsonName] = &Field{
			Fields:   childFields,
			Name:     structField.Name,
			retrieve: fun,
			Is:       is,
			IsAutoId: isAutoId,
		}
	}

	return fields
}

/*
HTTP GET http://www.appdomain.com/users
HTTP GET http://www.appdomain.com/users?size=20&page=5
HTTP GET http://www.appdomain.com/users/123
HTTP GET http://www.appdomain.com/users/123/address
 */

func (k *Kind) Name() string {
	return k.name
}

func (k *Kind) Type() reflect.Type {
	return k.t
}

func (k *Kind) New() interface{} {
	return reflect.New(k.t).Interface()
}

func (k *Kind) NewHolder(key *datastore.Key) *Holder {
	return &Holder{
		Kind:  k,
		value: k.New(),
		key:   key,
	}
}

/*func (k *Kind) get(w http.ResponseWriter, r *http.Request) {
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
*/
