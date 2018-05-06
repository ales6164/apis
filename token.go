package apis

import (
	"github.com/dgrijalva/jwt-go"
	"time"
)

type Claims struct {
	Roles               []string `json:"roles,omitempty"`
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
	PhoneNumberVerified bool     `json:"phone_number_verified,omitempty"` // true if phone number verified
	Nonce               string   `json:"nonce,omitempty"`                 // value used to associate client session with id token
	jwt.StandardClaims
}

// when logging in
func (ctx Context) authenticate(jwtID string, sessionID string, user *User, expiresAt int64) (string, error) {
	now := time.Now()

	claims := Claims{
		user.Roles,
		user.Name,
		user.GivenName,
		user.FamilyName,
		user.MiddleName,
		user.Nickname,
		user.Picture,
		user.Website,
		user.Email,
		user.EmailVerified,
		user.Locale,
		user.PhoneNumber,
		user.PhoneNumberVerified,
		sessionID,
		jwt.StandardClaims{
			Audience:  "all",
			Id:        jwtID,
			ExpiresAt: expiresAt,
			IssuedAt:  now.Unix(),
			Issuer:    ctx.a.options.AppName,
			NotBefore: now.Add(-time.Minute).Unix(),
			Subject:   user.UserID.Encode(),
		},
	}

	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(ctx.a.privateKey)
}
