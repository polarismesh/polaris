#!/bin/bash

set -e

workdir=$(dirname $(realpath $0))
version=$(cat version 2>/dev/null)
folder_name="polaris-server-release_${version}"
if [ -n "$GOOS" ] && [ -n "$GOARCH" ]
then
	folder_name="polaris-server-release_${version}.${GOOS}.${GOARCH}"
fi
pkg_name="${folder_name}.tar.gz"

cd $workdir

# 清理环境
rm -rf ${folder_name}
rm -f "${pkg_name}"

# 编译
rm -f polaris-server

build_date=$(date "+%Y%m%d.%H%M%S")
package="github.com/polarismesh/polaris-server/common/version"
go build -o polaris-server -ldflags="-X ${package}.Version=${version} -X ${package}.BuildDate=${build_date}"

# 打包
mkdir -p ${folder_name}
mv polaris-server ${folder_name}
cp polaris-server.yaml ${folder_name}
cp -r tool ${folder_name}/
tar -czvf "${pkg_name}" ${folder_name}
md5sum ${pkg_name} > "${pkg_name}.md5sum"