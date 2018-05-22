package apis

import (
	"github.com/ales6164/apis/kind"
	"net/http"
	"google.golang.org/appengine/datastore"
	"strconv"
	"strings"
	"github.com/ales6164/apis/errors"
	"google.golang.org/appengine/search"
	"reflect"
	"math"
)

type Route struct {
	a    *Apis
	kind *kind.Kind
	path string

	ui        *kind.UI
	listeners map[string]Listener
	searchListener func(ctx Context, query string) ([]interface{}, error)
	roles map[string]bool

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
	Count   int                      `json:"count,omitempty"`
	Total   int                      `json:"total,omitempty"`
	Results []interface{}            `json:"results,omitempty"`
	Filters map[string][]FacetOutput `json:"filters,omitempty"`
	Cursor  Cursor                   `json:"cursor,omitempty"`
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

func (R *Route) UI(ui *kind.UI) *Route {
	if R.kind.HasUI() {
		panic(errors.New("failed to set route UI on kind that already has UI"))
	}
	R.ui = ui
	R.kind.SetUI(ui, R.path, R.methods)
	return R
}

func (R *Route) Methods(ms ...string) *Route {
	R.methods = ms
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

type FacetOutput struct {
	Count int         `json:"count"`
	Value interface{} `json:"value"`
	Name  string      `json:"name"`
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
		if ok := ctx.HasPermission(R.kind, READ); !ok {
			ctx.PrintError(w, errors.ErrForbidden)
			return
		}

		q, next, autoFilterDiscovery, name, id, sort, limit, offset, ancestor := r.FormValue("q"), r.FormValue("next"), r.FormValue("autoFilterDiscovery"), r.FormValue("name"), r.FormValue("id"), r.FormValue("sort"), r.FormValue("limit"), r.FormValue("offset"), r.FormValue("ancestor")

		if err := R.trigger(BeforeRead, ctx, nil); err != nil {
			ctx.PrintError(w, err)
			return
		}

		if len(id) > 0 {
			// ordinary get
			key, err := R.kind.DecodeKey(id)
			if err != nil {
				ctx.PrintError(w, err)
				return
			}
			h := R.kind.NewHolder(ctx.UserKey())
			err = h.Get(ctx, key)
			if err != nil {
				ctx.PrintError(w, err)
				return
			}
			ctx.Print(w, h.Value())
			return
		} else if len(name) > 0 {
			// ordinary get
			var parent *datastore.Key
			if ancestor != "false" {
				parent = ctx.UserKey()
			}

			key := R.kind.NewKey(ctx, name, parent)
			h := R.kind.NewHolder(ctx.UserKey())
			err := h.Get(ctx, key)
			if err != nil {
				ctx.PrintError(w, err)
				return
			}
			ctx.Print(w, h.Value())
			return
		} else if R.kind.EnableSearch {
			index, err := OpenIndex(R.kind.Name)
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
				} else {
					for _, v := range val {
						fields, facets = R.kind.RetrieveSearchParameter(key, v, fields, facets)
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
					IDsOnly: R.kind.RetrieveByIDOnSearch,
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
				IDsOnly:       R.kind.RetrieveByIDOnSearch,
				Refinements:   facets,
				Cursor:        search.Cursor(next),
				CountAccuracy: 1000,
				Offset:        intOffset,
				Limit:         intLimit,
				Sort: &search.SortOptions{
					Expressions: sortExpr,
				}}); ; {
				var doc = reflect.New(R.kind.SearchType).Interface()
				docKey, err := t.Next(doc)
				if err == search.Done {
					break
				}
				if err != nil {
					ctx.PrintError(w, err)
					return
				}

				if key, err := R.kind.DecodeKey(docKey); err == nil {
					docKeys = append(docKeys, key)
				}

				results = append(results, doc)
			}

			// fetch real entries from datastore
			if R.kind.RetrieveByIDOnSearch {
				if len(docKeys) == len(results) {
					hs, err := kind.GetMulti(ctx, R.kind, docKeys...)
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

			ctx.Print(w, SearchOutput{
				Count:   len(results),
				Total:   t.Count(),
				Results: results,
				Filters: facsOutput,
				Cursor: Cursor{
					Next: string(t.Cursor()),
					Prev: next,
				},
			})

			return
		} else {
			// query
			limitInt, _ := strconv.Atoi(limit)
			offsetInt, _ := strconv.Atoi(offset)

			//q, next, autoFilterDiscovery, name, id, sort, limit, offset, ancestor

			var filters []kind.Filter
			for key, val := range r.URL.Query() {
				if key == "q" || key == "name" || key == "id" || key == "sort" || key == "limit" || key == "offset" || key == "ancestor" {
					continue
				}
				for _, v := range val {
					filters = append(filters, kind.Filter{
						FilterStr: key + " =",
						Value:     v,
					})
				}
			}

			var hs []*kind.Holder
			var err error
			if ancestor == "false" && ctx.HasRole(AdminRole) {
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
				if err := R.trigger(AfterRead, ctx, h); err != nil {
					ctx.PrintError(w, err)
					return
				}
				out = append(out, h.Value())
			}
			ctx.PrintResult(w, map[string]interface{}{
				"count":   len(out),
				"results": out,
			})
			return
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

		if ok := ctx.HasPermission(R.kind, CREATE); !ok {
			ctx.PrintError(w, errors.ErrForbidden)
			return
		}

		h := R.kind.NewHolder(ctx.UserKey())
		err := h.Parse(ctx.Body())
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		if err := R.trigger(BeforeCreate, ctx, h); err != nil {
			ctx.PrintError(w, err)
			return
		}

		err = h.Add(ctx)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		if err := R.trigger(AfterCreate, ctx, h); err != nil {
			ctx.PrintError(w, err)
			return
		}

		if R.kind.EnableSearch {
			// put to search

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
	type UpdateVal struct {
		Id    string      `json:"id"`
		Name  string      `json:"name"`
		Value interface{} `json:"value"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := R.NewContext(r)

		if ok := ctx.HasPermission(R.kind, UPDATE); !ok {
			ctx.PrintError(w, errors.ErrForbidden)
			return
		}

		id := r.URL.Query().Get("id")
		name := r.URL.Query().Get("name")

		/*var data = UpdateVal{}
		json.Unmarshal(ctx.Body(), &data)*/

		if len(id) == 0 && len(name) == 0 {
			ctx.PrintError(w, errors.New("must provide id or name"))
			return
		}

		h := R.kind.NewHolder(ctx.UserKey())
		err := h.Parse(ctx.Body())
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		var key *datastore.Key
		if len(id) > 0 {
			key, err = R.kind.DecodeKey(id)
			if err != nil {
				ctx.PrintError(w, err)
				return
			}
		} else {
			key = R.kind.NewKey(ctx, name, ctx.UserKey())
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
	return func(w http.ResponseWriter, r *http.Request) {}
}
