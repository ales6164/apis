package apis

import (
	"gopkg.in/ales6164/apis.v1/errors"
	"gopkg.in/ales6164/apis.v1/kind"
	"github.com/gorilla/mux"
	"google.golang.org/appengine/datastore"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

type Route struct {
	a    *Apis
	kind *kind.Kind
	path string

	listeners      map[string]Listener
	searchListener func(ctx Context, query string) ([]interface{}, error)
	roles          map[string]bool

	methods []string

	get    http.HandlerFunc
	post   http.HandlerFunc
	put    http.HandlerFunc
	delete http.HandlerFunc
}

type Cursor struct {
	Next string `json:"next,omitempty"`
	Prev string `json:"prev,omitempty"`
}

type SearchOutput struct {
	Count   int                      `json:"count"`
	Total   int                      `json:"total"`
	Results []interface{}            `json:"results"`
	Filters map[string][]FacetOutput `json:"filters,omitempty"`
	Cursor  *Cursor                  `json:"cursor,omitempty"`
}

type Listener func(ctx Context, h *kind.Holder) error

const (
	BeforeRead   = "beforeGet"
	BeforeCreate = "beforeCreate"
	BeforeUpdate = "beforeUpdate"
	BeforeDelete = "beforeDelete"

	AfterRead   = "afterRead"
	AfterCreate = "afterCreate"
	AfterUpdate = "afterUpdate"
	AfterDelete = "afterDelete"

	Search = "search"
)

var queryFilters = regexp.MustCompile(`(?m)filters\[(?P<num>[^\]]+)\]\[(?P<nam>[^\]]+)\]`)

// adds event listener
func (R *Route) On(event string, listener Listener) *Route {
	if R.listeners == nil {
		R.listeners = map[string]Listener{}
	}
	R.listeners[event] = listener
	return R
}
func (R *Route) trigger(e string, ctx Context, h *kind.Holder) error {
	if R.listeners != nil {
		if ls, ok := R.listeners[e]; ok {
			return ls(ctx, h)
		}
	}
	return nil
}

// custom search
func (R *Route) Search(searchListener func(ctx Context, query string) ([]interface{}, error)) *Route {
	R.searchListener = searchListener
	return R
}

func (R *Route) Roles(rs ...Role) *Route {
	R.roles = map[string]bool{}
	for _, r := range rs {
		R.roles[string(r)] = true
	}
	return R
}

func (R *Route) Methods(ms ...string) *Route {
	R.methods = ms
	R.kind.AddRouteSettings(R.Path(), ms)
	return R
}

func (R *Route) Get(x http.HandlerFunc) *Route {
	R.get = x
	return R
}
func (R *Route) Post(x http.HandlerFunc) *Route {
	R.post = x
	return R
}
func (R *Route) Put(x http.HandlerFunc) *Route {
	R.put = x
	return R
}
func (R *Route) Delete(x http.HandlerFunc) *Route {
	R.delete = x
	return R
}
func (R *Route) Path() string {
	return R.path
}

type FacetOutput struct {
	Count int         `json:"count"`
	Value interface{} `json:"value"`
	Name  string      `json:"name"`
}

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

func (R *Route) getHandler() http.HandlerFunc {
	if R.get != nil {
		return R.get
	}
	if R.kind == nil {
		return func(w http.ResponseWriter, r *http.Request) {}
	}
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := R.NewContext(r)
		var ok, isPrivate bool
		if ok, isPrivate = ctx.HasPermission(R.kind, GET); !ok {
			ctx.PrintError(w, errors.ErrForbidden)
			return
		}

		id := mux.Vars(r)["id"]

		key, err := R.kind.DecodeKey(id)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}
		if isPrivate && !key.Parent().Equal(ctx.UserKey()) {
			ctx.PrintError(w, errors.ErrForbidden)
			return
		}

		h := R.kind.NewHolder(ctx.UserKey())

		if err := R.trigger(BeforeRead, ctx, h); err != nil {
			ctx.PrintError(w, err)
			return
		}

		err = h.Get(ctx, key)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		if err := R.trigger(AfterRead, ctx, nil); err != nil {
			ctx.PrintError(w, err)
			return
		}

		ctx.Print(w, h.Value())
	}
}

