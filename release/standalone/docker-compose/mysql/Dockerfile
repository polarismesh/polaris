FROM mysql:8.0.23

LABEL maintainer="polaris"

# Set Timezone
ARG TZ=Asia/Shanghai
ENV TZ ${TZ}
RUN pwd \
	&& ln -snf /usr/share/zoneinfo/$TZ /etc/localtime \
	&& echo $TZ > /etc/timezone \
	&& chown -R mysql:root /var/lib/mysql/ \
	&& true

EXPOSE 3306
