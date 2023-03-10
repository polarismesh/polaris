#!/bin/bash
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


# 格式化 go.mod
go mod tidy -compat=1.17


# 处理 go imports 的格式化
rm -rf style_tool
rm -rf goimports-reviser

mkdir -p style_tool

cd style_tool

wget https://github.com/incu6us/goimports-reviser/releases/download/v3.1.1/goimports-reviser_3.1.1_linux_amd64.tar.gz
tar -zxvf goimports-reviser_3.1.1_linux_amd64.tar.gz
mv goimports-reviser ../

cd ../

# 处理 go 代码格式化
go fmt ./...

find . -name "*.go" -type f | grep -v .pb.go|grep -v test/tools/tools.go | grep -v ./plugin.go |\
xargs -I {} ./goimports-reviser -rm-unused -format {} -local github.com/polarismesh/specification -project-name github.com/polarismesh/polaris

