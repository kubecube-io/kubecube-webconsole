package handler

import (
	"context"
	logger "github.com/astaxie/beego/logs"
	clusterv1 "github.com/kubecube-io/kubecube/pkg/apis/cluster/v1"
	"github.com/kubecube-io/kubecube/pkg/multicluster"
	"k8s.io/apimachinery/pkg/types"
)

func GetClusterInfoByName(clusterName string) (clusterInfo *clusterv1.Cluster, err error) {
	if clusterName == "" {
		return nil, nil
	}
	var (
		client  = K8sClient
		ctx     = context.Background()
		cluster = clusterv1.Cluster{}
	)
	key := types.NamespacedName{Name: clusterName}
	err = client.Cache().Get(ctx, key, &cluster)
	if err != nil {
		logger.Error("get cluster failed: %v", err)
		return nil, err
	}
	return &cluster, nil

}

/**
  管控集群理论上只有一个，只返回第一个
*/
func GetControlClusterInfo() (clusterInfo []*clusterv1.Cluster, err error) {
	var controlClusters []*clusterv1.Cluster
	clusters := multicluster.Interface().FuzzyCopy()
	for _, fuzzyCluster := range clusters {
		cluster, err := GetClusterInfoByName(fuzzyCluster.Name)
		if cluster != nil && err != nil {
			if cluster.Spec.IsMemberCluster {
				controlClusters = append(controlClusters, cluster)
			}
		}
	}
	return controlClusters, nil
}
