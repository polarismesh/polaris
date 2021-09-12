#!/bin/bash

set -x

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

chmod 755 tool/*.sh $server_name

item="$workdir/tool/check.sh >>$workdir/log/check.log 2>&1"
exist=$(crontab -l | grep "$item" | grep -v "#" | wc -l)
if [[ $(uname -a) == 'Darwin' ]]; then
  # in mac os, if current user not crontab job will echo "crontab: no crontab for ${USER}" in stderr
  exist=$(crontab -l 2>&1 | grep "$item" | grep -v "#" | wc -l)
fi
if [ "$exist" == "0" ]; then
  start
  add_cron
fi

#------------------------------------------------------

cd $curpath
