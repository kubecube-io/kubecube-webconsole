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

package scout

import (
	"context"
	"fmt"
	"time"

	"github.com/kubecube-io/kubecube/pkg/clog"

	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/kubecube-io/kubecube/pkg/apis/cluster/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	defaultInitialDelaySeconds = 10
	defaultWaitTimeoutSeconds  = 10
)

// Scout collects information from warden
type Scout struct {
	// LastHeartbeat record last heartbeat
	LastHeartbeat time.Time

	// Heartbeat not receive timeout
	WaitTimeoutSeconds int

	// wait for warden start
	InitialDelaySeconds int

	// the cluster where the warden watch for
	Cluster string

	// is scout normal or not
	Normal bool

	// receive warden info form api
	Receiver chan WardenInfo

	// k8s client
	Client client.Client

	// use to stop scout for
	StopCh chan struct{}
}

// WardenInfo contains intelligence within communication
type WardenInfo struct {
	Cluster    string    `json:"cluster"`
	ReportTime time.Time `json:"reportTime"`
}

func NewScout(cluster string, initialDelay, waitTimeoutSeconds int, cli client.Client, stopCh chan struct{}) *Scout {
	if initialDelay == 0 {
		initialDelay = defaultInitialDelaySeconds
	}
	if waitTimeoutSeconds == 0 {
		waitTimeoutSeconds = defaultWaitTimeoutSeconds
	}

	s := &Scout{
		Cluster:             cluster,
		Receiver:            make(chan WardenInfo),
		InitialDelaySeconds: initialDelay,
		WaitTimeoutSeconds:  waitTimeoutSeconds,
		Client:              cli,
		StopCh:              stopCh,
	}

	return s
}

// Collect will scout a specified warden of cluster
func (s *Scout) Collect(ctx context.Context) {
	for {
		select {
		case info := <-s.Receiver:
			s.healthWarden(ctx, info)

		case <-time.Tick(time.Duration(s.WaitTimeoutSeconds) * time.Second):
			s.illWarden(ctx)

		case <-ctx.Done():
			clog.Warn("probe context exceed: %v", ctx.Err())
			return
		}
	}
}

// healthWarden be called once when receive heartbeat first
// todo: populate network delay with watden info
func (s *Scout) healthWarden(ctx context.Context, info WardenInfo) {
	s.LastHeartbeat = time.Now()

	if !s.Normal {
		state := v1.ClusterNormal
		reason := fmt.Sprintf("receive heartbeat from cluster %s", s.Cluster)

		err := s.updateClusterStatus(state, reason, ctx)
		if err != nil {
			clog.Error(err.Error())
		}
	}

	s.Normal = true
}

// illWarden do callback when warden ill
func (s *Scout) illWarden(ctx context.Context) {
	if s.Normal {
		state := v1.ClusterAbnormal
		reason := fmt.Sprintf("cluster %s disconnected", s.Cluster)
		clog.Info("%v, last heartbeat: %v", reason, s.LastHeartbeat)

		err := s.updateClusterStatus(state, reason, ctx)
		if err != nil {
			clog.Error(err.Error())
		}
	}

	s.Normal = false
}

func (s *Scout) updateClusterStatus(state v1.ClusterState, reason string, ctx context.Context) error {
	c := s.Client

	obj := &v1.Cluster{}
	err := c.Get(ctx, types.NamespacedName{Name: s.Cluster}, obj)
	if err != nil {
		return err
	}

	obj.Status.State = &state
	obj.Status.Reason = reason
	err = c.Status().Update(ctx, obj, &client.UpdateOptions{})
	if err != nil {
		return err
	}

	return nil
}
