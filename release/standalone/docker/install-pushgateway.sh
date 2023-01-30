#!/bin/bash

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
        exit -1
    fi

    local target_pgw_pkg=$(find . -name "pushgateway-*.tar.gz")
    local pgw_dirname=$(basename ${target_pgw_pkg} .tar.gz)
    if [ -e ${pgw_dirname} ]; then
        echo -e "${pgw_dirname} has exists, now remove it"
        rm -rf ${pgw_dirname}
    fi
    tar -xf ${target_pgw_pkg} >/dev/null

    pushd ${pgw_dirname}
    nohup ./pushgateway --web.enable-lifecycle --web.enable-admin-api --web.listen-address=:${pushgateway_port} >>pgw.out 2>&1 &
    echo "install pushgateway success"
    popd
}

installPushGateway
