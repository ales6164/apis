package apis

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"google.golang.org/appengine/datastore"
	"net/http"
	"strconv"
	"strings"
)

type Route struct {
	a          *Apis
	pathPrefix string
	kind       *Kind
	router     *mux.Router
}

type Cursor struct {
	Next string `json:"next"`
}

type Results struct {
	Results []interface{} `json:"results"`
	Count   int           `json:"count"`
	Cursor  *Cursor       `json:"cursor,omitempty"`
}

func (r *Route) init() {
	/*
	TODO:
	1. Use current handlers
	2. Upgrade to use REST API standard flow
	3. Add scopes

	 */

	// QUERY
	r.router.HandleFunc("", func(writer http.ResponseWriter, request *http.Request) {
		var err error
		ctx := NewContext(request)

		if ok := ctx.HasScope(r.kind.ScopeReadOnly, r.kind.ScopeReadWrite, r.kind.ScopeFullControl); !ok {
			http.Error(writer, "forbidden", http.StatusForbidden)
			return
		}

		var paramPairs []string

		var offset, limit int
		limit = 25 //default
		q := datastore.NewQuery(r.kind.name)
		var filterMap = map[string]map[string]string{}
		for name, values := range request.URL.Query() {
			switch name {
			case "order":
				v := values[len(values)-1]
				q = q.Order(v)
				paramPairs = append(paramPairs, "order="+v)
			case "limit":
				v := values[len(values)-1]
				l, err := strconv.Atoi(v)
				if err != nil {
					ctx.PrintError(writer, err)
					return
				}
				limit = l
				paramPairs = append(paramPairs, "limit="+v)
			case "offset":
				v := values[len(values)-1]
				l, err := strconv.Atoi(v)
				if err != nil {
					ctx.PrintError(writer, err)
					return
				}
				q = q.Offset(l)
				offset = l
				paramPairs = append(paramPairs, "offset="+v)
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

		var linkHeader []string
		var out []interface{}
		total, err := q.Count(ctx)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}
		t := q.Run(ctx)
		for {
			var h = r.kind.NewHolder(nil)
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
			linkHeader = append(linkHeader, "<"+getHost(request)+request.URL.Path+"?"+strings.Join(append(paramPairs, "offset="+strconv.Itoa(offset+count)), "&")+`>; rel="next"`)
			if (total - offset - count - limit) > 0 {
				// next is not last
				linkHeader = append(linkHeader, "<"+getHost(request)+request.URL.Path+"?"+strings.Join(append(paramPairs, "offset="+strconv.Itoa(total-limit)), "&")+`>; rel="last"`)
			}
		}
		if offset > 0 {
			// get previous link
			linkHeader = append(linkHeader, "<"+getHost(request)+request.URL.Path+"?"+strings.Join(append(paramPairs, "offset="+strconv.Itoa(offset+count)), "&")+`>; rel="prev"`)
			if offset-limit > 0 {
				// previous is not first
				linkHeader = append(linkHeader, "<"+getHost(request)+request.URL.Path+"?"+strings.Join(append(paramPairs, "offset=0"), "&")+`>; rel="first"`)
			}
		}

		ctx.Print(writer, out, "X-Total-Count", strconv.Itoa(total), "Link", strings.Join(linkHeader, ","))
	}).Methods(http.MethodGet)

	// GET
	r.router.HandleFunc("/{name}", func(writer http.ResponseWriter, request *http.Request) {
		ctx := NewContext(request)

		if ok := ctx.HasScope(r.kind.ScopeReadOnly, r.kind.ScopeReadWrite, r.kind.ScopeFullControl); !ok {
			http.Error(writer, "forbidden", http.StatusForbidden)
			return
		}

		name := mux.Vars(request)["name"]
		var key *datastore.Key
		if r.kind.dsUseName {
			key = datastore.NewKey(ctx, r.kind.name, name, 0, nil)
		} else {
			var err error
			key, err = datastore.DecodeKey(name)
			if err != nil {
				ctx.PrintError(writer, err)
				return
			}
		}
		h := r.kind.NewHolder(key)
		if err := datastore.Get(ctx, key, h); err != nil {
			ctx.PrintError(writer, err)
			return
		}
		ctx.Print(writer, h.GetValue())
	}).Methods(http.MethodGet)

	// GET FIELD
	r.router.HandleFunc(`/{name}/{rest:[a-zA-Z0-9=\-\/]+}`, func(writer http.ResponseWriter, request *http.Request) {
		ctx := NewContext(request)

		if ok := ctx.HasScope(r.kind.ScopeReadOnly, r.kind.ScopeReadWrite, r.kind.ScopeFullControl); !ok {
			http.Error(writer, "forbidden", http.StatusForbidden)
			return
		}

		vars := mux.Vars(request)
		var key *datastore.Key
		if r.kind.dsUseName {
			key = datastore.NewKey(ctx, r.kind.name, vars["name"], 0, nil)
		} else {
			var err error
			key, err = datastore.DecodeKey(vars["name"])
			if err != nil {
				ctx.PrintError(writer, err)
				return
			}
		}
		h := r.kind.NewHolder(key)
		if err := datastore.Get(ctx, key, h); err != nil {
			ctx.PrintError(writer, err)
			return
		}

		var err error
		for _, name := range strings.Split(vars["rest"], "/") {
			h, err = h.Get(r.a, ctx, name)
			if err != nil {
				ctx.PrintError(writer, err)
				return
			}
		}
		json.NewEncoder(writer).Encode(h.GetValue())
	}).Methods(http.MethodGet)

	// POST
	r.router.HandleFunc(``, func(writer http.ResponseWriter, request *http.Request) {
		ctx := NewContext(request)

		if ok := ctx.HasScope(r.kind.ScopeReadWrite, r.kind.ScopeFullControl); !ok {
			http.Error(writer, "forbidden", http.StatusForbidden)
			return
		}

		h := r.kind.NewHolder(nil)
		if err := h.Parse(ctx.Body()); err != nil {
			ctx.PrintError(writer, err)
			return
		}

		var name = r.kind.dsNameGenerator(ctx, h)
		var key = datastore.NewKey(ctx, r.kind.name, name, 0, nil)

		var err error
		if h.key, err = datastore.Put(ctx, key, h); err != nil {
			ctx.PrintError(writer, err)
			return
		}
		ctx.Print(writer, h.GetValue())
	}).Methods(http.MethodPost)

	// PUT
	r.router.HandleFunc(`/{name}`, func(writer http.ResponseWriter, request *http.Request) {
		ctx := NewContext(request)

		if ok := ctx.HasScope(r.kind.ScopeReadWrite, r.kind.ScopeFullControl); !ok {
			http.Error(writer, "forbidden", http.StatusForbidden)
			return
		}

		name := mux.Vars(request)["name"]

		var key *datastore.Key
		if r.kind.dsUseName {
			key = datastore.NewKey(ctx, r.kind.name, name, 0, nil)
		} else {
			var err error
			key, err = datastore.DecodeKey(name)
			if err != nil {
				ctx.PrintError(writer, err)
				return
			}
		}

		// TODO: Check scope before PUT

		h := r.kind.NewHolder(key)
		if err := h.Parse(ctx.Body()); err != nil {
			ctx.PrintError(writer, err)
			return
		}

		var err error
		if h.key, err = datastore.Put(ctx, key, h); err != nil {
			ctx.PrintError(writer, err)
			return
		}
		ctx.Print(writer, h.GetValue())
	}).Methods(http.MethodPut)

	// PUT FIELD
	// todo:
	r.router.HandleFunc(`/{name}`, func(writer http.ResponseWriter, request *http.Request) {
		ctx := NewContext(request)

		if ok := ctx.HasScope(r.kind.ScopeReadWrite, r.kind.ScopeFullControl); !ok {
			http.Error(writer, "forbidden", http.StatusForbidden)
			return
		}

		vars := mux.Vars(request)
		var key *datastore.Key
		if r.kind.dsUseName {
			key = datastore.NewKey(ctx, r.kind.name, vars["name"], 0, nil)
		} else {
			var err error
			key, err = datastore.DecodeKey(vars["name"])
			if err != nil {
				ctx.PrintError(writer, err)
				return
			}
		}
		h := r.kind.NewHolder(key)
		if err := datastore.Get(ctx, key, h); err != nil {
			ctx.PrintError(writer, err)
			return
		}
		var err error
		for _, name := range strings.Split(vars["rest"], "/") {
			h, err = h.Get(r.a, ctx, name)
			if err != nil {
				ctx.PrintError(writer, err)
				return
			}
		}
		if err := h.Delete(ctx); err != nil {
			ctx.PrintError(writer, err)
			return
		}
		json.NewEncoder(writer).Encode(h.GetValue())
	}).Methods(http.MethodPut)

	// DELETE
	r.router.HandleFunc(`/{name}`, func(writer http.ResponseWriter, request *http.Request) {
		ctx := NewContext(request)

		if ok := ctx.HasScope(r.kind.ScopeDelete, r.kind.ScopeFullControl); !ok {
			http.Error(writer, "forbidden", http.StatusForbidden)
			return
		}

		name := mux.Vars(request)["name"]
		var key *datastore.Key
		if r.kind.dsUseName {
			key = datastore.NewKey(ctx, r.kind.name, name, 0, nil)
		} else {
			var err error
			key, err = datastore.DecodeKey(name)
			if err != nil {
				ctx.PrintError(writer, err)
				return
			}
		}
		var err error
		if err = datastore.Delete(ctx, key); err != nil {
			ctx.PrintError(writer, err)
			return
		}
		ctx.Print(writer, "success")
	}).Methods(http.MethodDelete)

	// DELETE FIELD
	r.router.HandleFunc(`/{name}/{rest:[a-zA-Z0-9=\-\/]+}`, func(writer http.ResponseWriter, request *http.Request) {
		ctx := NewContext(request)

		if ok := ctx.HasScope(r.kind.ScopeDelete, r.kind.ScopeFullControl); !ok {
			http.Error(writer, "forbidden", http.StatusForbidden)
			return
		}

		vars := mux.Vars(request)
		var key *datastore.Key
		if r.kind.dsUseName {
			key = datastore.NewKey(ctx, r.kind.name, vars["name"], 0, nil)
		} else {
			var err error
			key, err = datastore.DecodeKey(vars["name"])
			if err != nil {
				ctx.PrintError(writer, err)
				return
			}
		}
		h := r.kind.NewHolder(key)
		if err := datastore.Get(ctx, key, h); err != nil {
			ctx.PrintError(writer, err)
			return
		}
		var err error
		for _, name := range strings.Split(vars["rest"], "/") {
			h, err = h.Get(r.a, ctx, name)
			if err != nil {
				ctx.PrintError(writer, err)
				return
			}
		}
		if err := h.Delete(ctx); err != nil {
			ctx.PrintError(writer, err)
			return
		}
		json.NewEncoder(writer).Encode(h.GetValue())
	}).Methods(http.MethodDelete)
}
