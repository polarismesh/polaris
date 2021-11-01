FROM alpine:3.13.6

RUN sed -i 's!http://dl-cdn.alpinelinux.org/!https://mirrors.tencent.com/!g' /etc/apk/repositories

RUN set -eux && \
    apk add tcpdump && \
    apk add tzdata && \
    apk add busybox-extras && \
    apk add curl && \
    apk add bash && \
    cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone && \
    date

COPY polaris-server /root/polaris-server

WORKDIR /root

CMD ["/root/polaris-server", "start"]