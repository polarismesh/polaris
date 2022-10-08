#!/bin/bash

if [ $# != 1 ]; then
    echo "e.g.: bash $0 v1.0"
    exit 1
fi

docker_tag=$1

docker_repository="polarismesh"

echo "docker repository : ${docker_repository}/polaris-server, tag : ${docker_tag}"

#arch_list=( "amd64" "arm64" )
arch_list=( "amd64" )
platforms=""

for arch in ${arch_list[@]}; do
    make build VERSION=${docker_tag} ARCH=${arch}

    if [ $? != 0 ]; then
        echo "build polaris-server failed"
        exit 1
    fi

    mv polaris-server polaris-server-${arch}
    platforms+="linux/${arch},"
done

platforms=${platforms::-1}
extra_tags=""

pre_release=`echo ${docker_tag}|egrep "(alpha|beta|rc|[T|t]est)"|wc -l`
if [ ${pre_release} == 0 ]; then
  extra_tags="-t ${docker_repository}/polaris-server:latest"
fi

docker buildx build --network=host -t ${docker_repository}/polaris-server:${docker_tag} ${extra_tags} --platform ${platforms} --push ./
