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

ps -ef | grep -w "$cmdline" | grep -v "grep"

#------------------------------------------------------

cd $curpath
