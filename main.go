package main

import (
	"fmt"
	logger "github.com/astaxie/beego/logs"
	"github.com/kubecube-io/kubecube/pkg/clients"
	"kubecube-webconsole/handler"
	"net/http"
	_ "net/http/pprof"
)

func init() {
	clients.InitCubeClientSetWithOpts(nil)
	handler.K8sClient = clients.Interface().Kubernetes(handler.PivotCluster)
}

func main() {
	http.Handle("/api/", handler.CreateHTTPAPIHandler())
	http.Handle("/api/sockjs/", handler.CreateAttachHandler("/api/sockjs"))
	http.HandleFunc("/healthz", func(response http.ResponseWriter, request *http.Request) {
		logger.Debug("Health check")
		response.WriteHeader(http.StatusOK)
	})
	http.HandleFunc("/leader", func(response http.ResponseWriter, request *http.Request) {
		logger.Debug("This is leader")
		response.WriteHeader(http.StatusOK)
	})

	err := http.ListenAndServe(fmt.Sprintf(":%d", *handler.ServerPort), nil)
	if err != nil {
		logger.Critical("ListenAndServe failed，error msg: %s", err.Error())
	}

}
