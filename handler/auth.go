package handler

import (
	"context"
	logger "github.com/astaxie/beego/logs"
	"github.com/emicklei/go-restful"
	v1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"kubecube-webconsole/utils"
	"net/http"
	ctrl "sigs.k8s.io/controller-runtime"
)

// podAuthorityVerify verify whether current account could access to pod
func PodAuthorityVerify(request *restful.Request, response *restful.Response, chain *restful.FilterChain) {
	logger.Info("request path parameters: %v", request.PathParameters())

	// 鉴权分两步：
	// 1. 判断用户是否有权限操作该namespace下的pod
	// 2. 判断操作的pod是否是属于传入的namespace
	if !isAuthValid(request) {
		response.WriteHeaderAndEntity(http.StatusUnauthorized, TerminalResponse{Message: "permission denied"})
		return
	}
	if isNsOrPodBelongToNamespace(request) {
		chain.ProcessFilter(request, response)
	} else {
		response.WriteHeaderAndEntity(http.StatusUnauthorized, TerminalResponse{Message: "permission denied"})
	}
}

// 判断用户是否有权限操作该namespace下的pod
func isAuthValid(request *restful.Request) bool {
	user := utils.GetUserFromReq(request)
	if user == "" {
		return false
	}
	namespace := request.PathParameter(NamespaceKey)
	accessReview := makeSubjectAccessReview(user, namespace)
	client, err := kubernetes.NewForConfig(ctrl.GetConfigOrDie())
	if err != nil {
		logger.Error("problem new raw k8s clientSet: %v", err)
		return false
	}
	r, err := client.AuthorizationV1().SubjectAccessReviews().Create(context.Background(), &accessReview, metav1.CreateOptions{})
	if err != nil {
		logger.Error("%v", err)
		return false
	}
	return r.Status.Allowed

}

// makeSubjectAccessReview consider user has visible view of given namespace
// if user can get pods in that namespace.
func makeSubjectAccessReview(user, namespace string) v1.SubjectAccessReview {
	return v1.SubjectAccessReview{
		Spec: v1.SubjectAccessReviewSpec{
			User: user,
			ResourceAttributes: &v1.ResourceAttributes{
				Name:      "pods",
				Namespace: namespace,
				Verb:      "get, list, watch",
			},
		},
	}
}

// 判断操作的pod是否是属于传入的namespace
func isNsOrPodBelongToNamespace(request *restful.Request) bool {
	podName := request.PathParameter("pod")
	namespace := request.PathParameter("namespace")
	var (
		client = K8sClient
		ctx    = context.Background()
		pod    = corev1.Pod{}
	)
	key := types.NamespacedName{Namespace: namespace, Name: podName}
	err := client.Cache().Get(ctx, key, &pod)
	if err != nil {
		logger.Error("get pod failed: %v", err)
		return false
	}
	if len(pod.Name) > 0 {
		return true
	}
	return false
}
