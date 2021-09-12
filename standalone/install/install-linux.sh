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
    exit 1
  fi
  
  local target_polaris_server_pkg=$(find . -name "polaris-server-release*.zip")
  local polaris_server_dirname=$(basename ${target_polaris_server_pkg} .zip)
  if [ -e ${polaris_server_dirname} ]
  then
    echo -e "${polaris_server_dirname} has exists, now remove it"
    rm -rf ${polaris_server_dirname}
  fi
  unzip ${target_polaris_server_pkg}

  pushd ${polaris_server_dirname}
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
    exit 1
  fi

  local target_polaris_console_pkg=$(find . -name "polaris-console-release*.zip")
  local polaris_console_dirname=$(basename ${target_polaris_console_pkg} .zip)
  if [ -e ${polaris_console_dirname} ]
  then
    echo -e "${polaris_console_dirname} has exists, now remove it"
    rm -rf ${polaris_console_dirname}
  fi
  unzip ${target_polaris_console_pkg}

  pushd ${polaris_console_dirname}
  /bin/bash ./tool/start.sh
  echo -e "install polaris console success"
  popd
}

function installPrometheus() {
  echo -e "install prometheus ... "
  local prometheus_num=$(ps -ef | grep prometheus | grep -v grep | wc -l)
  if [ ${prometheus_num} -ge 1 ]
  then
    echo -e "prometheus is running, exit"
    return -1
  fi

  local prometheus_pkg_num=$(find . -name "prometheus-*.tar.gz" | wc -l)
  if [ ${prometheus_pkg_num} != 1 ]; then
    echo -e "number of prometheus package not equals to 1, exit"
    exit 1
  fi

  local target_prometheus_pkg=$(find . -name "prometheus-*.tar.gz")
  local prometheus_dirname=$(basename ${target_prometheus_pkg} .tar.gz)
  if [ -e ${prometheus_dirname} ]
  then
    echo -e "${prometheus_dirname} has exists, now remove it"
    rm -rf ${prometheus_dirname}
  fi
  tar -xf ${target_prometheus_pkg}

  pushd ${prometheus_dirname}
  echo "" >> prometheus.yml
  echo "  - job_name: 'push-metrics'" >> prometheus.yml
  echo "    static_configs:" >> prometheus.yml
  echo "    - targets: ['localhost:9091']" >> prometheus.yml
  echo "    honor_labels: true" >> prometheus.yml
  nohup ./prometheus --web.enable-lifecycle --web.enable-admin-api >> prometheus.out 2>&1 &
  echo "install prometheus success"
  popd
}

function installPushGateway() {
  echo -e "install pushgateway ... "
  local pgw_num=$(ps -ef | grep pushgateway | grep -v grep | wc -l)
  if [ $pgw_num -ge 1 ]; then
    echo -e "pushgateway is running, exit"
    return -1
  fi

  local pgw_pkg_num=$(find . -name "pushgateway-*.tar.gz" | wc -l)
  if [ $pgw_pkg_num != 1 ]; then
    echo -e "number of pushgateway package not equals to 1, exit"
    exit 1
  fi

  local target_pgw_pkg=$(find . -name "pushgateway-*.tar.gz")
  local pgw_dirname=$(basename ${target_pgw_pkg} .tar.gz)
 if [ -e ${pgw_dirname} ]
  then
    echo -e "${pgw_dirname} has exists, now remove it"
    rm -rf ${pgw_dirname}
  fi
  tar -xf ${target_pgw_pkg}

  pushd ${pgw_dirname}
  nohup ./pushgateway --web.enable-lifecycle --web.enable-admin-api >> pgw.out 2>&1 &
  echo "install pushgateway success"
  popd
}

function checkPort() {
   ports=("8080" "8090" "8091" "7779" "9090" "9091")
   for port in ${ports[@]}
   do
    pid=`/usr/sbin/lsof -i :${port} | awk '{print $1 " " $2}'`
    if [ "${pid}" != "" ];
    then
      echo "port ${port} has been used, exit"
      exit 1
    fi
   done
}

# 检查端口占用
checkPort
# 安装server
installPolarisServer
# 安装console
installPolarisConsole
# 安装Prometheus
installPrometheus
# 安装PushGateWay
installPushGateway