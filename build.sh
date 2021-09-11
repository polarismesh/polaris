#!/bin/bash

set -e

workdir=$(dirname $(realpath $0))
version=$(cat version 2>/dev/null)
bin_name="polaris-server"
GOOS=`go env GOOS`
GOARCH=`go env GOARCH`
folder_name="polaris-server-release_${version}.${GOOS}.${GOARCH}"
pkg_name="${folder_name}.zip"
if [ ${GOOS} -eq "windows" ];then
  bin_name="polaris-server.exe"
fi
echo "binary name is ${bin_name}"

cd $workdir

# 清理环境
rm -rf ${folder_name}
rm -f "${pkg_name}"

# 编译
rm -f ${bin_name}

build_date=$(date "+%Y%m%d.%H%M%S")
package="github.com/polarismesh/polaris-server/common/version"
go build -o ${bin_name} -ldflags="-X ${package}.Version=${version} -X ${package}.BuildDate=${build_date}"

# 打包
mkdir -p ${folder_name}
mv ${bin_name} ${folder_name}
cp polaris-server.yaml ${folder_name}
cp -r tool ${folder_name}/
zip -r "${pkg_name}" ${folder_name}
md5sum ${pkg_name} > "${pkg_name}.md5sum"