#!/bin/bash

if [ $# != 1 ]; then
    echo "e.g.: bash $0 v1.0"
    exit 1
fi

docker_tag=$1

docker_repository="${DOCKER_REPOSITORY}"
if [[ "${docker_repository}" == "" ]]; then
    docker_repository="polarismesh"
fi

echo "docker repository : ${docker_repository}/polaris-prometheus, tag : ${docker_tag}"

docker buildx build --network=host -t ${docker_repository}/polaris-prometheus:${docker_tag} -t ${docker_repository}/polaris-prometheus:latest --platform linux/amd64,linux/arm64 --push ./
