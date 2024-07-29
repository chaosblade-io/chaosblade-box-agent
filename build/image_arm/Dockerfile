FROM multiarch/alpine:arm64-v3.14
LABEL maintainer="Mingxia Cai"

#RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.ustc.edu.cn/g' /etc/apk/repositories
RUN sed -i 's/https/http/' /etc/apk/repositories

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
