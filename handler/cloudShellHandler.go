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
	"encoding/json"
	"errors"
	"fmt"
	"kubecube-webconsole/constants"
	"kubecube-webconsole/errdef"
	"kubecube-webconsole/utils"
	"math/rand"
	"net/http"

	clog "github.com/astaxie/beego/logs"
	"github.com/emicklei/go-restful"
	"github.com/patrickmn/go-cache"
	v12 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
)

func handleCloudShellExec(request *restful.Request, response *restful.Response) {
	// check user simply
	token := utils.GetTokenFromReq(request)
	var user string
	if token != "" {
		claims := utils.ParseToken(token)
		if claims != nil {
			user = claims.UserInfo.Username
		}
	}

	if user == "" || token == "" {
		clog.Error("user is not exist")
		response.WriteHeaderAndEntity(http.StatusUnauthorized, TerminalResponse{Message: "permission denied"})
		return
	}

	// check cluster exists
	clusterName := request.PathParameter("cluster")
	clusterInfo, err := GetClusterInfoByName(clusterName)

	if err != nil || clusterInfo == nil {
		errdef.HandleInternalErrorByCode(response, errdef.ClusterInfoNotFound)
		return
	}
	// get information of pod and container in control cluster
	v, ok := configMap.Get(constants.ControlClusterName)
	var cfg *rest.Config

	if !ok {
		NCfg, err := getControlCluster()
		if err != nil {
			clog.Error("fail to fetch control cluster, msg: %v", err)
			errdef.HandleInternalErrorByCode(response, errdef.ControlClusterNotFound)
			return
		}
		cfg = NCfg
		configMap.Set(clusterName, cfg, cache.DefaultExpiration)
	} else {
		cfg = v.(*rest.Config)
	}

	controlRestClient, err := rest.RESTClientFor(cfg)

	if err != nil {
		clog.Info("Fail to new rest client from control pane cluster kube config data, from cfg: %#v", cfg)
		errdef.HandleInternalErrorByCode(response, errdef.InternalServerError)
		return
	}

	pods := v12.PodList{}
	err = controlRestClient.Get().Resource("pods").Namespace(CloudShellNs).Param("labelSelector", CloudShellLabelKey+"="+CloudShellDpName).Do(context.Background()).Into(&pods)
	if err != nil {
		clog.Info("Fetch pods of cloud shell fail, err msg: %v", err)
		errdef.HandleInternalError(response, errdef.InternalServerError)
		return
	}
	if len(pods.Items) == 0 {
		clog.Info("No pods of cloud shell available, err msg: %v", err)
		errdef.HandleInternalError(response, errdef.InternalServerError)
		return
	}

	// choose one pod in running status randomly
	runningPod := fetchRandomRunningPod(pods.Items)
	if runningPod == nil {
		clog.Info("No running pod of cloud shell available!")
		errdef.HandleInternalError(response, errdef.NoRunningPod)
		return
	}

	containerName := runningPod.Spec.Containers[0].Name
	podName := runningPod.Name

	shellConnInfo := ConnInfo{
		Namespace:        CloudShellNs,
		PodName:          podName,
		ContainerName:    containerName,
		ClusterName:      constants.ControlClusterName,
		UserName:         user,
		IsControlCluster: true,
		Token:            token,
	}

	connInfoBytes, _ := json.Marshal(shellConnInfo)

	sessionId, err := utils.GenTerminalSessionId()
	if err != nil {
		clog.Error("Generate session id failed. Error msg: " + err.Error())
		errdef.HandleInternalError(response, err)
		return
	}
	clog.Info("SessionId: %s", sessionId)

	// save container-connect info to memory
	connMap.Store(sessionId, string(connInfoBytes))
	response.WriteHeaderAndEntity(http.StatusOK, TerminalResponse{Id: sessionId})
}

func fetchRandomRunningPod(podArr []v12.Pod) *v12.Pod {
	var idxArr []int

	for idx, pod := range podArr {
		if isPodRunning(pod) {
			idxArr = append(idxArr, idx)
		}
	}
	if len(idxArr) == 0 {
		return nil
	}
	randomIdx := rand.Intn(len(idxArr))

	return &podArr[idxArr[randomIdx]]
}

// Returns true if given pod is in state ready or succeeded, false otherwise
func isPodRunning(pod v12.Pod) bool {
	if pod.Status.Phase == v12.PodRunning {
		for _, c := range pod.Status.Conditions {
			if c.Type == v12.PodReady {
				if c.Status == v12.ConditionFalse {
					return false
				}
			}
		}
		return true
	}
	return false
}

func getControlCluster() (cfg *rest.Config, err error) {
	controlCluster, err := GetClusterInfoByName(constants.ControlClusterName)
	if err != nil {
		clog.Error("get control cluster err")
		return nil, errdef.ControlClusterNotFound
	}

	tmpCfg := initKubeConf(string(controlCluster.Spec.KubeConfig))
	if tmpCfg == nil {
		clog.Info("fail to init cfg for control cluster [%s], config: %v", controlCluster.ClusterName, string(controlCluster.Spec.KubeConfig))
	}

	controlRestClient, err := rest.RESTClientFor(tmpCfg)
	if err != nil {
		msg := fmt.Sprintf("Fail to new rest client from control cluster [%s] with  kube config data, from cfg: %#v", controlCluster.ClusterName, tmpCfg)
		clog.Info(msg)
		return nil, errors.New(msg)
	}

	pods := v12.PodList{}
	err = controlRestClient.Get().Resource("pods").Namespace(CloudShellNs).Param("labelSelector", CloudShellLabelKey+"="+CloudShellDpName).Do(context.Background()).Into(&pods)
	if err != nil {
		msg := fmt.Sprintf("Fetch pods of cloud shell fail in control cluster [%s] fail, err msg: %v", controlCluster.ClusterName, err)
		clog.Info(msg)
		return nil, errors.New(msg)
	}

	if len(pods.Items) == 0 {
		msg := fmt.Sprintf("No pods of cloud shell in control cluster [%s]", controlCluster.ClusterName)
		clog.Info(msg)
		return nil, errors.New(msg)
	} else {
		cfg = tmpCfg
	}

	if cfg == nil {
		msg := fmt.Sprintf("Fail to get control cluster where pod of cloud-shell backend dp [%v] in namespace [%s] more than one!! please check if valid control cluster in Dd", CloudShellDpName, CloudShellNs)
		clog.Error(msg)
		return nil, errors.New(msg)
	}

	return cfg, nil

}
