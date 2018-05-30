package apis

import (
	"google.golang.org/appengine/datastore"
	"time"
	"github.com/gorilla/mux"
	"net/http"
	"github.com/ales6164/apis/kind"
	"reflect"
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
	Users     []*datastore.Key `json:"users"`
}

type Message struct {
	Id        *datastore.Key `apis:"id" datastore:"-" json:"id"`
	CreatedAt time.Time      `apis:"createdAt" json:"createdAt"`
	CreatedBy *datastore.Key `apis:"createdBy" json:"createdBy"`
	UpdatedAt time.Time      `apis:"updatedAt" json:"updatedAt"`
	UpdatedBy *datastore.Key `apis:"updatedBy" json:"updatedBy"`
	ChatGroup *datastore.Key `json:"-"`
	Message   []byte         `datastore:",noindex"`
}

// todo: user access - some roles have access to users some dont - make that
// todo: add special field to users that makes them avaialable for chat to others ... and such

var ChatGroupKind = kind.New(reflect.TypeOf(ChatGroup{}), &kind.Options{
	Name: "_chat_group",
})

var MessageKind = kind.New(reflect.TypeOf(Message{}), &kind.Options{
	Name: "_chat_message",
})

func initChat(a *Apis, r *mux.Router) {
	chatGroupRoute := &Route{
		kind:    ChatGroupKind,
		a:       a,
		path:    "/apis/chat/group",
		methods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete},
	}
	messageRoute := &Route{
		kind:    MessageKind,
		a:       a,
		path:    "/apis/chat/message",
		methods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete},
	}

	r.Handle(chatGroupRoute.path, a.middleware.Handler(chatGroupRoute.getHandler())).Methods(http.MethodGet)
	r.Handle(chatGroupRoute.path, a.middleware.Handler(chatGroupRoute.postHandler())).Methods(http.MethodPost)
	r.Handle(chatGroupRoute.path, a.middleware.Handler(chatGroupRoute.putHandler())).Methods(http.MethodPut)
	r.Handle(chatGroupRoute.path, a.middleware.Handler(chatGroupRoute.deleteHandler())).Methods(http.MethodDelete)

	r.Handle(messageRoute.path, a.middleware.Handler(messageRoute.getHandler())).Methods(http.MethodGet)
	r.Handle(messageRoute.path, a.middleware.Handler(messageRoute.postHandler())).Methods(http.MethodPost)
	r.Handle(messageRoute.path, a.middleware.Handler(messageRoute.putHandler())).Methods(http.MethodPut)
	r.Handle(messageRoute.path, a.middleware.Handler(messageRoute.deleteHandler())).Methods(http.MethodDelete)
}

// CHAT GROUP HANDLERS

// CHAT MESSAGE HANDLERS
