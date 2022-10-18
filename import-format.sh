#!/bin/bash

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

find . -name "*.go" -type f | grep -v .pb.go|grep -v test/tools/tools.go | grep -v ./plugin.go | xargs -I {} goimports-reviser -rm-unused -format {} -project-name github.com/polarismesh/polaris

# 处理 go 代码格式化
go fmt ./...
