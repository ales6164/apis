package apis

import (
	"net/http"
	"github.com/gorilla/mux"
	"google.golang.org/appengine/datastore"
	"github.com/ales6164/go-cms/kind"
)

func (a *Apis) GetHandler(e *kind.Kind) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := NewContext(r)

		vars := mux.Vars(r)
		id := vars["id"]

		key, err := datastore.DecodeKey(id)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		h, err := e.Get(ctx, key)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		ctx.PrintResult(w, h.Output())
	}
}

func (a *Apis) AddHandler(e *kind.Kind) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := NewContext(r)

		h := e.NewHolder(ctx, ctx.UserKey)
		h.ParseInput(ctx.Body())

		err := h.Add()
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		ctx.PrintResult(w, h.Output())
	}
}

/*

func (a *App) KindUpdateHandler(k *kind.Kind) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, h, err := NewContext(r).Parse(k)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		vars := mux.Vars(r)
		id := vars["id"]

		key, err := datastore.DecodeKey(id)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		err = h.Update(key)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		ctx.PrintResult(w, h.Output())
	}
}

func (a *App) KindDeleteHandler(k *kind.Kind) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, h, err := NewContext(r).Parse(k)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		vars := mux.Vars(r)
		id := vars["id"]

		key, err := datastore.DecodeKey(id)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		err = h.Delete(key)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		ctx.PrintResult(w, h.Output())
	}
}
*/
