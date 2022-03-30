#!/bin/bash

if [ $# != 1 ]; then
    echo "e.g.: bash $0 v1.0"
    exit 1
fi

docker_tag=$1

echo "docker repository : polarismesh/polaris-prometheus, tag : ${docker_tag}"

docker build --network=host -t polarismesh/polaris-prometheus:${docker_tag} ./

docker push polarismesh/polaris-prometheus:${docker_tag}
docker tag polarismesh/polaris-prometheus:${docker_tag} polarismesh/polaris-prometheus:latest
docker push polarismesh/polaris-prometheus:latest
