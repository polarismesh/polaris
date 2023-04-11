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

function runPolarisServer() {
    Write-Output "run polaris server ... "
    $polaris_server_num = (Get-Process | findstr "polaris-server" | Measure-Object -Line).Lines
    if ($polaris_server_num -gt 0) {
        Write-Output "polaris-server is running, skip"
        return
    }
    $polaris_server_dirname = (Get-ChildItem -Directory "polaris-server-release*")[0].Name
    Push-Location $polaris_server_dirname
    Start-Process -FilePath ".\\polaris-server.exe" -ArgumentList ('start') -WindowStyle Hidden
    Write-Output "run polaris server success"
    Pop-Location
}

function runPolarisConsole() {
    Write-Output "run polaris console ... "
    $polaris_console_num = (Get-Process | findstr "polaris-console" | Measure-Object -Line).Lines
    if ($polaris_console_num -gt 0) {
        Write-Output "polaris-console is running, skip"
        return
    }
    $polaris_console_dirname = (Get-ChildItem -Directory "polaris-console-release*")[0].Name
    Push-Location $polaris_console_dirname
    Start-Process -FilePath ".\\polaris-console.exe" -ArgumentList ('start') -WindowStyle Hidden
    Write-Output "run polaris console success"
    Pop-Location
}

function runPolarisLimiter() {
    Write-Output "run polaris limiter ... "
    $polaris_limiter_num = (Get-Process | findstr "polaris-limiter" | Measure-Object -Line).Lines
    if ($polaris_limiter_num -gt 0) {
        Write-Output "polaris-limiter is running, skip"
        return
    }
    $polaris_limiter_dirname = (Get-ChildItem -Directory "polaris-limiter-release*")[0].Name
    Push-Location $polaris_limiter_dirname
    Start-Process -FilePath ".\\polaris-limiter.exe" -ArgumentList ('start') -WindowStyle Hidden
    Write-Output "run polaris limiter success"
    Pop-Location
}

function runPrometheus() {
    Write-Output "run prometheus ... "
    $prometheus_num = (Get-Process | findstr "prometheus" | Measure-Object -Line).Lines
    if ($prometheus_num -gt 0) {
        Write-Output "prometheus is running, skip"
        return
    }
    $prometheus_dirname = (Get-ChildItem -Directory "prometheus-*")[0].Name
    Push-Location $prometheus_dirname
    Start-Process -FilePath ".\\prometheus.exe" -ArgumentList ('--web.enable-lifecycle', '--web.enable-admin-api') -RedirectStandardOutput prometheus.out -RedirectStandardError prometheus.err -WindowStyle Hidden
    Write-Output "run prometheus success"
    Pop-Location
}

function runPushGateway() {
    Write-Output "run pushgateway ... "
    $pgw_num = (Get-Process | findstr "pushgateway" | Measure-Object -Line).Lines
    if ($pgw_num -gt 0) {
        Write-Output "pushgateway is running, skip"
        return
    }
    $pgw_dirname = (Get-ChildItem -Directory "pushgateway-*")[0].Name
    Push-Location $pgw_dirname
    Start-Process -FilePath ".\\pushgateway.exe" -ArgumentList ('--web.enable-lifecycle', '--web.enable-admin-api') -RedirectStandardOutput pgw.out -RedirectStandardError pgw.err -WindowStyle Hidden
    Write-Output "run pushgateway success"
    Pop-Location
}


$fileContent = Get-Content port.properties
$fileContent = $fileContent -join [Environment]::NewLine
$config = ConvertFrom-StringData($fileContent)
$console_port = $config.'polaris_console_port'
$eureka_port = $config.'polaris_eureka_port'
$xdsv3_port = $config.'polaris_xdsv3_port'
$service_grpc_port = $config.'polaris_service_grpc_port'
$config_grpc_port = $config.'polaris_config_grpc_port'
$api_http_port = $config.'polaris_open_api_port'
$prometheus_port = $config.'prometheus_port'
$pushgateway_port = $config.'pushgateway_port'
$limiter_http_port = $config.'polaris_limiter_http_port'
$limiter_grpc_port = $config.'polaris_limiter_grpc_port'

function checkPort() {
    $ports = $console_port, $eureka_port, $xdsv3_port, $prometheus_sd_port, $service_grpc_port, $config_grpc_port, $api_http_port, $prometheus_port, $pushgateway_port, $limiter_http_port,$limiter_grpc_port
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

#检查端口占用
checkPort
# 运行server
runPolarisServer
# 运行console
runPolarisConsole
# 运行polaris-limiter
runPolarisLimiter
# 运行Prometheus
runPrometheus
# 运行Prometheus
runPushGateway