func (R *Route) queryHandler() http.HandlerFunc {
	if R.get != nil {
		return R.get
	}
	if R.kind == nil {
		return func(w http.ResponseWriter, r *http.Request) {}
	}
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := R.NewContext(r)
		var ok, isPrivate bool
		if ok, isPrivate = ctx.HasPermission(R.kind, QUERY); !ok {
			ctx.PrintError(w, errors.ErrForbidden)
			return
		}

		name, single, sort, limit, offset, ancestor := r.FormValue("name"), r.FormValue("single"), r.FormValue("sort"), r.FormValue("limit"), r.FormValue("offset"), r.FormValue("ancestor")

		if len(name) > 0 {
			// ordinary get
			var parent *datastore.Key
			if ancestor != "false" {
				parent = ctx.UserKey()
			}

			key := R.kind.NewKey(ctx, name, parent)
			if isPrivate && !key.Parent().Equal(ctx.UserKey()) {
				ctx.PrintError(w, errors.ErrForbidden)
				return
			}
			h := R.kind.NewHolder(ctx.UserKey())

			if err := R.trigger(BeforeRead, ctx, h); err != nil {
				ctx.PrintError(w, err)
				return
			}

			err := h.Get(ctx, key)
			if err != nil {
				ctx.PrintError(w, err)
				return
			}

			if err := R.trigger(AfterRead, ctx, h); err != nil {
				ctx.PrintError(w, err)
				return
			}

			ctx.Print(w, h.Value())
			return
		} else {
			limitInt, _ := strconv.Atoi(limit)
			offsetInt, _ := strconv.Atoi(offset)

			var filters []kind.Filter
			var filterMap = map[string]map[string]string{}
			for key, val := range r.URL.Query() {
				if strings.Split(key, "[")[0] == "filters" {
					fm := getParams(key)
					if len(fm["num"]) > 0 && len(fm["nam"]) > 0 {
						if m, ok := filterMap[fm["num"]]; ok {
							m[fm["nam"]] = val[0]
							var filterStr = m["filterStr"]
							var value = m["value"]
							if len(filterStr) > 0 && len(value) > 0 {
								filters = append(filters, kind.Filter{
									FilterStr: filterStr,
									Value:     value,
								})
							}
						} else {
							filterMap[fm["num"]] = map[string]string{
								fm["nam"]: val[0],
							}
						}
					}
				}
			}

			var hs []*kind.Holder
			var err error
			if ancestor == "false" {
				if isPrivate {
					ctx.PrintError(w, errors.ErrForbidden)
					return
				}
				hs, err = R.kind.Query(ctx, sort, limitInt, offsetInt, filters, nil)
				if err != nil {
					ctx.PrintError(w, err)
					return
				}
			} else {
				hs, err = R.kind.Query(ctx, sort, limitInt, offsetInt, filters, ctx.UserKey())
				if err != nil {
					ctx.PrintError(w, err)
					return
				}
			}

			var out []interface{}
			for _, h := range hs {
				out = append(out, h.Value())
			}

			if single == "true" {
				ctx.Print(w, out[0])
			} else {
				ctx.PrintResult(w, map[string]interface{}{
					"count":   len(out),
					"results": out,
				})
			}
		}
	}
}

func (R *Route) postHandler() http.HandlerFunc {
	if R.post != nil {
		return R.post
	}
	if R.kind == nil {
		return func(w http.ResponseWriter, r *http.Request) {}
	}
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := R.NewContext(r)

		if ok, _ := ctx.HasPermission(R.kind, CREATE); !ok {
			ctx.PrintError(w, errors.ErrForbidden)
			return
		}

		h := R.kind.NewHolder(ctx.UserKey())
		err := h.Parse(ctx.Body())
		if err != nil {
			ctx.PrintError(w, err, "error parsing")
			return
		}

		if err := R.trigger(BeforeCreate, ctx, h); err != nil {
			ctx.PrintError(w, err, "error on before create")
			return
		}

		err = h.Add(ctx)
		if err != nil {
			ctx.PrintError(w, err, "error adding")
			return
		}

		if err := R.trigger(AfterCreate, ctx, h); err != nil {
			ctx.PrintError(w, err, "error on after create")
			return
		}

		ctx.Print(w, h.Value())
	}
}

func (R *Route) putHandler() http.HandlerFunc {
	if R.put != nil {
		return R.put
	}
	if R.kind == nil {
		return func(w http.ResponseWriter, r *http.Request) {}
	}
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := R.NewContext(r)

		var ok, isPrivate bool
		if ok, isPrivate = ctx.HasPermission(R.kind, UPDATE); !ok {
			ctx.PrintError(w, errors.ErrForbidden)
			return
		}

		id := mux.Vars(r)["id"]

		key, err := R.kind.DecodeKey(id)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		if isPrivate && !key.Parent().Equal(ctx.UserKey()) {
			ctx.PrintError(w, errors.ErrForbidden)
			return
		}

		h := R.kind.NewHolder(ctx.UserKey())
		err = h.Parse(ctx.Body())
		if err != nil {
			ctx.PrintError(w, err)
			return
		}
		h.SetKey(key)

		if err := R.trigger(BeforeUpdate, ctx, h); err != nil {
			ctx.PrintError(w, err)
			return
		}

		err = h.Update(ctx)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		if err := R.trigger(AfterUpdate, ctx, h); err != nil {
			ctx.PrintError(w, err)
			return
		}

		ctx.Print(w, h.Value())
	}
}

// todo
func (R *Route) deleteHandler() http.HandlerFunc {
	if R.delete != nil {
		return R.delete
	}
	if R.kind == nil {
		return func(w http.ResponseWriter, r *http.Request) {}
	}
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := R.NewContext(r)

		var ok, isPrivate bool
		if ok, isPrivate = ctx.HasPermission(R.kind, DELETE); !ok {
			ctx.PrintError(w, errors.ErrForbidden)
			return
		}

		id := mux.Vars(r)["id"]

		key, err := R.kind.DecodeKey(id)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		if isPrivate && !key.Parent().Equal(ctx.UserKey()) {
			ctx.PrintError(w, errors.ErrForbidden)
			return
		}

		if err := R.trigger(BeforeDelete, ctx, nil); err != nil {
			ctx.PrintError(w, err)
			return
		}

		err = R.kind.Delete(ctx, key)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		if err := R.trigger(AfterDelete, ctx, nil); err != nil {
			ctx.PrintError(w, err)
			return
		}

		ctx.Print(w, ActionResult{
			Action:  "delete",
			Message: "success",
			Ids:     []string{id},
		})
	}
}

type ActionResult struct {
	Action  string   `json:"action,omitempty"`
	Message string   `json:"message,omitempty"`
	Error   string   `json:"error,omitempty"`
	Ids     []string `json:"ids,omitempty"`
}
