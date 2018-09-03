package middleware

import (
	"github.com/dgrijalva/jwt-go"
)

func AuthMiddleware(signingKey []byte) *JWTMiddleware {
	return New(MiddlewareOptions{
		Extractor: FromFirst(
			FromAuthHeader,
			FromParameter("key"),
			FromFormValue("key"),
		),
		ValidationKeyGetter: func(token *jwt.Token) (interface{}, error) {
			return signingKey, nil
		},
		SigningMethod:       jwt.SigningMethodHS256,
		CredentialsOptional: true,
	})
}
