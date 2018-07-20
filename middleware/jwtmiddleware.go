package middleware

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/context"
	"google.golang.org/appengine"
	clog "google.golang.org/appengine/log"
)

// A function called whenever an error is encountered
type errorHandler func(w http.ResponseWriter, r *http.Request, err string)

// TokenExtractor is a function that takes a request as input and returns
// either a token or an error.  An error should only be returned if an attempt
// to specify a token was found, but the information was somehow incorrectly
// formed.  In the case where a token is simply not present, this should not
// be treated as an error.  An empty string should be returned in that case.
type TokenExtractor func(r *http.Request) (string, error)

// Options is a struct for specifying configuration options for the middleware.
type MiddlewareOptions struct {
	// Redirect path
	RedirectOnError string
	// The function that will return the Key to validate the JWT.
	// It can be either a shared secret or a public key.
	// Default value: nil
	ValidationKeyGetter jwt.Keyfunc
	// The name of the property in the request where the auth information
	// from the JWT will be stored.
	// Default value: "auth"
	UserProperty string
	// The function that will be called when there's an error validating the token
	// Default value:
	ErrorHandler errorHandler
	// A boolean indicating if the credentials are required or not
	// Default value: false
	CredentialsOptional bool
	// A function that extracts the token from the request
	// Default: FromAuthHeader (i.e., from Authorization header as bearer token)
	Extractor TokenExtractor
	// Debug flag turns on debugging output
	// Default: false
	Debug bool
	// When set, all requests with the OPTIONS method will use authentication
	// Default: false
	EnableAuthOnOptions bool
	// When set, the middelware verifies that tokens are signed with the specific signing algorithm
	// If the signing method is not constant the ValidationKeyGetter callback can be used to implement additional checks
	// Important to avoid security issues described here: https://auth0.com/blog/2015/03/31/critical-vulnerabilities-in-json-web-token-libraries/
	// Default: nil
	SigningMethod jwt.SigningMethod
}

type JWTMiddleware struct {
	Options MiddlewareOptions
}

type Claims struct {
	/*Roles               []string `json:"roles,omitempty"`
	Name                string   `json:"name,omitempty"`
	GivenName           string   `json:"given_name,omitempty"`
	FamilyName          string   `json:"family_name,omitempty"`
	MiddleName          string   `json:"middle_name,omitempty"`
	Nickname            string   `json:"nickname,omitempty"`
	Picture             string   `json:"picture,omitempty"`               // profile picture URL
	Website             string   `json:"website,omitempty"`               // website URL
	Email               string   `json:"email,omitempty"`                 // preferred email
	EmailVerified       bool     `json:"email_verified,omitempty"`        // true if email verified
	Locale              string   `json:"locale,omitempty"`                // locale
	PhoneNumber         string   `json:"phone_number,omitempty"`          // preferred phone number
	PhoneNumberVerified bool     `json:"phone_number_verified,omitempty"` // true if phone number verified*/
	Nonce               string   `json:"nonce,omitempty"`                 // value used to associate client session with id token
	jwt.StandardClaims
}

func OnError(w http.ResponseWriter, r *http.Request, err string) {
	http.Error(w, err, http.StatusUnauthorized)
}

// New constructs a new Secure instance with supplied options.
func New(options ...MiddlewareOptions) *JWTMiddleware {

	var opts MiddlewareOptions
	if len(options) == 0 {
		opts = MiddlewareOptions{}
	} else {
		opts = options[0]
	}

	if opts.UserProperty == "" {
		opts.UserProperty = "auth"
	}

	if opts.ErrorHandler == nil {
		opts.ErrorHandler = OnError
	}

	if opts.Extractor == nil {
		opts.Extractor = FromAuthHeader
	}

	return &JWTMiddleware{
		Options: opts,
	}
}

func (m *JWTMiddleware) logf(format string, args ...interface{}) {
	if m.Options.Debug {
		log.Printf(format, args...)
	}
}

// Special implementation for optional auth authentication.
/*func (m *JWTMiddleware) HandlerWithOptionalAuth(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	err := m.CheckJWT(w, r)

	// If there was an error, do not call next.
	if err == nil && next != nil {
		next(w, r)
	}
}*/

// Special implementation for Negroni, but could be used elsewhere.
func (m *JWTMiddleware) HandlerWithNext(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	err := m.CheckJWT(w, r)

	// If there was an error, do not call next.
	if err == nil && next != nil {
		next(w, r)
	}
}

