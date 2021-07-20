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

package constants

const (
	// root route
	ApiPathRoot = "/api/v1/cube"

	// kubecube default namespace
	CubeNamespace = "kubecube-system"

	// pivot cluster name
	PivotCluster = "pivot-cluster"

	// warden deployment name
	WardenDeployment = "warden"

	// default pivot cube host
	DefaultPivotCubeHost = "cube.kubecube.io"

	// default pivot cube headless svc
	DefaultPivotCubeHeadlessSvc = "kubecube.kubecube-system.svc:7443"
	DefaultAuditSvc             = "audit.kubecube-system.svc:7443"

	HttpHeaderContentType        = "Content-type"
	HttpHeaderContentDisposition = "Content-Disposition"
	HttpHeaderContentTypeOctet   = "application/octet-stream"

	// TenantLabel represent which tenant resource relate with
	TenantLabel = "kubecube.io/tenant"

	// ProjectLabel represent which project resource relate with
	ProjectLabel = "kubecube.io/project"

	K8sResourceVersion   = "v1"
	K8sResourceNamespace = "namespaces"

	// audit
	EventName          = "event"
	EventTypeUserWrite = "userwrite"
	EventResourceType  = "resourceType"
	EventAccountId     = "accountId"

	// user
	AuthorizationHeader = "Authorization"

	// build-in cluster role
	ClusterRolePlatformAdmin = "platform-admin"
	ClusterRoleProjectAdmin  = "project-admin"
	ClusterRoleTenantAdmin   = "tenant-admin"
	ClusterRoleReviewer      = "reviewer"
)
