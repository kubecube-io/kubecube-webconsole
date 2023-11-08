# Copyright 2021 The KubeCube Authors. All rights reserved.
# Use of this source code is governed by a Apache license
# that can be found in the LICENSE file.

FROM harbor.cloud.netease.com/qztest/kubecube/ubuntu:remove-ssh-for-cloudshell-multi
COPY kubecube-chroot.sh /kubecube-chroot.sh
COPY kubectl /bin/kubectl
RUN chmod +x /kubecube-chroot.sh