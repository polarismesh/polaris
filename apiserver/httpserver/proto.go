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

	api "github.com/polarismesh/polaris-server/common/api/v1"
)

// NamespaceArr 命名空间数组定义
type NamespaceArr []*api.Namespace

// Reset
func (m *NamespaceArr) Reset() { *m = NamespaceArr{} }

// String return string
func (m *NamespaceArr) String() string { return proto.CompactTextString(m) }

// ProtoMessage
func (*NamespaceArr) ProtoMessage() {}

// ServiceArr 服务数组定义
type ServiceArr []*api.Service

// Reset
func (m *ServiceArr) Reset() { *m = ServiceArr{} }

// String
func (m *ServiceArr) String() string { return proto.CompactTextString(m) }

// ProtoMessage
func (*ServiceArr) ProtoMessage() {}

// InstanceArr 服务实例数组定义
type InstanceArr []*api.Instance

// Reset
func (m *InstanceArr) Reset() { *m = InstanceArr{} }

// String
func (m *InstanceArr) String() string { return proto.CompactTextString(m) }

// ProtoMessage
func (*InstanceArr) ProtoMessage() {}

// RoutingArr 路由规则数组定义
type RoutingArr []*api.Routing

// Reset
func (m *RoutingArr) Reset() { *m = RoutingArr{} }

// String
func (m *RoutingArr) String() string { return proto.CompactTextString(m) }

// ProtoMessage
func (*RoutingArr) ProtoMessage() {}

// RateLimitArr 限流规则数组定义
type RateLimitArr []*api.Rule

// Reset
func (m *RateLimitArr) Reset() { *m = RateLimitArr{} }

// String
func (m *RateLimitArr) String() string { return proto.CompactTextString(m) }

// ProtoMessage
func (*RateLimitArr) ProtoMessage() {}

// CircuitBreakerArr 熔断规则数组定义
type CircuitBreakerArr []*api.CircuitBreaker

// Reset
func (m *CircuitBreakerArr) Reset() { *m = CircuitBreakerArr{} }

// String
func (m *CircuitBreakerArr) String() string { return proto.CompactTextString(m) }

// ProtoMessage
func (*CircuitBreakerArr) ProtoMessage() {}

// ConfigReleaseArr 发布规则数组定义
type ConfigReleaseArr []*api.ConfigRelease

// Reset
func (m *ConfigReleaseArr) Reset() { *m = ConfigReleaseArr{} }

// String
func (m *ConfigReleaseArr) String() string { return proto.CompactTextString(m) }

// ProtoMessage
func (*ConfigReleaseArr) ProtoMessage() {}

// PlatformArr 平台数组定义
type PlatformArr []*api.Platform

// Reset proto reset
func (m *PlatformArr) Reset() { *m = PlatformArr{} }

// String proto string
func (m *PlatformArr) String() string { return proto.CompactTextString(m) }

// ProtoMessage proto message
func (m *PlatformArr) ProtoMessage() {}

// InstanceArr 服务实例数组定义
type ServiceAliasArr []*api.ServiceAlias

// Reset
func (m *ServiceAliasArr) Reset() { *m = ServiceAliasArr{} }

// String
func (m *ServiceAliasArr) String() string { return proto.CompactTextString(m) }

// ProtoMessage
func (*ServiceAliasArr) ProtoMessage() {}
