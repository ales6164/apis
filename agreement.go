package apis

import (
	"time"
	"google.golang.org/appengine/datastore"
	"github.com/gorilla/mux"
	"net/http"
	"github.com/ales6164/apis/kind"
	"reflect"
)

type Agreement struct {
	User          *datastore.Key `json:"user"`
	UserEmail     string         `json:"email"` // if signed by anonymous user (requires email)
	ClientRequest ClientRequest  `json:"clientRequest"`
	Signed        bool           `json:"signed"`
}

type Contract struct {
	CreatedAt time.Time      `json:"createdAt"`
	CreatedBy *datastore.Key `json:"createdBy"`
	UpdatedBy *datastore.Key `json:"createdBy"`
	UpdatedAt time.Time      `json:"updatedAt"`
	Title     string         `datastore:",noindex" json:"title"`
	Content   []byte         `datastore:",noindex" json:"content"`
}

// privacy policy, tos or contract agreement
var AgreementKind = kind.New(reflect.TypeOf(Agreement{}), &kind.Options{
	Name: "_agreement",
})

var ContractKind = kind.New(reflect.TypeOf(Contract{}), &kind.Options{
	Name: "_contract",
})

func initAgreement(a *Apis, r *mux.Router) {
	agreementRoute := &Route{
		kind:    AgreementKind,
		a:       a,
		path:    "/apis/agreement",
		methods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete},
	}
	contractRoute := &Route{
		kind:    ContractKind,
		a:       a,
		path:    "/apis/contract",
		methods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete},
	}

	r.Handle(agreementRoute.path, a.middleware.Handler(agreementRoute.getHandler())).Methods(http.MethodGet)
	r.Handle(agreementRoute.path, a.middleware.Handler(agreementRoute.postHandler())).Methods(http.MethodPost)
	r.Handle(agreementRoute.path, a.middleware.Handler(agreementRoute.putHandler())).Methods(http.MethodPut)
	r.Handle(agreementRoute.path, a.middleware.Handler(agreementRoute.deleteHandler())).Methods(http.MethodDelete)

	r.Handle(contractRoute.path, a.middleware.Handler(contractRoute.getHandler())).Methods(http.MethodGet)
	r.Handle(contractRoute.path, a.middleware.Handler(contractRoute.postHandler())).Methods(http.MethodPost)
	r.Handle(contractRoute.path, a.middleware.Handler(contractRoute.putHandler())).Methods(http.MethodPut)
	r.Handle(contractRoute.path, a.middleware.Handler(contractRoute.deleteHandler())).Methods(http.MethodDelete)
}
