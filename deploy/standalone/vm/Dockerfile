FROM ubuntu:latest

LABEL cn.polarismesh.image.authors="polaris"
LABEL cn.polarismesh.image.documentation="https://polarismesh.cn/#/"

RUN apt-get -y update
RUN apt-get install -y lsof curl

COPY polaris-server-release.zip /root/polaris-server-release.zip
COPY polaris-console-release.zip /root/polaris-console-release.zip
COPY prometheus-release.tar.gz /root/prometheus-release.tar.gz
COPY install.sh /root/install.sh
COPY port.properties /root/port.properties

WORKDIR /root

EXPOSE 8091 8090 8761 8093 8080 9000 8761 15010 9090

CMD ["/bin/bash", "install.sh"]

