package apis

import (
	"errors"
	"net/http"

	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
	"io/ioutil"
)

type Apis struct {
	options *Options
	routes  []*Route

	middleware *JWTMiddleware
	privateKey []byte
	permissions
	allowedTranslations map[string]bool

	OnUserSignUp func(ctx Context, user User, token Token)
	//OnUserSignIn func(ctx context.Context, user User)
	OnUserVerified func(ctx Context, user User, token Token)
}

type Options struct {
	PrivateKeyPath           string // for password hashing
	AuthorizedOrigins        []string
	AllowUserRegistration    bool
	DefaultRole              Role
	RequireEmailConfirmation bool
	HasTranslationsFor       []string
	DefaultLanguage          string
	ProjectID                string
	/*UserProfileKind          *Kind*/
	RequireTrackingID bool // todo:generated from pages - track users - stored as session cookie
	Permissions
}

func New(opt *Options) (*Apis, error) {
	a := &Apis{
		options:             opt,
		allowedTranslations: map[string]bool{},
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
	a.middleware = AuthMiddleware(a.privateKey)

	// languages
	for _, l := range opt.HasTranslationsFor {
		a.allowedTranslations[l] = true
	}

	if len(opt.ProjectID) == 0 {
		return a, errors.New("missing project id")
	}

	return a, nil
}

func (a *Apis) Handle(p string, kind *Kind) *Route {
	if kind != nil {
		kind.ProjectID = a.options.ProjectID
	}
	r := &Route{
		kind:      kind,
		ProjectID: a.options.ProjectID,
		a:         a,
		path:      p,
		methods:   []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete},
	}

	a.routes = append(a.routes, r)
	return r
}

func (a *Apis) Handler(pathPrefix string) http.Handler {
	r := mux.NewRouter().PathPrefix(pathPrefix).Subrouter()

	for _, route := range a.routes {
		for _, method := range route.methods {
			switch method {
			case http.MethodGet:
				r.Handle(route.path, a.middleware.Handler(route.getHandler())).Methods(http.MethodGet)
			case http.MethodPost:
				r.Handle(route.path, a.middleware.Handler(route.postHandler())).Methods(http.MethodPost)
			case http.MethodPut:
				r.Handle(route.path, a.middleware.Handler(route.putHandler())).Methods(http.MethodPut)
			case http.MethodDelete:
				r.Handle(route.path, a.middleware.Handler(route.deleteHandler())).Methods(http.MethodDelete)
			}
		}
	}

	authRoute := &Route{
		ProjectID: a.options.ProjectID,
		a:         a,
		methods:   []string{},
	}
	r.Handle("/auth/login", authRoute.loginHandler()).Methods(http.MethodPost)
	if a.options.AllowUserRegistration {
		r.Handle("/auth/register", authRoute.registrationHandler(a.options.DefaultRole)).Methods(http.MethodPost)
	}
	r.Handle("/auth/confirm", a.middleware.Handler(authRoute.confirmEmailHandler()))
	r.Handle("/auth/password", a.middleware.Handler(authRoute.changePasswordHandler())).Methods(http.MethodPost)
	r.Handle("/auth/meta", a.middleware.Handler(authRoute.updateMeta())).Methods(http.MethodPost)

	r.Handle("/user", a.middleware.Handler(authRoute.getUserHandler())).Methods(http.MethodGet)
	r.Handle("/users", a.middleware.Handler(authRoute.getUsersHandler())).Methods(http.MethodGet)

	return &Server{r}
}

func (a *Apis) SignToken(token *jwt.Token) (*Token, error) {
	signedToken, err := token.SignedString(a.privateKey)
	if err != nil {
		return nil, err
	}
	return &Token{Id: signedToken, ExpiresAt: token.Claims.(jwt.MapClaims)["exp"].(int64)}, nil
}
