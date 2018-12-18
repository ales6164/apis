package apis

import (
	"github.com/gorilla/context"
	"github.com/gorilla/mux"
	"net/http"
)

func UserKeyMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		vars["key"] = context.Get(r, "userKey").(string)
		h.ServeHTTP(w, mux.SetURLVars(r, vars))
	})
}
