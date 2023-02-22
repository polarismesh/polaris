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
        Write-Output "polaris-server is running, skip"
        return
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
    sed "conf/polaris-server.yaml" "listenPort: 8761" "listenPort: ${eureka_port}"
    sed "conf/polaris-server.yaml" "listenPort: 15010" "listenPort: ${xdsv3_port}"
    sed "conf/polaris-server.yaml" "listenPort: 8091" "listenPort: ${service_grpc_port}"
    sed "conf/polaris-server.yaml" "listenPort: 8093" "listenPort: ${config_grpc_port}"
    sed "conf/polaris-server.yaml" "listenPort: 8090" "listenPort: ${api_http_port}"
    Start-Process -FilePath ".\\polaris-server.exe" -ArgumentList ('start') -WindowStyle Hidden
    Write-Output "install polaris server success"
    Pop-Location
}


function installPolarisConsole() {
    Write-Output "install polaris console ... "
    $polaris_console_num = (Get-Process | findstr "polaris-console" | Measure-Object -Line).Lines
    if ($polaris_console_num -gt 0) {
        Write-Output "polaris-console is running, skip"
        return
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
    sed "polaris-console.yaml" "listenPort: 8080" "listenPort: ${console_port}"
    sed "polaris-console.yaml" "address: '127.0.0.1:8090'" "address: '127.0.0.1:'${api_http_port}'"
    sed "polaris-console.yaml" "address: '127.0.0.1:9090'" "address: '127.0.0.1:'${prometheus_port}'"
    Start-Process -FilePath ".\\polaris-console.exe" -ArgumentList ('start') -WindowStyle Hidden
    Write-Output "install polaris console success"
    Pop-Location
}

function installPolarisLimiter() {
    Write-Output "install polaris limiter ... "
    $polaris_limiter_num = (Get-Process | findstr "polaris-limiter" | Measure-Object -Line).Lines
    if ($polaris_limiter_num -gt 0) {
        Write-Output "polaris-limiter is running, skip"
        return
    }
    $polaris_limiter_pkg_num = (Get-ChildItem "polaris-limiter-release*.zip" | Measure-Object -Line).Lines
    if ($polaris_limiter_pkg_num -ne 1) {
        Write-Output "number of polaris limiter package not equals to 1, exit"
        exit -1
    }
    $target_polaris_limiter_pkg = (Get-ChildItem "polaris-limiter-release*.zip")[0].Name
    $polaris_limiter_dirname = ([io.fileinfo]$target_polaris_limiter_pkg).basename
    if (Test-Path $polaris_limiter_dirname) {
        Write-Output "$polaris_limiter_dirname has exists, now remove it"
        Remove-Item $polaris_limiter_dirname -Recurse
    }
    Expand-Archive -Path $target_polaris_limiter_pkg -DestinationPath .
    Push-Location $polaris_limiter_dirname
    sed "polaris-limiter.yaml" "polaris-server-address: 127.0.0.1:8091" "polaris-server-address: 127.0.0.1:${service_grpc_port}"
    sed "polaris-limiter.yaml" "port: 8100" "port: ${limiter_http_port}"
    sed "polaris-limiter.yaml" "port: 8101" "port: ${limiter_grpc_port}"
    Start-Process -FilePath ".\\polaris-limiter.exe" -ArgumentList ('start') -WindowStyle Hidden
    Write-Output "install polaris limiter success"
    Pop-Location
}


function installPrometheus() {
    Write-Output "install prometheus ... "
    $prometheus_num = (Get-Process | findstr "prometheus" | Measure-Object -Line).Lines
    if ($prometheus_num -gt 0) {
        Write-Output "prometheus is running, skip"
        return
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
    Add-Content prometheus.yml "    http_sd_configs:"
    Add-Content prometheus.yml "    - url: http://localhost:8090/prometheus/v1/clients"
    Add-Content prometheus.yml ""
    Add-Content prometheus.yml "  - job_name: 'push-metrics'"
    Add-Content prometheus.yml "    static_configs:"
    Add-Content prometheus.yml "    - targets: ['localhost:9091']"
    Add-Content prometheus.yml "    honor_labels: true"
    Start-Process -FilePath ".\\prometheus.exe" -ArgumentList ('--web.enable-lifecycle', '--web.enable-admin-api') -RedirectStandardOutput prometheus.out -RedirectStandardError prometheus.err -WindowStyle Hidden
    Write-Output "install prometheus success"
    Pop-Location
}

function installPushGateway() {
    Write-Output "install pushgateway ... "
    $pgw_num = (Get-Process | findstr "pushgateway" | Measure-Object -Line).Lines
    if ($pgw_num -gt 0) {
        Write-Output "pushgateway is running, skip"
        return
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
    Start-Process -FilePath ".\\pushgateway.exe" -ArgumentList ('--web.enable-lifecycle', '--web.enable-admin-api') -RedirectStandardOutput pgw.out -RedirectStandardError pgw.err -WindowStyle Hidden
    Write-Output "install pushgateway success"
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

function sed($Filename, $Oldvalue, $Newvalue) {
    if (Test-Path $Filename) {
        $content = get-content $Filename
        clear-content $Filename
        foreach($line in $content) {
            $liner=$line.Replace($Oldvalue, $Newvalue)
            Add-content $Filename -Value $liner
        }
    }
}


# 检查端口占用
checkPort
# 安装server
installPolarisServer
# 安装console
installPolarisConsole
# 安装polaris-limiter
installPolarisLimiter
# 安装Prometheus
installPrometheus
# 安装Prometheus
installPushGateway
