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

set -ex # Exit on error; debugging enabled.

cur_dir=$(pwd)

# apiserver 模块的包信息
apiserver_pkg=(
    "github.com/polarismesh/polaris/apiserver"
    "github.com/polarismesh/polaris/apiserver/eurekaserver"
    "github.com/polarismesh/polaris/apiserver/xdsserverv3"
    "github.com/polarismesh/polaris/apiserver/nacosserver"
    "github.com/polarismesh/polaris/apiserver/nacosserver/core"
    "github.com/polarismesh/polaris/apiserver/nacosserver/v1"
    "github.com/polarismesh/polaris/apiserver/nacosserver/v1/discover"
    "github.com/polarismesh/polaris/apiserver/nacosserver/v1/config"
    "github.com/polarismesh/polaris/apiserver/nacosserver/v2"
    "github.com/polarismesh/polaris/apiserver/nacosserver/v2/discover"
    "github.com/polarismesh/polaris/apiserver/nacosserver/v2/config"
    "github.com/polarismesh/polaris/apiserver/nacosserver/v2/remote"
)

# 鉴权模块的包信息
auth_pkg=(
    "github.com/polarismesh/polaris/auth/user"
    "github.com/polarismesh/polaris/auth/policy"
)

# cache 模块的包信息
cache_pkg=(
    "github.com/polarismesh/polaris/cache"
    "github.com/polarismesh/polaris/cache/service"
    "github.com/polarismesh/polaris/cache/config"
    "github.com/polarismesh/polaris/cache/gray"
    "github.com/polarismesh/polaris/cache/auth"
    "github.com/polarismesh/polaris/cache/client"
)

# 注册发现模块的包信息
discover_pkg=(
    "github.com/polarismesh/polaris/service"
    "github.com/polarismesh/polaris/service/batch"
    "github.com/polarismesh/polaris/service/healthcheck"
)

# 配置模块
config_pkg=(
    "github.com/polarismesh/polaris/config"
)

# 存储模块
store_pkg=(
    "github.com/polarismesh/polaris/store/boltdb"
    "github.com/polarismesh/polaris/store/mysql"
)

# 插件模块
plugin_pkg=(
    "github.com/polarismesh/polaris/plugin"
    "github.com/polarismesh/polaris/plugin/healthchecker/leader"
    "github.com/polarismesh/polaris/plugin/healthchecker/memory"
    "github.com/polarismesh/polaris/plugin/healthchecker/redis"
)

# 普通包模块
common_pkg=(
    "github.com/polarismesh/polaris/common/eventhub"
    "github.com/polarismesh/polaris/common/redispool"
    "github.com/polarismesh/polaris/common/timewheel"
)

coverpkg=""

for val in ${apiserver_pkg[@]}; do
    if [ "${coverpkg}" == "" ]; then
        coverpkg="${val}"
    else
        coverpkg="${coverpkg},${val}"
    fi
done
for val in ${auth_pkg[@]}; do
    if [ "${coverpkg}" == "" ]; then
        coverpkg="${val}"
    else
        coverpkg="${coverpkg},${val}"
    fi
done
for val in ${cache_pkg[@]}; do
    if [ "${coverpkg}" == "" ]; then
        coverpkg="${val}"
    else
        coverpkg="${coverpkg},${val}"
    fi
done
for val in ${discover_pkg[@]}; do
    if [ "${coverpkg}" == "" ]; then
        coverpkg="${val}"
    else
        coverpkg="${coverpkg},${val}"
    fi
done
for val in ${config_pkg[@]}; do
    if [ "${coverpkg}" == "" ]; then
        coverpkg="${val}"
    else
        coverpkg="${coverpkg},${val}"
    fi
done
for val in ${store_pkg[@]}; do
    if [ "${coverpkg}" == "" ]; then
        coverpkg="${val}"
    else
        coverpkg="${coverpkg},${val}"
    fi
done
for val in ${plugin_pkg[@]}; do
    if [ "${coverpkg}" == "" ]; then
        coverpkg="${val}"
    else
        coverpkg="${coverpkg},${val}"
    fi
