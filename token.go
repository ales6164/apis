package apis

import (
	"github.com/dgrijalva/jwt-go"
	"time"
	"github.com/ales6164/apis/middleware"
)

// when logging in
func (ctx Context) authenticate(sess *ClientSession, sessionID string, user *User, expiresAt int64) (string, error) {
	now := time.Now()

	claims := middleware.Claims{
		user.Roles,
		user.Profile.Name,
		user.Profile.GivenName,
		user.Profile.FamilyName,
		user.Profile.MiddleName,
		user.Profile.Nickname,
		user.Profile.Picture,
		user.Profile.Website,
		user.Email,
		user.EmailVerified,
		user.Profile.Locale,
		user.PhoneNumber,
		user.PhoneNumberVerified,
		sessionID,
		jwt.StandardClaims{
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
