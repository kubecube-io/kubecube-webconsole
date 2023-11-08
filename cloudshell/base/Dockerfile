# Copyright 2021 The KubeCube Authors. All rights reserved.
# Use of this source code is governed by a Apache license
# that can be found in the LICENSE file.

FROM ubuntu:16.04
MAINTAINER netease
RUN apt-get update \
    && apt-get update && apt-get install -y vim tar wget curl rsync bzip2 iptables tcpdump less telnet net-tools lsof sysstat cron supervisor inetutils-ping \
    && rm -rf /var/lib/apt/lists/*
#RUN cp -f /usr/share/zoneinfo/Asia/Shanghai /etc/localtime
CMD ["/usr/bin/supervisord"]