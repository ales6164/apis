package apis

import (
	"github.com/ales6164/apis/collection"
	"github.com/ales6164/apis/iam"
	"google.golang.org/appengine/datastore"
	"net/http"
	"strconv"
	"strings"
)

func (a *Apis) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := getPath(r.URL.Path)

	rules := a.Rules

	ctx := a.IAM.NewContext(w, r)

	var parent *datastore.Key
	var group collection.Doc
	var document collection.Doc

	// analyse path in pairs
	for i := 0; i < len(path); i += 2 {
		// get collection kind and match it to rules
		if k, ok := a.kinds[path[i]]; ok && rules != nil {
			if rules, ok = rules.Match[k]; ok {
				// got latest rules

				// create key
				var key *datastore.Key
				if (i + 1) < len(path) {
					key = k.Key(ctx, path[i+1], parent, ctx.Member())
					if key == nil {
						ctx.PrintError("error decoding key", http.StatusBadRequest)
						return
					}
				}
				parent = key

				document = k.Doc(key, group)
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
		if ctx, ok := iam.CheckAccess(ctx, document, ctx.Member(), iam.ReadOnly, iam.ReadWrite, iam.FullControl); ok {
			if document.Key() != nil && !document.Key().Incomplete() {
				document, err = document.Get(ctx)
				if err != nil {
					if err == datastore.ErrNoSuchEntity {
						ctx.PrintError(http.StatusText(http.StatusNotFound), http.StatusNotFound)
						return
					}
					ctx.PrintError(err.Error(), http.StatusInternalServerError)
					return
				}
				ctx.PrintJSON(document.Kind().Data(document, ctx.HasIncludeMetaHeader, ctx.HasResolveMetaRefHeader), http.StatusOK)
			} else {
				queryResults, err := Query(ctx, document, r, r.URL.Query())
				if err != nil {
					ctx.PrintError(err.Error(), http.StatusBadRequest)
					return
				}

				if ctx.HasIncludeMetaHeader {
					ctx.PrintJSON(map[string]interface{}{
						"results":    queryResults.Items,
						"lastUpdate": queryResults.UpdatedAt.String(),
					}, queryResults.StatusCode, "X-Total-Count", strconv.Itoa(queryResults.Total), "Link", queryResults.LinkHeader)
				} else {
					ctx.PrintJSON(queryResults.Items, queryResults.StatusCode, "X-Total-Count", strconv.Itoa(queryResults.Total), "Link", queryResults.LinkHeader)
				}
			}
		} else {
			ctx.PrintError(http.StatusText(http.StatusForbidden), http.StatusForbidden)
		}
	case http.MethodPost:
		if ctx, ok := iam.CheckAccess(ctx, document, ctx.Member(), iam.ReadWrite, iam.FullControl); ok {
			document.SetOwner(ctx.Member())
			document, err = document.Add(ctx, ctx.Body())
			if err != nil {
				ctx.PrintError(err.Error(), http.StatusInternalServerError)
				return
			}
			err = iam.SetAccess(ctx, document, ctx.Member(), iam.FullControl)
			if err != nil {
				ctx.PrintError(err.Error(), http.StatusInternalServerError)
				return
			}
			ctx.PrintJSON(document.Kind().Data(document, ctx.HasIncludeMetaHeader, ctx.HasResolveMetaRefHeader), http.StatusOK)
		} else {
			ctx.PrintError(http.StatusText(http.StatusForbidden), http.StatusForbidden)
		}
	case http.MethodDelete:
		if ctx, ok := iam.CheckAccess(ctx, document, ctx.Member(), iam.Delete, iam.FullControl); ok {
			if !document.Key().Incomplete() {
				ctx.PrintError(http.StatusText(http.StatusNotImplemented), http.StatusNotImplemented)
			} else {
				err = document.Delete(ctx)
				if err != nil {
					ctx.PrintError(err.Error(), http.StatusInternalServerError)
					return
				}
				ctx.PrintStatus(http.StatusText(http.StatusOK), http.StatusOK)
			}
		} else {
			ctx.PrintError(http.StatusText(http.StatusForbidden), http.StatusForbidden)
		}
	case http.MethodPut:
		if ctx, ok := iam.CheckAccess(ctx, document, ctx.Member(), iam.ReadWrite, iam.FullControl); ok {
			document.SetOwner(ctx.Member())
			if document.Key() == nil || document.Key().Incomplete() {
				ctx.PrintError(http.StatusText(http.StatusNotImplemented), http.StatusNotImplemented)
			} else {
				document, err = document.Set(ctx, ctx.Body())
				if err != nil {
					ctx.PrintError(err.Error(), http.StatusInternalServerError)
					return
				}

				err = iam.SetAccess(ctx, document, ctx.Member(), iam.FullControl)
				if err != nil {
					ctx.PrintError(err.Error(), http.StatusInternalServerError)
					return
				}

				//ctx.PrintJSON(ctx.Context, 200)

				ctx.PrintJSON(document.Kind().Data(document, ctx.HasIncludeMetaHeader, ctx.HasResolveMetaRefHeader), http.StatusOK)
			}
		} else {
			ctx.PrintError(http.StatusText(http.StatusForbidden), http.StatusForbidden)
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
