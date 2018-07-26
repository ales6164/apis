package apis

import (
	"encoding/json"
	"gopkg.in/ales6164/apis.v1/errors"
	"gopkg.in/ales6164/apis.v1/kind"
	"gopkg.in/ales6164/apis.v1/middleware"
	"github.com/dgrijalva/jwt-go"
	gcontext "github.com/gorilla/context"
	"github.com/gorilla/mux"
	"golang.org/x/net/context"
	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Context struct {
	*Route
	*ClientRequest
	claims                 middleware.Claims
	hasActiveClientSession bool
	ClientSession          ClientSession
	error                  error
	clientRequestKey       *datastore.Key
	r                      *http.Request
	context.Context
	*body
}

type body struct {
	hasReadBody bool
	body        []byte
}

type Device struct {
	UserAgent      string
	AcceptLanguage string
	IP             string
	Country        string
	Region         string
	City           string
	CityLatLng     appengine.GeoPoint
}

type ClientRequest struct {
	Time            time.Time
	Device          Device
	URL             string
	Method          string
	ClientSession   *datastore.Key
	Error           string
	IsAuthenticated bool
	IsBlocked       bool
	IsExpired       bool
	Body            []byte
}

// authenticated request is necessary - not yet
// logs every request
func (R *Route) NewContext(r *http.Request) Context {
	return R.a.newContext(r, R)
}
func (a *Apis) newContext(r *http.Request, R *Route) Context {
	clientReq := new(ClientRequest)
	ctx := Context{
		ClientRequest: clientReq,
		Route:         R,
		r:             r,
		Context:       appengine.NewContext(r),
		body:          &body{hasReadBody: false},
	}
	clientReq.Time = time.Now()
	clientReq.URL = r.URL.String()
	clientReq.Method = r.Method
	clientReq.Device = Device{
		IP:             r.RemoteAddr,
		UserAgent:      r.UserAgent(),
		AcceptLanguage: r.Header.Get("accept-language"),
		City:           r.Header.Get("X-AppEngine-City"),
		Country:        r.Header.Get("X-AppEngine-Country"),
		Region:         r.Header.Get("X-AppEngine-Region"),
	}
	latlng := strings.Split(r.Header.Get("X-AppEngine-CityLatLong"), ",")
	if len(latlng) == 2 {
		lat, _ := strconv.ParseFloat(latlng[0], 64)
		lng, _ := strconv.ParseFloat(latlng[1], 64)
		clientReq.Device.CityLatLng = appengine.GeoPoint{Lat: lat, Lng: lng}
	}

	tkn := gcontext.Get(r, "auth")
	if tkn != nil {
		if token, ok := tkn.(*jwt.Token); ok && token.Valid {
			if claims, ok := token.Claims.(*middleware.Claims); ok {

				// todo: decode claims.Nonce as ClientSession key and compare with datastore entry;
				// todo: implement and check if session was blocked

				// authenticated
				ctx.claims = *claims

				clientReq.ClientSession, ctx.error = datastore.DecodeKey(claims.Nonce)
				ctx.check()

				ctx.error = datastore.Get(ctx, clientReq.ClientSession, &ctx.ClientSession)
				ctx.hasActiveClientSession = ctx.error == nil
				ctx.check()
			} else {
				ctx.error = errors.New("claims not of type *Claims")
				ctx.check()
			}
		} else {
			ctx.error = errors.New("token not valid")
			ctx.check()
		}
	} else {
		log.Debugf(ctx, "token is nil")
	}

	// check if session is ok
	if ctx.hasActiveClientSession {
		ctx.IsExpired = ctx.ClientSession.ExpiresAt.Before(ctx.Time)
		if !ctx.IsExpired {
			ctx.IsBlocked = ctx.ClientSession.IsBlocked
			ctx.IsAuthenticated = !ctx.IsBlocked
		}
	}

	// store to datastore
	ctx.clientRequestKey = datastore.NewIncompleteKey(ctx, "_clientRequest", nil)
	ctx.clientRequestKey, ctx.error = datastore.Put(ctx, ctx.clientRequestKey, clientReq)
	ctx.check()

	return ctx
}

// logs error if any
func (ctx Context) check() {
	if ctx.error != nil {
		log.Criticalf(ctx, "context of %v check error: %v", ctx.ClientRequest, ctx.error)
	}
}

// reads body once and stores contents
func (ctx Context) Body() []byte {
	if !ctx.body.hasReadBody {
		ctx.body.body, _ = ioutil.ReadAll(ctx.r.Body)
		ctx.r.Body.Close()
		ctx.body.hasReadBody = true
	}
	return ctx.body.body
}

func (ctx Context) User() *User {
	u, _ := getUser(ctx, ctx.UserKey())
	return u
}

func (ctx Context) UserKey() *datastore.Key {
	key, _ := datastore.DecodeKey(ctx.claims.StandardClaims.Subject)
	return key
}

func (ctx Context) Claims() middleware.Claims {
	return ctx.claims
}

func (ctx Context) Roles() []string {
	if len(ctx.claims.Roles) > 0 {
		return ctx.claims.Roles
	}
	return PublicRoles
}

func (ctx Context) Id() string {
	return mux.Vars(ctx.r)["id"]
}

func (ctx Context) SetNamespace(namespace string) (Context, error) {
	var err error
	ctx.Context, err = appengine.Namespace(ctx, namespace)
	return ctx, err
}

func (ctx Context) HasRole(role Role) bool {
	sr := string(role)
	for _, r := range ctx.claims.Roles {
		if r == sr {
			return true
		}
	}
	return false
}

// AdminRole has all permissions
// searches if user has permission - also goes through all roles to find if also has access to global scope (not private)
func (ctx Context) HasPermission(k *kind.Kind, scope Scope) (ok bool, isPrivateOnly bool) {
	roles := ctx.Roles()
	hasPermission := false
	ok = false
	isPrivateOnly = true
	for _, role := range roles {
		if val1, ok := ctx.a.permissions[role]; ok {
			if val2, ok := val1[k]; ok {
				if val3, ok := val2[scope]; ok {
					if ctx.roles != nil {
						if _, ok := ctx.roles[role]; ok {
							hasPermission = true
							isPrivateOnly = val3
						}
					} else {
						hasPermission = true
						isPrivateOnly = val3
					}
				}
			}
		}
		if role == string(AdminRole) {
			hasPermission = true
			isPrivateOnly = false
		}
		if hasPermission && !isPrivateOnly {
			break
		}
	}
	return hasPermission, isPrivateOnly
}

/**
RESPONSE
*/

func (ctx *Context) Print(w http.ResponseWriter, result interface{}) {
	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(result)
}

func (ctx *Context) PrintBytes(w http.ResponseWriter, result []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.Write(result)
}

func (ctx *Context) PrintResult(w http.ResponseWriter, result map[string]interface{}) {
	w.Header().Set("Content-Type", "application/json")

	bs, err := json.Marshal(result)
	if err != nil {
		ctx.PrintError(w, err)
		return
	}
	w.Write(bs)
}

func (ctx *Context) PrintError(w http.ResponseWriter, err error, descriptors ...string) {
	ctx.ClientRequest.Error = err.Error()
	for i, d := range descriptors {
		ctx.ClientRequest.Error += `\n[descriptor"` + strconv.Itoa(i) + `","` + d + `"]`
	}
	log.Errorf(ctx, "context error: %s", ctx.ClientRequest.Error)
	ctx.ClientRequest.Body = ctx.Body()
	datastore.Put(ctx, ctx.clientRequestKey, ctx.ClientRequest)
	if err == errors.ErrUnathorized {
		w.WriteHeader(http.StatusUnauthorized)
	} else if err == errors.ErrForbidden {
		w.WriteHeader(http.StatusForbidden)
	} else if _, ok := err.(*errors.Error); ok {
		w.WriteHeader(http.StatusBadRequest)
	} else {
		w.WriteHeader(http.StatusInternalServerError)
	}
	w.Write([]byte(err.Error()))
}
