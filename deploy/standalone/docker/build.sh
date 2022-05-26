#!/bin/bash

if [[ $# != 1 ]]; then
    echo "invalid args, eg. bash $0 version"
    exit 1
fi

version=$1

cp ../../../polaris-server.yaml ./

docker build --build-arg= VERSION="${version}" --network=host -t polarismesh/polaris-server-standalone:${docker_tag} ./

docker push polarismesh/polaris-server-standalone:${docker_tag}
docker tag polarismesh/polaris-server-standalone:${docker_tag} polarismesh/polaris-server-standalone:latest
docker push polarismesh/polaris-server-standalone:latest
