package apis

import (
	"gopkg.in/ales6164/apis.v1/errors"
	"gopkg.in/ales6164/apis.v1/kind"
	"github.com/gorilla/mux"
	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"net/http"
	"reflect"
	"time"
)

type ChatGroup struct {
	Id        *datastore.Key   `search:"-" apis:"id" datastore:"-" json:"id"`
	CreatedAt time.Time        `search:"-" apis:"createdAt" json:"createdAt"`
	CreatedBy *datastore.Key   `search:"-" apis:"createdBy" json:"createdBy"`
	UpdatedAt time.Time        `search:"-" apis:"updatedAt" json:"updatedAt"`
	UpdatedBy *datastore.Key   `search:"-" apis:"updatedBy" json:"updatedBy"`
	Users     []string         `search:"-" json:"users"`
	UserKeys  []*datastore.Key `search:"-" json:"-"`
	GroupName string           `search:"-" json:"groupName"`
}

type Message struct {
	Id        *datastore.Key `search:"-" apis:"id" datastore:"-" json:"id"`
	CreatedAt time.Time      `search:"-" apis:"createdAt" json:"createdAt"`
	CreatedBy *datastore.Key `search:"-" apis:"createdBy" json:"createdBy"`
	UpdatedAt time.Time      `search:"-" apis:"updatedAt" json:"updatedAt"`
	UpdatedBy *datastore.Key `search:"-" apis:"updatedBy" json:"updatedBy"`
	Body      string         `search:"-" datastore:",noindex" json:"body"`
}

var ChatKind = kind.New(reflect.TypeOf(ChatGroup{}), &kind.Options{
	Name: "_chatGroup",
})

var messageKind = kind.New(reflect.TypeOf(Message{}), &kind.Options{
	Name: "_chatMessage",
})

// todo: implement with pub/sub

func getChatGroupsHandler(R *Route) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := R.NewContext(r)

		if ok, _ := ctx.HasPermission(ChatKind, GET); !ok {
			ctx.PrintError(w, errors.ErrForbidden)
			return
		}

		id := mux.Vars(r)["group"]
		if len(id) > 0 {
			// get messages for the group
			chatGroupKey, err := R.kind.DecodeKey(id)
			if err != nil {
				ctx.PrintError(w, err)
				return
			}

			if !chatGroupKey.Parent().Equal(ctx.UserKey()) {
				ctx.PrintError(w, errors.ErrForbidden)
				return
			}

			cgCtx, err := appengine.Namespace(ctx, "chatGroup_"+chatGroupKey.StringID())
			if err != nil {
				ctx.PrintError(w, err, "ns error")
				return
			}

			hs, err := messageKind.Query(cgCtx, "-CreatedAt", 0, 0, nil, nil)
			if err != nil {
				ctx.PrintError(w, err)
				return
			}

			var out []*Message
			for _, h := range hs {
				out = append(out, h.Value().(*Message))
			}
			ctx.PrintResult(w, map[string]interface{}{
				"count":   len(out),
				"results": out,
			})
		} else {
			// get groups
			hs, err := R.kind.Query(ctx, "-UpdatedAt", 0, 0, nil, ctx.UserKey())
			if err != nil {
				ctx.PrintError(w, err)
				return
			}

			var out []*ChatGroup
			for _, h := range hs {
				chatGroup := h.Value().(*ChatGroup)
				out = append(out, chatGroup)
			}
			ctx.PrintResult(w, map[string]interface{}{
				"count":   len(out),
				"results": out,
			})
		}
	}
}

func createChatGroupMessageHandler(R *Route) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := R.NewContext(r)

		if ok, _ := ctx.HasPermission(ChatKind, CREATE); !ok {
			ctx.PrintError(w, errors.ErrForbidden)
			return
		}

		id := mux.Vars(r)["group"]
		if len(id) == 0 {
			ctx.PrintError(w, errors.New("must provide id or name"))
			return
		}

		decodedGroupKey, err := ChatKind.DecodeKey(id)
		if err != nil {
			ctx.PrintError(w, err, "decoding error")
			return
		}

		if !decodedGroupKey.Parent().Equal(ctx.UserKey()) {
			ctx.PrintError(w, errors.ErrForbidden)
			return
		}

		// check if user has access to user group
		chatGroupHolder := ChatKind.NewHolder(ctx.UserKey())
		if err := chatGroupHolder.Get(ctx, decodedGroupKey); err != nil {
			ctx.PrintError(w, err, "parsing error")
			return
		}

		h := messageKind.NewHolder(ctx.UserKey())
		err = h.Parse(ctx.Body())
		if err != nil {
			ctx.PrintError(w, err, "parsing error")
			return
		}

		cgCtx, err := appengine.Namespace(ctx, "chatGroup_"+decodedGroupKey.StringID())
		if err != nil {
			ctx.PrintError(w, err, "ns error")
			return
		}
		h.SetKey(h.Kind.NewIncompleteKey(cgCtx, nil))
		err = h.Add(cgCtx)
		if err != nil {
			ctx.PrintError(w, err, "add error")
			return
		}

		// update chat group
		err = chatGroupHolder.Update(ctx)
		if err != nil {
			ctx.PrintError(w, err, "updating chat groupł error")
			return
		}

		ctx.Print(w, h.Value())
	}
}

func initChat(a *Apis, r *mux.Router) {
	chatGroupRoute := &Route{
		kind:    ChatKind,
		a:       a,
		path:    "/chat",
		methods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete},
	}
	messageRoute := &Route{
		kind:    messageKind,
		a:       a,
		path:    "/chat",
		methods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete},
	}

	r.Handle("/chat", a.middleware.Handler(getChatGroupsHandler(chatGroupRoute))).Methods(http.MethodGet)
	r.Handle("/chat/{group}", a.middleware.Handler(getChatGroupsHandler(chatGroupRoute))).Methods(http.MethodGet)
	r.Handle("/chat", a.middleware.Handler(chatGroupRoute.postHandler())).Methods(http.MethodPost)
	r.Handle("/chat/{group}", a.middleware.Handler(createChatGroupMessageHandler(messageRoute))).Methods(http.MethodPost)

	chatGroupRoute.On(BeforeCreate, func(ctx Context, h *kind.Holder) error {
		chatId := RandStringBytesMaskImprSrc(LetterBytes, 16)
		chatGroup := h.Value().(*ChatGroup)
		var hasSelf bool
		for _, u := range chatGroup.Users {
			decodedUserKey, err := datastore.DecodeKey(u)
			if err != nil {
				return err
			}
			chatGroup.UserKeys = append(chatGroup.UserKeys, decodedUserKey)
			if decodedUserKey.Equal(ctx.UserKey()) {
				hasSelf = true
			}
		}
		if !hasSelf {
			chatGroup.Users = append(chatGroup.Users, ctx.UserKey().Encode())
			chatGroup.UserKeys = append(chatGroup.UserKeys, ctx.UserKey())
			h.SetValue(chatGroup)
		}
		for _, u := range chatGroup.UserKeys {
			if !u.Equal(ctx.UserKey()) {
				h.SetKey(ChatKind.NewKey(ctx, chatId, u))
				err := h.Add(ctx)
				if err != nil {
					return err
				}
			}
		}

		h.SetKey(ChatKind.NewKey(ctx, chatId, ctx.UserKey()))
		return nil
	})
}

// CHAT GROUP HANDLERS

// CHAT MESSAGE HANDLERS
