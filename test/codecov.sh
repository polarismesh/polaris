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

set -ex         # Exit on error; debugging enabled.
set -o pipefail # Fail a pipe if any sub-command fails.

function run_codecov() {
    test_standalone &
    prepare_cluster_env
    test_cluster_auth &
    test_cluster_discovery &
    test_cluster_config &

    for pid in $(jobs -p); do
        wait $pid
        status=$?
        if [ $status != 0 ]; then
            echo "$pid status is $status have some error!"
            exit 1
        fi
    done
}

function test_standalone() {
    export STORE_MODE=""
    # bash coverage.sh
    go mod vendor
    go test -timeout 120m ./... -v -covermode=count -coverprofile=coverage_1.cover -coverpkg=github.com/polarismesh/polaris/apiserver,github.com/polarismesh/polaris/apiserver/eurekaserver,github.com/polarismesh/polaris/auth/defaultauth,github.com/polarismesh/polaris/service,github.com/polarismesh/polaris/service/batch,github.com/polarismesh/polaris/service/healthcheck,github.com/polarismesh/polaris/cache,github.com/polarismesh/polaris/store/boltdb,github.com/polarismesh/polaris/store/mysql,github.com/polarismesh/polaris/plugin,github.com/polarismesh/polaris/config,github.com/polarismesh/polaris/plugin/healthchecker/leader,github.com/polarismesh/polaris/plugin/healthchecker/memory,github.com/polarismesh/polaris/plugin/healthchecker/redis,github.com/polarismesh/polaris/common/batchjob,github.com/polarismesh/polaris/common/eventhub,github.com/polarismesh/polaris/common/redispool,github.com/polarismesh/polaris/common/timewheel >test_standalone.log
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
    pushd ./auth/defaultauth
    go mod vendor && go test -v -timeout 120m -v -covermode=count -coverprofile=coverage_sqldb_1.cover -coverpkg=github.com/polarismesh/polaris/apiserver,github.com/polarismesh/polaris/apiserver/eurekaserver,github.com/polarismesh/polaris/auth/defaultauth,github.com/polarismesh/polaris/service,github.com/polarismesh/polaris/service/batch,github.com/polarismesh/polaris/service/healthcheck,github.com/polarismesh/polaris/cache,github.com/polarismesh/polaris/store/boltdb,github.com/polarismesh/polaris/store/mysql,github.com/polarismesh/polaris/plugin,github.com/polarismesh/polaris/config,github.com/polarismesh/polaris/plugin/healthchecker/leader,github.com/polarismesh/polaris/plugin/healthchecker/memory,github.com/polarismesh/polaris/plugin/healthchecker/redis,github.com/polarismesh/polaris/common/batchjob,github.com/polarismesh/polaris/common/eventhub,github.com/polarismesh/polaris/common/redispool,github.com/polarismesh/polaris/common/timewheel >test_cluster_auth.log
    mv coverage_sqldb_1.cover ../../
    popd
}

function test_cluster_config() {
    # 测试配置中心
    export STORE_MODE=sqldb
    echo "cur STORE MODE=${STORE_MODE}, MYSQL_DB_USER=${MYSQL_DB_USER}, MYSQL_DB_PWD=${MYSQL_DB_PWD}"
    pushd ./config
    go mod vendor && go test -v -timeout 120m -v -covermode=count -coverprofile=coverage_sqldb_2.cover -coverpkg=github.com/polarismesh/polaris/apiserver,github.com/polarismesh/polaris/apiserver/eurekaserver,github.com/polarismesh/polaris/auth/defaultauth,github.com/polarismesh/polaris/service,github.com/polarismesh/polaris/service/batch,github.com/polarismesh/polaris/service/healthcheck,github.com/polarismesh/polaris/cache,github.com/polarismesh/polaris/store/boltdb,github.com/polarismesh/polaris/store/mysql,github.com/polarismesh/polaris/plugin,github.com/polarismesh/polaris/config,github.com/polarismesh/polaris/plugin/healthchecker/leader,github.com/polarismesh/polaris/plugin/healthchecker/memory,github.com/polarismesh/polaris/plugin/healthchecker/redis,github.com/polarismesh/polaris/common/batchjob,github.com/polarismesh/polaris/common/eventhub,github.com/polarismesh/polaris/common/redispool,github.com/polarismesh/polaris/common/timewheel >test_cluster_discovery.log
    mv coverage_sqldb_2.cover ../
    popd
}

function test_cluster_discovery() {
    # 测试服务、治理
    export STORE_MODE=sqldb
    echo "cur STORE MODE=${STORE_MODE}, MYSQL_DB_USER=${MYSQL_DB_USER}, MYSQL_DB_PWD=${MYSQL_DB_PWD}"
    pushd ./service
    go mod vendor && go test -v -timeout 120m -v -covermode=count -coverprofile=coverage_sqldb_3.cover -coverpkg=github.com/polarismesh/polaris/apiserver,github.com/polarismesh/polaris/apiserver/eurekaserver,github.com/polarismesh/polaris/auth/defaultauth,github.com/polarismesh/polaris/service,github.com/polarismesh/polaris/service/batch,github.com/polarismesh/polaris/service/healthcheck,github.com/polarismesh/polaris/cache,github.com/polarismesh/polaris/store/boltdb,github.com/polarismesh/polaris/store/mysql,github.com/polarismesh/polaris/plugin,github.com/polarismesh/polaris/config,github.com/polarismesh/polaris/plugin/healthchecker/leader,github.com/polarismesh/polaris/plugin/healthchecker/memory,github.com/polarismesh/polaris/plugin/healthchecker/redis,github.com/polarismesh/polaris/common/batchjob,github.com/polarismesh/polaris/common/eventhub,github.com/polarismesh/polaris/common/redispool,github.com/polarismesh/polaris/common/timewheel >test_cluster_config.log
    mv coverage_sqldb_3.cover ../
    popd
}

run_codecov

echo "===================== TEST STANDALONE LOG "====================="
cat test_standalone.log

echo "===================== TEST CLUSTER AUTH LOG "====================="
cat test_cluster_auth.log

echo "===================== TEST CLUSTER DISCOVERY LOG "====================="
cat test_cluster_discovery.log

echo "===================== TEST CLUSTER CONFIG LOG "====================="
cat test_cluster_config.log
