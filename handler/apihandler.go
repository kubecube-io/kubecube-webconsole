package handler

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	logger "github.com/astaxie/beego/logs"
	"github.com/emicklei/go-restful"
	"github.com/patrickmn/go-cache"
	"gopkg.in/igm/sockjs-go.v2/sockjs"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"kubecube-webconsole/errdef"
	"kubecube-webconsole/utils"
	"net/http"
)

func init() {
	logger.Info("webconsole initializing")
	flag.Parse()

	initConfig()

	logger.Info("webconsole initialized")
}

func CreateHTTPAPIHandler() http.Handler {
	wsContainer := restful.NewContainer()
	wsContainer.EnableContentEncoding(true)

	apiV1Ws := new(restful.WebService)
	apiV1Ws.Filter(PodAuthorityVerify)

	apiV1Ws.Path("/api/v1").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	apiV2Ws := new(restful.WebService)
	//apiV2Ws.Filter(adminAuthorityVerify) // 放开，普通用户可访问

	apiV2Ws.Path("/api/v1/extends").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	wsContainer.Add(apiV1Ws)
	wsContainer.Add(apiV2Ws)

	apiV1Ws.Route(
		apiV1Ws.GET("{cluster}/pod/{namespace}/{pod}/shell/{container}").
			To(handleExecShell).
			Writes(TerminalResponse{}))
	apiV2Ws.Route(
		apiV2Ws.GET("cloudShell/clusters/{cluster}").
			To(handleCloudShellExec).
			Writes(TerminalResponse{}))

	return wsContainer

}

// CreateAttachHandler is called from main for /api/sockjs
func CreateAttachHandler(path string) http.Handler {
	return sockjs.NewHandler(path, sockjs.DefaultOptions, handleTerminalSession)
}

// Handles execute shell API call
func handleExecShell(request *restful.Request, response *restful.Response) {
	name := request.PathParameter("cluster")

	sessionId, err := utils.GenTerminalSessionId()
	if err != nil {
		logger.Error("generate session id failed. Error msg: " + err.Error())
		errdef.HandleInternalError(response, err)
		return
	}
	logger.Info("sessionId: %s", sessionId)

	// 根据集群名从map中获取相应的restClient配置
	_, err = getNonControlCfgFromCacheAndDbIfCacheFail(name)
	if err != nil {
		// 如果获取不到配置，可能是动态增加的一个新集群信息，需要重新获取
		logger.Error("fail to fetch rest.config for cluster [%s], msg: %v", name, err)
		errdef.HandleInternalErrorByCode(response, errdef.ClusterInfoNotFound)
		return
	}

	cInfo := getConnInfo(request)
	C.Set(sessionId, cInfo, KeyExpiredSeconds)

	response.WriteHeaderAndEntity(http.StatusOK, TerminalResponse{Id: sessionId})
}

func cacheConnInfo(sessionId string, info *ConnInfo) {
	v, _ := json.Marshal(info)

	// save container-connect info to sync.Map
	connMap.Store(sessionId, string(v))
	// save container info to cache with an expire time
	C.Set(sessionId, v, KeyExpiredSeconds)
}

//todo
func getConnInfo(request *restful.Request) *ConnInfo {
	tenantId := request.PathParameter("tenantId")
	clusterName := request.PathParameter("cluster")
	namespace := request.PathParameter("namespace")
	podName := request.PathParameter("pod")
	containerName := request.PathParameter("container")

	scriptUser := request.QueryParameter("user")
	scriptUID := request.QueryParameter("uid")
	scriptUserAuth := request.QueryParameter("auth")

	// Audit related info
	webUser := request.QueryParameter("webuser")
	platform := request.QueryParameter("platform")
	// 未传入platform信息时，认为是轻舟页面传入的
	if platform == "" {
		platform = PlatformSkiff
	}

	remoteIP := request.QueryParameter("remote_ip")
	if remoteIP == "" {
		remoteIP = request.HeaderParameter("X-Forwarded-For")
	}

	ua := request.QueryParameter("user_agent")
	if ua == "" {
		ua = request.HeaderParameter("User-Agent")
	}

	return &ConnInfo{
		TenantId:       tenantId,
		Namespace:      namespace,
		PodName:        podName,
		ContainerName:  containerName,
		ClusterName:    clusterName,
		ScriptUID:      scriptUID,
		ScriptUser:     scriptUser,
		ScriptUserAuth: scriptUserAuth,
		AuditRawInfo: &AuditRawInfo{
			RemoteIP:  remoteIP,
			UserAgent: ua,
			WebUser:   webUser,
			Platform:  platform,
		},
	}
}

// 通过kubeconfig文件内容初始化rest.Config
func initKubeConf(kubeConfData string) *rest.Config {
	var err error
	cfg, err := clientcmd.RESTConfigFromKubeConfig([]byte(kubeConfData))
	if err != nil {
		logger.Critical("init kubeconfig failed, error msg: %s", err.Error())
		return nil
	}
	groupVersion := schema.GroupVersion{
		Group:   "",
		Version: "v1",
	}
	cfg.GroupVersion = &groupVersion
	cfg.APIPath = "/api"
	cfg.ContentType = runtime.ContentTypeJSON
	cfg.NegotiatedSerializer = scheme.Codecs
	return cfg
}

/**
  从cache中根据集群名字获取cfg，cache可能失效，则尝试从数据库获取并且更新cache
*/
func getNonControlCfgFromCacheAndDbIfCacheFail(clusterName string) (cfg *rest.Config, err error) {
	v, ok := configMap.Get(clusterName)
	if ok {
		return v.(*rest.Config), nil
	}
	// 按照正常方式从数据库表中获取
	logger.Info("cluster [%s] config expire ot not exist in cache, try to fetch in db", clusterName)
	ci, err := GetClusterInfoByName(clusterName)
	if err != nil {
		return nil, err
	}
	data := string(ci.Spec.KubeConfig)
	// 初始化restClient的配置，存入map中
	NCfg := initKubeConf(data)
	if NCfg != nil {
		logger.Info("init rest client for cluster [%s] from config from db success", clusterName)
		configMap.Set(clusterName, NCfg, cache.DefaultExpiration)
	} else {
		msg := fmt.Sprintf("init rest client for cluster [%s] from config from db Fail, config data: %v", clusterName, data)
		logger.Error(msg)
		return nil, errors.New(msg)
	}
	return NCfg, nil
}
