@echo off

rem Tencent is pleased to support the open source community by making Polaris available.
rem Copyright (C) 2019 THL A29 Limited, a Tencent company. All rights reserved.
rem Licensed under the BSD 3-Clause License (the "License");
rem you may not use this file except in compliance with the License.
rem You may obtain a copy of the License at
rem
rem https://opensource.org/licenses/BSD-3-Clause
rem
rem Unless required by applicable law or agreed to in writing, software distributed
rem under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR
rem CONDITIONS OF ANY KIND, either express or implied. See the License for the
rem specific language governing permissions and limitations under the License.

setlocal

rem Guess POLARIS_HOME if not defined
set "SERVER_NAME=polaris-server.exe"
set "CURRENT_DIR=%cd%"
if not "%POLARIS_HOME%" == "" goto gotHome
set "POLARIS_HOME=%CURRENT_DIR%"
if exist "%POLARIS_HOME%\%SERVER_NAME%" goto okHome
cd ..
set "POLARIS_HOME=%cd%"
cd "%CURRENT_DIR%"
:gotHome
if exist "%POLARIS_HOME%\%SERVER_NAME%" goto okHome
echo The POLARIS_HOME environment variable is not defined correctly
echo This environment variable is needed to run this program
goto end
:okHome

set "EXECUTABLE=%POLARIS_HOME%\%SERVER_NAME%"

rem Check that target executable exists
if exist "%EXECUTABLE%" goto okExec
echo Cannot find "%EXECUTABLE%"
echo This file is needed to run this program
goto end
:okExec

echo "start %EXECUTABLE%"

set/p SERVER_PROCESS=tasklist | findstr "%SERVER_NAME%"
if not "%SERVER_PROCESS%" equ ""  (
   echo "%SERVER_PROCESS% started"
   goto end
)

pushd %POLARIS_HOME%
start "" "%EXECUTABLE%" start
popd
goto end

:end