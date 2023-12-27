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

ACTUAL_ARCH="$(/usr/bin/uname -m)"
EXPECT_ARCH=$(cat arch.txt)

actual_is_arm=$(/usr/bin/uname -m | grep "arm|aarch64" | wc -l)
expect_is_arm=$(cat arch.txt | grep -E "arm" | wc -l)

if [ ${actual_is_arm} -ne ${expect_is_arm} ]; then
  echo "machine arch is ${ACTUAL_ARCH}, but Installation package arch is ${EXPECT_ARCH}"
  exit 1
fi

function getProperties() {
  result=""
  proFilePath="./port.properties"
  key="$1"
  if [ "WJA${key}" = "WJA" ]; then
    echo "invalid param, pls set key"
    echo "" >&2
    exit 1
  fi
  if [ ! -r ${proFilePath} ]; then
    echo "current use not file ${proFilePath} read and write permission"
    echo "" >&2
    exit 1
  fi
  keyLength=$(echo ${key} | awk '{print length($0)}')
  lineNumStr=$(cat ${proFilePath} | wc -l)
  lineNum=$((${lineNumStr}))
  for ((i = 1; i <= ${lineNum}; i++)); do
    oneLine=$(sed -n ${i}p ${proFilePath})
    if [ "${oneLine:0:((keyLength))}" = "${key}" ] && [ "${oneLine:$((keyLength)):1}" = "=" ]; then
      result=${oneLine#*=}
      break
    fi
  done
  echo ${result}
}

console_port=$(getProperties polaris_console_port)

eureka_port=$(getProperties polaris_eureka_port)
xdsv3_port=$(getProperties polaris_xdsv3_port)
service_grpc_port=$(getProperties polaris_service_grpc_port)
config_grpc_port=$(getProperties polaris_config_grpc_port)
api_http_port=$(getProperties polaris_open_api_port)
nacos_port=$(getProperties nacos_http_port)

prometheus_port=$(getProperties prometheus_port)
pushgateway_port=$(getProperties pushgateway_port)

limiter_http_port=$(getProperties "polaris_limiter_http_port")
limiter_grpc_port=$(getProperties "polaris_limiter_grpc_port")

echo "prepare install polaris standalone..."

echo "polaris-console listen port info"
echo "console_port=${console_port}"
echo ""
echo "polaris-server listen port info"
echo "eureka_port=${eureka_port}"
echo "xdsv3_port=${xdsv3_port}"
echo "service_grpc_port=${service_grpc_port}"
echo "config_grpc_port=${config_grpc_port}"
echo "api_http_port=${api_http_port}"
echo "nacos_port=${nacos_port}"
echo ""
echo "polaris-limiter-server listen port info"
echo "polaris_limiter_http_port=${limiter_http_port}"
echo "polaris_limiter_grpc_port=${limiter_grpc_port}"
echo ""
echo "prometheus-server listen port info"
echo "prometheus_server_port=${prometheus_port}"
echo ""
echo "pushgateway-server listen port info"
echo "pushgateway_server_port=${pushgateway_port}"

function installPolarisServer() {
  echo -e "install polaris server ... "
  local polaris_server_num=$(ps -ef | grep polaris-server | grep -v grep | wc -l)
  if [ ${polaris_server_num} -ge 1 ]; then
    echo -e "polaris-server is running, exit"
    return -1
  fi

  local polaris_server_pkg_num=$(find . -name "polaris-server-release*.zip" | wc -l)
  if [ ${polaris_server_pkg_num} != 1 ]; then
    echo -e "number of polaris server package not equals to 1, exit"
    exit -1
  fi

  local target_polaris_server_pkg=$(find . -name "polaris-server-release*.zip")
  local polaris_server_dirname=$(basename ${target_polaris_server_pkg} .zip)
  if [ ! -e ${polaris_server_dirname} ]; then
    unzip ${target_polaris_server_pkg} >/dev/null
  else
    echo -e "${target_polaris_server_pkg} has been decompressed, skip."
  fi

  pushd ${polaris_server_dirname}

  # 备份 polaris-server.yaml
  cp conf/polaris-server.yaml conf/polaris-server.yaml.bak

  # 修改 polaris-server eureka 端口信息
  sed -i "s/listenPort: 8761/listenPort: ${eureka_port}/g" conf/polaris-server.yaml
  # 修改 polaris-server xdsv3 端口信息
  sed -i "s/listenPort: 15010/listenPort: ${xdsv3_port}/g" conf/polaris-server.yaml
  # 修改 polaris-server service-grpc 端口信息
  sed -i "s/listenPort: 8091/listenPort: ${service_grpc_port}/g" conf/polaris-server.yaml
  # 修改 polaris-server config-grpc 端口信息
  sed -i "s/listenPort: 8093/listenPort: ${config_grpc_port}/g" conf/polaris-server.yaml
  # 修改 polaris-server http-api 端口信息
  sed -i "s/listenPort: 8090/listenPort: ${api_http_port}/g" conf/polaris-server.yaml
  # 修改 polaris-server nacos 端口信息
  sed -i "s/listenPort: 8848/listenPort: ${nacos_port}/g" conf/polaris-server.yaml

  /bin/bash ./tool/start.sh
  echo -e "install polaris server success"
  popd
}

function installPolarisConsole() {
  echo -e "install polaris console ... "
  local polaris_console_num=$(ps -ef | grep polaris-console | grep -v grep | wc -l)
  if [ ${polaris_console_num} -ge 1 ]; then
    echo -e "polaris-console is running, exit"
    return -1
  fi

  local polaris_console_pkg_num=$(find . -name "polaris-console-release*.zip" | wc -l)
  if [ ${polaris_console_pkg_num} != 1 ]; then
    echo -e "number of polaris console package not equals to 1, exit"
    exit -1
  fi

  local target_polaris_console_pkg=$(find . -name "polaris-console-release*.zip")
  local polaris_console_dirname=$(basename ${target_polaris_console_pkg} .zip)
  if [ ! -e ${polaris_console_dirname} ]; then
    unzip ${target_polaris_console_pkg} >/dev/null
  else
    echo -e "${target_polaris_console_pkg} has been decompressed, skip."
  fi

  pushd ${polaris_console_dirname}

  # 备份 polaris-console.yaml
  cp polaris-console.yaml polaris-console.yaml.bak

  # 修改 polaris-console 端口信息
  sed -i "s/listenPort: 8080/listenPort: ${console_port}/g" polaris-console.yaml
  # 修改监听的 polaris-server http 端口信息
  sed -i "s/address: \"127.0.0.1:8090\"/address: \"127.0.0.1:${api_http_port}\"/g" polaris-console.yaml
  # 修改监听的 prometheus 端口信息
  sed -i "s/address: \"127.0.0.1:9090\"/address: \"127.0.0.1:${prometheus_port}\"/g" polaris-console.yaml

  /bin/bash ./tool/start.sh
  echo -e "install polaris console success"
  popd
}

function installPrometheus() {
  echo -e "install prometheus ... "
  local prometheus_num=$(ps -ef | grep polaris-prometheus | grep -v grep | wc -l)
  if [ ${prometheus_num} -ge 1 ]; then
    echo -e "polaris-prometheus is running, skip install polaris-prometheus"
    return 0
  fi

  local prometheus_pkg_num=$(find . -name "prometheus-*.tar.gz" | wc -l)
  if [ ${prometheus_pkg_num} != 1 ]; then
    echo -e "number of prometheus package not equals to 1, exit"
    exit -1
  fi

  local target_prometheus_pkg=$(find . -name "prometheus-*.tar.gz")
  local prometheus_dirname=$(basename ${target_prometheus_pkg} .tar.gz)
  if [ ! -e ${prometheus_dirname} ]; then
    tar -xf ${target_prometheus_pkg} >/dev/null
  else
    echo -e "${target_prometheus_pkg} has been decompressed, skip."
  fi

  cp prometheus-help.sh ${prometheus_dirname}/
  pushd ${prometheus_dirname}
  local push_count=$(cat prometheus.yml | grep "push-metrics" | wc -l)
  if [ $push_count -eq 0 ]; then
    echo "    http_sd_configs:" >>prometheus.yml
    echo "    - url: http://localhost:8090/prometheus/v1/clients" >>prometheus.yml
    echo "" >>prometheus.yml
    echo "  - job_name: 'push-metrics'" >>prometheus.yml
    echo "    static_configs:" >>prometheus.yml
    echo "    - targets: ['localhost:9091']" >>prometheus.yml
    echo "    honor_labels: true" >>prometheus.yml
  fi
  if [ ! -e polaris-prometheus ]; then
    mv prometheus polaris-prometheus
  fi
  chmod +x polaris-prometheus
  # nohup ./polaris-prometheus --web.enable-lifecycle --web.enable-admin-api --web.listen-address=:${prometheus_port} >>prometheus.out 2>&1 &
  bash prometheus-help.sh start ${prometheus_port}
  echo "install polaris-prometheus success"
  popd
}

function installPushGateway() {
  echo -e "install polaris-pushgateway ... "
  local pgw_num=$(ps -ef | grep polaris-pushgateway | grep -v grep | wc -l)
  if [ $pgw_num -ge 1 ]; then
    echo -e "polaris-pushgateway is running, exit"
    return -1
  fi

  local pgw_pkg_num=$(find . -name "pushgateway-*.tar.gz" | wc -l)
  if [ $pgw_pkg_num != 1 ]; then
    echo -e "number of pushgateway package not equals to 1, exit"
    exit -1
  fi

  local target_pgw_pkg=$(find . -name "pushgateway-*.tar.gz")
  local pgw_dirname=$(basename ${target_pgw_pkg} .tar.gz)
  if [ ! -e ${pgw_dirname} ]; then
    tar -xf ${target_pgw_pkg} >/dev/null
  else
    echo -e "pushgateway has been decompressed, skip."
  fi

  pushd ${pgw_dirname}
  if [ ! -e "polaris-pushgateway" ]; then
    mv pushgateway polaris-pushgateway
  fi
  chmod +x polaris-pushgateway
  nohup ./polaris-pushgateway --web.enable-lifecycle --web.enable-admin-api --web.listen-address=:${pushgateway_port} >>pgw.out 2>&1 &
  echo "install polaris-pushgateway success"
  popd
}

# 安装北极星分布式限流服务端
function installPolarisLimiter() {
  echo -e "install polaris limiter ... "
  local polaris_limiter_num=$(ps -ef | grep polaris-limiter | grep -v grep | wc -l)
  if [ $polaris_limiter_num -ge 1 ]; then
    echo -e "polaris-limiter is running, skip."
    return
  fi

  local polaris_limiter_tarnum=$(find . -name "polaris-limiter-release*.zip" | wc -l)
  if [ $polaris_limiter_tarnum != 1 ]; then
    echo -e "number of polaris limiter tar not equal 1, exit."
    exit -1
  fi

  local polaris_limiter_tarname=$(find . -name "polaris-limiter-release*.zip")
  local polaris_limiter_dirname=$(basename ${polaris_limiter_tarname} .zip)
  if [ ! -e $polaris_limiter_dirname ]; then
    unzip $polaris_limiter_tarname >/dev/null
  else
    echo -e "polaris-limiter-release.tar.gz has been decompressed, skip."
  fi

  pushd ${polaris_limiter_dirname}

  # 备份 polaris-limiter.yaml
  cp polaris-limiter.yaml polaris-limiter.yaml.bak

  # 修改 polaris-server grpc 端口信息
  sed -i "s/polaris-server-address: 127.0.0.1:8091/polaris-server-address: 127.0.0.1:${service_grpc_port}/g" polaris-limiter.yaml
  # 修改监听的 polaris-limiter http 端口信息
  sed -i "s/port: 8100/port: ${limiter_http_port}/g" polaris-limiter.yaml
  # 修改监听的 polaris-limiter grpc 端口信息
  sed -i "s/port: 8101/port: ${limiter_grpc_port}/g" polaris-limiter.yaml

  /bin/bash ./tool/start.sh
  echo -e "install polaris limiter finish."
  popd
}

function checkPort() {
  proFilePath="./port.properties"
  if [ ! -f ${proFilePath} ]; then
    echo "file ${proFilePath} not exist"
    echo "" >&2
    exit 1
  fi
  lineNumStr=$(cat ${proFilePath} | wc -l)
  lineNum=$((${lineNumStr}))
  for ((i = 1; i <= ${lineNum}; i++)); do
    oneLine=$(sed -n ${i}p ${proFilePath})
    port=${oneLine#*=}
    pid=$(lsof -i :${port} | awk '{print $1 " " $2}')
    if [ "${pid}" != "" ]; then
      echo "port ${port} already used, you can modify port.properties to adjust port"
      exit -1
    else
      echo "port ${port} is checked ,and is not used"
    fi
  done
}

# 检查端口占用
checkPort
# 安装server
installPolarisServer
# 安装console
installPolarisConsole
# 安装 polaris-limiter
installPolarisLimiter
# 安装Prometheus
installPrometheus
# 安装PushGateway
installPushGateway
