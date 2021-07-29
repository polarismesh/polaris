#!/bin/bash

curpath=$(pwd)

if [ "${0:0:1}" == "/" ]; then
    dir=$(dirname "$0")
else
    dir=$(pwd)/$(dirname "$0")
fi

cd $dir/..
workdir=$(pwd)

#------------------------------------------------------
source tool/include

pids=$(ps -e -o pid,cmd | grep -w "$cmdline" | grep -v "grep" | awk '{print $1}')
array=($pids)
if [ "${#array[@]}" == "0" ]; then
    start
fi

#------------------------------------------------------

cd $curpath
