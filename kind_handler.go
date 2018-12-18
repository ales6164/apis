package apis

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

	q := datastore.NewQuery(k.Path)
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
	r.Total, err = Count(ctx, k.Path)
	if err != nil {
		return r, err
	}

	t := q.Run(ctx)
	for {
		var h = k.NewHolder(nil)
		h.Key, err = t.Next(h)

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
		q := ctx.r.URL.Query()
		q.Set("offset", strconv.Itoa(r.Offset+r.Count))
		linkHeader = append(linkHeader, "<"+getSchemeAndHost(ctx.r)+ctx.r.URL.Path+"?"+q.Encode()+`>; rel="next"`)
		if (r.Total - r.Offset - r.Count - r.Limit) > 0 {
			// next is not last
			q := ctx.r.URL.Query()
			q.Set("offset", strconv.Itoa(r.Total+r.Limit))
			linkHeader = append(linkHeader, "<"+getSchemeAndHost(ctx.r)+ctx.r.URL.Path+"?"+q.Encode()+`>; rel="last"`)
		}
	}
	if r.Offset > 0 {
		// get previous link
		q := ctx.r.URL.Query()
		offset := r.Offset - r.Limit
		if offset < 0 {
			offset = 0
		}
		q.Set("offset", strconv.Itoa(r.Offset-r.Limit))
		linkHeader = append(linkHeader, "<"+getSchemeAndHost(ctx.r)+ctx.r.URL.Path+"?"+q.Encode()+`>; rel="prev"`)
		if offset > 0 {
			// previous is not first
			q.Set("offset", "0")
			linkHeader = append(linkHeader, "<"+getSchemeAndHost(ctx.r)+ctx.r.URL.Path+"?"+q.Encode()+`>; rel="first"`)
		}
	}

	r.LinkHeader = strings.Join(linkHeader, ",")

	return r, nil
}

/*
/kinds QUERY, POST
/kinds/{key} GET, PUT
/kinds/{key}/{path} GET, PUT
 */

func (k *Kind) QueryHandler(ctx Context) {
	queryResults, err := k.Query(ctx, ctx.r.URL.Query())
	if err != nil {
		ctx.PrintError(err.Error(), http.StatusBadRequest)
		return
	}
	ctx.PrintJSON(queryResults.Items, queryResults.StatusCode, "X-Total-Count", strconv.Itoa(queryResults.Total), "Link", queryResults.LinkHeader)
}

