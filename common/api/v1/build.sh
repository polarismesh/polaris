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

#!/bin/bash

# 安装protoc和protoc-gen-go插件
#
# 注意：
# grpc包引入github.com/golang/protobuf/proto v1.2.0
# protoc-gen-go插件和引入proto包的版本必须保持一致
#
# github.com/golang/protobuf/
#   protoc-gen-go：在pb.go文件中插入proto.ProtoPackageIsVersionX
#   proto：在lib.go中定义ProtoPackageIsVersionX
#
# ProtoPackageIsVersion并非表示proto2/proto3

CURRENT_OS=$(uname -s)
CURRENT_ARCH=$(uname -m)
PROTOC=../protoc
PROTO_FILES="model.proto client.proto service.proto routing.proto ratelimit.proto circuitbreaker.proto configrelease.proto \
             platform.proto request.proto response.proto grpcapi.proto config_file.proto config_file_response.proto \
             grpc_config_api.proto auth.proto"

if [ "$CURRENT_ARCH" != "x86_64" ]; then
    echo "Current only support x86_64"
    exit 1
fi

if [ "$CURRENT_OS" == "Linux" ]; then
    ${PROTOC}/bin/protoc \
    --plugin=protoc-gen-go=${PROTOC}/bin/protoc-gen-go \
    --go_out=plugins=grpc:. \
    --proto_path=${PROTOC}/include \
    --proto_path=. \
    ${PROTO_FILES}

    ${PROTOC}/bin/protoc-go-inject-tag -input="*.pb.go"
else
    docker run --rm -it -v "$(dirname $(pwd))":/app --workdir /app/v1 debian:buster ./build.sh
fi
