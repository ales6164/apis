package apis

import (
	"github.com/ales6164/apis/kind"
	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"net/http"
	"strings"
)

func (a *Apis) serve(w http.ResponseWriter, r *http.Request) {
	path := getPath(r.URL.Path)

	rules := a.Rules

	var isOk bool

	var lastKind kind.Kind
	var lastKey *datastore.Key
	var lastGroupKey *datastore.Key
	var lastGroupId string
	var isAfterFirst bool

	ctx := appengine.NewContext(r)

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


						if isAfterFirst {
							// this key is inside a group -- check namespace

							lastGroupKey, lastGroupId, err = getGroupId(ctx, lastGroupKey, key)
							if err != nil {
								http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
								return
							}

							// todo:

							ctx, err = appengine.Namespace(ctx, lastGroupId)
							if err != nil {
								http.Error(w, err.Error(), http.StatusInternalServerError)
								return
							}
						} else {
							key = datastore.NewKey(ctx, k.Name(), id, 0, nil)
						}
					} else if isAfterFirst {
						// check namespace

						lastGroupKey, lastGroupId, err = getGroupId(ctx, lastGroupKey, key)
						if err != nil {
							http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
							return
						}

						ctx, err = appengine.Namespace(ctx, lastGroupId)
						if err != nil {
							http.Error(w, err.Error(), http.StatusInternalServerError)
							return
						}
					} else if len(key.Namespace()) > 0 {
						// key should not have namespace defined
						break
					}

					lastKey = key
				} else {
					// this request is without final id

				}

				isAfterFirst = true
				lastKind = k
				isOk = true
				continue
			}
		}
		break
	}

	if isOk {
		// todo:
		// check if has group access
		// check rules


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
