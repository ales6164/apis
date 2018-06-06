package apis

import (
	"google.golang.org/appengine/datastore"
	"time"
	"github.com/gorilla/mux"
	"net/http"
	"github.com/ales6164/apis/kind"
	"reflect"
	"github.com/ales6164/apis/errors"
	"strings"
	"strconv"
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
	Group     *datastore.Key `json:"group"`
	Message   string         `datastore:",noindex" json:"message"`
	Read      bool           `json:"read"`
	User      BasicUser      `datastore:"-" json:"user"`
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

		hs, err := R.kind.Query(ctx, "-UpdatedAt", 20, 0, []kind.Filter{{FilterStr: "Users =", Value: ctx.UserKey()}}, nil)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		var out []*ChatGroup
		for _, h := range hs {
			value := h.Value().(*ChatGroup)

			if len(value.Name) == 0 {
				// populate with generated name
				var usrsToF []*datastore.Key
				for _, k := range value.Users {
					if !k.Equal(ctx.UserKey()) {
						usrsToF = append(usrsToF, k)
					}
				}

				var usrs = make([]*Account, len(usrsToF))
				err := datastore.GetMulti(ctx, usrsToF, usrs)
				if err != nil {
					ctx.PrintError(w, err)
					return
				}

				for i, u := range usrs {
					if i > 0 {
						value.Name += ", "
					}
					value.Name += strings.Join([]string{u.User.Profile.Name, u.User.Profile.GivenName, u.User.Profile.FamilyName}, " ")
				}
			}

			out = append(out, value)
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

type BasicUser struct {
	Id      *datastore.Key `json:"id"`
	Name    string         `json:"name"`
	Picture string         `json:"picture"`
}

func getChatGroupMessagesHandler(R *Route) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := R.NewContext(r)

		if ok, _ := ctx.HasPermission(R.kind, READ); !ok {
			ctx.PrintError(w, errors.ErrForbidden)
			return
		}

		id, offset := r.FormValue("id"), r.FormValue("offset")
		groupKey, err := datastore.DecodeKey(id)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		offsetInt, _ := strconv.Atoi(offset)

		// get chat group
		var chatGroup = new(ChatGroup)
		err = datastore.Get(ctx, groupKey, chatGroup)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		// get chat group users
		var users = map[string]BasicUser{}
		var chatGroupUsers = make([]*Account, len(chatGroup.Users))
		err = datastore.GetMulti(ctx, chatGroup.Users, chatGroupUsers)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}
		for i, u := range chatGroupUsers {
			users[u.Email] = BasicUser{
				Name:    strings.Join([]string{u.User.Profile.Name, u.User.Profile.GivenName, u.User.Profile.FamilyName}, " "),
				Picture: u.User.Profile.Picture,
				Id:      chatGroup.Users[i],
			}
		}

		// get messages
		hs, err := R.kind.Query(ctx, "-CreatedAt", 20, offsetInt, []kind.Filter{{FilterStr: "Group =", Value: groupKey}}, nil)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		var out []*Message
		for _, h := range hs {
			value := h.Value().(*Message)
			if uk, ok := users[value.CreatedBy.StringID()]; ok {
				value.User = uk
			}
			out = append(out, value)
		}
		ctx.PrintResult(w, map[string]interface{}{
			"count":   len(out),
			"results": out,
		})
	}
}

func createChatGroupMessageHandler(R *Route) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := R.NewContext(r)

		if ok, _ := ctx.HasPermission(R.kind, CREATE); !ok {
			ctx.PrintError(w, errors.ErrForbidden)
			return
		}

		id := r.URL.Query().Get("id")

		if len(id) == 0 {
			ctx.PrintError(w, errors.New("must provide id or name"))
			return
		}

		key, err := datastore.DecodeKey(id)
		if err != nil {
			ctx.PrintError(w, err, "decoding error")
			return
		}

		h := R.kind.NewHolder(ctx.UserKey())
		err = h.Parse(ctx.Body())
		if err != nil {
			ctx.PrintError(w, err, "parsing error")
			return
		}
		value := h.Value().(*Message)
		value.Group = key
		h.SetValue(value)

		err = h.Add(ctx)
		if err != nil {
			ctx.PrintError(w, err, "add error")
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
	messageRoute := &Route{
		kind:    MessageKind,
		a:       a,
		path:    "/chat/message",
		methods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete},
	}

	r.Handle(chatGroupRoute.path, a.middleware.Handler(getChatGroupsHandler(chatGroupRoute))).Methods(http.MethodGet)
	r.Handle(chatGroupRoute.path, a.middleware.Handler(createChatGroupHandler(chatGroupRoute))).Methods(http.MethodPost)
	//r.Handle(chatGroupRoute.path, a.middleware.Handler(chatGroupRoute.putHandler())).Methods(http.MethodPut)
	//r.Handle(chatGroupRoute.path, a.middleware.Handler(chatGroupRoute.deleteHandler())).Methods(http.MethodDelete)

	r.Handle(messageRoute.path, a.middleware.Handler(getChatGroupMessagesHandler(messageRoute))).Methods(http.MethodGet)
	r.Handle(messageRoute.path, a.middleware.Handler(createChatGroupMessageHandler(messageRoute))).Methods(http.MethodPost)
	/*r.Handle(messageRoute.path, a.middleware.Handler(messageRoute.putHandler())).Methods(http.MethodPut)
	r.Handle(messageRoute.path, a.middleware.Handler(messageRoute.deleteHandler())).Methods(http.MethodDelete)*/
}

// CHAT GROUP HANDLERS

// CHAT MESSAGE HANDLERS
