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

package errdef

import (
	"encoding/json"
	"fmt"
	"net/http"

	logger "github.com/astaxie/beego/logs"
)

type ErrorInfo struct {
	Code      int    `json:"Code"`
	ErrorCode string `json:"ErrorCode"`
	Msg       string `json:"Message"`
}

var (
	ClusterInfoNotFound    = ErrorInfo{http.StatusInternalServerError, "ClusterInfoNotFound", "Cluster not found."}
	InternalServerError    = ErrorInfo{http.StatusInternalServerError, "InternalServerError", "Internal server error."}
	NoRunningPod           = ErrorInfo{http.StatusInternalServerError, "NoRunningPod", "No running pod available."}
	ControlClusterNotFound = ErrorInfo{http.StatusInternalServerError, "ControlClusterNotFound", "Control cluster not found."}
	InvalidToken           = &ErrorInfo{http.StatusUnauthorized, "InvalidToken", "Token invalid."}
)

func (ei ErrorInfo) WithMarshal() []byte {
	res, err := json.Marshal(ei)
	if err != nil {
		logger.Error("Json marshal failed, %s", err.Error())
	}
	return res
}

func (ei ErrorInfo) Error() string {
	return fmt.Sprintf("errorCode: %d, errorMsg %s", ei.Code, ei.Msg)
}
