package apis

import (
	"github.com/ales6164/apis/errors"
	"google.golang.org/appengine/datastore"
	"time"
	"github.com/dgrijalva/jwt-go"
	"github.com/ales6164/apis/middleware"
	"github.com/gorilla/mux"
	"net/http"
)

var (
	ErrEmailUndefined    = errors.New("email undefined")
	ErrPasswordUndefined = errors.New("password undefined")
	ErrInvalidCallback   = errors.New("callback is not a valid URL")
	ErrInvalidEmail      = errors.New("email is not valid")
	ErrPasswordTooLong   = errors.New("password must be exactly or less than 128 characters long")
	ErrPasswordTooShort  = errors.New("password must be at least 6 characters long")
)

type ClientSession struct {
	CreatedAt time.Time
	ExpiresAt time.Time
	JwtID     string
	IsBlocked bool
	Roles     []string
	Account   *datastore.Key
	User      *datastore.Key
}

type Token struct {
	Id        string `json:"id"`
	ExpiresAt int64  `json:"expiresAt"`
}

func initAuth(a *Apis, r *mux.Router) {
	authRoute := &Route{
		a:       a,
		methods: []string{},
	}

	r.HandleFunc("/auth/login", loginHandler(authRoute)).Methods(http.MethodPost)
	if a.options.AllowUserRegistration {
		r.Handle("/auth/register", registrationHandler(authRoute)).Methods(http.MethodPost)
	}
	r.Handle("/auth/password", a.middleware.Handler(changePasswordHandler(authRoute))).Methods(http.MethodPut)
}

func createSession(ctx Context, usrKey *datastore.Key, usrEmail string, roles []string) (Token, error) {
	var signedToken Token
	now := time.Now()
	expiresAt := now.Add(time.Hour * time.Duration(72))
	sess := new(ClientSession)
	sess.CreatedAt = now
	sess.ExpiresAt = expiresAt
	sess.User = usrKey
	sess.Roles = roles
	sess.JwtID = RandStringBytesMaskImprSrc(LetterBytes, 16)
	sessKey := datastore.NewIncompleteKey(ctx, "_clientSession", nil)
	sessKey, err := datastore.Put(ctx, sessKey, sess)
	if err != nil {
		return signedToken, err
	}
	return ctx.authenticate(usrEmail, sess, sessKey.Encode(), usrKey.Encode(), expiresAt.Unix())
}

func (ctx Context) authenticate(accEmail string, sess *ClientSession, sessionID string, encodedUsrKey string, expiresAt int64) (Token, error) {
	var err error
	now := time.Now()
	claims := middleware.Claims{
		Nonce: sessionID,
		StandardClaims: jwt.StandardClaims{
			Audience:  "all",
			Id:        sess.JwtID,
			ExpiresAt: expiresAt,
			IssuedAt:  now.Unix(),
			Issuer:    ctx.a.options.AppName,
			NotBefore: now.Add(-time.Minute).Unix(),
			Subject:   encodedUsrKey,
		},
	}
	token := Token{
		ExpiresAt: expiresAt,
	}
	token.Id, err = jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(ctx.a.privateKey)
	return token, err
}
