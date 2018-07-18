package apis

import (
	"github.com/ales6164/apis-v1/middleware"
	"github.com/dgrijalva/jwt-go"
	"time"
)

// when logging in
func (ctx Context) authenticate(sess *ClientSession, sessionID string, user *User, expiresAt int64) (string, error) {
	now := time.Now()

	claims := middleware.Claims{
		Roles:               user.Roles,
		Name:                user.Profile.Name,
		GivenName:           user.Profile.GivenName,
		FamilyName:          user.Profile.FamilyName,
		MiddleName:          user.Profile.MiddleName,
		Nickname:            user.Profile.Nickname,
		Picture:             user.Profile.Picture,
		Website:             user.Profile.Website,
		Email:               user.Email,
		EmailVerified:       user.EmailVerified,
		Locale:              user.Locale,
		PhoneNumber:         user.PhoneNumber,
		PhoneNumberVerified: user.PhoneNumberVerified,
		Nonce:               sessionID,
		StandardClaims: jwt.StandardClaims{
			Audience:  "all",
			Id:        sess.JwtID,
			ExpiresAt: expiresAt,
			IssuedAt:  now.Unix(),
			Issuer:    ctx.a.options.AppName,
			NotBefore: now.Add(-time.Minute).Unix(),
			Subject:   user.UserID.Encode(),
		},
	}

	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(ctx.a.privateKey)
}
