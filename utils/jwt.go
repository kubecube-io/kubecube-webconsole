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
	"context"
	logger "github.com/astaxie/beego/logs"
	"github.com/dgrijalva/jwt-go"
	"github.com/kubecube-io/kubecube/pkg/clients"
	"k8s.io/api/authentication/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"kubecube-webconsole/constants"
)

const JwtSecretName = "jwt-secret"

type Claims struct {
	UserInfo v1beta1.UserInfo
	jwt.StandardClaims
}

func ParseToken(token string) *Claims {

	claims := &Claims{}

	// empty bearer tokens aren't valid
	if len(token) == 0 {
		return nil
	}
	jwtSecret, err := getJwtSecret()
	if err != nil {
		return nil
	}
	newToken, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
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

func getJwtSecret() ([]byte, error) {
	var (
		client = clients.Interface().Kubernetes(constants.ControlClusterName)
		ctx    = context.Background()
		secret = corev1.Secret{}
	)
	key := types.NamespacedName{Name: JwtSecretName}
	err := client.Cache().Get(ctx, key, &secret)
	if err != nil {
		logger.Error("get jwt secret failed: %v", err)
		return []byte(""), err
	}
	return secret.Data["secret"], nil
}
