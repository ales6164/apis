package apis

import (
	"github.com/ales6164/apis/kind"
	"google.golang.org/appengine/datastore"
	"net/http"
	"strconv"
	"strings"
)

func (a *Apis) serve(w http.ResponseWriter, r *http.Request) {
	path := getPath(r.URL.Path)

	rules := a.Rules

	ctx := a.NewContext(w, r)

	var document kind.Doc

	// analyse path in pairs
	for i := 0; i < len(path); i += 2 {
		// get collection kind and match it to rules
		if k, ok := a.kinds[path[i]]; ok {
			if rules, ok = rules.Match[k]; ok {
				// got latest rules
				var err error

				// create key
				var key *datastore.Key
				if (i + 1) < len(path) {
					var err error
					key, err = datastore.DecodeKey(path[i+1])
					if err != nil {
						key = datastore.NewKey(ctx, k.Name(), path[i+1], 0, nil)
					}
				}

				document, err = k.Doc(ctx, key, document)

				if err != nil {
					ctx.PrintError(err.Error(), http.StatusBadRequest)
					return
				}
				continue
			}
		}
		ctx.PrintError(http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	//document.SetMember(ctx.Member(), ctx.session.isAuthenticated)

	// TODO: Check api.Rules for access
	// TODO: document.HasRole ...

	var err error

	switch r.Method {
	case http.MethodGet:
		// check rules
		if ok := ctx.HasAccess(rules, ReadOnly, ReadWrite, FullControl); !ok {
			ctx.PrintError(http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}

		// check group access
		if document.HasAncestor() {
			if ok := document.Ancestor().HasRole(ctx.Member(), ReadOnly, ReadWrite, FullControl); !ok {
				ctx.PrintError(http.StatusText(http.StatusForbidden), http.StatusForbidden)
				return
			}
		}

		if !document.Key().Incomplete() {
			document, err = document.Get()
			if err != nil {
				if err == datastore.ErrNoSuchEntity {
					ctx.PrintError(http.StatusText(http.StatusNotFound), http.StatusNotFound)
					return
				}
				ctx.PrintError(err.Error(), http.StatusInternalServerError)
				return
			}
			ctx.PrintJSON(document.Kind().Data(document), http.StatusOK)
		} else {
			queryResults, err := Query(document, ctx.r, ctx.r.URL.Query())
			if err != nil {
				ctx.PrintError(err.Error(), http.StatusBadRequest)
				return
			}

			ctx.PrintJSON(queryResults.Items, queryResults.StatusCode, "X-Total-Count", strconv.Itoa(queryResults.Total), "Link", queryResults.LinkHeader)
		}
	case http.MethodPost:
		// check rules
		if ok := ctx.HasAccess(rules, ReadWrite, FullControl); !ok {
			ctx.PrintError(http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}

		// check group access
		if document.HasAncestor() {
			if ok := document.Ancestor().HasRole(ctx.Member(), ReadWrite, FullControl); !ok {
				ctx.PrintError(http.StatusText(http.StatusForbidden), http.StatusForbidden)
				return
			}
		}

		if !document.Key().Incomplete() {
			ctx.PrintError(http.StatusText(http.StatusNotImplemented), http.StatusNotImplemented)
		} else {
			document, err = document.Add(ctx.Body())
			if err != nil {
				ctx.PrintError(err.Error(), http.StatusInternalServerError)
				return
			}
			err = document.SetRole(ctx.Member(), FullControl)
			if err != nil {
				ctx.PrintError(err.Error(), http.StatusInternalServerError)
				return
			}
			ctx.PrintJSON(document.Kind().Data(document), http.StatusOK)
		}
	case http.MethodDelete:
		// check rules
		if ok := ctx.HasAccess(rules, Delete, FullControl); !ok {
			ctx.PrintError(http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}

		// check group access
		if document.HasAncestor() {
			if ok := document.Ancestor().HasRole(ctx.Member(), Delete, FullControl); !ok {
				ctx.PrintError(http.StatusText(http.StatusForbidden), http.StatusForbidden)
				return
			}
		}

		if !document.Key().Incomplete() {
			ctx.PrintError(http.StatusText(http.StatusNotImplemented), http.StatusNotImplemented)
		} else {
			err = document.Delete()
			if err != nil {
				ctx.PrintError(err.Error(), http.StatusInternalServerError)
				return
			}

			ctx.PrintStatus(http.StatusText(http.StatusOK), http.StatusOK)
		}

	case http.MethodPut:
		// check rules
		if ok := ctx.HasAccess(rules, ReadWrite, FullControl); !ok {
			ctx.PrintError(http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}

		// check group access
		if document.HasAncestor() {
			if ok := document.Ancestor().HasRole(ctx.Member(), ReadWrite, FullControl); !ok {
				ctx.PrintError(http.StatusText(http.StatusForbidden), http.StatusForbidden)
				return
			}
		}

		if document.Key().Incomplete() {
			ctx.PrintError(http.StatusText(http.StatusNotImplemented), http.StatusNotImplemented)
		} else {
			document, err = document.Set(ctx.Body())
			if err != nil {
				ctx.PrintError(err.Error(), http.StatusInternalServerError)
				return
			}
			ctx.PrintJSON(document.Kind().Data(document), http.StatusOK)
		}
	default:
		ctx.PrintError(http.StatusText(http.StatusNotImplemented), http.StatusNotImplemented)

	}

	//collector.ServeContent(r.Method, rules)
}

/*func ContainsScope(arr []string, scopes ...string) bool {
	for _, scp := range scopes {
		for _, r := range arr {
			if r == scp {
				return true
			}
		}
	}
	return false
}*/

func getPath(p string) []string {
	if p[:1] == "/" {
		p = p[1:]
	}
	return strings.Split(p, "/")
}
