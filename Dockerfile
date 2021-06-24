# Copyright 2021 The KubeCube Authors. All rights reserved.
# Use of this source code is governed by a Apache license
# that can be found in the LICENSE file.

# Build the manager binary
FROM golang:1.15 as builder

# Copy in the go src
WORKDIR /go/src/kubecube-webconsole
COPY . .

RUN git config --global url."https://JiahuiZhao11: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@github.com".insteadOf "https://github.com"
RUN go mod download


# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o webconsole main.go

# Copy the ripple into a thin image
FROM debian:stretch-slim
WORKDIR /
COPY --from=builder /go/src/kubecube-webconsole/webconsole .
ENTRYPOINT ["/webconsole"]