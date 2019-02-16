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

	var parentKey *datastore.Key

	var ancestorKey *datastore.Key
	var isAncestorAccessControl bool
	var namespace string

	//var group collection.Doc
	var document collection.Doc
	var accessController collection.Doc

	// analyse path in pairs
	for i := 0; i < len(path); i += 2 {
		// get collection kind and match it to rules
		if k, ok := a.kinds[path[i]]; ok && rules != nil {
			if rules, ok = rules.Match[k]; ok {
				// got latest rules

				// create key
				var key *datastore.Key
				var err error

				// load namespace
				if isAncestorAccessControl {
					// if this is true, we have parent key and now we can get group and change namespace

					// namespace change
					var groupKey = datastore.NewKey(ctx.Default(), "_group", ancestorKey.Encode(), 0, nil)
					var group = new(collection.Group)
					err = datastore.Get(ctx.Default(), groupKey, group)
					if err != nil {
						ctx.PrintError(http.StatusText(http.StatusConflict), http.StatusConflict)
						return
					}
					ctx, err = ctx.SetNamespace(group.Namespace)
					if err != nil {
						ctx.PrintError(http.StatusText(http.StatusConflict), http.StatusConflict)
						return
					}
					namespace = group.Namespace

					if parentKey.Namespace() != namespace {
						// becase can't have parent with different ns
						parentKey = nil
					}
				}

				if (i + 1) < len(path) {
					var id = path[i+1]

					key, err = datastore.DecodeKey(id)
					if err != nil {
						key = datastore.NewKey(ctx.Context, k.Name(), id, 0, parentKey)
						if key.Namespace() != namespace {
							ctx.PrintError(key.Namespace(), http.StatusConflict)
							return
						}
					} else {
						if parentKey != nil {
							if !parentKey.Equal(key) {
								ctx.PrintError(http.StatusText(http.StatusConflict), http.StatusConflict)
								return
							}
						}
						if key.Namespace() != namespace {
							ctx.PrintError(http.StatusText(http.StatusConflict), http.StatusConflict)
							return
						}
					}

					// key should be okay by this point
				}
				parentKey = key

				// todo: create key from id and parent and rules.AccessControl (retrieve group namespace)
				// todo: pass on group namespace inside Doc (get rid of access controller) - or maybe not?
				// todo: check for access (retrieve _rel) on operations (also save/delete _rel when adding/deleting)
				// todo: if group creator (doc) as current and is being deleted also delete everything inside group?

				document = k.Doc(key, document)
				if rules.AccessControl {
					accessController = document
					ancestorKey = key
					isAncestorAccessControl = rules.AccessControl
				}
				continue
			}
		}
		ctx.PrintError(http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	var userScopes []string
	var err error
	userScopes, err = iam.GetScopes(ctx, accessController, rules.Permissions)
	if err != nil {
		ctx.PrintError(http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}

	// 1. If doc has access control ancestor or itself is access controller, then retrieve _rel
	// 2. Store _rel roles AND rules.Permissions inside some object which is then used to check access

	switch r.Method {
	case http.MethodGet:
		if ok := ContainsScope(userScopes, iam.ReadOnly, iam.ReadWrite, iam.FullControl); ok {
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
		if ok := ContainsScope(userScopes, iam.ReadWrite, iam.FullControl); ok {
			document, err = document.Add(ctx, ctx.Body(), ctx.Member())
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
		if ok := ContainsScope(userScopes, iam.Delete, iam.FullControl); ok {
			//TODO: delete groups and group relationships
			if document.Key() == nil || document.Key().Incomplete() {
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
		if ok := ContainsScope(userScopes, iam.ReadWrite, iam.FullControl); ok {
			if document.Key() == nil || document.Key().Incomplete() {
				ctx.PrintError(http.StatusText(http.StatusNotImplemented), http.StatusNotImplemented)
			} else {
				document, err = document.Set(ctx, ctx.Body(), ctx.Member())
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
