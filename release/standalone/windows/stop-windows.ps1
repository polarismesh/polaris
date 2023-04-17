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

$ErrorActionPreference = "Stop"

function stopPolarisServer {
    Write-Output "stop polaris server ... "
    Get-Process | ForEach-Object($_.name) {
        if($_.name -eq "polaris-server") {
            $process_pid = $_.Id
            Write-Output "start to kill polaris-server process $process_pid"
            Stop-Process -Id $process_pid
            Start-Sleep -Seconds 2
        }
    }
    Write-Output "stop polaris server success"
}

function stopPolarisConsole {
    Write-Output "stop polaris-console ... "
    Get-Process | ForEach-Object($_.name) {
        if($_.name -eq "polaris-console") {
            $process_pid = $_.Id
            Write-Output "start to kill polaris-console process $process_pid"
            Stop-Process -Id $process_pid
            Start-Sleep -Seconds 2
        }
    }
    Write-Output "stop polaris console success"
}

function stopPolarisLimiter {
    Write-Output "stop polaris-limiter ... "
    Get-Process | ForEach-Object($_.name) {
        if($_.name -eq "polaris-limiter") {
            $process_pid = $_.Id
            Write-Output "start to kill polaris-limiter process $process_pid"
            Stop-Process -Id $process_pid
            Start-Sleep -Seconds 2
        }
    }
    Write-Output "stop polaris limiter success"
}

function stopPrometheus {
    Write-Output "stop prometheus ... "
    Get-Process | ForEach-Object($_.name) {
        if($_.name -eq "prometheus") {
            $process_pid = $_.Id
            Write-Output "start to kill prometheus process $process_pid"
            Stop-Process -Id $process_pid
            Start-Sleep -Seconds 2
        }
    }
    Write-Output "stop prometheus success"
}

function stopPushGateway {
    Write-Output "stop pushgateway ... "
    Get-Process | ForEach-Object($_.name) {
        if($_.name -eq "pushgateway") {
            $process_pid = $_.Id
            Write-Output "start to kill pushgateway process $process_pid"
            Stop-Process -Id $process_pid
            Start-Sleep -Seconds 2
        }
    }
    Write-Output "stop pushgateway success"
}

# 停止 server
stopPolarisServer
# 停止 console
stopPolarisConsole
# 停止 limiter
stopPolarisLimiter
# 停止 prometheus
stopPrometheus
# 停止 pushgateway
stopPushGateway
