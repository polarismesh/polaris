#!/bin/bash

rm -rf style_tool
rm -rf goimports-reviser

mkdir -p style_tool

cd style_tool

wget https://github.com/incu6us/goimports-reviser/releases/download/v3.1.1/goimports-reviser_3.1.1_linux_amd64.tar.gz

tar -zxvf goimports-reviser_3.1.1_linux_amd64.tar.gz

mv goimports-reviser ../

cd ../

# find . -name "*.go" -type f | grep -v .pb.go|grep -v test/tools/tools.go | grep -v ./plugin.go | xargs -I {} goimports-reviser -rm-unused -format {} -project-name github.com/polarismesh/polaris
# find . -name "*.go" -type f |grep -v .pb.go|grep -v test/tools/tools.go | grep -v ./plugin.go | xargs -I {} goimports -local github.com/polarismesh/polaris -w {}