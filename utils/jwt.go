package utils

import (
	logger "github.com/astaxie/beego/logs"
	"github.com/dgrijalva/jwt-go"
	"k8s.io/api/authentication/v1beta1"
)

type Claims struct {
	UserInfo v1beta1.UserInfo
	jwt.StandardClaims
}

func ParseToken(token string) *Claims {

	claims := &Claims{}

	// Empty bearer tokens aren't valid
	if len(token) == 0 {
		return nil
	}

	newToken, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(JwtSecret()), nil
	})
	if err != nil {
		logger.Error("parse token error: %s", err)
		return nil
	}
	if claims, ok := newToken.Claims.(*Claims); ok && newToken.Valid {
		return claims
	}
	return nil
}
