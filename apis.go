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
	HandlerPathPrefix      string
	PrivateKeyPath         string // for password hashing
	Kinds                  []*kind.Kind
	AuthorizedOrigins      []string
	AuthorizedCallbackURIs []string
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

	// add kind endpoints
	for _, k := range a.options.Kinds {
		a.withKind(k)
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
	//a.router.Handle("/{project}/api/"+name+"/{id}", authMiddleware.Handler(a.KindUpdateHandler(e))).Methods(http.MethodPut)    // UPDATE
	//a.router.Handle("/{project}/api/"+name+"/{id}", authMiddleware.Handler(a.KindDeleteHandler(e))).Methods(http.MethodDelete) // DELETE
}

func (a *Apis) Handle(path string, handler http.Handler) *mux.Route {
	return a.router.Handle(path, handler)
}

func (a *Apis) HandleFunc(path string, f func(http.ResponseWriter, *http.Request)) *mux.Route {
	return a.router.HandleFunc(path, f)
}

func (a *Apis) SignToken(token *jwt.Token) (*Token, error) {
	signedToken, err := token.SignedString(a.privateKey)
	if err != nil {
		return nil, err
	}

	return &Token{Id: signedToken, ExpiresAt: token.Claims.(jwt.MapClaims)["exp"].(int64)}, nil
}
