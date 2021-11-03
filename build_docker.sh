#!/bin/bash

if [ $# != 2 ]; then
    echo "e.g.: bash $0 polaris_mesh/polaris-server v1.0"
    exit 1
fi

docker_repository=$1
docker_tag=$2

echo "docker repository : ${docker_repository}, tag : ${docker_tag}"

bash build.sh

if [ $? != 0 ]; then
    echo "build polaris-server failed"
fi

docker build ${docker_repository}:${docker_tag} ./
