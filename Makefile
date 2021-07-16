# Copyright 2021 The KubeCube Authors. All rights reserved.
# Use of this source code is governed by a Apache license
# that can be found in the LICENSE file.

IMG ?= hub.c.163.com/kubecube/webconsole:v1.0.0
MULTI_ARCH ?= false

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

GOFILES=$(shell find . -name "*.go" -type f -not -path "./vendor/*")

all: build

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

vendor:
	go mod vendor

docker-build-webconsole: vendor #test ## Build docker image with the manager.
	docker build -f ./Dockerfile -t ${IMG} .

docker-build-webconsole-multi-arch: vendor #test
	MULTI_ARCH=true
	docker buildx build -f ./Dockerfile -t ${IMG} --platform=linux/arm,linux/arm64,linux/amd64 . --push

docker-build-cloudshell: vendor #test
	docker build -f ./cloudshell/Dockerfile -t ${IMG} .

docker-build-cloudshell-multi-arch: vendor #test
	MULTI_ARCH=true
	docker buildx build -f ./cloudshell/Dockerfile -t ${IMG} --platform=linux/arm,linux/arm64,linux/amd64 . --push