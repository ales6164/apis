package apis

import (
	"gopkg.in/ales6164/apis.v1/errors"
	"gopkg.in/ales6164/apis.v1/kind"
	"github.com/gorilla/mux"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/search"
	"math"
	"net/http"
	"strconv"
	"strings"
)

func initSearch(a *Apis, r *mux.Router) {
	R := &Route{
		a:       a,
		methods: []string{http.MethodGet},
	}
	R.get = func(w http.ResponseWriter, r *http.Request) {
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

		index, err := search.Open(kindName)
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
			var doc = new(kind.Document)
			docKey, err := t.Next(doc)
			if err == search.Done {
				break
			}
			if err != nil {
				ctx.PrintError(w, err)
				return
			}

			if k.RetrieveByIDOnSearch {
				if key, err := k.DecodeKey(docKey); err == nil {
					if (isPrivate && key.Parent().Equal(ctx.UserKey())) || !isPrivate {
						docKeys = append(docKeys, key)
						results = append(results, nil)
					}
				}
			} else {
				results = append(results, doc.Parse(k))
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

	r.Handle("/search/{kind}", a.middleware.Handler(R.getHandler())).Methods(http.MethodGet)
}
