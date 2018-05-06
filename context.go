package apis

import (
	"golang.org/x/net/context"
	gcontext "github.com/gorilla/context"
	"google.golang.org/appengine"
	"io/ioutil"
	"net/http"
	"github.com/dgrijalva/jwt-go"
	"encoding/json"
	"github.com/gorilla/mux"
	"google.golang.org/appengine/datastore"
	"github.com/ales6164/apis/kind"
	"google.golang.org/appengine/log"
	"github.com/ales6164/apis/errors"
	"strings"
	"strconv"
	"time"
)

type Context struct {
	*Route
	*ClientRequest
	claims           Claims
	ClientSession    *ClientSession
	error            error
	clientRequestKey *datastore.Key
	r                *http.Request
	context.Context
	*body
}

type body struct {
	hasReadBody bool
	body        []byte
}

type Device struct {
	UserAgent  string
	IP         string
	Country    string
	Region     string
	City       string
	CityLatLng appengine.GeoPoint
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
}

// authenticated request is necessary - not yet
// logs every request
func (R *Route) NewContext(r *http.Request) Context {
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
		IP:        r.RemoteAddr,
		UserAgent: r.UserAgent(),
		City:      r.Header.Get("X-AppEngine-City"),
		Country:   r.Header.Get("X-AppEngine-Country"),
		Region:    r.Header.Get("X-AppEngine-Region"),
	}
	latlng := strings.Split(r.Header.Get("X-AppEngine-CityLatLong"), ",")
	if len(latlng) == 2 {
		lat, _ := strconv.ParseFloat(latlng[0], 64)
		lng, _ := strconv.ParseFloat(latlng[1], 64)
		clientReq.Device.CityLatLng = appengine.GeoPoint{Lat: lat, Lng: lng}
	}

	tkn := gcontext.Get(r, "auth")
	if tkn != nil {
		if token, ok := tkn.(*jwt.Token); ok {
			if claims, ok := token.Claims.(*Claims); ok && token.Valid {

				// todo: decode claims.Nonce as ClientSession key and compare with datastore entry;
				// todo: implement and check if session was blocked

				// authenticated
				ctx.claims = *claims

				clientReq.ClientSession, ctx.error = datastore.DecodeKey(claims.Nonce)
				ctx.check()

				ctx.error = datastore.Get(ctx, clientReq.ClientSession, ctx.ClientSession)
				ctx.check()
			}
		}
	}

	// check if session is ok
	if ctx.ClientSession != nil {
		ctx.IsExpired = ctx.ClientSession.ExpiresAt.After(ctx.Time)
		if !ctx.IsExpired {
			ctx.IsBlocked = ctx.ClientSession.IsBlocked
			ctx.IsAuthenticated = !ctx.IsBlocked
		}
	}

	// store to datastore
	ctx.clientRequestKey = datastore.NewIncompleteKey(ctx, "_clientRequest", nil)
	ctx.clientRequestKey, ctx.error = datastore.Put(ctx, ctx.clientRequestKey, nil)
	ctx.check()

	return ctx
}

// logs error if any
func (ctx Context) check() {
	if ctx.error != nil {
		log.Criticalf(ctx, "context error: %v", ctx.error)
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

func (ctx Context) UserKey() *datastore.Key {
	key, _ := datastore.DecodeKey(ctx.claims.StandardClaims.Subject)
	return key
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
func (ctx Context) HasPermission(k *kind.Kind, scope Scope) bool {
	roles := ctx.Roles()
	for _, role := range roles {
		if val1, ok := ctx.a.permissions[role]; ok {
			if val2, ok := val1[scope]; ok {
				if val3, ok := val2[k]; ok && val3 {
					if ctx.roles != nil {
						if _, ok := ctx.roles[role]; ok {
							return true
						}
					} else {
						return true
					}
				}
			}
		}
		if role == string(AdminRole) {
			return true
		}
	}
	return false
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

func (ctx *Context) PrintError(w http.ResponseWriter, err error) {
	log.Errorf(ctx, "context error: %v", err)
	ctx.ClientRequest.Error = err.Error()
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
