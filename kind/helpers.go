package kind

import (
	"github.com/gorilla/mux"
	"net/http"
	"strings"
)

func CollectionMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		if val, ok := vars["key"]; ok {
			vars["collection"] = val
			delete(vars, "key")
		}
		if val, ok := vars["path"]; ok {
			path := strings.Split(val, "/")
			if len(path) > 0 {
				vars["kind"] = path[0]
				if len(path) > 1 {
					vars["key"] = path[1]
					if len(path) > 2 {
						vars["path"] = strings.Join(path[2:], "/")
					} else {
						delete(vars, "path")
					}
				} else {
					delete(vars, "path")
				}
			} else {
				delete(vars, "path")
			}
		}
		h.ServeHTTP(w, mux.SetURLVars(r, vars))
	})
}
