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

function uninstallPolarisServer() {
  echo -e "uninstall polaris server ... "
  local polaris_server_dirname=$(find . -name "polaris-server-release*" -type d | awk 'NR==1{print}')
  if [ ! -e ${polaris_server_dirname} ]; then
    echo -e "${polaris_server_dirname} not exists, skip"
    return
  fi
  pushd ${polaris_server_dirname}
  echo -e "start to execute polaris-server uninstall script"
  /bin/bash ./tool/stop.sh
  popd
  echo -e "start to remove ${polaris_server_dirname}"
  rm -rf ${polaris_server_dirname}
  echo -e "uninstall polaris server success"
}

function uninstallPolarisConsole() {
  echo -e "uninstall polaris console ... "
  local polaris_console_dirname=$(find . -name "polaris-console-release*" -type d | awk 'NR==1{print}')
  if [ ! -e ${polaris_console_dirname} ]; then
    echo -e "${polaris_console_dirname} not exists, skip"
    return
  fi
  pushd ${polaris_console_dirname}
  echo -e "start to execute polaris-console uninstall script"
  /bin/bash ./tool/stop.sh
  popd
  echo -e "start to remove ${polaris_console_dirname}"
  rm -rf ${polaris_console_dirname}
  echo -e "uninstall polaris console success"
}

function uninstallPolarisLimiter() {
  echo -e "uninstall polaris limiter ... "
  local polaris_limiter_dirname=$(find . -name "polaris-limiter-release*" -type d | awk 'NR==1{print}')
  if [ ! -e ${polaris_limiter_dirname} ]; then
    echo -e "${polaris_limiter_dirname} not exists, skip"
    return
  fi
  pushd ${polaris_limiter_dirname}
  echo -e "start to execute polaris-limiter uninstall script"
  /bin/bash ./tool/stop.sh
  popd
  echo -e "start to remove ${polaris_limiter_dirname}"
  rm -rf ${polaris_limiter_dirname}
  echo -e "uninstall polaris limiter success"
}

function uninstallPrometheus() {
  echo -e "uninstall polaris-prometheus ... "
  local pid=$(ps -ef | grep polaris-prometheus | grep -v grep | awk '{print $2}')
  if [ "${pid}" != "" ]; then
    echo -e "start to kill polaris-prometheus process ${pid}"
    kill ${pid}
  fi
  local prometheus_dirname=$(find . -name "prometheus*" -type d | awk 'NR==1{print}')
  if [ -e ${prometheus_dirname} ]; then
    echo -e "start to remove ${prometheus_dirname}"
    rm -rf ${prometheus_dirname}
  fi
  echo -e "uninstall polaris-prometheus success"
}

function uninstallPushGateway() {
  echo -e "uninstall polaris-pushgateway ... "
  local pid=$(ps -ef | grep polaris-pushgateway | grep -v grep | awk '{print $2}')
  if [ "${pid}" != "" ]; then
    echo -e "start to kill polaris-pushgateway process ${pid}"
    kill ${pid}
  fi
  local pushgateway_dirname=$(find . -name "pushgateway*" -type d | awk 'NR==1{print}')
  if [ -e ${pushgateway_dirname} ]; then
    echo -e "start to remove ${pushgateway_dirname}"
    rm -rf ${pushgateway_dirname}
  fi
  echo -e "uninstall polaris-pushgateway success"
}

# 卸载 server
uninstallPolarisServer
# 卸载 console
uninstallPolarisConsole
# 卸载 polaris-limiter
uninstallPolarisLimiter
# 卸载 prometheus
uninstallPrometheus
# 安装PushGateWay
uninstallPushGateway
