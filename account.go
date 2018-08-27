package apis

import (
	"reflect"
	"github.com/ales6164/apis/kind"
	"github.com/ales6164/apis/providers"
	"github.com/gorilla/mux"
	"net/http"
	"github.com/ales6164/apis/errors"
)

var accountKind = kind.New(reflect.TypeOf(providers.Account{}), &kind.Options{
	Name: "_account",
})

var UserKind *kind.Kind

func initUser(a *Apis, r *mux.Router) {
	userRoute := &Route{
		a:       a,
		path:    "/user",
		methods: []string{},
	}

	r.Handle("/user", a.middleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := userRoute.NewContext(r)
		if ok := ctx.IsAuthenticated; !ok {
			ctx.PrintError(w, errors.ErrUnathorized)
			return
		}

		h := UserKind.NewHolder(nil)
		if err := h.Get(ctx, ctx.UserKey()); err != nil {
			ctx.PrintError(w, err)
			return
		}

		ctx.Print(w, h.Value())
	}))).Methods(http.MethodGet)
	r.Handle("/user/{id}", a.middleware.Handler(userRoute.getHandler())).Methods(http.MethodGet)
	r.Handle("/user", a.middleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := userRoute.NewContext(r)
		if ok := ctx.IsAuthenticated; !ok {
			ctx.PrintError(w, errors.ErrUnathorized)
			return
		}

		h := UserKind.NewHolder(nil)
		if err := h.Parse(ctx.Body()); err != nil {
			ctx.PrintError(w, err)
			return
		}
		h.SetKey(ctx.UserKey())
		if err := h.Update(ctx); err != nil {
			ctx.PrintError(w, err)
			return
		}
		ctx.Print(w, h.Value())
	}))).Methods(http.MethodPut)
	r.Handle("/user/{id}", a.middleware.Handler(userRoute.putHandler())).Methods(http.MethodPut)
}
