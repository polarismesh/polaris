#!/bin/bash

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
    exit 1
  fi

  local polaris_server_tarname=$(find . -name "polaris-server-release*.zip")
  local polaris_server_dirname=$(basename ${polaris_server_tarname} .zip)
  if [ ! -e $polaris_server_dirname ]; then
    unzip $polaris_server_tarname
  else
    echo -e "polaris-server-release.tar.gz has been decompressed, skip."
  fi

  cd ${polaris_server_dirname} || (echo "no such directory ${polaris_server_dirname}"; exit 1)
  /bin/bash ./tool/start.sh
  echo -e "install polaris server finish."
  cd ${install_path} || (echo "no such directory ${install_path}"; exit 1)
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
    exit 1
  fi

  local polaris_console_tarname=$(find . -name "polaris-console-release*.zip")
  local polaris_console_dirname=$(basename ${polaris_console_tarname} .zip)
  if [ ! -e $polaris_console_dirname ]; then
    unzip $polaris_console_tarname
  else
    echo -e "polaris-console-release.tar.gz has been decompressed, skip."
  fi

  cd ${polaris_console_dirname} || (echo "no such directory ${polaris_console_dirname}"; exit 1)
  /bin/bash ./tool/start.sh
  echo -e "install polaris console finish."
  cd ${install_path} || (echo "no such directory ${install_path}"; exit 1)
}

function installPrometheus() {
  echo -e "install prometheus ... "
  local prometheus_num=$(ps -ef | grep prometheus | grep -v grep | wc -l)
  if [ $prometheus_num -ge 1 ]; then
    echo -e "prometheus is running, skip."
    return
  fi

  local target_prometheus="prometheus-2.28.0.darwin-amd64.tar.gz"
  local target_prometheus_dir="prometheus-2.28.0.darwin-amd64"
  if [[ "$UNAME_MACHINE" == "arm64" ]]; then
    target_prometheus="prometheus-2.28.0.darwin-arm64.tar.gz"
    target_prometheus_dir="prometheus-2.28.0.darwin-arm64"
  fi

  if [ ! -f $target_prometheus ]; then
    echo "file ${target_prometheus} not exists, exit."
  fi

  tar -xf $target_prometheus
  cd ${target_prometheus_dir} || (
    echo "not such directory ${target_prometheus_dir}"
    exit 1
  )
  echo "" >>prometheus.yml
  echo "  - job_name: 'push-metrics'" >>prometheus.yml
  echo "    static_configs:" >>prometheus.yml
  echo "    - targets: ['localhost:9091']" >>prometheus.yml
  echo "    honor_labels: true" >>prometheus.yml
  nohup ./prometheus --web.enable-lifecycle --web.enable-admin-api >>prometheus.out 2>&1 &

  echo "install prometheus success."
  cd ${install_path} || (echo "no such directory ${install_path}"; exit 1)
}

function installPushGateway() {
  echo -e "install pushgateway ... "
  local pgw_num=$(ps -ef | grep pushgateway | grep -v grep | wc -l)
  if [ $pgw_num -ge 1 ]; then
    echo -e "pushgateway is running, skip."
    return
  fi

  local target_pgw="pushgateway-1.4.1.darwin-amd64.tar.gz"
  local target_pgw_dir="pushgateway-1.4.1.darwin-amd64"
  if [[ "$UNAME_MACHINE" == "arm64" ]]; then
    target_pgw="pushgateway-1.4.1.darwin-arm64.tar.gz"
    target_pgw_dir="pushgateway-1.4.1.darwin-arm64"
  fi

  if [ ! -f "$target_pgw" ]; then
    echo "file ${target_pgw} not exists, exit."
  fi

  tar -xf $target_pgw
  cd ${target_pgw_dir} || (echo "not such directory ${target_pgw_dir}"; exit 1)
  nohup ./pushgateway --web.enable-lifecycle --web.enable-admin-api >>pgw.out 2>&1 &

  echo "install pushgateway success."
  cd ${install_path} || (echo "no such directory ${install_path}"; exit 1)
}

function checkPort() {
  ports=(8080 8090 8091 7779)
  for port in ${ports[@]}; do
    pid=$(/usr/sbin/lsof -i :${port} | awk '{print $1 " " $2}')
    if [ "${pid}" != "" ]; then
      echo "port ${port} has been used, exit."
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
