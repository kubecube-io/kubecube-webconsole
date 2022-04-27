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
	"fmt"
	clog "github.com/astaxie/beego/logs"
	clusterv1 "github.com/kubecube-io/kubecube/pkg/apis/cluster/v1"
	"github.com/kubecube-io/kubecube/pkg/clients"
	"github.com/kubecube-io/kubecube/pkg/utils/constants"
	"k8s.io/apimachinery/pkg/types"
)

var pivotCluster *clusterv1.Cluster

func GetClusterInfoByName(clusterName string) (clusterInfo *clusterv1.Cluster, err error) {
	if clusterName == "" {
		return nil, nil
	}
	var (
		pivotClient = clients.Interface().Kubernetes(constants.LocalCluster)
		ctx         = context.Background()
		cluster     = clusterv1.Cluster{}
	)
	key := types.NamespacedName{Name: clusterName}
	err = pivotClient.Cache().Get(ctx, key, &cluster)
	if err != nil {
		clog.Error("get cluster failed: %v", err)
		return nil, err
	}
	return &cluster, nil
}

func GetPivotCluster() (*clusterv1.Cluster, error) {
	if pivotCluster != nil {
		return pivotCluster, nil
	}

	list := &clusterv1.ClusterList{}
	if err := clients.Interface().Kubernetes(constants.LocalCluster).Cache().List(context.Background(), list); err != nil {
		return nil, fmt.Errorf("list clusters failed")
	}

	for _, cluster := range list.Items {
		if cluster.Spec.IsMemberCluster == false {
			pivotCluster = &cluster
			clog.Info("found pivot cluster %v", pivotCluster.Name)
			return pivotCluster, nil
		}
	}

	return nil, fmt.Errorf("can not found pivot cluster")
}
