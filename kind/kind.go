package kind

import (
	"errors"
	"github.com/asaskevich/govalidator"
	"github.com/gorilla/mux"
	"golang.org/x/net/context"
	"google.golang.org/appengine/datastore"
	"net/http"
	"reflect"
	"strconv"
	"strings"
)

type Kind struct {
	t                     reflect.Type
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

	http.Handler
	*Options
}

type Options struct {
	// control entry access
	// even if field is false gotta store who created entry? so that is switched to true, creators still have access - if no owner is stored nobody has access
	EnableEntryScope bool
	Name             string
	Type             interface{}
	t                reflect.Type
	KindProvider     *KindProvider
	IsCollection     bool
}

type Field struct {
	Name     string                                                 // real field name
	Fields   map[string]*Field                                      // map key is json representation for field name
	retrieve func(value reflect.Value, path []string) reflect.Value // if *datastore.Key, fetches and returns resource; if array, returns item at index; otherwise returns the value
	Is       string

	IsAutoId bool
}

func New(opt *Options) *Kind {
	if opt == nil {
		panic(errors.New("kind options can't be nil"))
	}

	k := &Kind{
		Options: opt,
		t:       reflect.TypeOf(opt.Type),
		dsNameGenerator: func(ctx context.Context, holder *Holder) string {
			return ""
		},
	}

	if len(opt.Name) == 0 || !govalidator.IsAlphanumeric(opt.Name) {
		panic(errors.New("name must be at least one character and can contain only a-Z0-9"))
	}

	k.ScopeFullControl = opt.Name + ".fullcontrol"
	k.ScopeReadOnly = opt.Name + ".readonly"
	k.ScopeReadWrite = opt.Name + ".readwrite"
	k.ScopeDelete = opt.Name + ".delete"

	if k.KindProvider != nil {
		k.KindProvider.RegisterKind(k)
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

func (k *Kind) Get(ctx context.Context, key *datastore.Key) (h *Holder, err error) {
	h = k.NewHolder(key)
	err = datastore.Get(ctx, key, h)
	return h, err
}

func (k *Kind) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	var err error
	var h *Holder
	var path []string
	var key *datastore.Key
	var hasPath, hasKey bool
	if encodedKey, ok := vars["key"]; ok {
		if key, err = datastore.DecodeKey(encodedKey); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		} else {
			hasKey = true
		}
	}
	if _path, ok := vars["path"]; ok {
		path = strings.Split(_path, "/")
		hasPath = len(path) > 0
	}

	ctx := NewContext(r)

	switch r.Method {
	case http.MethodGet:
		if ok := ctx.HasScope(k.ScopeReadOnly, k.ScopeReadWrite, k.ScopeFullControl); !ok {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		if hasKey {
			h, err := k.Get(ctx, key)
			if err != nil {
				ctx.PrintError(w, "not found", http.StatusNotFound, err.Error())
				return
			}

			if hasPath {
				var value interface{}
				h, value, err = h.Get(ctx, path)
				if err != nil {
					ctx.PrintError(w, "not found", http.StatusNotFound, err.Error())
					return
				}
				ctx.Print(w, value, http.StatusOK)
			} else {
				ctx.Print(w, h.GetValue(), http.StatusOK)
			}
		} else {
			// DO QUERY
			var paramPairs []string
			var offset, limit int
			limit = 25 //default
			q := datastore.NewQuery(k.Name)
			var filterMap = map[string]map[string]string{}
			for name, values := range r.URL.Query() {
				switch name {
				case "order":
					v := values[len(values)-1]
					q = q.Order(v)
					paramPairs = append(paramPairs, "order="+v)
				case "limit":
					v := values[len(values)-1]
					l, err := strconv.Atoi(v)
					if err != nil {
						ctx.PrintError(w, err.Error(), http.StatusBadRequest)
						return
					}
					limit = l
					paramPairs = append(paramPairs, "limit="+v)
				case "offset":
					v := values[len(values)-1]
					l, err := strconv.Atoi(v)
					if err != nil {
						ctx.PrintError(w, err.Error(), http.StatusBadRequest)
						return
					}
					offset = l
				default:
					if strings.Split(name, "[")[0] == "filters" {
						fm := getParams(name)
						if len(fm["num"]) > 0 && len(fm["nam"]) > 0 {
							if m, ok := filterMap[fm["num"]]; ok {
								m[fm["nam"]] = values[len(values)-1]
								var filterStr = m["filterStr"]
								var value = m["value"]
								if len(filterStr) > 0 && len(value) > 0 {
									q = q.Filter(filterStr, value)
									paramPairs = append(paramPairs, "filters["+fm["num"]+"][filterStr]="+filterStr)
									paramPairs = append(paramPairs, "filters["+fm["num"]+"][value]="+value)
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

			// set limit
			q = q.Limit(limit)
			// set offset
			q = q.Offset(offset)

			total, err := Count(ctx, k.Name)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			var linkHeader []string
			var out = []interface{}{}
			t := q.Run(ctx)
			for {
				var h = k.NewHolder(nil)
				h.key, err = t.Next(h)
				if err == datastore.Done {
					break
				}
				out = append(out, h.GetValue())
			}

			// pagination links
			count := len(out)
			if (total - offset - count) > 0 {
				// has more items to fetch
				linkHeader = append(linkHeader, "<"+getSchemeAndHost(r)+r.URL.Path+"?"+strings.Join(append(paramPairs, "offset="+strconv.Itoa(offset+count)), "&")+`>; rel="next"`)
				if (total - offset - count - limit) > 0 {
					// next is not last
					linkHeader = append(linkHeader, "<"+getSchemeAndHost(r)+r.URL.Path+"?"+strings.Join(append(paramPairs, "offset="+strconv.Itoa(total-limit)), "&")+`>; rel="last"`)
				}
			}
			if offset > 0 {
				// get previous link
				linkHeader = append(linkHeader, "<"+getSchemeAndHost(r)+r.URL.Path+"?"+strings.Join(append(paramPairs, "offset="+strconv.Itoa(offset-limit)), "&")+`>; rel="prev"`)
				if offset-limit > 0 {
					// previous is not first
					linkHeader = append(linkHeader, "<"+getSchemeAndHost(r)+r.URL.Path+"?"+strings.Join(append(paramPairs, "offset=0"), "&")+`>; rel="first"`)
				}
			}

			ctx.Print(w, out, http.StatusOK, "X-Total-Count", strconv.Itoa(total), "Link", strings.Join(linkHeader, ","))
		}
	case http.MethodPost:
		if ok := ctx.HasScope(k.ScopeReadWrite, k.ScopeFullControl); !ok {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		if hasPath || hasKey {
			http.Error(w, "not implemented", http.StatusNotImplemented)
			return
		}

		h = k.NewHolder(nil)
		if err := h.Parse(ctx.Body()); err != nil {
			ctx.PrintError(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var name = k.dsNameGenerator(ctx, h)
		h.key = datastore.NewKey(ctx, k.Name, name, 0, nil)

		if h.key.Incomplete() {
			h.key, err = datastore.Put(ctx, h.key, h)
		} else {
			err = datastore.RunInTransaction(ctx, func(tc context.Context) error {
				if _, err := k.Get(tc, h.key); err != nil {
					if err == datastore.ErrNoSuchEntity {
						h.key, err = datastore.Put(tc, h.key, h)
						return err
					}
					return err
				}
				return errors.New("entry already exists")
			}, nil)
		}

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

		_ = Increment(ctx, k.Name)

		var location string
		locationUrl, err := mux.CurrentRoute(r).URL()
		if err == nil {
			location = strings.Join(append(strings.Split(locationUrl.Path, "/"), h.key.Encode()), "/")
		}

		ctx.Print(w, h.GetValue(), http.StatusCreated, "Location", location)
	case http.MethodPut:
		if ok := ctx.HasScope(k.ScopeReadWrite, k.ScopeFullControl); !ok {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		if hasKey {
			h, err = k.Get(ctx, key)
			if err != nil {
				ctx.PrintError(w, "not found", http.StatusNotFound, err.Error())
				return
			}
			h, err = h.Set(ctx, path, ctx.Body())
			if err != nil {
				ctx.PrintError(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if h.key, err = datastore.Put(ctx, key, h); err != nil {
				ctx.PrintError(w, err.Error(), http.StatusInternalServerError)
				return
			}
			ctx.Print(w, h.GetValue(), http.StatusOK)
		} else {
			r.Method = http.MethodPost
			k.ServeHTTP(w, r)
			return
		}
	case http.MethodDelete:
		if ok := ctx.HasScope(k.ScopeDelete, k.ScopeFullControl); !ok {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		if hasKey {
			if hasPath {
				if h, err = k.Get(ctx, key); err != nil {
					ctx.PrintError(w, "not found", http.StatusNotFound, err.Error())
					return
				}

				h, err = h.Delete(ctx, path)
				if err != nil {
					ctx.PrintError(w, err.Error(), http.StatusInternalServerError)
					return
				}

				_, err = datastore.Put(ctx, h.key, h)
				if err != nil {
					ctx.PrintError(w, err.Error(), http.StatusInternalServerError)
					return
				}

				ctx.Print(w, h.GetValue(), http.StatusOK)
			} else {
				if err = datastore.Delete(ctx, key); err != nil {
					ctx.PrintError(w, "not found", http.StatusNotFound, err.Error())
					return
				}

				_ = Decrement(ctx, k.Name)

				ctx.Print(w, "ok", http.StatusOK)
			}
		} else {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
	}
}

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

func (k *Kind) Type() reflect.Type {
	return k.t
}

func (k *Kind) New() reflect.Value {
	return reflect.New(k.t)
}

func (k *Kind) NewHolder(key *datastore.Key) *Holder {
	return &Holder{
		Kind:         k,
		reflectValue: k.New(),
		key:          key,
	}
}
