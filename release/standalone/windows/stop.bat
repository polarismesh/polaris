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
set "CURRENT_DIR=%cd%"
rem powershell -c "Set-ExecutionPolicy RemoteSigned"
echo allowed to use powershell
powershell -File %CURRENT_DIR%\stop-windows.ps1
endlocal

pause
