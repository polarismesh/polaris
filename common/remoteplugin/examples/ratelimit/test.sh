#!/usr/bin/env sh

echo 'client building ....'
go build -o rate-limit-client client/client.go

echo 'server-v1 building ...'
go build -o rate-limit-server-v1 server/v1/server.go

echo 'server-v2 building ...'
go build -o rate-limit-server-v2 server/v2/server.go


echo 'server-v3 building...'
go build -o rate-limit-server-v3 server/v3/server.go

ps aux | grep "rate-limit-server-v3"


echo 'clean old server...'
ps aux | grep -v "grep" | grep "rate-limit-server" | awk '{print $2}' | xargs kill -9

echo 'server-v3 running in nohup...'
./rate-limit-server-v3 >> /dev/null 2>&1 &

echo 'start client and call plugin server...'
./rate-limit-client