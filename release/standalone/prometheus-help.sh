#!/bin/bash

if [ $# -lt 1 ]; then
    echo "$0 start port|stop"
    exit 0
fi

command=$1

if [ ${command} == "start" ]; then
    prometheus_port=$2
    if [ "${prometheus_port}w" == "w" ]; then
        prometheus_port="9090"
    fi

    nohup ./polaris-prometheus --web.enable-lifecycle --web.enable-admin-api --web.listen-address=:${prometheus_port} >>prometheus.out 2>&1 &
fi

if [ ${command} == "stop" ]; then
    pid=$(ps -ef | grep polaris-prometheus | grep -v grep | awk '{print $2}')
    if [ "${pid}" != "" ]; then
        echo -e "start to kill polaris-prometheus process ${pid}"
        kill -9 ${pid}
    else
        echo "not found running polaris-prometheus"
    fi
fi