func (k *Kind) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var err error
	var ok bool
	var h *Holder

	var encodedCollection string
	var key, collection *datastore.Key
	var path []string
	var hasKey, hasPath, hasCollection bool

	vars := mux.Vars(r)

	if encodedKey, ok := vars["key"]; ok {
		if key, err = datastore.DecodeKey(encodedKey); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		} else {
			hasKey = true
		}
	}
	if encodedCollection, ok = vars["collection"]; ok {
		if collection, err = datastore.DecodeKey(encodedCollection); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		} else {
			hasCollection = true
			if collection != nil {
				//remove this if
			}
		}
	}
	if _path, ok := vars["path"]; ok {
		path = strings.Split(_path, "/")
		hasPath = len(path) > 0
	}

	ctx := NewContext(w, r)

	// todo: if has collection ...
	if hasCollection {
		if hasKey && key.Namespace() != encodedCollection {
			// key namespace doesn't match collection
			ctx.PrintError("key namespace doesn't match collection", http.StatusConflict)
			return
		}

		// collection key + user key
		//datastore.Get(ctx, collection)

		// 1. Get collection permission datastore table and check if user has entry (also check if allUsers has access)
		// 2. Load collection permissions to context for future scope checks
		// 3. Update context namespace
	} else {
		/*if hasKey && len(key.Namespace()) > 0 {
			// key namespace doesn't match collection
			ctx.PrintError(w, http.StatusConflict, "key namespace doesn't match collection")
			return
		}*/

		// 1. Check if key has namespace and if so check access
	}

	switch r.Method {
	case http.MethodGet:
		if ok := ctx.HasScope(k.ScopeReadOnly, k.ScopeReadWrite, k.ScopeFullControl); !ok {
			ctx.PrintError(http.StatusText(http.StatusForbidden), http.StatusInternalServerError)
			return
		}
		if hasKey {
			h, err := k.Get(ctx, key)
			if err != nil {
				if err == datastore.ErrNoSuchEntity {
					ctx.PrintError(http.StatusText(http.StatusNotFound), http.StatusNotFound)
					return
				}
				ctx.PrintError(err.Error(), http.StatusInternalServerError)
				return
			}
			if hasPath {
				var value interface{}
				h, value, err = h.Get(ctx, path)
				if err != nil {
					if err == datastore.ErrNoSuchEntity {
						ctx.PrintError(http.StatusText(http.StatusNotFound), http.StatusNotFound)
						return
					}
					ctx.PrintError(err.Error(), http.StatusInternalServerError)
					return
				}
				ctx.PrintJSON(value, http.StatusOK)
			} else {
				ctx.PrintJSON(h.GetValue(), http.StatusOK)
			}
		} else {
			// DO QUERY
			queryResults, err := k.Query(ctx, r.URL.Query())
			if err != nil {
				ctx.PrintError(err.Error(), http.StatusBadRequest)
				return
			}
			ctx.PrintJSON(queryResults.Items, queryResults.StatusCode, "X-Total-Count", strconv.Itoa(queryResults.Total), "Link", queryResults.LinkHeader)
		}
	case http.MethodPost:
		if ok := ctx.HasScope(k.ScopeReadWrite, k.ScopeFullControl); !ok {
			ctx.PrintError(http.StatusText(http.StatusForbidden), http.StatusInternalServerError)
			return
		}

		if hasPath || hasKey {
			http.Error(w, "not implemented", http.StatusNotImplemented)
			return
		}

		h = k.NewHolder(nil)
		if err := h.Parse(ctx.Body()); err != nil {
			ctx.PrintError(err.Error(), http.StatusInternalServerError)
			return
		}

		var name = k.dsNameGenerator(ctx, h)
		h.Key = datastore.NewKey(ctx, k.Path, name, 0, nil)

		err = datastore.RunInTransaction(ctx, func(tc context.Context) error {
			err = k.Create(tc, h)
			if err != nil {
				return err
			}

			// create iam entry
			//iam := IAMKind.NewHolder(nil)
			//iam.Key = datastore.NewKey(tc, iam.Kind.Name, )

			return errors.New("entry already exists")
		}, nil)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

		_ = Increment(ctx, k.Path)

		var location string
		locationUrl, err := mux.CurrentRoute(r).URL()
		if err == nil {
			location = strings.Join(append(strings.Split(locationUrl.Path, "/"), h.Key.Encode()), "/")
		}

		ctx.PrintJSON(h.GetValue(), http.StatusCreated, "Location", location)
	case http.MethodPut:
		if ok := ctx.HasScope(k.ScopeReadWrite, k.ScopeFullControl); !ok {
			ctx.PrintError(http.StatusText(http.StatusForbidden), http.StatusInternalServerError)
			return
		}

		if hasKey {
			h, err = k.Get(ctx, key)
			if err != nil {
				ctx.PrintError(err.Error(), http.StatusNotFound)
				return
			}
			h, err = h.Set(ctx, path, ctx.Body())
			if err != nil {
				ctx.PrintError(err.Error(), http.StatusInternalServerError)
				return
			}
			if h.Key, err = datastore.Put(ctx, key, h); err != nil {
				ctx.PrintError(err.Error(), http.StatusInternalServerError)
				return
			}
			ctx.PrintJSON(h.GetValue(), http.StatusOK)
		} else {
			r.Method = http.MethodPost
			k.ServeHTTP(w, r)
			return
		}
	case http.MethodDelete:
		if ok := ctx.HasScope(k.ScopeDelete, k.ScopeFullControl); !ok {
			ctx.PrintError(http.StatusText(http.StatusForbidden), http.StatusInternalServerError)
			return
		}

		if hasKey {
			if hasPath {
				if h, err = k.Get(ctx, key); err != nil {
					ctx.PrintError(err.Error(), http.StatusNotFound)
					return
				}

				h, err = h.Delete(ctx, path)
				if err != nil {
					ctx.PrintError(err.Error(), http.StatusInternalServerError)
					return
				}

				_, err = datastore.Put(ctx, h.Key, h)
				if err != nil {
					ctx.PrintError(err.Error(), http.StatusInternalServerError)
					return
				}

				ctx.PrintJSON(h.GetValue(), http.StatusOK)
			} else {
				if err = datastore.Delete(ctx, key); err != nil {
					ctx.PrintError(err.Error(), http.StatusNotFound)
					return
				}

				_ = Decrement(ctx, k.Path)

				ctx.PrintStatus(http.StatusText(http.StatusOK), http.StatusOK)
			}
		} else {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
	}
}
