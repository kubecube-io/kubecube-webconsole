package utils

import (
	logger "github.com/astaxie/beego/logs"
	"github.com/emicklei/go-restful"
	"strings"
)

const (
	authorizationHeader = "Authorization"
	bearerTokenPrefix   = "Bearer"
)

func GetTokenFromReq(request *restful.Request) string {
	// get token from header
	var bearerToken = request.HeaderParameter(authorizationHeader)
	logger.Debug("get bearer token from header: %s", bearerToken)
	if bearerToken == "" {
		// get token from cookie
		cookie, err := request.Request.Cookie(authorizationHeader)
		if err != nil {
			logger.Error("get token from cookie error: %s", err)
		}
		bearerToken = cookie.Value
		logger.Info("get bearer token from cookie: %s", bearerToken)
		if bearerToken == "" {
			return ""
		}
	}

	// parse bearer token
	parts := strings.Split(bearerToken, " ")
	if len(parts) < 2 || strings.ToLower(parts[0]) != strings.ToLower(bearerTokenPrefix) {
		return ""
	}
	return parts[1]
}

func GetUserFromReq(request *restful.Request) string {
	token := GetTokenFromReq(request)
	if token != "" {
		claims := ParseToken(token)
		if claims != nil {
			return claims.UserInfo.Username
		}
	}
	return ""
}
