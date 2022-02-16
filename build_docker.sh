#!/bin/bash

if [ $# != 4 ]; then
    echo "e.g.: bash $0 polaris_mesh/polaris-server v1.0 docker_username docekr_user_password"
    exit 1
fi

docker_repository=$1
docker_tag=$2
docker_username=$3
docker_password=$4

echo "docker repository : ${docker_repository}, tag : ${docker_tag}"

bash build.sh

if [ $? != 0 ]; then
    echo "build polaris-server failed"
    exit 1
fi

docker build --network=host -t ${docker_repository}:${docker_tag} ./

docker login --username=${docker_username} --password=${docker_password}

if [[ $? != 0 ]]; then
    echo "docker login failed"
fi

docker push ${docker_repository}:${docker_tag}
docker tag ${docker_repository}:${docker_tag} ${docker_repository}:latest
docker push ${docker_repository}:latest