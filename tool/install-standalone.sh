# REQUIRED OS Centos7.8

#!/bin/bash
if [ "${0:0:1}" == "/" ]; then
  install_path=$(dirname "$0")
else
  install_path=$(pwd)/$(dirname "$0")
fi

# 脚本名称
program_name="install_standalone"
# 数据库使用密码
general_password="Tse@123456"
# 是否启用数据库
db_enable=false
db_name="polaris_server"
network_name="eth0"
db_ip=""
db_port=""
db_username=""
db_password=""
redis_enable=false
redis_ip=""
redis_port=""
redis_password=""

function Usage(){
  echo "HELP: "
  echo -e "$program_name [-h] [-v] [-e db_enable] [-i <db_ip>] [-p <db_port>] [-u <db_username>] [-w <db_password>] [-n <network_name>]"
}

function showParam() {
  echo "PARAM INFO: "
  echo -e "\t -db option"
  echo -e "\t\t db_enable: $db_enable; db_ip: $db_ip; db_port: $db_port; db_username: $db_username; db_password: $db_password;"
}

function installMysql() {
  if [ $(command -v mysql) ]; then
    echo "mysql has installed, please use \"$program_name -e\" to install polaris."
    Usage
    exit -1
  fi

  local target_mysql_rpm=mysql57-community-release-el7-11.noarch.rpm
  if [ ! -f $target_mysql_rpm ]; then
    wget -T10 -t3 https://repo.mysql.com//${target_mysql_rpm}
    if [ $? -ne 0 ]; then
      echo "download $target_mysql_rpm to $install_path fail, exit."
      exit -1
    else
      echo "download $target_mysql_rpm success."
    fi
  fi

  # 安装rpm
  yum -y module disable mysql
  yum -y install ${target_mysql_rpm}
  yum -y install mysql-community-server
  systemctl start mysqld

  # 检查mysql是否存在
  local mysql_num=$(ps -ef | grep mysql | grep -v grep | wc -l)
  if [ $mysql_num -eq 0 ]; then
    echo "mysql is not started, exit."
    exit -1
  fi

  local pwdText=$(cat /var/log/mysqld.log |grep "temporary password")
  if [ -z "$pwdText" ]; then
    echo "cannot get tempopary password, exit."
    exit -1
  fi

  local tmpPwd=${pwdText#*root@localhost: }
  echo "replace old password $tmpPwd with general password $general_password ."
  mysql -uroot -p$tmpPwd -e"ALTER USER USER() IDENTIFIED BY '$general_password'" --connect-expired-password
  mysql -uroot -p$general_password -e "Grant all privileges on *.* to 'root'@'%' identified by '${general_password}' with grant option"
  if [ $? -ne 0 ]; then
    echo "replace password failed, exit."
    exit -1
  fi
  
  db_enable=true
  db_ip=$(ifconfig ${network_name} | grep inet |grep -v inet6|awk '{print $2}'|tr -d "addr:")
  db_port="3306"
  db_username="root"
  db_password=$general_password
  if [ $? -ne 0 ]; then
    echo "failed to configure variable, exit."
    exit -1
  fi
  
  if [ -z "$db_ip" ]; then
    echo "failed to get ip, exit."
    exit -1
  fi

  cd $install_path
}

function paramCheck() {
  if [ $db_enable = "true" ]
  then
    if [ -z $db_ip ] || [ -z $db_port ] || [ -z $db_username ] || [ -z $db_password ]
    then
      echo -e "not all required database options \033[31m[db_ip, db_port, db_username, db_password]\033[0m are configured, exit."
      exit -1
    else
      echo -e "database parameter check success."
    fi
  else
    echo -e "database not enable, skip."
  fi
  
  if [ $redis_enable = "true" ]
  then
    if [ -z $redis_ip ] || [ -z $redis_port ] || [ -z $redis_password ]
    then
      echo -e "not all required redis options \033[31m[redis_ip, redis_port, redis_password]\033[0m are configured, exit."
      exit -1
    else
      echo -e "redis parameter check success."
    fi
  else
    echo -e "redis not enable, skip."
  fi

  cd $install_path
}

function connectCheck() {
  mysql -u$db_username -h$db_ip -p$db_password -P$db_port -e";"
  if [ $? -ne 0 ]; then
    echo "mysql connect check fail, exit."
    exit -1
  else
    echo "mysql connect check success."
  fi

  cd $install_path
}

function importSchema() {
  echo -e "import schema ..."
  cd $install_path/database
  
  local database_num=$(mysql -u$db_username -h$db_ip -p$db_password -P$db_port -e "select count(*) from information_schema.SCHEMATA where SCHEMA_NAME=\"${db_name}\";")
  if [ ${database_num:0-1:1} == 0 ];
  then
    mysql -u$db_username -h$db_ip -p$db_password -P$db_port < polaris_server.sql
    local result=$? 
    if [ "$result" == "0" ]; 
    then
      echo -e "install database finish. "
    else
      echo -e "install database encountered err, exit."
      exit $?
    fi
  else
    echo -e "database has created, skip."
  fi
  
  cd $install_path
}

function installPolarisServer() {
  echo -e "install polaris server ... "
  local polaris_server_num=$(ps -ef | grep polaris-server | grep -v grep | wc -l)
  if [ $polaris_server_num -ge 1 ]; then
    echo -e "polaris-server is running, skip."
    return
  fi
  
  local polaris_server_tarnum=$(find . -name "polaris-server-release*.tar.gz" | wc -l)
  if [ $polaris_server_tarnum != 1 ]; then
    echo -e "number of polaris server tar not equal 1, exit."
    exit -1
  fi
  
  local polaris_server_tarname=$(find . -name "polaris-server-release*.tar.gz")
  local polaris_server_config_filename="polaris-server.yaml"
  local polaris_server_dirname=$(basename ${polaris_server_tarname} .tar.gz)
  if [ ! -e $polaris_server_dirname ]
  then
    tar -xf $polaris_server_tarname
  else
    echo -e "polaris-server-release.tar.gz has been decompressed, skip."
  fi
  
  cd $polaris_server_dirname
  sed -i "s/##DB_ADDR##/${db_ip}:${db_port}/g" $polaris_server_config_filename
  sed -i "s/##DB_USER##/${db_username}/g" $polaris_server_config_filename
  sed -i "s/##DB_PWD##/${db_password}/g" $polaris_server_config_filename 
  sed -i "s/##DB_NAME##/${db_name}/g" $polaris_server_config_filename
  if [ $? != 0 ]; then
    echo -e "error happen when prepare polaris server config file, exit."
    exit $?
  fi
  
  /bin/bash ./tool/install.sh
  echo -e "install polaris server finish." 
  cd $install_path
}

function installPolarisConsole() {
  echo -e "install polaris console ... "
  local polaris_console_num=$(ps -ef | grep polaris-console | grep -v grep | wc -l)
  if [ $polaris_console_num -ge 1 ]; then
    echo -e "polaris-console is running, skip."
    return
  fi

  local polaris_console_tarnum=$(find . -name "polaris-console-release*.tar.gz" | wc -l)
  if [ $polaris_console_tarnum != 1 ]; then
    echo -e "number of polaris console tar not equal 1, exit."
    exit -1
  fi

  local polaris_console_tarname=$(find . -name "polaris-console-release*.tar.gz")
  local polaris_console_dirname=$(basename ${polaris_console_tarname} .tar.gz)
  if [ ! -e $polaris_console_dirname ]
  then
    tar -xf $polaris_console_tarname
  else
    echo -e "polaris-console-release.tar.gz has been decompressed, skip."
  fi

  cd $polaris_console_dirname
  /bin/bash ./tool/install.sh
  echo -e "install polaris console finish."
  cd $install_path
}

function installPrometheus() {
  echo -e "install prometheus ... "
  local promethues_num=$(ps -ef | grep prometheus | grep -v grep | wc -l)
  if [ $promethues_num -ge 1 ]
  then
    echo -e "prometheus is running, skip."
    return
  fi

  local target_prometheus="prometheus-2.28.0.linux-amd64.tar.gz"
  if [ ! -f $target_prometheus ]
  then
    wget -T10 -t3 https://github.com/prometheus/prometheus/releases/download/v2.28.0/${target_prometheus} --no-check-certificate
    if [ $? -ne 0 ]; then
      echo "download $target_prometheus to $install_path fail, exit."
      exit -1
    else
      echo "download $target_prometheus success."
    fi
  fi

  tar -xf $target_prometheus
  cd prometheus-2.28.0.linux-amd64
  echo "" >> prometheus.yml
  echo "  - job_name: 'push-metrics'" >> prometheus.yml
  echo "    static_configs:" >> prometheus.yml
  echo "    - targets: ['localhost:9091']" >> prometheus.yml
  echo "    honor_labels: true" >> prometheus.yml
  nohup ./prometheus --web.enable-lifecycle --web.enable-admin-api >> prometheus.out 2>&1 &

  echo "install prometheus success."
  cd $install_path
}

function installPushGateway() {
  echo -e "install pushgateway ... "
  local pgw_num=$(ps -ef | grep pushgateway | grep -v grep | wc -l)
  if [ $pgw_num -ge 1 ]; then
    echo -e "pushgateway is running, skip."
    return
  fi

  local target_pgw=pushgateway-1.4.1.linux-amd64.tar.gz
  if [ ! -f "$target_pgw" ]; then
    wget -T10 -t3 https://github.com/prometheus/pushgateway/releases/download/v1.4.1/${target_pgw} --no-check-certificate
    if [ $? -ne 0 ]; then
      echo "download $target_pgw to $install_path fail, exit."
      exit -1
    else
      echo "download $target_pgw success."
    fi
  fi

  tar -xf $target_pgw
  cd pushgateway-1.4.1.linux-amd64
  nohup ./pushgateway --web.enable-lifecycle --web.enable-admin-api >> pgw.out 2>&1 &

  echo "install pushgateway success."
  cd $install_path
}

function createPolarisService() {
  local create_discover_rsp=$(curl -w "##%{http_code}" -H 'Content-Type: application/json;charset=UTF-8' -d '[{"name":"polaris-server","namespace":"Polaris","owners":"polaris","business":"polaris server","comment":"","metadata":{}}]' http://127.0.0.1/naming/v1/services)
  local result=$?
  if [ "$result" == "0" ];
  then
    local http_code=${create_discover_rsp##*##}
    local rsp_body=${create_discover_rsp%%##*}
    echo -e "create polaris-server service response: $(echo ${rsp_body%%##*} | sed ":a;N;s/[  \t \n]//g;ta" )"

    if [ "$http_code" == "200" ];
    then
      echo -e "create polaris-server service success."
    else
      local rsp_check=$(echo ${rsp_body%%##*}| grep "existed resource" | wc -l)
      if [ $rsp_check -ge 1 ];then
        echo "polaris-server is existed, skip."
        return
      fi
      
      echo -e "create polaris-server service fail, http_code=$http_code"
      exit -1
    fi
  else
    echo -e "curl create polaris service fail: ret=$result"
    exit -1
  fi
  cd $install_path
}

function aliasDiscover() {
  local alias_rsp=$(curl -w "##%{http_code}" -H 'Content-Type: application/json;charset=UTF-8' -d '{"service":"polaris-server","namespace":"Polaris","type":0,"alias":"polaris.discover"}' http://127.0.0.1/naming/v1/service/alias)
  local result=$?
  if [ "$result" == "0" ];
  then
    local http_code=${alias_rsp##*##}
    local rsp_body=${alias_rsp%%##*}
    echo -e "alias polaris.discover service response: $(echo ${rsp_body%%##*} | sed ":a;N;s/[  \t \n]//g;ta" )"

    if [ "$http_code" == "200" ];
    then
      echo -e "alias polaris.discover service success."
    else
      local rsp_check=$(echo ${rsp_body%%##*}| grep "existed resource" | wc -l)
      if [ $rsp_check -ge 1 ];then
        echo "polaris.discover is existed, skip."
        return
      fi

      echo -e "alias polaris.discover service fail, http_code=$http_code"
      exit -1
    fi
  else
    echo -e "curl alias discover service fail: ret=$result"
    exit -1
  fi
  cd $install_path
}

function aliasHealthCheck() {
  local alias_rsp=$(curl -w "##%{http_code}" -H 'Content-Type: application/json;charset=UTF-8' -d '{"service":"polaris-server","namespace":"Polaris","type":0,"alias":"polaris.healthcheck"}' http://127.0.0.1/naming/v1/service/alias)
  local result=$?
  if [ "$result" == "0" ];
  then
    local http_code=${alias_rsp##*##}
    local rsp_body=${alias_rsp%%##*}
    echo -e "alias polaris.healthcheck service response: $(echo ${rsp_body%%##*} | sed ":a;N;s/[  \t \n]//g;ta" )"

    if [ "$http_code" == "200" ];
    then
      echo -e "alias polaris.healthcheck service success."
    else
      local rsp_check=$(echo ${rsp_body%%##*}| grep "existed resource" | wc -l)
      if [ $rsp_check -ge 1 ];then
        echo "alias polaris.healthcheck is existed, skip."
        return
      fi

      echo -e "alias healthcheck service fail, http_code=$http_code"
      exit -1
    fi
  else
    echo -e "alias healthcheck service fail: ret=$result"
    exit -1
  fi
  cd $install_path
}

function createPushGatewayService() {
  local pgw_rsp=$(curl -w "##%{http_code}" -H 'Content-Type: application/json;charset=UTF-8' -d '[{"name":"polaris.monitor","namespace":"Polaris","owners":"polaris","business":"polaris monitor","comment":"","metadata":{}}]' http://127.0.0.1/naming/v1/services)
  local result=$?
  if [ "$result" == "0" ];
  then
    local http_code=${pgw_rsp##*##}
    local rsp_body=${pgw_rsp%%##*}
    echo -e "create push gateway service response: $(echo ${rsp_body%%##*} | sed ":a;N;s/[  \t \n]//g;ta" )"

    if [ "$http_code" == "200" ];
    then
      echo -e "create push gateway service success."
    else
      local rsp_check=$(echo ${rsp_body%%##*}| grep "existed resource" | wc -l)
      if [ $rsp_check -ge 1 ];then
        echo "push gateway service is existed, skip."
        return
      fi

      echo -e "create push gateway fail, http_code=$http_code"
      exit -1
    fi
  else
    echo -e "create push gateway fail: ret=$result"
    exit -1
  fi
  cd $install_path
}

function registerPushGateway() {
  local local_host=$(ifconfig ${network_name} | grep inet |grep -v inet6|awk '{print $2}'|tr -d "addr:")
  if [ -z "$local_host" ];then
     echo "get ip fail, exit."
     exit
  fi

  local pgw_rsp=$(curl -w "##%{http_code}" -H 'Content-Type: application/json;charset=UTF-8' -d '[{"service":"polaris.monitor","namespace":"Polaris","weight":100,"healthy":true,"isolate":false,"port":9091,"host":"'${local_host}'","enable_health_check":false,"metadata":{}}]' http://127.0.0.1/naming/v1/instances)
  local result=$?
  if [ "$result" == "0" ];
  then
    local http_code=${pgw_rsp##*##}
    local rsp_body=${pgw_rsp%%##*}
    echo -e "register push gateway service response: $(echo ${rsp_body%%##*} | sed ":a;N;s/[  \t \n]//g;ta" )"

    if [ "$http_code" == "200" ];
    then
      echo -e "register push gateway service success."
    else
      local rsp_check=$(echo ${rsp_body%%##*}| grep "existed resource" | wc -l)
      if [ $rsp_check -ge 1 ];then
        echo "register push gateway is existed, skip."
        return
      fi

      echo -e "register push gateway fail, http_code=$http_code"
      exit -1
    fi
  else
    echo -e "register push gateway fail: ret=$result"
    exit -1
  fi
  cd $install_path
}

function addRouteStrategy() {
  local add_route_req_json='[{"namespace":"Polaris","service":"polaris-server","inbounds":'$(cat ${install_path}/route-rule/rule.yaml | sed ":a;N;s/[  \t \n]//g;ta")',"outbounds":[]}]'
  local add_route_rsp=$(curl -w "##%{http_code}" -H 'Content-Type: application/json;charset=UTF-8' -d $add_route_req_json http://127.0.0.1/naming/v1/routings)
  local result=$?
  if [ "$result" == "0" ];
  then
    local http_code=${add_route_rsp##*##}
    local rsp_body=${add_route_rsp%%##*}
    echo -e "add route response: $(echo ${rsp_body%%##*} | sed ":a;N;s/[  \t \n]//g;ta" )"

    if [ "$http_code" == "200" ];
    then
      echo -e "add route success."
    else
      local rsp_check=$(echo ${rsp_body%%##*}| grep "existed resource" | wc -l)
      if [ $rsp_check -ge 1 ];then
        echo "polaris.discover route is existed, skip."
        return
      fi

      echo -e "add route fail, http_code=$http_code, exit."
      exit -1
    fi
  else
    echo -e "curl route fail: ret=$result, exit."
    exit -1
  fi
  cd $install_path
}

function restartPolarisServer() {
  cd $install_path
  local polaris_discover_tarname=$(find . -name "polaris-server-release*.tar.gz")
  local polaris_discover_config_filename="polaris-server.yaml"
  local polaris_discover_dirname=$(basename ${polaris_discover_tarname} .tar.gz)
  cd $polaris_discover_dirname
  
  local enable_register_line=$(grep -n "enable_register:" $polaris_discover_config_filename | awk -F ":" '{print $1}')
  sed -i ''$enable_register_line's/enable_register: false/enable_register: true/' $polaris_discover_config_filename
  sed -i 's/isolated: true/isolated: false/' $polaris_discover_config_filename

  echo -e "restart polaris server... "
  /bin/bash ./tool/stop.sh
  sleep 2s
  /bin/bash ./tool/start.sh
  cd $install_path
}

while getopts ':hvn:ei:p:u:w:EI:P:W:' varname; do
  case $varname in
  h)
    Usage
    exit 1
    ;;
  v)
    echo "version 1.0"
    exit 1
    ;;
  n)
    network_name=$OPTARG
    ;;
  e)
    db_enable=true
    ;;
  i)
    db_ip=$OPTARG
    ;;
  p)
    db_port=$OPTARG
    ;;
  u)
    db_username=$OPTARG
    ;;
  w)
    db_password=$OPTARG
    ;;
  E)
    redis_enable=true
    ;;
  I)
    redis_ip=$OPTARG
    ;;
  P)
    redis_port=$OPTARG
    ;;
  W)
    redis_password=$OPTARG
    ;;
  *)
    echo "unexpected option '-$OPTARG'"
    exit -1
    ;;
  esac
done

# 确认mysql是否已经安装
if [ $db_enable = "false" ]
then
  # 如果mysql已安装，则要求使用 -e 选项启动安装脚本
  # 如果mysql未安装，则安装mysql并初始化必要的变量
  installMysql
fi

if [ $db_enable = "true" ]
then
  # 参数检查
  paramCheck
  # 数据库连通性检查
  connectCheck
  # 导入数据
  importSchema
  # 安装server
  installPolarisServer
  # 安装console
  installPolarisConsole
  # 配置server
  createPolarisService
  aliasDiscover
  aliasHealthCheck
  addRouteStrategy
  restartPolarisServer
  # 安装Prometheus和PushGateWay
  installPrometheus
  installPushGateway
  createPushGatewayService
  registerPushGateway
else
  echo -e "database config needed."
  exit -1
fi
