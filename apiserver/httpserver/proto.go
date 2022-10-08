/**
 * Tencent is pleased to support the open source community by making Polaris available.
 *
 * Copyright (C) 2019 THL A29 Limited, a Tencent company. All rights reserved.
 *
 * Licensed under the BSD 3-Clause License (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * https://opensource.org/licenses/BSD-3-Clause
 *
 * Unless required by applicable law or agreed to in writing, software distributed
 * under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR
 * CONDITIONS OF ANY KIND, either express or implied. See the License for the
 * specific language governing permissions and limitations under the License.
 */

package httpserver

import (
	"github.com/golang/protobuf/proto"

	api "github.com/polarismesh/polaris/common/api/v1"
)

// NamespaceArr 命名空间数组定义
type NamespaceArr []*api.Namespace

// Reset 重置初始化
func (m *NamespaceArr) Reset() { *m = NamespaceArr{} }

// String return string
func (m *NamespaceArr) String() string { return proto.CompactTextString(m) }

// ProtoMessage return proto message
func (*NamespaceArr) ProtoMessage() {}

// ConfigReleaseArr 发布规则数组定义
type ConfigReleaseArr []*api.ConfigRelease

// Reset reset initialization
func (m *ConfigReleaseArr) Reset() { *m = ConfigReleaseArr{} }

// String return string
func (m *ConfigReleaseArr) String() string { return proto.CompactTextString(m) }

// ProtoMessage return proto message
func (*ConfigReleaseArr) ProtoMessage() {}

// PlatformArr 平台数组定义
type PlatformArr []*api.Platform

// Reset proto reset
func (m *PlatformArr) Reset() { *m = PlatformArr{} }

// String proto string
func (m *PlatformArr) String() string { return proto.CompactTextString(m) }

// ProtoMessage proto message
func (m *PlatformArr) ProtoMessage() {}

// ServiceAliasArr 服务实例数组定义
type ServiceAliasArr []*api.ServiceAlias

// Reset reset initialization
func (m *ServiceAliasArr) Reset() { *m = ServiceAliasArr{} }

// String return string
func (m *ServiceAliasArr) String() string { return proto.CompactTextString(m) }

// ProtoMessage proto message
func (*ServiceAliasArr) ProtoMessage() {}

// ConfigFileArr 配置文件数组定义
type ConfigFileArr []*api.ConfigFile

// Reset reset initialization
func (m *ConfigFileArr) Reset() { *m = ConfigFileArr{} }

// String return string
func (m *ConfigFileArr) String() string { return proto.CompactTextString(m) }

// ProtoMessage proto message
func (*ConfigFileArr) ProtoMessage() {}
