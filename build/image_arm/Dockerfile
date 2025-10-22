# Copyright 2025 The ChaosBlade Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

FROM multiarch/alpine:arm64-edge
LABEL maintainer="Mingxia Cai"

RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.ustc.edu.cn/g' /etc/apk/repositories

RUN apk add --update bash iproute2 util-linux curl openssl gcc musl-dev libpcap-dev audit coreutils \
    && mkdir -p /lib/modules/$(uname -r) \
    && rm -rf /var/cache/apk/*

RUN curl -L https://get.helm.sh/helm-v3.6.0-linux-arm64.tar.gz | tar xz \
    && mv linux-arm64/helm /bin/helm && rm -rf linux-arm64

ENV HELM_EXPERIMENTAL_OCI 1

WORKDIR /opt
WORKDIR /root/chaos

COPY ./agent /root/chaos
ARG BLADE_VERSION=0.0.1

RUN curl -L https://chaosblade.oss-cn-hangzhou.aliyuncs.com/agent/github/${BLADE_VERSION}/chaosblade-${BLADE_VERSION}-linux-arm64.tar.gz | tar xz \
    && mv chaosblade-${BLADE_VERSION} /opt/chaosblade

ENTRYPOINT ["/root/chaos/agent"]
