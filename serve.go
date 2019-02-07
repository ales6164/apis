package apis

import (
	"github.com/ales6164/apis/collection"
	"github.com/ales6164/apis/kind"
	"google.golang.org/appengine/datastore"
	"net/http"
	"strconv"
	"strings"
)

func (a *Apis) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := getPath(r.URL.Path)

	rules := a.Rules

	ctx := a.NewContext(w, r)

	var group kind.Doc
	var document kind.Doc

	// analyse path in pairs
	for i := 0; i < len(path); i += 2 {
		// get collection kind and match it to rules
		if k, ok := a.kinds[path[i]]; ok && rules != nil {
			if rules, ok = rules.Match[k]; ok {
				// got latest rules

				// create key
				var key *datastore.Key
				if (i + 1) < len(path) {
					key = k.Key(ctx, path[i+1], ctx.Member())
					if key == nil {
						ctx.PrintError("error decoding key", http.StatusBadRequest)
						return
					}
				}

				document = k.Doc(ctx, key, group)
				if rules.AccessControl {
					document.SetAccessControl(rules.AccessControl)
					group = document
				}

				continue
			}
		}
		ctx.PrintError(http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	// 1. Variable "lastAccessCheckDocument" saves the latest collection document that has defined "EnableAccessControl" under rules
	// 2. Use that variable in method switch cases to check for the new HasAccess function (kind.Document) instead of general group access check

	var err error

	switch r.Method {
	case http.MethodGet:
		// check rules
		if ok := ctx.HasAccess(*rules, ReadOnly, ReadWrite, FullControl); !ok {
			ctx.PrintError(http.StatusText(http.StatusForbidden), http.StatusForbidden)
			/*if ctx.authError != nil {
				ctx.PrintError(ctx.authError.Error(), http.StatusInternalServerError)
				return
			}
			if ctx.sessError != nil {
				ctx.PrintError(ctx.sessError.Error(), http.StatusInternalServerError)
				return
			}
			ctx.PrintJSON(ctx.session, http.StatusForbidden)*/
			return
		}

		// check access
		collection.CheckAccess(document)
		if ok := document.HasAccess(ctx.Member(), ReadOnly, ReadWrite, FullControl); !ok {
			ctx.PrintError(http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
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
			ctx.PrintJSON(document.Kind().Data(document, ctx.hasIncludeMetaHeader), http.StatusOK)
		} else {
			queryResults, err := Query(document, ctx.r, ctx.r.URL.Query())
			if err != nil {
				ctx.PrintError(err.Error(), http.StatusBadRequest)
				return
			}

			ctx.PrintJSON(map[string]interface{}{
				"value":      queryResults.Items,
				"lastUpdate": queryResults.UpdatedAt.String(),
			}, queryResults.StatusCode, "X-Total-Count", strconv.Itoa(queryResults.Total), "Link", queryResults.LinkHeader)
		}
	case http.MethodPost:
		// check rules
		if ok := ctx.HasAccess(*rules, ReadWrite, FullControl); !ok {
			ctx.PrintError(http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}

		// check group access
		if document.HasAncestor() {
			if ok := document.Ancestor().HasAccess(ctx.Member(), ReadWrite, FullControl); !ok {
				ctx.PrintError(http.StatusText(http.StatusForbidden), http.StatusForbidden)
				return
			}
		}

		document.SetOwner(ctx.Member())

		document, err = document.Add(ctx.Body())
		if err != nil {
			ctx.PrintError(err.Error(), http.StatusInternalServerError)
			return
		}
		err = document.SetAccess(ctx.Member(), FullControl)
		if err != nil {
			ctx.PrintError(err.Error(), http.StatusInternalServerError)
			return
		}

		ctx.PrintJSON(document.Kind().Data(document, ctx.hasIncludeMetaHeader), http.StatusOK)
	case http.MethodDelete:
		// check rules
		if ok := ctx.HasAccess(*rules, Delete, FullControl); !ok {
			ctx.PrintError(http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}

		// check group access
		if document.HasAncestor() {
			if ok := document.Ancestor().HasAccess(ctx.Member(), Delete, FullControl); !ok {
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
		if ok := ctx.HasAccess(*rules, ReadWrite, FullControl); !ok {
			ctx.PrintError(http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}

		// check group access
		if document.HasAncestor() {
			if ok := document.Ancestor().HasAccess(ctx.Member(), ReadWrite, FullControl); !ok {
				ctx.PrintError(http.StatusText(http.StatusForbidden), http.StatusForbidden)
				return
			}
		}

		document.SetOwner(ctx.Member())

		if document.Key().Incomplete() {
			ctx.PrintError(http.StatusText(http.StatusNotImplemented), http.StatusNotImplemented)
		} else {
			document, err = document.Set(ctx.Body())
			if err != nil {
				ctx.PrintError(err.Error(), http.StatusInternalServerError)
				return
			}
			err = document.SetAccess(ctx.Member(), FullControl)
			if err != nil {
				ctx.PrintError(err.Error(), http.StatusInternalServerError)
				return
			}
			ctx.PrintJSON(document.Kind().Data(document, ctx.hasIncludeMetaHeader), http.StatusOK)
		}
	default:
		ctx.PrintError(http.StatusText(http.StatusNotImplemented), http.StatusNotImplemented)

	}
}

func getPath(p string) []string {
	if p[:1] == "/" {
		p = p[1:]
	}
	return strings.Split(p, "/")
}
