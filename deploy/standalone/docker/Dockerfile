ARG VERSION=v1.0.0

FROM polarismesh/polaris-server:${VERSION}

COPY ./polaris-server.yaml /root/

EXPOSE 8091 8090 8761 8093 8080 9000 8761 15010

CMD ["/root/polaris-server", "start"]
