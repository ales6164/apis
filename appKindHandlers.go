package apis

import (
	"net/http"
	"github.com/gorilla/mux"
	"google.golang.org/appengine/datastore"
	"github.com/ales6164/apis/kind"
)

func (a *Apis) QueryHandler(e *kind.Kind) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := a.NewContext(r)
		ctx, err := ctx.HasPermission(e, Read)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		hs, err := e.Query(ctx, "", 100, 0)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		var out []map[string]interface{}
		for _, h := range hs {
			out = append(out, h.Output())
		}

		ctx.PrintResult(w, out)
	}
}

func (a *Apis) GetHandler(e *kind.Kind) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := a.NewContext(r)

		vars := mux.Vars(r)
		id := vars["id"]

		key, err := datastore.DecodeKey(id)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		ctx, err = ctx.HasPermission(e, Read)
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

/*
func getGroup(ctx Context, group string, ) (error) {
	groupKey, err := datastore.DecodeKey(group)
	if err != nil {
		return err
	}

	ctx, err = ctx.SetGroup(group)
	if err != nil {
		ctx.PrintError(w, err)
		return
	}
}*/

func (a *Apis) AddHandler(e *kind.Kind) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := a.NewContext(r)

		h := e.NewHolder(ctx, ctx.UserKey)
		err := h.ParseInput(ctx.Body())
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		ctx, err = ctx.HasPermission(e, Create)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		err = h.Add()
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
