package errdef

import (
	"encoding/json"
	"fmt"
	"net/http"

	logger "github.com/astaxie/beego/logs"
)

type errorInfo struct {
	Code      int    `json:"Code"`
	ErrorCode string `json:"ErrorCode"`
	Msg       string `json:"Message"`
}

var (
	ClusterInfoNotFound    = errorInfo{http.StatusInternalServerError, "ClusterInfoNotFound", "Cluster info not found"}
	InternalServerError    = errorInfo{http.StatusInternalServerError, "InternalServerError", "Internal server error"}
	NoRunningPod           = errorInfo{http.StatusInternalServerError, "NoRunningPod", "No running pod available"}
	ControlClusterNotFound = errorInfo{http.StatusInternalServerError, "ControlClusterNotFound", "No control cluster in Db"}
)

func (ei errorInfo) WithMarshal() []byte {
	res, err := json.Marshal(ei)
	if err != nil {
		logger.Error("Json marshal failed, %s", err.Error())
	}
	return res
}

func (ei errorInfo) Error() string {
	return fmt.Sprintf("errorCode: %d, errorMsg %s", ei.Code, ei.Msg)
}
