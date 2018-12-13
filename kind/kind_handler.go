package kind

import (
	"errors"
	"github.com/gorilla/mux"
	"golang.org/x/net/context"
	"google.golang.org/appengine/datastore"
	"net/http"
	"strconv"
	"strings"
)

type QueryResult struct {
	Items      []interface{}
	Total      int
	Count      int
	Limit      int
	Offset     int
	Order      string
	LinkHeader string
	StatusCode int
}

/*
Valid params are order, limit, offset, name, id, and filters.
Filters param is an array of filter pairs:
filters[0][filterStr] "fieldName >"
filters[0][value] "fieldValue"
 */
func (k *Kind) Query(ctx Context, params map[string][]string) (QueryResult, error) {
	r := QueryResult{
		Limit: 25,
		Items: []interface{}{},
	}

	q := datastore.NewQuery(k.Name)
	var filterMap = map[string]map[string]string{}
	for name, values := range params {
		switch name {
		case "order":
			v := values[len(values)-1]
			q = q.Order(v)

		case "limit":
			v := values[len(values)-1]
			l, err := strconv.Atoi(v)
			if err != nil {
				return r, err
			}
			r.Limit = l
		case "offset":
			v := values[len(values)-1]
			l, err := strconv.Atoi(v)
			if err != nil {
				return r, err
			}
			r.Offset = l
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

	// set limit
	q = q.Limit(r.Limit)
	// set offset
	q = q.Offset(r.Offset)

	var err error
	r.Total, err = Count(ctx, k.Name)
	if err != nil {
		return r, err
	}

	t := q.Run(ctx)
	for {
		var h = k.NewHolder(nil)
		h.key, err = t.Next(h)

		if err == datastore.Done {
			break
		}

		r.Count++
		r.Items = append(r.Items, h.GetValue())
	}

	if r.Count > 0 {
		r.StatusCode = http.StatusOK
	} else {
		r.StatusCode = http.StatusNoContent
	}

	var linkHeader []string
	if (r.Total - r.Offset - r.Count) > 0 {
		// has more items to fetch
		q := ctx.request.URL.Query()
		q.Set("offset", strconv.Itoa(r.Offset+r.Count))
		linkHeader = append(linkHeader, "<"+getSchemeAndHost(ctx.request)+ctx.request.URL.Path+"?"+q.Encode()+`>; rel="next"`)
		if (r.Total - r.Offset - r.Count - r.Limit) > 0 {
			// next is not last
			q := ctx.request.URL.Query()
			q.Set("offset", strconv.Itoa(r.Total+r.Limit))
			linkHeader = append(linkHeader, "<"+getSchemeAndHost(ctx.request)+ctx.request.URL.Path+"?"+q.Encode()+`>; rel="last"`)
		}
	}
	if r.Offset > 0 {
		// get previous link
		q := ctx.request.URL.Query()
		offset := r.Offset - r.Limit
		if offset < 0 {
			offset = 0
		}
		q.Set("offset", strconv.Itoa(r.Offset-r.Limit))
		linkHeader = append(linkHeader, "<"+getSchemeAndHost(ctx.request)+ctx.request.URL.Path+"?"+q.Encode()+`>; rel="prev"`)
		if offset > 0 {
			// previous is not first
			q.Set("offset", "0")
			linkHeader = append(linkHeader, "<"+getSchemeAndHost(ctx.request)+ctx.request.URL.Path+"?"+q.Encode()+`>; rel="first"`)
		}
	}

	r.LinkHeader = strings.Join(linkHeader, ",")

	return r, nil
}

func (k *Kind) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var err error
	var h *Holder

	var key *datastore.Key
	var path []string
	var hasKey, hasPath bool

	vars := mux.Vars(r)

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
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}
		if hasKey {
			h, err := k.Get(ctx, key)
			if err != nil {
				if err == datastore.ErrNoSuchEntity {
					ctx.PrintError(w, http.StatusNotFound)
					return
				}
				ctx.PrintError(w, http.StatusInternalServerError, err.Error())
				return
			}
			if hasPath {
				var value interface{}
				h, value, err = h.Get(ctx, path)
				if err != nil {
					if err == datastore.ErrNoSuchEntity {
						ctx.PrintError(w, http.StatusNotFound)
						return
					}
					ctx.PrintError(w, http.StatusInternalServerError, err.Error())
					return
				}
				ctx.Print(w, value, http.StatusOK)
			} else {
				ctx.Print(w, h.GetValue(), http.StatusOK)
			}
		} else {
			// DO QUERY
			queryResults, err := k.Query(ctx, r.URL.Query())
			if err != nil {
				ctx.PrintError(w, http.StatusBadRequest, err.Error())
				return
			}
			ctx.Print(w, queryResults.Items, queryResults.StatusCode, "X-Total-Count", strconv.Itoa(queryResults.Total), "Link", queryResults.LinkHeader)
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
			ctx.PrintError(w, http.StatusInternalServerError, err.Error())
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
				ctx.PrintError(w, http.StatusNotFound, err.Error())
				return
			}
			h, err = h.Set(ctx, path, ctx.Body())
			if err != nil {
				ctx.PrintError(w, http.StatusInternalServerError, err.Error())
				return
			}
			if h.key, err = datastore.Put(ctx, key, h); err != nil {
				ctx.PrintError(w, http.StatusInternalServerError, err.Error())
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
					ctx.PrintError(w, http.StatusNotFound, err.Error())
					return
				}

				h, err = h.Delete(ctx, path)
				if err != nil {
					ctx.PrintError(w, http.StatusInternalServerError, err.Error())
					return
				}

				_, err = datastore.Put(ctx, h.key, h)
				if err != nil {
					ctx.PrintError(w, http.StatusInternalServerError, err.Error())
					return
				}

				ctx.Print(w, h.GetValue(), http.StatusOK)
			} else {
				if err = datastore.Delete(ctx, key); err != nil {
					ctx.PrintError(w, http.StatusNotFound, err.Error())
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
