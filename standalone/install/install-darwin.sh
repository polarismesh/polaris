#!/bin/bash

echo "To allow polaris to be installed on your Mac, we need to open the install from anywhere 'sudo spctl
--master-disable'"

sudo spctl --master-disable > /dev/null

if [ "${0:0:1}" == "/" ]; then
  install_path=$(dirname "$0")
else
  install_path=$(pwd)/$(dirname "$0")
fi

# Get Darwin CPU is AMD64 or ARM64
UNAME_MACHINE="$(/usr/bin/uname -m)"

function installPolarisServer() {
  echo -e "install polaris server ... "
  local polaris_server_num=$(ps -ef | grep polaris-server | grep -v grep | wc -l)
  if [ $polaris_server_num -ge 1 ]; then
    echo -e "polaris-server is running, skip."
    return
  fi

  local polaris_server_tarnum=$(find . -name "polaris-server-release*.zip" | wc -l)
  if [ $polaris_server_tarnum != 1 ]; then
    echo -e "number of polaris server tar not equal 1, exit."
    exit -1
  fi

  local polaris_server_tarname=$(find . -name "polaris-server-release*.zip")
  local polaris_server_dirname=$(basename ${polaris_server_tarname} .zip)
  if [ ! -e $polaris_server_dirname ]; then
    unzip $polaris_server_tarname > /dev/null
  else
    echo -e "polaris-server-release.tar.gz has been decompressed, skip."
  fi

  cd ${polaris_server_dirname} || (echo "no such directory ${polaris_server_dirname}"; exit -1)
  /bin/bash ./tool/start.sh
  echo -e "install polaris server finish."
  cd ${install_path} || (echo "no such directory ${install_path}"; exit -1)
}

function installPolarisConsole() {
  echo -e "install polaris console ... "
  local polaris_console_num=$(ps -ef | grep polaris-console | grep -v grep | wc -l)
  if [ $polaris_console_num -ge 1 ]; then
    echo -e "polaris-console is running, skip."
    return
  fi

  local polaris_console_tarnum=$(find . -name "polaris-console-release*.zip" | wc -l)
  if [ $polaris_console_tarnum != 1 ]; then
    echo -e "number of polaris console tar not equal 1, exit."
    exit -1
  fi

  local polaris_console_tarname=$(find . -name "polaris-console-release*.zip")
  local polaris_console_dirname=$(basename ${polaris_console_tarname} .zip)
  if [ ! -e $polaris_console_dirname ]; then
    unzip $polaris_console_tarname > /dev/null
  else
    echo -e "polaris-console-release.tar.gz has been decompressed, skip."
  fi

  cd ${polaris_console_dirname} || (echo "no such directory ${polaris_console_dirname}"; exit -1)
  /bin/bash ./tool/start.sh
  echo -e "install polaris console finish."
  cd ${install_path} || (echo "no such directory ${install_path}"; exit -1)
}

function installPrometheus() {
  echo -e "install prometheus ... "
  local prometheus_num=$(ps -ef | grep prometheus | grep -v grep | wc -l)
  if [ ${prometheus_num} -ge 1 ]
  then
    echo -e "prometheus is running, skip"
    return
  fi

  local prometheus_pkg_num=$(find . -name "prometheus-*.tar.gz" | wc -l)
  if [ ${prometheus_pkg_num} != 1 ]; then
    echo -e "number of prometheus package not equals to 1, exit"
    exit -1
  fi

  local target_prometheus_pkg=$(find . -name "prometheus-*.tar.gz")
  local prometheus_dirname=$(basename ${target_prometheus_pkg} .tar.gz)
  if [ -e ${prometheus_dirname} ]
  then
    echo -e "${prometheus_dirname} has exists, now remove it"
  else
    tar -xf ${target_prometheus_pkg}
  fi
  tar -xf ${target_prometheus_pkg} > /dev/null

  pushd ${prometheus_dirname}
  local push_count=$(cat prometheus.yml | grep "push-metrics" | wc -l)
  if [ $push_count -eq 0 ];then
  echo "" >> prometheus.yml
  echo "  - job_name: 'push-metrics'" >> prometheus.yml
  echo "    static_configs:" >> prometheus.yml
  echo "    - targets: ['localhost:9091']" >> prometheus.yml
  echo "    honor_labels: true" >> prometheus.yml
  fi
  nohup ./prometheus --web.enable-lifecycle --web.enable-admin-api >> prometheus.out 2>&1 &
  echo "install prometheus success"
  popd
}

function installPushGateway() {
  echo -e "install pushgateway ... "
  local pgw_num=$(ps -ef | grep pushgateway | grep -v grep | wc -l)
  if [ $pgw_num -ge 1 ]; then
    echo -e "pushgateway is running, skip"
    return
  fi

  local pgw_pkg_num=$(find . -name "pushgateway-*.tar.gz" | wc -l)
  if [ $pgw_pkg_num != 1 ]; then
    echo -e "number of pushgateway package not equals to 1, exit"
    exit -1
  fi

  local target_pgw_pkg=$(find . -name "pushgateway-*.tar.gz")
  local pgw_dirname=$(basename ${target_pgw_pkg} .tar.gz)
  if [ -e ${pgw_dirname} ]
  then
    echo -e "${pgw_dirname} has exists, now remove it"
  else
    tar -xf ${target_pgw_pkg}
  fi
  tar -xf ${target_pgw_pkg} > /dev/null

  pushd ${pgw_dirname}
  nohup ./pushgateway --web.enable-lifecycle --web.enable-admin-api >> pgw.out 2>&1 &
  echo "install pushgateway success"
  popd
}

function checkPort() {
  ports=(8080 8090 8091 7779)
  for port in ${ports[@]}; do
    pid=$(/usr/sbin/lsof -i :${port} | grep LISTEN | awk '{print $1 " " $2}')
    if [ "${pid}" != "" ]; then
      echo "port ${port} has been used, exit."
      exit -1:
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

echo "now, we finish install polaris in your mac, we will exec rollback 'sudo spctl --master-enable'"

sudo spctl --master-enable
