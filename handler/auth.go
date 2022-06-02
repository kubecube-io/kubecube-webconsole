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
	v1 "github.com/kubecube-io/kubecube/pkg/apis/cluster/v1"
	"github.com/kubecube-io/kubecube/pkg/clients"
	"github.com/kubecube-io/kubecube/pkg/utils/constants"
	"github.com/kubecube-io/kubecube/pkg/utils/kubeconfig"
	"io/ioutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
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
	resp, err := c.Post("https://"+utils.GetKubeCubeSvc()+"/api/v1/cube/authorization/access",
		"application/json", strings.NewReader(string(bytesData)))
	if err != nil {
		clog.Error(err.Error())
		return false
	}
	if resp == nil {
		clog.Error("request to kubecube for auth failed, response is nil")
		return false
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	if string(body) == "true" {
		return true
	}
	clog.Debug("kubecube auth response is false.")
	return false
}

// determine whether the operated pod belongs to the namespace
func isNsOrPodBelongToNamespace(request *restful.Request) bool {
	podName := request.PathParameter("pod")
	namespace := request.PathParameter("namespace")
	clusterName := request.PathParameter("cluster")
	pivotClient := clients.Interface().Kubernetes(constants.LocalCluster)
	memberCluster := v1.Cluster{}
	pivotClient.Cache().Get(request.Request.Context(), types.NamespacedName{Name: clusterName}, &memberCluster)

	config := memberCluster.Spec.KubeConfig
	kubeConfig, err := kubeconfig.LoadKubeConfigFromBytes(config)
	if err != nil {
		clog.Error("convert kubeconfig error: %s", err)
		return false
	}
	clientSet, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		clog.Error("problem new raw k8s clientSet: %v", err)
		return false
	}

	pod, err := clientSet.CoreV1().Pods(namespace).Get(context.Background(), podName, metav1.GetOptions{})
	if err != nil {
		clog.Error("get pod error: %s", err)
	}
	if len(pod.Name) > 0 {
		return true
	}
	return false
}
