package apis

import "net/http"

type Handler struct {
	*Kind

	http.Handler
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.get(w, r)
	case http.MethodPost:
		h.post(w, r)
	case http.MethodPut:
		h.put(w, r)
	case http.MethodDelete:
		h.delete(w, r)
	}
}
