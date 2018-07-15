package apis

/*type Agreement struct {
	User          *datastore.Key `search:"-" json:"user"`
	UserEmail     string         `search:"-" json:"email"` // if signed by anonymous user (requires email)
	ClientRequest ClientRequest  `search:"-" json:"clientRequest"`
	Signed        bool           `search:"-" json:"signed"`
}

type Contract struct {
	CreatedAt time.Time      `search:"-" json:"createdAt"`
	CreatedBy *datastore.Key `search:"-" json:"createdBy"`
	UpdatedBy *datastore.Key `search:"-" json:"createdBy"`
	UpdatedAt time.Time      `search:"-" json:"updatedAt"`
	Title     string         `search:"-" datastore:",noindex" json:"title"`
	Content   []byte         `search:"-" datastore:",noindex" json:"content"`
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
		path:    "/agreement",
		methods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete},
	}
	contractRoute := &Route{
		kind:    ContractKind,
		a:       a,
		path:    "/contract",
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
*/