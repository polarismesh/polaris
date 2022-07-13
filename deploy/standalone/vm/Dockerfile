FROM centos:centos7.9.2009

ARG SERVER_VERSION=v1.10.0
ARG CONSOLE_VERSION=v1.7.1
ARG GOOS=linux
ARG GOARCH=amd64

LABEL cn.polarismesh.image.authors="polaris"
LABEL cn.polarismesh.image.documentation="https://polarismesh.cn/#/"

RUN yum install -y lsof curl unzip vixie-cron crontabs

COPY polaris-server-release_${SERVER_VERSION}.${GOOS}.${GOARCH}.zip /root/polaris-server-release_${SERVER_VERSION}.${GOOS}.${GOARCH}.zip
COPY polaris-console-release_${CONSOLE_VERSION}.${GOOS}.${GOARCH}.zip /root/polaris-console-release_${CONSOLE_VERSION}.${GOOS}.${GOARCH}.zip
COPY prometheus-2.28.0.${GOOS}-${GOARCH}.tar.gz /root/prometheus-2.28.0.${GOOS}-${GOARCH}.tar.gz
COPY install.sh /root/install.sh
COPY port.properties /root/port.properties
COPY run.sh /root/run.sh

WORKDIR /root

EXPOSE 8091 8090 8761 8093 8080 9000 8761 15010 9090

CMD ["/bin/bash", "run.sh"]
