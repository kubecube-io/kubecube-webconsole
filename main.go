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

package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	logger "github.com/astaxie/beego/logs"
	"github.com/golang/glog"
	"github.com/kubecube-io/kubecube/pkg/clients"
	"github.com/kubecube-io/kubecube/pkg/clog"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	_ "net/http/pprof"
	ctrl "sigs.k8s.io/controller-runtime"

	consolelog "kubecube-webconsole/clog"
	"kubecube-webconsole/handler"
)

// leader flag
var leader = false

func init() {
	clients.InitCubeClientSetWithOpts(nil)
}

func main() {
	clog.InitCubeLoggerWithOpts(consolelog.NewLogConfig())
	// hostname is the key to select the master, so it must be terminated if it fails
	hostname, err := os.Hostname()
	if err != nil {
		glog.Fatalf("failed to get hostname: %v", err)
	}

	client, err := kubernetes.NewForConfig(ctrl.GetConfigOrDie())
	if err != nil {
		logger.Error("problem new raw k8s clientSet: %v", err)
		return
	}

	runAPIServer()

	rl, err := resourcelock.New(resourcelock.ConfigMapsResourceLock,
		handler.LeaderElectionNamespace,
		handler.LeaderElectionKey,
		client.CoreV1(), nil,
		resourcelock.ResourceLockConfig{
			Identity: hostname,
		})
	if err != nil {
		glog.Errorf("error creating lock: %v", err)
	}

	le, err := leaderelection.NewLeaderElector(leaderelection.LeaderElectionConfig{
		Lock:          rl,
		LeaseDuration: 15 * time.Second,
		RenewDeadline: 10 * time.Second,
		RetryPeriod:   2 * time.Second,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(ctx context.Context) {
				leader = true
			},
			OnStoppedLeading: func() {
				leader = false
				glog.Infoln("leader election lost")
			},
		},
	})
	if err != nil {
		glog.Errorf("leader election fail, be member")
	}
	le.Run(context.Background())
}

func runAPIServer() {
	// provide api for livenessProbe
	http.HandleFunc("/healthz", func(response http.ResponseWriter, request *http.Request) {
		logger.Debug("Health check")
		response.WriteHeader(http.StatusOK)
	})
	http.Handle("/api/", handler.CreateHTTPAPIHandler())
	http.Handle("/api/sockjs/", handler.CreateAttachHandler("/api/sockjs"))
	// provide api for readinessProbe，avoid service flow into in-leader pod
	http.HandleFunc("/leader", func(response http.ResponseWriter, request *http.Request) {
		statusCode := http.StatusOK
		if !leader {
			statusCode = http.StatusBadRequest
		}
		response.WriteHeader(statusCode)
	})

	go func() {
		err := http.ListenAndServe(fmt.Sprintf(":%d", *handler.ServerPort), nil)
		if err != nil {
			logger.Critical("ListenAndServe failed，error msg: %s", err.Error())
		}
	}()
}
