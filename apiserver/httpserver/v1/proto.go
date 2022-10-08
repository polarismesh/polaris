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

package v1

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

// ServiceArr 服务数组定义
type ServiceArr []*api.Service

// Reset 重置初始化
func (m *ServiceArr) Reset() { *m = ServiceArr{} }

// String return string
func (m *ServiceArr) String() string { return proto.CompactTextString(m) }

// ProtoMessage return proto message
func (*ServiceArr) ProtoMessage() {}

// InstanceArr 服务实例数组定义
type InstanceArr []*api.Instance

// Reset reset initialization
func (m *InstanceArr) Reset() { *m = InstanceArr{} }

// String
func (m *InstanceArr) String() string { return proto.CompactTextString(m) }

// ProtoMessage return proto message
func (*InstanceArr) ProtoMessage() {}

// RoutingArr 路由规则数组定义
type RoutingArr []*api.Routing

// Reset reset initialization
func (m *RoutingArr) Reset() { *m = RoutingArr{} }

// String return string
func (m *RoutingArr) String() string { return proto.CompactTextString(m) }

// ProtoMessage return proto message
func (*RoutingArr) ProtoMessage() {}

// RateLimitArr 限流规则数组定义
type RateLimitArr []*api.Rule

// Reset reset initialization
func (m *RateLimitArr) Reset() { *m = RateLimitArr{} }

// String
func (m *RateLimitArr) String() string { return proto.CompactTextString(m) }

// ProtoMessage return proto message
func (*RateLimitArr) ProtoMessage() {}

// CircuitBreakerArr 熔断规则数组定义
type CircuitBreakerArr []*api.CircuitBreaker

// Reset reset initialization
func (m *CircuitBreakerArr) Reset() { *m = CircuitBreakerArr{} }

// String
func (m *CircuitBreakerArr) String() string { return proto.CompactTextString(m) }

// ProtoMessage return proto message
func (*CircuitBreakerArr) ProtoMessage() {}

// ConfigReleaseArr 发布规则数组定义
type ConfigReleaseArr []*api.ConfigRelease

// Reset reset initialization
func (m *ConfigReleaseArr) Reset() { *m = ConfigReleaseArr{} }

// String return string
func (m *ConfigReleaseArr) String() string { return proto.CompactTextString(m) }

// ProtoMessage return proto message
func (*ConfigReleaseArr) ProtoMessage() {}

// ServiceAliasArr 服务实例数组定义
type ServiceAliasArr []*api.ServiceAlias

// Reset reset initialization
func (m *ServiceAliasArr) Reset() { *m = ServiceAliasArr{} }

// String return string
func (m *ServiceAliasArr) String() string { return proto.CompactTextString(m) }

// ProtoMessage proto message
func (*ServiceAliasArr) ProtoMessage() {}
