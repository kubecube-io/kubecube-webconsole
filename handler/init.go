package handler

import (
	"github.com/kubecube-io/kubecube/pkg/clients/kubernetes"
	"github.com/patrickmn/go-cache"
)

const (
	KeyExpiredSeconds    = 300
	CacheCleanupInterval = 600
	PivotCluster         = "pivot-cluster"
)

var K8sClient kubernetes.Client

var C *cache.Cache

func initConfig() {
	CloudShellDpName = *cloudShellDpName
	CloudShellNs = *appNamespace
	C = cache.New(KeyExpiredSeconds, CacheCleanupInterval)
}
