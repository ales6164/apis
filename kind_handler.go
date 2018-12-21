package apis

import (
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
/kinds/{key} GET, PUT, DELETE
/kinds/{key}/{path} GET, PUT, DELETE
 */

func (k *Kind) QueryHandler(ctx Context) {
	queryResults, err := k.Query(ctx, ctx.r.URL.Query())
	if err != nil {
		ctx.PrintError(err.Error(), http.StatusBadRequest)
		return
	}
	ctx.PrintJSON(queryResults.Items, queryResults.StatusCode, "X-Total-Count", strconv.Itoa(queryResults.Total), "Link", queryResults.LinkHeader)
}

func (k *Kind) GetHandler(ctx Context, key *datastore.Key, path ...string) {
	h, err := k.Get(ctx, key)
	if err != nil {
		if err == datastore.ErrNoSuchEntity {
			ctx.PrintError(http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}
		ctx.PrintError(err.Error(), http.StatusInternalServerError)
		return
	}
	if len(path) > 0 {
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
}

func (k *Kind) PostHandler(ctx Context, key *datastore.Key, path ...string) {
	if key != nil || len(path) > 0 {
		ctx.PrintError(http.StatusText(http.StatusNotImplemented), http.StatusNotImplemented)
		return
	}

	h := k.NewHolder(nil)
	if err := h.Parse(ctx.Body()); err != nil {
		ctx.PrintError(err.Error(), http.StatusInternalServerError)
		return
	}

	var name = k.dsNameGenerator(ctx, h)
	h.Key = datastore.NewKey(ctx, k.Name, name, 0, nil)

	err := datastore.RunInTransaction(ctx, func(tc context.Context) error {
		err := k.Create(tc, h)
		if err != nil {
			return err
		}

		err = OwnerIAM(tc, ctx.Member(), h.Key)
		if err != nil {
			return err
		}

		return IncrementTransactionless(tc, k.Name)
	}, &datastore.TransactionOptions{XG: true})
	if err != nil {
		ctx.PrintError(err.Error(), http.StatusInternalServerError)
		return
	}

	var location string
	locationUrl, err := mux.CurrentRoute(ctx.r).URL()
	if err == nil {
		location = strings.Join(append(strings.Split(locationUrl.Path, "/"), h.Key.Encode()), "/")
	}

	ctx.PrintJSON(h.GetValue(), http.StatusCreated, "Location", location)
}

func (k *Kind) PutHandler(ctx Context, key *datastore.Key, path ...string) {
	if key == nil {
		ctx.PrintError(http.StatusText(http.StatusNotImplemented), http.StatusNotImplemented)
		return
	}

	var err error
	var h *Holder
	err = datastore.RunInTransaction(ctx, func(tc context.Context) error {
		h, err = k.Get(tc, key)
		if err != nil {
			return err
		}
		h, err = h.Set(ctx, path, ctx.Body())
		if err != nil {
			return err
		}
		h.Key, err = datastore.Put(ctx, h.Key, h)
		return err
	}, nil)
	if err != nil {
		if err == datastore.ErrNoSuchEntity {
			ctx.PrintError(http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}
		ctx.PrintError(err.Error(), http.StatusInternalServerError)
		return
	}

	ctx.PrintJSON(h.GetValue(), http.StatusOK)
}

func (k *Kind) DeleteHandler(ctx Context, key *datastore.Key, path ...string) {
	if key == nil {
		ctx.PrintError(http.StatusText(http.StatusNotImplemented), http.StatusNotImplemented)
		return
	}
	if len(path) > 0 {
		var err error
		var h *Holder
		err = datastore.RunInTransaction(ctx, func(tc context.Context) error {
			h, err = k.Get(tc, key)
			if err != nil {
				return err
			}
			h, err = h.Delete(ctx, path)
			if err != nil {
				return err
			}
			h.Key, err = datastore.Put(ctx, h.Key, h)
			return err
		}, nil)
		if err != nil {
			if err == datastore.ErrNoSuchEntity {
				ctx.PrintError(http.StatusText(http.StatusNotFound), http.StatusNotFound)
				return
			}
			ctx.PrintError(err.Error(), http.StatusInternalServerError)
			return
		}
		ctx.PrintJSON(h.GetValue(), http.StatusOK)
	} else {
		var err error
		err = datastore.RunInTransaction(ctx, func(tc context.Context) error {
			err = datastore.Delete(ctx, key)
			if err != nil {
				return err
			}
			return DecrementTransactionless(tc, k.Name)
		}, &datastore.TransactionOptions{XG: true})
		if err != nil {
			if err == datastore.ErrNoSuchEntity {
				ctx.PrintError(http.StatusText(http.StatusNotFound), http.StatusNotFound)
				return
			}
			ctx.PrintError(err.Error(), http.StatusInternalServerError)
			return
		}
		ctx.PrintStatus(http.StatusText(http.StatusOK), http.StatusOK)
	}

}
