/*
Copyright 2021 KubeCube Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package utils

import (
	clog "github.com/astaxie/beego/logs"
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
	clog.Debug("get bearer token from header: %s", bearerToken)
	if bearerToken == "" {
		// get token from cookie
		cookie, err := request.Request.Cookie(authorizationHeader)
		if err != nil {
			clog.Error("get token from cookie error: %s", err)
			return ""
		}
		if cookie == nil {
			clog.Error("cookie is nil")
			return ""
		}
		bearerToken = cookie.Value
		clog.Debug("get bearer token from cookie: %s", bearerToken)
		if bearerToken == "" {
			return ""
		}
	}

	// parse bearer token
	parts := strings.Split(bearerToken, string(bearerToken[6]))
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
