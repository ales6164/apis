package apis

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/dgrijalva/jwt-go"
	"io/ioutil"
	"github.com/ales6164/apis/middleware"
	"github.com/ales6164/apis/kind"
)

type Apis struct {
	options    *Options
	router     *mux.Router
	Handler    http.Handler
	middleware *middleware.JWTMiddleware
	privateKey []byte
	kinds      map[string]*kind.Kind
	permissions
}

type Options struct {
	HandlerPathPrefix     string
	PrivateKeyPath        string // for password hashing
	Kinds                 []*kind.Kind
	AuthorizedOrigins     []string
	AllowUserRegistration bool
	DefaultRole           Role
	UserProfileKind       *kind.Kind
	RequireTrackingID     bool // todo:generated from pages - track users - stored as session cookie
	Permissions
}

func New(opt *Options) (*Apis, error) {
	a := &Apis{
		options: opt,
		kinds:   map[string]*kind.Kind{},
		router:  mux.NewRouter().PathPrefix(opt.HandlerPathPrefix).Subrouter(),
	}

	// read private key
	var err error
	a.privateKey, err = ioutil.ReadFile(opt.PrivateKeyPath)
	if err != nil {
		return a, err
	}

	// parse permissions
	a.permissions, err = a.options.Permissions.parse()
	if err != nil {
		return a, err
	}

	// set auth middleware
	a.middleware = middleware.AuthMiddleware(a.privateKey)

	// init and add kind endpoints
	for _, k := range a.options.Kinds {
		err = k.Init()
		if err != nil {
			return a, err
		}
		a.withKind(k)
	}

	// add kind endpoints for groups
	for _, k := range a.kinds {
		a.withGroupKind(k)
	}

	// add login handler
	a.router.Handle("/auth/login", a.AuthLoginHandler()).Methods(http.MethodPost)

	// add register handler
	if a.options.AllowUserRegistration {
		a.router.Handle("/auth/register", a.AuthRegistrationHandler(a.options.DefaultRole)).Methods(http.MethodPost)
	}

	// add profile handlers
	if a.options.UserProfileKind != nil {
		a.router.Handle("/auth/profile", a.middleware.Handler(a.AuthGetProfile(a.options.UserProfileKind))).Methods(http.MethodGet)
		a.router.Handle("/auth/profile", a.middleware.Handler(a.AuthUpdateProfile(a.options.UserProfileKind))).Methods(http.MethodPost)
		a.router.Handle("/auth/meta", a.middleware.Handler(a.AuthUpdateMeta())).Methods(http.MethodPost)
	}

	// create handler
	a.Handler = &Server{a.router}

	return a, nil
}

func (a *Apis) withKind(kind *kind.Kind) {
	a.kinds[kind.Name] = kind

	a.router.Handle("/"+kind.Name, a.middleware.Handler(a.QueryHandler(kind))).Methods(http.MethodGet)
	a.router.Handle("/"+kind.Name+"/{id}", a.middleware.Handler(a.GetHandler(kind))).Methods(http.MethodGet)

	//a.router.Handle("/"+name+"/draft", authMiddleware.Handler(a.AddDraftHandler(ent))).Methods(http.MethodPost) // ADD
	a.router.Handle("/"+kind.Name, a.middleware.Handler(a.AddHandler(kind))).Methods(http.MethodPost) // ADD
	//a.router.Handle("/"+name+"/{id}", authMiddleware.Handler(a.KindGetHandler(e))).Methods(http.MethodGet)       // GET
	a.router.Handle("/"+kind.Name+"/{id}", a.middleware.Handler(a.UpdateHandler(kind))).Methods(http.MethodPut)    // UPDATE
	//a.router.Handle("/{project}/api/"+name+"/{id}", authMiddleware.Handler(a.KindDeleteHandler(e))).Methods(http.MethodDelete) // DELETE
}

func (a *Apis) withGroupKind(kind *kind.Kind) {
	a.router.Handle("/{group}/"+kind.Name, a.middleware.Handler(a.QueryHandler(kind))).Methods(http.MethodGet)
	a.router.Handle("/{group}/"+kind.Name+"/{id}", a.middleware.Handler(a.GetHandler(kind))).Methods(http.MethodGet)

	//a.router.Handle("/"+name+"/draft", authMiddleware.Handler(a.AddDraftHandler(ent))).Methods(http.MethodPost) // ADD
	a.router.Handle("/{group}/"+kind.Name, a.middleware.Handler(a.AddHandler(kind))).Methods(http.MethodPost) // ADD
	//a.router.Handle("/"+name+"/{id}", authMiddleware.Handler(a.KindGetHandler(e))).Methods(http.MethodGet)       // GET
	//a.router.Handle("/{project}/api/"+name+"/{id}", authMiddleware.Handler(a.KindUpdateHandler(e))).Methods(http.MethodPut)    // UPDATE
	//a.router.Handle("/{project}/api/"+name+"/{id}", authMiddleware.Handler(a.KindDeleteHandler(e))).Methods(http.MethodDelete) // DELETE
}

func (a *Apis) Handle(path string, handler http.Handler) *mux.Route {
	return a.router.Handle(path, a.middleware.Handler(handler))
}

func (a *Apis) HandleFunc(path string, f func(http.ResponseWriter, *http.Request)) *mux.Route {
	return a.router.Handle(path, a.middleware.Handler(http.HandlerFunc(f)))
}

func (a *Apis) SignToken(token *jwt.Token) (*Token, error) {
	signedToken, err := token.SignedString(a.privateKey)
	if err != nil {
		return nil, err
	}

	return &Token{Id: signedToken, ExpiresAt: token.Claims.(jwt.MapClaims)["exp"].(int64)}, nil
}
