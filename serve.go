package apis

import (
	"golang.org/x/net/context"
	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"net/http"
	"strings"
)





func (a *Apis) serve(w http.ResponseWriter, r *http.Request) {
	path := getPath(r.URL.Path)

	rules := a.Rules

	ctx := a.NewContext(w, r)
	collector := NewCollector(ctx)

	// analyse path in pairs
	for i := 0; i < len(path); i += 2 {
		// get collection kind and match it to rules
		if k, ok := a.kinds[path[i]]; ok {
			if rules, ok = rules.Match[k]; ok {
				// got latest rules
				var err error

				if (i + 1) < len(path) {
					collector, err = collector.Fetch(k, path[i+1])
				} else {
					collector, err = collector.Fetch(k, "")
				}

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

	var ok bool
	if ctx, ok = ctx.WithSession(); !ok {
		return
	}

	c := appengine.NewContext(ctx.r)

	if collector.collection.entryKey != nil {
		if len(collector.collection.entry.ParentNamespace) > 0 {
			c, _ = appengine.Namespace(c, collector.collection.entry.ParentNamespace)
		}

		switch r.Method {
		case http.MethodGet:
			if ok := ctx.HasRole(rules.ReadOnly, rules.ReadWrite, rules.FullControl); ok {
				doc, err := collector.collection.kind.Doc(c, collector.collection.entryKey).Get()
				if err != nil {
					ctx.PrintError(err.Error(), http.StatusInternalServerError)
					return
				}
				ctx.PrintJSON(collector.collection.kind.Data(doc), http.StatusOK)
			} else {
				ctx.PrintError(http.StatusText(http.StatusForbidden), http.StatusForbidden)
			}
		}
	} else {
		switch r.Method {
		case http.MethodGet:
		case http.MethodPost:
			if ok := ctx.HasRole(rules.ReadWrite, rules.FullControl); ok {
				err := datastore.RunInTransaction(c, func(tc context.Context) error {
					doc, err := collector.collection.kind.Doc(tc, nil).Add(ctx.Body())
					if err != nil {
						return err
					} else {
						ctx.PrintJSON(collector.collection.kind.Data(doc), http.StatusOK)
					}
					return nil
				}, nil)
				if err != nil {
					ctx.PrintError(err.Error(), http.StatusInternalServerError)
				}
			} else {
				ctx.PrintError(http.StatusText(http.StatusForbidden), http.StatusForbidden)
			}
		}
	}

}

func getPath(p string) []string {
	if p[:1] == "/" {
		p = p[1:]
	}
	return strings.Split(p, "/")
}
