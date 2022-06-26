#!/bin/bash

if [ $# != 1 ]; then
    echo "e.g.: bash $0 v1.0"
    exit 1
fi

docker_tag=$1

echo "docker repository : polarismesh/polaris-server, tag : ${docker_tag}"

bash build.sh ${docker_tag}

if [ $? != 0 ]; then
    echo "build polaris-server failed"
    exit 1
fi

docker build --network=host -t polarismesh/polaris-server:${docker_tag} ./

docker push polarismesh/polaris-server:${docker_tag}
docker tag polarismesh/polaris-server:${docker_tag} polarismesh/polaris-server:latest
docker push polarismesh/polaris-server:latest
