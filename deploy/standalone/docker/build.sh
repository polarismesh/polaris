#!/bin/bash

if [[ $# != 1 ]]; then
    echo "invalid args, eg. bash $0 version"
    exit 1
fi

version=$1

cp ../../../polaris-server.yaml ./

docker build --network=host --build-arg VERSION="${version}" -t polarismesh/polaris-server-standalone:${version} ./

docker push polarismesh/polaris-server-standalone:${version}
docker tag polarismesh/polaris-server-standalone:${version} polarismesh/polaris-server-standalone:latest
docker push polarismesh/polaris-server-standalone:latest
