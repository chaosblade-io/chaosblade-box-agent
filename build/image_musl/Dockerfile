FROM alpine:3.8
LABEL maintainer="Mingxia Cai"

RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.tuna.tsinghua.edu.cn/g' /etc/apk/repositories

RUN apk add --update bash iproute2 util-linux curl openssl gcc musl-dev libpcap-dev audit coreutils \
    && mkdir -p /lib/modules/$(uname -r) \
    && rm -rf /var/cache/apk/*

RUN curl -L https://get.helm.sh/helm-v3.6.0-linux-amd64.tar.gz | tar xz \
    && mv linux-amd64/helm /bin/helm && rm -rf linux-amd64

ENV HELM_EXPERIMENTAL_OCI 1

RUN mkdir -p /opt
WORKDIR /opt
WORKDIR /root/chaos

COPY ./agent /root/chaos
ARG BLADE_VERSION=1.7.5

RUN curl -L https://chaosblade.oss-cn-hangzhou.aliyuncs.com/agent/github/1.7.5/chaosblade-1.7.5-linux_amd64.tar.gz | tar xz \
    && mv chaosblade-1.7.5-linux_amd64 /opt/chaosblade

ENTRYPOINT ["/root/chaos/agent"]