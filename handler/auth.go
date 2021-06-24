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

package handler

import (
	"context"
	"crypto/tls"
	"encoding/json"
	clog "github.com/astaxie/beego/logs"
	"github.com/emicklei/go-restful"
	"github.com/kubecube-io/kubecube/pkg/clients"
	"io/ioutil"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"kubecube-webconsole/utils"
	"net/http"
	"strings"
	"time"
)

type attributes struct {
	User            string `json:"user"`
	Verb            string `json:"verb"`
	Namespace       string `json:"namespace"`
	APIGroup        string `json:"apiGroup"`
	APIVersion      string `json:"apiVersion"`
	Resource        string `json:"resource"`
	Subresource     string `json:"subresource"`
	Name            string `json:"name"`
	ResourceRequest bool   `json:"resourceRequest"`
	Path            string `json:"path"`
}

// podAuthorityVerify verify whether current user could access to pod
func PodAuthorityVerify(request *restful.Request, response *restful.Response, chain *restful.FilterChain) {
	clog.Info("request path parameters: %v", request.PathParameters())

	// two steps：
	// 1. determine whether the user has permission to operate the pod under the namespace
	// 2. determine whether the operated pod belongs to the namespace
	if !isAuthValid(request) {
		clog.Info("user has no permission to operate the pod or the pod does not belong to the namespace")
		response.WriteHeaderAndEntity(http.StatusUnauthorized, TerminalResponse{Message: "permission denied"})
		return
	}
	if isNsOrPodBelongToNamespace(request) {
		chain.ProcessFilter(request, response)
	} else {
		response.WriteHeaderAndEntity(http.StatusUnauthorized, TerminalResponse{Message: "the pod is not found"})
	}
}

// determine whether the user has permission to operate the pod under the namespace
func isAuthValid(request *restful.Request) bool {
	user := utils.GetUserFromReq(request)
	if user == "" {
		clog.Error("the user is not exists")
		return false
	}
	namespace := request.PathParameter(NamespaceKey)
	attribute := &attributes{user, "get", namespace, "", "", "pods",
		"", "", true, ""}
	bytesData, err := json.Marshal(attribute)
	if err != nil {
		clog.Error("marshal json error: %s", err)
		return false
	}
	// skip tsl verify
	c := &http.Client{
		Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}},
		Timeout:   5 * time.Second,
	}
	resp, _ := c.Post("https://"+utils.GetKubeCubeSvc()+"/api/v1/cube/authorization/access",
		"application/x-www-form-urlencoded", strings.NewReader(string(bytesData)))
	if resp == nil {
		clog.Error("request to kubecube for auth failed, response is nil")
		return false
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	if string(body) == "true" {
		return true
	}
	return false
}

// determine whether the operated pod belongs to the namespace
func isNsOrPodBelongToNamespace(request *restful.Request) bool {
	podName := request.PathParameter("pod")
	namespace := request.PathParameter("namespace")
	clusterName := request.PathParameter("cluster")
	var (
		client = clients.Interface().Kubernetes(clusterName)
		ctx    = context.Background()
		pod    = corev1.Pod{}
	)
	key := types.NamespacedName{Namespace: namespace, Name: podName}
	err := client.Cache().Get(ctx, key, &pod)
	if err != nil {
		clog.Error("get pod failed: %v", err)
		return false
	}
	if len(pod.Name) > 0 {
		return true
	}
	return false
}
