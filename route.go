package apis

import (
	"github.com/gorilla/mux"
	"net/http"
	"encoding/json"
	"google.golang.org/appengine/datastore"
	"strings"
	"strconv"
	"google.golang.org/appengine/log"
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
		ctx := r.a.NewContext(request)
		q := datastore.NewQuery(r.kind.name)
		var filterMap = map[string]map[string]string{}
		for name, values := range request.URL.Query() {
			switch name {
			case "order":
				q = q.Order(values[len(values)-1])
			case "limit":
				l, err := strconv.Atoi(values[len(values)-1])
				if err != nil {
					ctx.PrintError(writer, err)
					return
				}
				q = q.Limit(l)
			case "start":
				curr, err := datastore.DecodeCursor(values[len(values)-1])
				if err != nil {
					ctx.PrintError(writer, err)
					return
				}
				q = q.Start(curr)
			case "end":
				// ???
				curr, err := datastore.DecodeCursor(values[len(values)-1])
				if err != nil {
					ctx.PrintError(writer, err)
					return
				}
				q = q.End(curr)
			case "offset":
				l, err := strconv.Atoi(values[len(values)-1])
				if err != nil {
					ctx.PrintError(writer, err)
					return
				}
				q = q.Offset(l)
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
		var cursor *Cursor
		var out []interface{}
		t := q.Run(ctx)
		for {
			var h = r.kind.NewHolder(nil)
			h.key, err = t.Next(h)
			if err == datastore.Done {
				if c, err := t.Cursor(); err == nil && len(c.String()) > 0 {
					cursor = new(Cursor)
					cursor.Next = c.String()
				}
				break
			}
			out = append(out, h.Value())
		}
		ctx.Print(writer, Results{
			Count:   len(out),
			Results: out,
			Cursor:  cursor,
		})
	}).Methods(http.MethodGet)

	// GET
	r.router.HandleFunc("/{name}", func(writer http.ResponseWriter, request *http.Request) {
		ctx := r.a.NewContext(request)
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
		ctx.Print(writer, h.Value())
	}).Methods(http.MethodGet)

	// GET FIELD
	r.router.HandleFunc(`/{name}/{rest:[a-zA-Z0-9=\-\/]+}`, func(writer http.ResponseWriter, request *http.Request) {
		ctx := r.a.NewContext(request)
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
			log.Debugf(ctx, "%s", name)
			h, err = h.Get(ctx, name)
			if err != nil {
				ctx.PrintError(writer, err)
				return
			}
		}
		json.NewEncoder(writer).Encode(h.value)
	}).Methods(http.MethodGet)

	// PUT
	r.router.HandleFunc(`/{name}`, func(writer http.ResponseWriter, request *http.Request) {
		ctx := r.a.NewContext(request)
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
		if err := h.Parse(ctx.Body()); err != nil {
			ctx.PrintError(writer, err)
			return
		}

		var err error
		if h.key, err = datastore.Put(ctx, key, h); err != nil {
			ctx.PrintError(writer, err)
			return
		}
		ctx.Print(writer, h.Value())
	}).Methods(http.MethodPut)
}
