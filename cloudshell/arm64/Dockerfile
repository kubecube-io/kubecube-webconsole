# Copyright 2021 The KubeCube Authors. All rights reserved.
# Use of this source code is governed by a Apache license
# that can be found in the LICENSE file.

FROM hub.c.163.com/kubecube/ubuntu:16.04-tools
COPY ./arm64/kubecube-chroot.sh /kubecube-chroot.sh
COPY ./arm64/kubectl /bin/kubectl
RUN chmod +x /kubecube-chroot.sh