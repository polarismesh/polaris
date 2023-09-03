FROM centos:centos7.9.2009

ARG SERVER_VERSION=v1.12.0
ARG CONSOLE_VERSION=v1.9.0
ARG LIMITER_VERSION=v1.0.2
ARG GOOS=linux
ARG TARGETARCH=amd64

LABEL cn.polarismesh.image.authors="polaris"
LABEL cn.polarismesh.image.documentation="https://polarismesh.cn/#/"

RUN yum install -y lsof curl unzip vixie-cron crontabs net-tools

COPY polaris-server-release_${SERVER_VERSION}.${GOOS}.${TARGETARCH}.zip /root/polaris-server-release_${SERVER_VERSION}.${GOOS}.${TARGETARCH}.zip
COPY polaris-console-release_${CONSOLE_VERSION}.${GOOS}.${TARGETARCH}.zip /root/polaris-console-release_${CONSOLE_VERSION}.${GOOS}.${TARGETARCH}.zip
COPY polaris-limiter-release_${LIMITER_VERSION}.${GOOS}.${TARGETARCH}.zip /root/polaris-limiter-release_${LIMITER_VERSION}.${GOOS}.${TARGETARCH}.zip
COPY prometheus-2.28.0.${GOOS}-${TARGETARCH}.tar.gz /root/prometheus-2.28.0.${GOOS}-${TARGETARCH}.tar.gz
COPY pushgateway-1.6.0.${GOOS}-${TARGETARCH}.tar.gz /root/pushgateway-1.6.0.${GOOS}-${TARGETARCH}.tar.gz
COPY install.sh /root/install.sh
COPY prometheus-help.sh /root/prometheus-help.sh
COPY port.properties /root/port.properties
COPY run.sh /root/run.sh

WORKDIR /root

EXPOSE 8091 8090 8761 8093 8080 8100 8101 15010 9090 9091

CMD ["/bin/bash", "run.sh"]
