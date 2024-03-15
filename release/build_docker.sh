#!/bin/bash
# Tencent is pleased to support the open source community by making Polaris available.
#
# Copyright (C) 2019 THL A29 Limited, a Tencent company. All rights reserved.
#
# Licensed under the BSD 3-Clause License (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# https://opensource.org/licenses/BSD-3-Clause
#
# Unless required by applicable law or agreed to in writing, software distributed
# under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR
# CONDITIONS OF ANY KIND, either express or implied. See the License for the
# specific language governing permissions and limitations under the License.


if [ $# != 1 ]; then
    echo "e.g.: bash $0 v1.0"
    exit 1
fi

docker_tag=$1

docker_repository="${DOCKER_REPOSITORY}"
if [[ "${docker_repository}" == "" ]]; then
    docker_repository="polarismesh"
fi

echo "docker repository : ${docker_repository}/polaris-server, tag : ${docker_tag}"

arch_list=( "amd64" "arm64" )
# arch_list=( "amd64" )
platforms=""

for arch in ${arch_list[@]}; do
    export GOARCH=${arch}
    export GOOS="linux"
    make build VERSION=${docker_tag} ARCH=${arch}

    if [ $? != 0 ]; then
        echo "build polaris-server failed"
        exit 1
    fi

    mv polaris-server polaris-server-${arch}
    platforms+="linux/${arch},"
done

echo "${platforms}"
platforms=${platforms%?}
extra_tags=""

pre_release=`echo ${docker_tag}|egrep "(alpha|beta|rc|[T|t]est)"|wc -l`
if [ ${pre_release} == 0 ]; then
  extra_tags="-t ${docker_repository}/polaris-server:latest"
fi

docker buildx build -f ./release/Dockerfile -t ${docker_repository}/polaris-server:${docker_tag} ${extra_tags} --platform ${platforms} --push ./
