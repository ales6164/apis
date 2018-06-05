package apis

import (
	"google.golang.org/appengine/datastore"
	"time"
	"github.com/gorilla/mux"
	"net/http"
	"github.com/ales6164/apis/kind"
	"reflect"
	"github.com/ales6164/apis/errors"
)

type ChatOptions struct {
	Enabled            bool
	MessagePermissions Permissions
	GroupPermissions   Permissions
}

type ChatGroup struct {
	Id        *datastore.Key   `apis:"id" datastore:"-" json:"id"`
	CreatedAt time.Time        `apis:"createdAt" json:"createdAt"`
	CreatedBy *datastore.Key   `apis:"createdBy" json:"createdBy"`
	UpdatedAt time.Time        `apis:"updatedAt" json:"updatedAt"`
	UpdatedBy *datastore.Key   `apis:"updatedBy" json:"updatedBy"`
	Name      string           `json:"name"`
	Users     []*datastore.Key `json:"-"`
}

// parent is chat group
type Message struct {
	Id        *datastore.Key `apis:"id" datastore:"-" json:"id"`
	CreatedAt time.Time      `apis:"createdAt" json:"createdAt"`
	CreatedBy *datastore.Key `apis:"createdBy" json:"createdBy"`
	UpdatedAt time.Time      `apis:"updatedAt" json:"updatedAt"`
	UpdatedBy *datastore.Key `apis:"updatedBy" json:"updatedBy"`
	Message   []byte         `datastore:",noindex" json:"message"`
	Read      bool           `json:"read"`
}

// todo: user access - some roles have access to users some dont - make that
// todo: add special field to users that makes them avaialable for chat to others ... and such

var ChatGroupKind = kind.New(reflect.TypeOf(ChatGroup{}), &kind.Options{
	Name: "_chatGroup",
})

var MessageKind = kind.New(reflect.TypeOf(Message{}), &kind.Options{
	Name: "_chatMessage",
})

func getChatGroupsHandler(R *Route) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := R.NewContext(r)

		if ok, _ := ctx.HasPermission(R.kind, READ); !ok {
			ctx.PrintError(w, errors.ErrForbidden)
			return
		}

		hs, err := R.kind.Query(ctx, "-UpdatedAt", 1000, 0, []kind.Filter{{FilterStr: "Users =", Value: ctx.UserKey()}}, nil)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		var out []interface{}
		for _, h := range hs {
			out = append(out, h.Value())
		}
		ctx.PrintResult(w, map[string]interface{}{
			"count":   len(out),
			"results": out,
		})
	}
}

func createChatGroupHandler(R *Route) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := R.NewContext(r)
		if ok, _ := ctx.HasPermission(R.kind, CREATE); !ok {
			ctx.PrintError(w, errors.ErrForbidden)
			return
		}

		h := R.kind.NewHolder(ctx.UserKey())
		err := h.Parse(ctx.Body())
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		val := h.Value().(*ChatGroup)
		val.Users = append(val.Users, ctx.UserKey())
		h.SetValue(val)

		err = h.Add(ctx)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		ctx.Print(w, h.Value())
	}
}

func initChat(a *Apis, r *mux.Router) {
	chatGroupRoute := &Route{
		kind:    ChatGroupKind,
		a:       a,
		path:    "/chat/group",
		methods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete},
	}
	/*messageRoute := &Route{
		kind:    MessageKind,
		a:       a,
		path:    "/apis/chat/message",
		methods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete},
	}*/

	r.Handle(chatGroupRoute.path, a.middleware.Handler(getChatGroupsHandler(chatGroupRoute))).Methods(http.MethodGet)
	r.Handle(chatGroupRoute.path, a.middleware.Handler(createChatGroupHandler(chatGroupRoute))).Methods(http.MethodPost)
	//r.Handle(chatGroupRoute.path, a.middleware.Handler(chatGroupRoute.putHandler())).Methods(http.MethodPut)
	//r.Handle(chatGroupRoute.path, a.middleware.Handler(chatGroupRoute.deleteHandler())).Methods(http.MethodDelete)

	/*r.Handle(messageRoute.path, a.middleware.Handler(messageRoute.getHandler())).Methods(http.MethodGet)
	r.Handle(messageRoute.path, a.middleware.Handler(messageRoute.postHandler())).Methods(http.MethodPost)
	r.Handle(messageRoute.path, a.middleware.Handler(messageRoute.putHandler())).Methods(http.MethodPut)
	r.Handle(messageRoute.path, a.middleware.Handler(messageRoute.deleteHandler())).Methods(http.MethodDelete)*/
}

// CHAT GROUP HANDLERS

// CHAT MESSAGE HANDLERS
