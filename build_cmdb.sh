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

#!/bin/bash

workdir=$(dirname $(realpath $0))
version=$(cat version 2>/dev/null)
folder_name="polaris-cmdb-syncer-release_${version}"
pkg_name="${folder_name}.tar.gz"

cd $workdir/plugin/cmdb/tencent/cmdbSyncer

# 清理环境
rm -rf ${folder_name}
rm -f "${pkg_name}"

# 编译
rm -f polaris-cmdb-syncer

build_date=$(date "+%Y%m%d.%H%M%S")
package="github.com/polarismesh/polaris-server/common/version"


go build -mod=vendor -o polaris-cmdb-syncer -ldflags="-X ${package}.Version=${version} -X ${package}.BuildDate=${build_date}"

# 打包
mkdir -p ${folder_name}
mv polaris-cmdb-syncer ${folder_name}
cp config.yaml ${folder_name}
cp -r ../cmdb-tools/ ${folder_name}/cmdb-tools
cp -r ../tool/ ${folder_name}/tool
tar -czvf "${pkg_name}" ${folder_name}
mv "${pkg_name}" $workdir
cd $workdir
md5sum ${pkg_name} > "${pkg_name}.md5sum"