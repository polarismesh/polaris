#!/bin/bash

if [ $# != 3 ]; then
    echo "e.g.: bash $0 v1.0 docker_username docekr_user_password"
    exit 1
fi

docker_tag=$1
docker_username=$2
docker_password=$3

echo "docker repository : polarismesh/polaris-server, tag : ${docker_tag}"

bash build.sh

if [ $? != 0 ]; then
    echo "build polaris-server failed"
    exit 1
fi

docker build --network=host -t polarismesh/polaris-server:${docker_tag} ./

docker login --username=${docker_username} --password=${docker_password}

if [[ $? != 0 ]]; then
    echo "docker login failed"
fi

docker push polarismesh/polaris-server:${docker_tag}
docker tag polarismesh/polaris-server:${docker_tag} polarismesh/polaris-server:latest
docker push polarismesh/polaris-server:latest