func (m *JWTMiddleware) Handler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Let secure process the request. If it returns an error,
		// that indicates the request should not continue.
		err := m.CheckJWT(w, r)

		// If there was an error, do not continue.
		if err != nil {

			ctx := appengine.NewContext(r)
			clog.Debugf(ctx, "auth error: %s", err.Error())

			if len(m.Options.RedirectOnError) > 0 {
				redirect(w, r, m.Options.RedirectOnError)
			} else {
				//panic(err)
				//printError(w, err, http.StatusUnauthorized)
			}
			return
		}

		h.ServeHTTP(w, r)
	})
}

// FromAuthHeader is a "TokenExtractor" that takes a give request and extracts
// the JWT token from the Authorization header.
func FromAuthHeader(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", nil // No error, just no token
	}

	// TODO: Make this a bit more robust, parsing-wise
	authHeaderParts := strings.Split(authHeader, " ")
	if len(authHeaderParts) != 2 || strings.ToLower(authHeaderParts[0]) != "bearer" {
		return "", fmt.Errorf("Authorization header format must be Bearer {token}")
	}

	return authHeaderParts[1], nil
}

// FromParameter returns a function that extracts the token from the specified
// query string parameter
func FromParameter(param string) TokenExtractor {
	return func(r *http.Request) (string, error) {
		return r.URL.Query().Get(param), nil
	}
}

func FromFormValue(param string) TokenExtractor {
	return func(r *http.Request) (string, error) {
		return r.FormValue(param), nil
	}
}

// FromParameter returns a function that extracts the token from the specified
// query string parameter
/*func FromSession(sessionName string) TokenExtractor {
	return func(r *http.Request) (string, error) {
		session, err := session.Get(r, sessionName)
		if err != nil {
			return "", err
		}
		return session.Values["id_token"].(string), nil
	}
}*/

// FromFirst returns a function that runs multiple token extractors and takes the
// first token it finds
func FromFirst(extractors ...TokenExtractor) TokenExtractor {
	return func(r *http.Request) (string, error) {
		for _, ex := range extractors {
			token, err := ex(r)
			if err != nil {
				return "", err
			}
			if token != "" {
				return token, nil
			}
		}
		return "", nil
	}
}

func redirect(w http.ResponseWriter, r *http.Request, path string) {
	http.Redirect(w, r, path, 302)
}

func (m *JWTMiddleware) CheckJWT(w http.ResponseWriter, r *http.Request) error {
	if !m.Options.EnableAuthOnOptions {
		if r.Method == "OPTIONS" {
			return nil
		}
	}

	// Use the specified token extractor to extract a token from the request
	token, err := m.Options.Extractor(r)

	// If debugging is turned on, log the outcome
	if err != nil {
		m.logf("Error extracting JWT: %v", err)
	} else {
		m.logf("Token extracted: %s", token)
	}

	// If an error occurs, call the error handler and return an error
	if err != nil {
		//m.Options.ErrorHandler(w, r, err.Error())
		return fmt.Errorf("Error extracting token: %v", err)
	}

	// If the token is empty...
	if token == "" {
		// Check if it was required
		if m.Options.CredentialsOptional {
			m.logf("  No credentials found (CredentialsOptional=true)")
			// No error, just no token (and that is ok given that CredentialsOptional is true)
			return nil
		}

		// If we get here, the required token is missing
		errorMsg := "Required authorization token not found"
		//m.Options.ErrorHandler(w, r, errorMsg)
		m.logf("  Error: No credentials found (CredentialsOptional=false)")
		return fmt.Errorf(errorMsg)
	}

	// Now parse the token
	parser := new(jwt.Parser)
	parser.SkipClaimsValidation = true
	parsedToken, err := parser.ParseWithClaims(token, &Claims{}, m.Options.ValidationKeyGetter)

	// Check if there was an error in parsing...
	if err != nil {
		m.logf("Error parsing token: %v", err)
		//m.Options.ErrorHandler(w, r, err.Error())
		return fmt.Errorf("Error parsing token: %v", err)
	}

	if m.Options.SigningMethod != nil && m.Options.SigningMethod.Alg() != parsedToken.Header["alg"] {
		message := fmt.Sprintf("Expected %s signing method but token specified %s",
			m.Options.SigningMethod.Alg(),
			parsedToken.Header["alg"])
		m.logf("Error validating token algorithm: %s", message)
		//m.Options.ErrorHandler(w, r, errors.New(message).Error())
		return fmt.Errorf("Error validating token algorithm: %s", message)
	}

	// Check if the parsed token is valid...
	if !parsedToken.Valid {
		m.logf("Token is invalid")
		//m.Options.ErrorHandler(w, r, "The token isn't valid")
		return fmt.Errorf("Token is invalid")
	}

	m.logf("JWT: %v", parsedToken)

	// If we get here, everything worked and we can set the
	// auth property in instance.
	context.Set(r, m.Options.UserProperty, parsedToken)

	return nil
}
