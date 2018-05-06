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
