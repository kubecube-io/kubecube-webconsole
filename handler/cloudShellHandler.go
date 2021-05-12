package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"kubecube-webconsole/errdef"
	"kubecube-webconsole/utils"
	"math/rand"
	"net/http"

	logger "github.com/astaxie/beego/logs"
	"github.com/emicklei/go-restful"
	"github.com/patrickmn/go-cache"
	v12 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
)

/**
  处理页面使用kubectl的逻辑
*/

func handleCloudShellExec(request *restful.Request, response *restful.Response) {
	// 简单检查一下accountId
	accountId := request.HeaderParameter(AccountIdKey)
	if accountId == "" {
		response.WriteHeaderAndEntity(http.StatusUnauthorized, TerminalResponse{Message: "permission denied"})
		return
	}

	// 检查集群是否存在
	clusterName := request.PathParameter("cluster")
	clusterInfo, err := GetClusterInfoByName(clusterName)

	if err != nil || clusterInfo == nil {
		errdef.HandleInternalErrorByCode(response, errdef.ClusterInfoNotFound)
		return
	}
	// 获取管控集群上的pod和container信息
	v, ok := configMap.Get(ControlClusterName)
	var cfg *rest.Config

	if !ok {

		clusterName, NCfg, err := getControlCluster()
		if err != nil {
			logger.Error("fail to fetch control cluster info from db, msg: %v", err)
			errdef.HandleInternalErrorByCode(response, errdef.ControlClusterNotFound)
			return
		}

		ControlClusterName = clusterName
		cfg = NCfg
		configMap.Set(clusterName, cfg, cache.DefaultExpiration)
	} else {
		cfg = v.(*rest.Config)
	}

	controlRestClient, err := rest.RESTClientFor(cfg)

	if err != nil {
		logger.Info("Fail to new rest client from control pane cluster kube config data, from cfg: %#v", cfg)
		errdef.HandleInternalErrorByCode(response, errdef.InternalServerError)
		return
	}

	pods := v12.PodList{}
	err = controlRestClient.Get().Resource("pods").Namespace(CloudShellNs).Param("labelSelector", NCS_CLOUD_SHELL_LABEL_KEY+"="+CloudShellDpName).Do(context.Background()).Into(&pods)
	if err != nil {
		logger.Info("Fetch pods of cloud shell fail, err msg: %v", err)
		errdef.HandleInternalError(response, errdef.InternalServerError)
		return
	}
	if len(pods.Items) == 0 {
		logger.Info("No pods of cloud shell available, err msg: %v", err)
		errdef.HandleInternalError(response, errdef.InternalServerError)
		return
	}

	// 选择running状态的pod，并且随机选择一个
	runningPod := fetchRandomRunningPod(pods.Items)
	if runningPod == nil {
		logger.Info("No running pod of cloud shell available!")
		errdef.HandleInternalError(response, errdef.NoRunningPod)
		return
	}

	containerName := runningPod.Spec.Containers[0].Name
	podName := runningPod.Name

	shellConnInfo := ConnInfo{
		TenantId:         "1",
		Namespace:        CloudShellNs,
		PodName:          podName,
		ContainerName:    containerName,
		ClusterName:      ControlClusterName,
		ClusterId:        clusterName,
		AccountId:        accountId,
		IsControlCluster: true,
	}

	connInfoBytes, _ := json.Marshal(shellConnInfo)

	sessionId, err := utils.GenTerminalSessionId()
	if err != nil {
		logger.Error("Generate session id failed. Error msg: " + err.Error())
		errdef.HandleInternalError(response, err)
		return
	}
	logger.Info("SessionId: %s", sessionId)

	// Save container-connect info to memory
	connMap.Store(sessionId, string(connInfoBytes))
	// Save container-connect info to cache
	C.Set(sessionId, connInfoBytes, KeyExpiredSeconds)
	response.WriteHeaderAndEntity(http.StatusOK, TerminalResponse{Id: sessionId})
}

// admin auth verify whether operator is admin
func adminAuthorityVerify(request *restful.Request, response *restful.Response, chain *restful.FilterChain) {
	// 鉴权，判断是否有权操作集群即可
	if !isRequestAdmin(request) {
		response.WriteHeaderAndEntity(http.StatusUnauthorized, TerminalResponse{Message: "permission denied"})
		return
	}
	chain.ProcessFilter(request, response)
}

