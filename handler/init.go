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
	"github.com/patrickmn/go-cache"
	"k8s.io/klog/v2"
	"net/http"
	"time"
)

func initConfig() {
	CloudShellDpName = *cloudShellDpName
	CloudShellNs = *appNamespace
	configMap = cache.New(5*time.Minute, 5*time.Minute)
}

func initAudit() {
	if !*enableAudit {
		klog.Info("audit disabled")
		return
	}

	// 初始化http client
	httpClient := &http.Client{
		Transport: &http.Transport{
			MaxIdleConnsPerHost: MaxIdlePerHost,
		},
		Timeout: RequestTimeout * time.Second,
	}
	AuditAdapter = &auditAdapter{}
	AuditAdapter.HttpClient = httpClient
	AuditAdapter.URL = *auditURL
	AuditAdapter.Method = *auditMethod
	AuditAdapter.Header = *auditHeader
	klog.Infof("audit init success, url: %v", AuditAdapter.URL)
}
