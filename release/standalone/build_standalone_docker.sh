#!/bin/bash

set -e
workdir=${WORKDIR}

cd release/standalone

POLARIS_GIT_PATH=https://github.com/polarismesh

DIR_NAME=polaris-standalone-release_${SERVER_VERSION}.${GOOS}
rm -rf ${DIR_NAME}
mkdir ${DIR_NAME}
cd ${DIR_NAME}

arch_list=("amd64" "arm64")
platforms=""

for GOARCH in ${arch_list[@]}; do
    SERVER_PKG_NAME=polaris-server-release_${SERVER_VERSION}.${GOOS}.${GOARCH}.zip
    wget -T10 -t3 ${POLARIS_GIT_PATH}/polaris/releases/download/${SERVER_VERSION}/${SERVER_PKG_NAME} --no-check-certificate
    
    CONSOLE_PKG_NAME=polaris-console-release_${CONSOLE_VERSION}.${GOOS}.${GOARCH}.zip
    wget -T10 -t3 ${POLARIS_GIT_PATH}/polaris-console/releases/download/${CONSOLE_VERSION}/${CONSOLE_PKG_NAME} --no-check-certificate
    
    LIMITER_PKG_NAME=polaris-limiter-release_${LIMITER_VERSION}.${GOOS}.${GOARCH}.zip
    wget -T10 -t3 ${POLARIS_GIT_PATH}/polaris-limiter/releases/download/${LIMITER_VERSION}/${LIMITER_PKG_NAME} --no-check-certificate

    wget -T10 -t3 https://github.com/prometheus/prometheus/releases/download/v2.28.0/prometheus-2.28.0.${GOOS}-${GOARCH}.tar.gz --no-check-certificate
    wget -T10 -t3 https://github.com/prometheus/pushgateway/releases/download/v1.6.0/pushgateway-1.6.0.${GOOS}-${GOARCH}.tar.gz --no-check-certificate

    platforms+="${GOOS}/${GOARCH},"
done

platforms=${platforms::-1}

cp ../linux/install.sh ./install.sh
cp ../linux/uninstall.sh ./uninstall.sh
cp ../prometheus-help.sh ./prometheus-help.sh
cp ../port.properties ./port.properties
cp ../docker/Dockerfile ./Dockerfile

echo "#!/bin/bash" >"run.sh"
echo "" >>"run.sh"
echo "bash install.sh" >>"run.sh"
echo "while ((1))" >>"run.sh"
echo "do" >>"run.sh"
echo "   sleep 1" >>"run.sh"
echo "done" >>"run.sh"

docker_repository="polarismesh"
docker_image="polaris-standalone"
docker_tag=${SERVER_VERSION}

docker buildx build --network=host --build-arg SERVER_VERSION="${SERVER_VERSION}" --build-arg CONSOLE_VERSION="${CONSOLE_VERSION}" --build-arg LIMITER_VERSION="${LIMITER_VERSION}" -t ${docker_repository}/${docker_image}:${docker_tag} -t ${docker_repository}/${docker_image}:latest --platform ${platforms} --push ./
