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

	"kubecube-webconsole/handler"
)

func init() {
	clients.InitCubeClientSetWithOpts(nil)
}

const (
	initPhase = iota
	masterPhase
	subsPhase
)

type httpRunner struct {
	server *http.Server
	// phase is the runner current phase
	phase int
	// subStopCn notify sub exit
	subStopCn chan struct{}
	// subExitCh notify sub exit complete
	subExitCh chan struct{}
	// masterExitCh notify master exit complete
	masterExitCh chan struct{}
}

func (r *httpRunner) registerApis(asMaster bool) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		logger.Debug("Health check")
		w.WriteHeader(http.StatusOK)
	})

	if asMaster {
		mux.Handle("/api/", handler.CreateHTTPAPIHandler())
		mux.Handle("/api/sockjs/", handler.CreateAttachHandler("/api/sockjs"))
		mux.HandleFunc("/leader", func(response http.ResponseWriter, request *http.Request) {
			logger.Debug("This is leader")
			response.WriteHeader(http.StatusOK)
		})
	}

	return mux
}

func (r *httpRunner) runAsMaster(ctx context.Context) {
	clog.Info("run as master server")

	if r.phase == subsPhase {
		// notify subs exit
		r.subStopCn <- struct{}{}
		// wait subs exit
		<-r.subExitCh
	}

	r.phase = masterPhase

	mux := r.registerApis(true)
	r.server.Handler = mux

	go func() {
		if err := r.server.ListenAndServe(); err != nil {
			clog.Fatal("ListenAndServe failed，error msg: %s", err.Error())
		}
	}()

	// lost leader then exit
	<-ctx.Done()

	clog.Info("shutting down master server")

	timeoutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := r.server.Shutdown(timeoutCtx); err != nil {
		clog.Fatal("cube apiserver forced to shutdown: %v", err)
	}

	clog.Info("master server exiting")
	r.masterExitCh <- struct{}{}
}

func (r *httpRunner) runAsSub() {
	if r.phase == masterPhase {
		// wait mater exit
		<-r.masterExitCh
	}

	// only called once as subs
	if r.phase != subsPhase {
		clog.Info("run as subsidiary server")
		mux := r.registerApis(false)
		r.server.Handler = mux
		go func() {
			if err := r.server.ListenAndServe(); err != nil {
				clog.Fatal("ListenAndServe failed，error msg: %s", err.Error())
			}
		}()

		r.phase = subsPhase

		// become leader then exit
		<-r.subStopCn

		clog.Info("shutting down subsidiary server")

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := r.server.Shutdown(ctx); err != nil {
			clog.Fatal("cube apiserver forced to shutdown: %v", err)
		}

		clog.Info("subsidiary server exiting")
		r.subExitCh <- struct{}{}
	}
}

func main() {
	clog.InitCubeLoggerWithOpts(&clog.Config{LogLevel: "info", StacktraceLevel: "error"})
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

	runner := &httpRunner{
		server: &http.Server{
			Addr: fmt.Sprintf(":%d", *handler.ServerPort),
		},
		subStopCn: make(chan struct{}, 1),
		subExitCh: make(chan struct{}, 1),
		phase:     initPhase,
	}

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
				runner.runAsMaster(ctx)
			},
			OnStoppedLeading: func() {
				glog.Infoln("leader election lost")
			},
			OnNewLeader: func(identity string) {
				if identity != hostname {
					runner.runAsSub()
				}
			},
		},
	})
	if err != nil {
		glog.Errorf("leader election fail, be member")
	}
	le.Run(context.Background())
}