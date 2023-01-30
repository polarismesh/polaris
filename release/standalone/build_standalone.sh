#!/bin/bash

set -e
workdir=${WORKDIR}

if [ ${GOOS} == "kubernetes" ]; then
    # ---------------------- 出简单 kubernetes 安装包 ----------------------
    cd ${workdir}
    cd release/cluster

    sed -i "s/##POLARIS_SERVER_VERSION##/${SERVER_VERSION}/g" kubernetes/03-polaris-server.yaml
    sed -i "s/##POLARIS_CONSOLE_VERSION##/${CONSOLE_VERSION}/g" kubernetes/03-polaris-server.yaml
    sed -i "s/##POLARIS_PROMETHEUS_VERSION##/${SERVER_VERSION}/g" kubernetes/04-prometheus.yaml
    sed -i "s/##POLARIS_LIMITER_VERSION##/${LIMITER_VERSION}/g" kubernetes/06-polaris-limiter.yaml

    DIR_NAME=polaris-standalone-release_${SERVER_VERSION}.${GOOS}
    mkdir ${DIR_NAME}
    pushd ${DIR_NAME}
    cp -rf ../kubernetes/* ./
    popd

    PACKAGE_NAME=${DIR_NAME}.zip
    zip -r ${PACKAGE_NAME} ${DIR_NAME}
    rm -rf ${DIR_NAME}
    mv ${PACKAGE_NAME} ../../

    # ---------------------- 出 docker-compose 安装包 ----------------------
    cd ${workdir}
    cd release/standalone

    sed -i "s/##POLARIS_SERVER_VERSION##/${SERVER_VERSION}/g" docker-compose/docker-compose.yaml
    sed -i "s/##POLARIS_CONSOLE_VERSION##/${CONSOLE_VERSION}/g" docker-compose/docker-compose.yaml
    sed -i "s/##POLARIS_PROMETHEUS_VERSION##/${SERVER_VERSION}/g" docker-compose/docker-compose.yaml
    sed -i "s/##POLARIS_LIMITER_VERSION##/${LIMITER_VERSION}/g" docker-compose/docker-compose.yaml

    DOCKER_COMPOSE_DIR_NAME=polaris-standalone-release_${SERVER_VERSION}.docker-compose
    mkdir ${DOCKER_COMPOSE_DIR_NAME}
    pushd ${DOCKER_COMPOSE_DIR_NAME}
    cp -rf ../docker-compose/* ./
    popd

    DOCKER_COMPOSE_PACKAGE_NAME=${DOCKER_COMPOSE_DIR_NAME}.zip
    zip -r ${DOCKER_COMPOSE_PACKAGE_NAME} ${DOCKER_COMPOSE_DIR_NAME}
    rm -rf ${DOCKER_COMPOSE_DIR_NAME}
    mv ${DOCKER_COMPOSE_PACKAGE_NAME} ../../

    # ---------------------- 出 helm 安装包 ----------------------
    cd ${workdir}
    cd release/cluster

    sed -i "s/##POLARIS_SERVER_VERSION##/${SERVER_VERSION}/g" helm/values.yaml
    sed -i "s/##POLARIS_CONSOLE_VERSION##/${CONSOLE_VERSION}/g" helm/values.yaml
    sed -i "s/##POLARIS_PROMETHEUS_VERSION##/${SERVER_VERSION}/g" helm/values.yaml
    sed -i "s/##POLARIS_LIMITER_VERSION##/${LIMITER_VERSION}/g" helm/values.yaml

    HELM_DIR_NAME=polaris-helm-release_${SERVER_VERSION}.${GOOS}
    mkdir ${HELM_DIR_NAME}
    pushd ${HELM_DIR_NAME}
    cp -rf ../helm/* ./
    popd

    HELM_PACKAGE_NAME=${HELM_DIR_NAME}.zip
    zip -r ${HELM_PACKAGE_NAME} ${HELM_DIR_NAME}
    rm -rf ${HELM_DIR_NAME}
    mv ${HELM_PACKAGE_NAME} ../
else
    cd release/standalone
    
    POLARIS_GIT_PATH=https://github.com/polarismesh
    
    DIR_NAME=polaris-standalone-release_${SERVER_VERSION}.${GOOS}.${GOARCH}
    
    mkdir ${DIR_NAME}
    pushd ${DIR_NAME}
    
    SERVER_PKG_NAME=polaris-server-release_${SERVER_VERSION}.${GOOS}.${GOARCH}.zip
    wget -T10 -t3 ${POLARIS_GIT_PATH}/polaris/releases/download/${SERVER_VERSION}/${SERVER_PKG_NAME} --no-check-certificate
    
    CONSOLE_PKG_NAME=polaris-console-release_${CONSOLE_VERSION}.${GOOS}.${GOARCH}.zip
    wget -T10 -t3 ${POLARIS_GIT_PATH}/polaris-console/releases/download/${CONSOLE_VERSION}/${CONSOLE_PKG_NAME} --no-check-certificate
    
    LIMITER_PKG_NAME=polaris-limiter-release_${LIMITER_VERSION}.${GOOS}.${GOARCH}.zip
    wget -T10 -t3 ${POLARIS_GIT_PATH}/polaris-limiter/releases/download/${LIMITER_VERSION}/${LIMITER_PKG_NAME} --no-check-certificate
    
    if [ ${GOOS} == "windows" ]; then
        wget -T10 -t3 https://github.com/prometheus/prometheus/releases/download/v2.28.0/prometheus-2.28.0.${GOOS}-${GOARCH}.zip
        mv ../${GOOS}/install.bat ./install.bat
        mv ../${GOOS}/install-windows.ps1 ./install-windows.ps1
        mv ../${GOOS}/uninstall.bat ./uninstall.bat
        mv ../${GOOS}/uninstall-windows.ps1 ./uninstall-windows.ps1
        mv ../port.properties ./port.properties
    else
        wget -T10 -t3 https://github.com/prometheus/prometheus/releases/download/v2.28.0/prometheus-2.28.0.${GOOS}-${GOARCH}.tar.gz
        mv ../${GOOS}/install.sh ./install.sh
        mv ../${GOOS}/uninstall.sh ./uninstall.sh
        mv ../port.properties ./port.properties
        mv ../prometheus-help.sh ./prometheus-help.sh
    fi
    echo "${GOARCH}" > arch.txt
    popd
    PACKAGE_NAME=${DIR_NAME}.zip
    zip -r ${PACKAGE_NAME} ${DIR_NAME}
    rm -rf ${DIR_NAME}
    mv ${PACKAGE_NAME} ../../
    echo ::set-output name=name::${PACKAGE_NAME}
fi
