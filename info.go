package apis

import (
	"net/http"
	"github.com/ales6164/apis/errors"
	"github.com/ales6164/apis/kind"
)

// todo: add get, put, post, delete handlers
// todo: add simple search but index.put has to be delayed
// simple search output could be displayed in order that fields are defined
// todo: add "label" tag to display proper label with input

func infoHandler(R *Route) http.HandlerFunc {
	var isInited bool
	var kinds map[*kind.Kind]*kind.Kind
	var fun = func() {
		kinds = map[*kind.Kind]*kind.Kind{}
		for _, r := range R.a.routes {
			if r.kind != nil && kinds[r.kind] == nil {
				kinds[r.kind] = r.kind
			}
		}
		isInited = true
	}
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := R.NewContext(r)
		if !ctx.HasRole(AdminRole) {
			ctx.PrintError(w, errors.ErrUnathorized)
			return
		}

		if !isInited {
			fun()
		}

		// get all kinds
		for _, k := range R.a.kinds {
			k.Type.
		}

		ctx.Print(w, nil)
	}
}
