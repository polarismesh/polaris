FROM polarismesh/polaris-server:polaris-server-tag

LABEL MAINTAINER = huyuanxin
LABEL EMAIL = huyuanxin1999@outlook.com

COPY ../../../polaris-server.yaml /root/

EXPOSE 8091 8090 8761 8093 8080 9000 8761 15010

CMD ["/root/polaris-server", "start"]
