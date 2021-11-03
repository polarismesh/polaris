#!/bin/bash

if [ $# != 2 ]; then
    echo "e.g.: bash $0 polaris_mesh/polaris-server v1.0"
    exit 1
fi

docker_repository=$1
docker_tag=$2

echo "docker repository : ${docker_repository}, tag : ${docker_tag}"

set -e

workdir=$(dirname $(realpath $0))
version=$(cat version 2>/dev/null)
bin_name="polaris-server"
if [ "${GOOS}" == "" ]; then
    GOOS=$(go env GOOS)
fi
if [ "${GOARCH}" == "" ]; then
    GOARCH=$(go env GOARCH)
fi

if [ "${GOOS}" == "windows" ]; then
    echo "need to run on linux os"
    exit 1
fi
echo "GOOS is ${GOOS}, binary name is ${bin_name}"

cd $workdir

# 编译
rm -f ${bin_name}

# 禁止 CGO_ENABLED 参数打开
export CGO_ENABLED=0

build_date=$(date "+%Y%m%d.%H%M%S")
package="github.com/polarismesh/polaris-server/common/version"
go build -o ${bin_name} -ldflags="-X ${package}.Version=${version} -X ${package}.BuildDate=${build_date}"

docker build ${docker_repository}:${docker_tag} ./
