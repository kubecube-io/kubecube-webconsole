# Copyright 2021 The KubeCube Authors. All rights reserved.
# Use of this source code is governed by a Apache license
# that can be found in the LICENSE file.

FROM ubuntu:16.04

RUN apt-get update \
    && apt-get update && apt-get install -y kubectl vim tar wget curl rsync bzip2 iptables tcpdump less telnet net-tools lsof sysstat cron supervisor inetutils-ping \
    && rm -rf /var/lib/apt/lists/*

COPY ./cloudshell/build/supervisord.conf /etc/supervisor/conf.d/supervisord.conf
COPY ./cloudshell/build/kubecube-chroot.sh /kubecube-chroot.sh
RUN chmod +x /kubecube-chroot.sh

CMD ["/usr/bin/supervisord"]