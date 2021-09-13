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

function installPolarisServer() {
    Write-Output "install polaris server ... "
    $polaris_server_num = (Get-Process | findstr "polaris-server" | Measure-Object -Line).Lines
    if ($polaris_server_num -gt 0) {
        Write-Output "polaris-server is running, exit"
        exit -1
    }
    $polaris_server_pkg_num = (Get-ChildItem "polaris-server-release*.zip" | Measure-Object -Line).Lines
    if ($polaris_server_pkg_num -ne 1) {
        Write-Output "number of polaris server package not equals to 1, exit"
        exit -1
    }
    $target_polaris_server_pkg = (Get-ChildItem "polaris-server-release*.zip")[0].Name
    $polaris_server_dirname = ([io.fileinfo]$target_polaris_server_pkg).basename
    if (Test-Path $polaris_server_dirname) {
        Write-Output "$polaris_server_dirname has exists, now remove it"
        Remove-Item $polaris_server_dirname -Recurse
    }
    Expand-Archive -Path $target_polaris_server_pkg -DestinationPath .
    Push-Location $polaris_server_dirname
    Start-Process -FilePath ".\\polaris-server.exe" -ArgumentList ('start')
    Write-Output "install polaris server success"
    Pop-Location
}


function installPolarisConsole() {
    Write-Output "install polaris console ... "
    $polaris_console_num = (Get-Process | findstr "polaris-console" | Measure-Object -Line).Lines
    if ($polaris_console_num -gt 0) {
        Write-Output "polaris-console is running, exit"
        exit -1
    }
    $polaris_console_pkg_num = (Get-ChildItem "polaris-console-release*.zip" | Measure-Object -Line).Lines
    if ($polaris_console_pkg_num -ne 1) {
        Write-Output "number of polaris console package not equals to 1, exit"
        exit -1
    }
    $target_polaris_console_pkg = (Get-ChildItem "polaris-console-release*.zip")[0].Name
    $polaris_console_dirname = ([io.fileinfo]$target_polaris_console_pkg).basename
    if (Test-Path $polaris_console_dirname) {
        Write-Output "$polaris_console_dirname has exists, now remove it"
        Remove-Item $polaris_console_dirname -Recurse
    }
    Expand-Archive -Path $target_polaris_console_pkg -DestinationPath .
    Push-Location $polaris_console_dirname
    Start-Process -FilePath ".\\polaris-console.exe" -ArgumentList ('start')
    Write-Output "install polaris console success"
    Pop-Location
}

function installPrometheus() {
    Write-Output "install prometheus ... "
    $prometheus_num = (Get-Process | findstr "prometheus" | Measure-Object -Line).Lines
    if ($prometheus_num -gt 0) {
        Write-Output "prometheus is running, exit"
        exit -1
    }
    $prometheus_pkg_num = (Get-ChildItem "prometheus-*.zip" | Measure-Object -Line).Lines
    if ($prometheus_pkg_num -ne 1) {
        Write-Output "number of prometheus package not equals to 1, exit"
        exit -1
    }
    $target_prometheus_pkg =  (Get-ChildItem "prometheus-*.zip")[0].Name
    $prometheus_dirname = ([io.fileinfo]$target_prometheus_pkg).basename
    if (Test-Path $prometheus_dirname) {
        Write-Output "$prometheus_dirname has exists, now remove it"
        Remove-Item $prometheus_dirname -Recurse
    }
    Expand-Archive -Path $target_prometheus_pkg -DestinationPath .
    Push-Location $prometheus_dirname
    Add-Content prometheus.yml ""
    Add-Content prometheus.yml "  - job_name: 'push-metrics'"
    Add-Content prometheus.yml "    static_configs:"
    Add-Content prometheus.yml "    - targets: ['localhost:9091']"
    Add-Content prometheus.yml "    honor_labels: true"
    Start-Process -FilePath ".\\prometheus.exe" -ArgumentList ('--web.enable-lifecycle', '--web.enable-admin-api') -RedirectStandardOutput prometheus.out -RedirectStandardError prometheus.err
    Write-Output "install prometheus success"
    Pop-Location
}

function installPushGateway() {
    Write-Output "install pushgateway ... "
    $pgw_num = (Get-Process | findstr "pushgateway" | Measure-Object -Line).Lines
    if ($pgw_num -gt 0) {
        Write-Output "pushgateway is running, exit"
        exit -1
    }
    $pgw_pkg_num = (Get-ChildItem "pushgateway-*.zip" | Measure-Object -Line).Lines
    if ($pgw_pkg_num -ne 1) {
        Write-Output "number of pushgateway package not equals to 1, exit"
        exit -1
    }
    $target_pgw_pkg =  (Get-ChildItem "pushgateway-*.zip")[0].Name
    $pgw_dirname = ([io.fileinfo]$target_pgw_pkg).basename
    if (Test-Path $pgw_dirname) {
        Write-Output "$pgw_dirname has exists, now remove it"
        Remove-Item $pgw_dirname -Recurse
    }
    Expand-Archive -Path $target_pgw_pkg -DestinationPath .
    Push-Location $pgw_dirname
    Start-Process -FilePath ".\\pushgateway.exe" -ArgumentList ('--web.enable-lifecycle', '--web.enable-admin-api') -RedirectStandardOutput pgw.out -RedirectStandardError pgw.err
    Write-Output "install pushgateway success"
    Pop-Location
}

function checkPort() {
    $ports = "8080", "8090", "8091", "7779", "9090", "9091"
    foreach ($port in $ports)
    {
        $processInfo = netstat -ano | findstr "LISTENING" | findstr $port
        if($processInfo)
        {
            Write-Output $processInfo
            Write-Output "port $port has been used, exit"
            exit -1
        }
    }
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

