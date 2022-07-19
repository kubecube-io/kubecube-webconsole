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
	"net/http"

	"encoding/json"

	logger "github.com/astaxie/beego/logs"
	"github.com/emicklei/go-restful"
	"k8s.io/apimachinery/pkg/api/errors"
)

// HandleInternalError writes the given error to the response and sets appropriate HTTP status headers.
func HandleInternalError(response *restful.Response, err error) {
	clog.Error(err)
	statusCode := http.StatusInternalServerError
	statusError, ok := err.(*errors.StatusError)
	if ok && statusError.Status().Code > 0 {
		statusCode = int(statusError.Status().Code)
	}
	response.AddHeader("Content-Type", "text/plain")
	response.WriteErrorString(statusCode, err.Error()+"\n")
}

func HandleInternalErrorByCode(response *restful.Response, errCode ErrorInfo) {
	clog.Error(errCode)
	response.AddHeader("Content-Type", "text/plain")
	msg, _ := json.Marshal(errCode)
	response.WriteErrorString(errCode.Code, string(msg))
}
