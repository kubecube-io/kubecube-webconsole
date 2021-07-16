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

package clients

import (
	"github.com/kubecube-io/kubecube/pkg/clients/kubernetes"
	"github.com/kubecube-io/kubecube/pkg/clog"
	"github.com/kubecube-io/kubecube/pkg/multicluster"
)

// Clients aggregates all clients of cube needed
type Clients interface {
	Kubernetes(cluster string) kubernetes.Client
}

var (
	genericClientSet = &cubeClientSet{}
)

type cubeClientSet struct {
	k8s multicluster.MultiClustersManager
}

func InitCubeClientSetWithOpts(opts *Config) {
	genericClientSet.k8s = multicluster.Interface()
}

// Interface the entry for cube client
func Interface() *cubeClientSet {
	return genericClientSet
}

func (c *cubeClientSet) Kubernetes(cluster string) kubernetes.Client {
	client, err := c.k8s.GetClient(cluster)
	if err != nil {
		clog.Error("get internal cluster of cluster %v failed: %v", cluster, err)
	}

	return client
}
