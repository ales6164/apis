package apis

import (
	"github.com/ales6164/apis/kind"
	"google.golang.org/appengine/datastore"
	"net/http"
	"strings"
)

func (a *Apis) serve(w http.ResponseWriter, r *http.Request) {
	path := getPath(r.URL.Path)

	rules := a.Rules

	ctx := a.NewContext(w, r)
	var ok bool
	if ctx, ok = ctx.WithSession(); !ok {
		return
	}

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

				document = k.Doc(ctx, key, document)

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

	document.SetMember(ctx.Member(), ctx.session.isAuthenticated)

	var err error

	switch r.Method {
	case http.MethodGet:
		if document.Key() != nil {
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
			ctx.PrintError(http.StatusText(http.StatusNotImplemented), http.StatusNotImplemented)
		}
	case http.MethodPost:
		if document.Key() != nil {
			ctx.PrintError(http.StatusText(http.StatusNotImplemented), http.StatusNotImplemented)
		} else {
			document, err = document.Add(ctx.Body())
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

func getPath(p string) []string {
	if p[:1] == "/" {
		p = p[1:]
	}
	return strings.Split(p, "/")
}
