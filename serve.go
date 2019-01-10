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

	var isOk bool
	var namespace string
	var lastKind kind.Kind

	// analyse path in pairs
	for i := 0; i < len(path); i += 2 {
		isOk = false

		// get collection kind and match it to rules
		if k, ok := a.kinds[path[i]]; ok {
			if rules, ok = rules.Match[k]; ok {
				// got latest rules

				// is id provided in path?
				if (i + 1) < len(path) {

					id := path[i+1]
					key, err := datastore.DecodeKey(id)
					if err != nil {
						// id is not encoded key
						key =
					}


				} else {




				}

				isOk = true
			}
		}
		break
	}

	if isOk {

	} else {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
	}
}

func getPath(p string) []string {
	if p[:1] == "/" {
		p = p[1:]
	}
	return strings.Split(p, "/")
}