done
for val in ${common_pkg[@]}; do
    if [ "${coverpkg}" == "" ]; then
        coverpkg="${val}"
    else
        coverpkg="${coverpkg},${val}"
    fi
done

echo "${coverpkg}"

function test_standalone() {
    cd ${cur_dir}
    export STORE_MODE=""
    go mod vendor
    go test -timeout 40m ./... -v -covermode=count -coverprofile=coverage_1.cover -coverpkg=${coverpkg}
}

function prepare_cluster_env() {
    # 测试配置
    echo "cur STORE MODE=${STORE_MODE}, MYSQL_DB_USER=${MYSQL_DB_USER}, MYSQL_DB_PWD=${MYSQL_DB_PWD}"
    # 设置严格模式
    mysql -h127.0.0.1 -P3306 -u${MYSQL_DB_USER} -p"${MYSQL_DB_PWD}" -e "set sql_mode='STRICT_TRANS_TABLES,NO_ENGINE_SUBSTITUTION'"
    # 清空数据
    mysql -h127.0.0.1 -P3306 -u${MYSQL_DB_USER} -p"${MYSQL_DB_PWD}" -e "DROP DATABASE IF EXISTS polaris_server"
    # 初始化 polaris 数据库
    mysql -h127.0.0.1 -P3306 -u${MYSQL_DB_USER} -p"${MYSQL_DB_PWD}" <store/mysql/scripts/polaris_server.sql
    # 临时放开 DB 的最大连接数
    mysql -h127.0.0.1 -P3306 -u${MYSQL_DB_USER} -p"${MYSQL_DB_PWD}" -e "set GLOBAL max_connections = 3000;"
}

function test_cluster_auth() {
    # 测试鉴权
    export STORE_MODE=sqldb
    echo "cur STORE MODE=${STORE_MODE}, MYSQL_DB_USER=${MYSQL_DB_USER}, MYSQL_DB_PWD=${MYSQL_DB_PWD}"

    cd ${cur_dir}
    pushd ./auth/user
    go mod vendor && go test -v -timeout 40m -v -covermode=count -coverprofile=user_coverage.cover -coverpkg=${coverpkg}
    mv user_coverage.cover ../../

    cd ${cur_dir}
    pushd ./auth/policy
    go mod vendor && go test -v -timeout 40m -v -covermode=count -coverprofile=policy_coverage.cover -coverpkg=${coverpkg}
    mv policy_coverage.cover ../../
}

function test_cluster_config() {
    cd ${cur_dir}
    # 测试配置中心
    export STORE_MODE=sqldb
    echo "cur STORE MODE=${STORE_MODE}, MYSQL_DB_USER=${MYSQL_DB_USER}, MYSQL_DB_PWD=${MYSQL_DB_PWD}"
    pushd ./config
    go mod vendor && go test -v -timeout 40m -v -covermode=count -coverprofile=coverage_sqldb_2.cover -coverpkg=${coverpkg}
    mv coverage_sqldb_2.cover ../
}

function test_cluster_discovery() {
    cd ${cur_dir}
    # 测试服务、治理
    export STORE_MODE=sqldb
    echo "cur STORE MODE=${STORE_MODE}, MYSQL_DB_USER=${MYSQL_DB_USER}, MYSQL_DB_PWD=${MYSQL_DB_PWD}"
    pushd ./service
    go mod vendor && go test -v -timeout 40m -v -covermode=count -coverprofile=coverage_sqldb_3.cover -coverpkg=${coverpkg}
    mv coverage_sqldb_3.cover ../
}

if [[ "${RUN_MODE}" == "STANDALONE" ]]; then
    test_standalone
else
    prepare_cluster_env
    test_cluster_auth
    test_cluster_discovery
    test_cluster_config
fi

# for pid in $(jobs -p); do
#     wait $pid
#     status=$?
#     if [ $status != 0 ]; then
#         echo "$pid status is $status have some error!"
#         exit 1
#     fi
# done
