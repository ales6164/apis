package apis

import (
	"github.com/ales6164/apis/collection"
	"github.com/ales6164/apis/iam"
	"github.com/ales6164/apis/kind"
	"google.golang.org/appengine/datastore"
	"net/http"
	"strconv"
	"strings"
)

func (a *Apis) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := getPath(r.URL.Path)

	rules := a.Rules

	defaultCtx := a.IAM.NewContext(w, r)

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
					key = k.Key(defaultCtx, path[i+1], defaultCtx.Member())
					if key == nil {
						defaultCtx.PrintError("error decoding key", http.StatusBadRequest)
						return
					}
				}

				document = k.Doc(key, group)
				if rules.AccessControl {
					document.SetAccessControl(rules.AccessControl)
					group = document
				}

				continue
			}
		}
		defaultCtx.PrintError(http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	// 1. Variable "lastAccessCheckDocument" saves the latest collection document that has defined "EnableAccessControl" under rules
	// 2. Use that variable in method switch cases to check for the new HasAccess function (kind.Document) instead of general group access check

	var err error

	switch r.Method {
	case http.MethodGet:
		if ctx, ok := collection.CheckAccess(defaultCtx, document, defaultCtx.Member(), iam.ReadOnly, iam.ReadWrite, iam.FullControl); ok {
			if !document.Key().Incomplete() {
				document, err = document.Get(ctx)
				if err != nil {
					if err == datastore.ErrNoSuchEntity {
						defaultCtx.PrintError(http.StatusText(http.StatusNotFound), http.StatusNotFound)
						return
					}
					defaultCtx.PrintError(err.Error(), http.StatusInternalServerError)
					return
				}
				defaultCtx.PrintJSON(document.Kind().Data(document, defaultCtx.HasIncludeMetaHeader), http.StatusOK)
			} else {
				queryResults, err := Query(defaultCtx, document, r, r.URL.Query())
				if err != nil {
					defaultCtx.PrintError(err.Error(), http.StatusBadRequest)
					return
				}

				defaultCtx.PrintJSON(map[string]interface{}{
					"value":      queryResults.Items,
					"lastUpdate": queryResults.UpdatedAt.String(),
				}, queryResults.StatusCode, "X-Total-Count", strconv.Itoa(queryResults.Total), "Link", queryResults.LinkHeader)
			}
		} else {
			defaultCtx.PrintError(http.StatusText(http.StatusForbidden), http.StatusForbidden)
		}
	case http.MethodPost:
		if ctx, ok := collection.CheckAccess(defaultCtx, document, defaultCtx.Member(), iam.ReadWrite, iam.FullControl); ok {
			document.SetOwner(defaultCtx.Member())
			document, err = document.Add(ctx, defaultCtx.Body())
			if err != nil {
				defaultCtx.PrintError(err.Error(), http.StatusInternalServerError)
				return
			}
			err = collection.SetAccess(defaultCtx, document, defaultCtx.Member(), iam.FullControl)
			if err != nil {
				defaultCtx.PrintError(err.Error(), http.StatusInternalServerError)
				return
			}
			defaultCtx.PrintJSON(document.Kind().Data(document, defaultCtx.HasIncludeMetaHeader), http.StatusOK)
		} else {
			defaultCtx.PrintError(http.StatusText(http.StatusForbidden), http.StatusForbidden)
		}
	case http.MethodDelete:
		if ctx, ok := collection.CheckAccess(defaultCtx, document, defaultCtx.Member(), iam.Delete, iam.FullControl); ok {
			if !document.Key().Incomplete() {
				defaultCtx.PrintError(http.StatusText(http.StatusNotImplemented), http.StatusNotImplemented)
			} else {
				err = document.Delete(ctx)
				if err != nil {
					defaultCtx.PrintError(err.Error(), http.StatusInternalServerError)
					return
				}
				defaultCtx.PrintStatus(http.StatusText(http.StatusOK), http.StatusOK)
			}

		} else {
			defaultCtx.PrintError(http.StatusText(http.StatusForbidden), http.StatusForbidden)
		}
	case http.MethodPut:
		if ctx, ok := collection.CheckAccess(defaultCtx, document, defaultCtx.Member(), iam.ReadWrite, iam.FullControl); ok {
			document.SetOwner(defaultCtx.Member())
			if document.Key().Incomplete() {
				defaultCtx.PrintError(http.StatusText(http.StatusNotImplemented), http.StatusNotImplemented)
			} else {
				document, err = document.Set(ctx, defaultCtx.Body())
				if err != nil {
					defaultCtx.PrintError(err.Error(), http.StatusInternalServerError)
					return
				}

				err = collection.SetAccess(defaultCtx, document, defaultCtx.Member(), iam.FullControl)
				if err != nil {
					defaultCtx.PrintError(err.Error(), http.StatusInternalServerError)
					return
				}

				defaultCtx.PrintJSON(document.Kind().Data(document, defaultCtx.HasIncludeMetaHeader), http.StatusOK)
			}
		} else {
			defaultCtx.PrintError(http.StatusText(http.StatusForbidden), http.StatusForbidden)
		}
	default:
		defaultCtx.PrintError(http.StatusText(http.StatusNotImplemented), http.StatusNotImplemented)

	}
}

func getPath(p string) []string {
	if p[:1] == "/" {
		p = p[1:]
	}
	return strings.Split(p, "/")
}
