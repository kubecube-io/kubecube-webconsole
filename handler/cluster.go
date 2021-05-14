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
	logger "github.com/astaxie/beego/logs"
	clusterv1 "github.com/kubecube-io/kubecube/pkg/apis/cluster/v1"
	"github.com/kubecube-io/kubecube/pkg/clients"
	"k8s.io/apimachinery/pkg/types"
)

func GetClusterInfoByName(clusterName string) (clusterInfo *clusterv1.Cluster, err error) {
	if clusterName == "" {
		return nil, nil
	}
	var (
		client  = clients.Interface().Kubernetes(clusterName)
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
