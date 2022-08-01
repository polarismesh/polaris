#!/bin/bash

set -e

if [[ $(uname) == 'Darwin' ]]; then
  realpath() {
    [[ $1 = /* ]] && echo "$1" || echo "$PWD/${1#./}"
  }

  md5sum() {
    md5 $*
  }
fi

workdir=$(dirname $(realpath $0))
version=$(cat version 2>/dev/null)
bin_name="polaris-server"

if [ $# == 1 ]; then
  version=$1
fi

if [ "${GOOS}" == "windows" ]; then
  bin_name="polaris-server.exe"
fi

if [ "${GOOS}" == "" ]; then
  GOOS=$(go env GOOS)
fi

if [ "${GOARCH}" == "" ]; then
  GOARCH=$(go env GOARCH)
fi

folder_name="polaris-server-release_${version}.${GOOS}.${GOARCH}"
pkg_name="${folder_name}.zip"
echo "GOOS is ${GOOS}, binary name is ${bin_name}"

cd $workdir

# 清理环境
rm -rf ${folder_name}
rm -f "${pkg_name}"
rm -f ${bin_name}

# 禁止 CGO_ENABLED 参数打开
export CGO_ENABLED=0

build_date=$(date "+%Y%m%d.%H%M%S")
package="github.com/polarismesh/polaris-server/common/version"
i18n_res="apiserver/httpserver/i18n"
go build -o ${bin_name} -ldflags="-X ${package}.Version=${version} -X ${package}.BuildDate=${build_date}"

# 打包
mkdir -p ${folder_name}
cp ${bin_name} ${folder_name}
cp polaris-server.yaml ${folder_name}
cp -r tool ${folder_name}/
mkdir -p ${folder_name}/${i18n_res}
cp -r ${i18n_res}/*.toml ${folder_name}/${i18n_res}
zip -r "${pkg_name}" ${folder_name}
md5sum ${pkg_name} >"${pkg_name}.md5sum"
