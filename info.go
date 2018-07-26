package apis

import (
	"gopkg.in/ales6164/apis.v1/kind"
	"net/http"
)

// todo: add get, put, post, delete handlers
// todo: add simple search but index.put has to be delayed
// simple search output could be displayed in order that fields are defined
// todo: add "label" tag to display proper label with input

func infoHandler(R *Route) http.HandlerFunc {
	var isInited bool
	var kinds map[*kind.Kind]*kind.Kind
	var infos []*kind.KindInfo
	//var routes = map[*kind.Kind][]*Route{}
	var fun = func() {
		kinds = map[*kind.Kind]*kind.Kind{}
		for _, r := range R.a.routes {
			if r.kind != nil && kinds[r.kind] == nil {
				kinds[r.kind] = r.kind
			}
		}

		// get routes
		/*for _, r := range R.a.routes {
			if r.kind != nil {
				routes[r.kind] = append(routes[r.kind], r)
			}
		}*/

		for _, k := range kinds {
			infos = append(infos, k.Info())
		}

		isInited = true
	}

	return func(w http.ResponseWriter, r *http.Request) {
		ctx := R.NewContext(r)
		/*if !ctx.HasRole(AdminRole) {
			ctx.PrintError(w, errors.ErrUnathorized)
			return
		}*/

		if !isInited {
			fun()
		}

		// get all kinds

		ctx.PrintResult(w, map[string]interface{}{
			"kinds": infos,
		})
	}
}