func isRequestAdmin(request *restful.Request) bool {
	accountId := request.HeaderParameter(AccountIdKey)
	jwtToken := request.HeaderParameter(JwtTokenKey)

	authResult := authRequest(SystemScopeId, ParentOfSystem, accountId, jwtToken, ClusterAdd, ClusterRes)

	if authResult == nil {
		return false
	}

	realAccountId := accountId

	for _, opResult := range authResult.AuthenticationResults {
		if !opResult.HasRole {
			return false
		}
		if len(opResult.AccountId) > 0 {
			realAccountId = opResult.AccountId
		}
	}

	request.SetAttribute(AccountIdKey, realAccountId)

	return true
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

/**
鉴权请求函数
*/
func authRequest(scopeId string, parentId string, accountId string, jwtToken string, operation string, opRes string) *AuthResp {
	authInfo := AuthInfo{
		Jwt:               jwtToken,
		Operation:         operation,
		ResourceType:      opRes,
		Index:             0,
		ParentId:          parentId,
		PermissionScopeId: scopeId,
		AccountId:         accountId,
	}
	authReq := AuthReq{
		AuthenticationParams: []AuthInfo{authInfo},
		ServiceModule:        ServiceModeTypeNCS,
	}

	url := *authEndpoint + PlatformAuthUrl

	reqBody, _ := json.Marshal(authReq)

	client := &http.Client{}

	req, err := http.NewRequest("POST", url, bytes.NewReader(reqBody))

	if err != nil {
		logger.Info("fail to make auth request")
	}

	// 响应流最后务必关闭
	resp, err := client.Do(req)

	if err != nil {
		logger.Error("error when perform auth request!")
		return nil
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger.Error("read from response reqBody failed, error msg: %v", err)
		return nil
	}

	var authResp AuthResp

	err = json.Unmarshal(body, &authResp)

	if err != nil {
		logger.Error("unmarshal response body failed, error msg: %v", err)
		return nil
	}

	return &authResp
}

func getControlCluster() (clusterName string, cfg *rest.Config, err error) {
	controlClusters, err := GetControlClusterInfo()
	if err != nil {
		logger.Error("No control cluster info in db, please add control cluster info into db!!!!")
		return "", nil, errors.New("No control cluster info in db, please add control cluster info into db!!!!")
	}
	for _, controlCluster := range controlClusters {

		tmpCfg := initKubeConf(string(controlCluster.Spec.KubeConfig))

		if tmpCfg == nil {
			logger.Info("fail to init cfg for control cluster [%s], config: %v", controlCluster.ClusterName, string(controlCluster.Spec.KubeConfig))
			continue
		}

		controlRestClient, err := rest.RESTClientFor(tmpCfg)

		if err != nil {
			logger.Info("Fail to new rest client from control cluster [%s] with  kube config data, from cfg: %#v", controlCluster.ClusterName, tmpCfg)
			continue
		}

		pods := v12.PodList{}
		err = controlRestClient.Get().Resource("pods").Namespace(CloudShellNs).Param("labelSelector", NCS_CLOUD_SHELL_LABEL_KEY+"="+CloudShellDpName).Do(context.Background()).Into(&pods)
		if err != nil {
			logger.Info("Fetch pods of cloud shell fail in control cluster [%s] fail, err msg: %v", controlCluster.ClusterName, err)
			continue
		}

		if len(pods.Items) == 0 {
			logger.Info("No pods of cloud shell in control cluster [%s], try to find it in next control cluster, if more exist", controlCluster.ClusterName)
		} else {
			cfg = tmpCfg
			clusterName = controlCluster.ClusterName
			break
		}
	}

	if cfg == nil {
		msg := fmt.Sprintf("Fail to get any control cluster where pod of cloud-shell backend dp [%v] in namespace [%s] more than one!! please check if valid control cluster in Dd", CloudShellDpName, CloudShellNs)
		logger.Error(msg)
		return "", nil, errors.New(msg)
	}

	return

}
