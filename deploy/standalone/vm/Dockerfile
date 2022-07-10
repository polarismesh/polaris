FROM ubuntu:latest

LABEL cn.polarismesh.image.authors="polaris"
LABEL cn.polarismesh.image.documentation="https://polarismesh.cn/#/"

COPY polaris-server.zip /root/polaris-server.zip
COPY polaris-console.zip /root/polaris-server.zip
COPY prometheus.tar.gz /root/prometheus.tar.gz
COPY install.sh /root/install.sh
COPY port.properties /root/port.properties

WORKDIR /root

EXPOSE 8091 8090 8761 8093 8080 9000 8761 15010 9090

CMD ["/bin/bash", "install.sh"]

