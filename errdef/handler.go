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
	logger.Error(err)
	statusCode := http.StatusInternalServerError
	statusError, ok := err.(*errors.StatusError)
	if ok && statusError.Status().Code > 0 {
		statusCode = int(statusError.Status().Code)
	}
	response.AddHeader("Content-Type", "text/plain")
	response.WriteErrorString(statusCode, err.Error()+"\n")
}

func HandleInternalErrorByCode(response *restful.Response, errCode errorInfo) {
	logger.Error(errCode)
	response.AddHeader("Content-Type", "text/plain")
	msg, _ := json.Marshal(errCode)
	response.WriteErrorString(errCode.Code, string(msg))
}
