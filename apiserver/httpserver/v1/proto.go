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
	apifault "github.com/polarismesh/specification/source/go/api/v1/fault_tolerance"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"
	apitraffic "github.com/polarismesh/specification/source/go/api/v1/traffic_manage"
)

// NamespaceArr 命名空间数组定义
type NamespaceArr []*apimodel.Namespace

// Reset 重置初始化
func (m *NamespaceArr) Reset() { *m = NamespaceArr{} }

// String return string
func (m *NamespaceArr) String() string { return proto.CompactTextString(m) }

// ProtoMessage return proto message
func (*NamespaceArr) ProtoMessage() {}

// ServiceArr 服务数组定义
type ServiceArr []*apiservice.Service

// Reset 重置初始化
func (m *ServiceArr) Reset() { *m = ServiceArr{} }

// String return string
func (m *ServiceArr) String() string { return proto.CompactTextString(m) }

// ProtoMessage return proto message
func (*ServiceArr) ProtoMessage() {}

// InstanceArr 服务实例数组定义
type InstanceArr []*apiservice.Instance

// Reset reset initialization
func (m *InstanceArr) Reset() { *m = InstanceArr{} }

// String
func (m *InstanceArr) String() string { return proto.CompactTextString(m) }

// ProtoMessage return proto message
func (*InstanceArr) ProtoMessage() {}

// RoutingArr 路由规则数组定义
type RoutingArr []*apitraffic.Routing

// Reset reset initialization
func (m *RoutingArr) Reset() { *m = RoutingArr{} }

// String return string
func (m *RoutingArr) String() string { return proto.CompactTextString(m) }

// ProtoMessage return proto message
func (*RoutingArr) ProtoMessage() {}

// RateLimitArr 限流规则数组定义
type RateLimitArr []*apitraffic.Rule

// Reset reset initialization
func (m *RateLimitArr) Reset() { *m = RateLimitArr{} }

// String
func (m *RateLimitArr) String() string { return proto.CompactTextString(m) }

// ProtoMessage return proto message
func (*RateLimitArr) ProtoMessage() {}

// CircuitBreakerArr 熔断规则数组定义
type CircuitBreakerArr []*apifault.CircuitBreaker

// Reset reset initialization
func (m *CircuitBreakerArr) Reset() { *m = CircuitBreakerArr{} }

// String
func (m *CircuitBreakerArr) String() string { return proto.CompactTextString(m) }

// ProtoMessage return proto message
func (*CircuitBreakerArr) ProtoMessage() {}

// ConfigReleaseArr 发布规则数组定义
type ConfigReleaseArr []*apiservice.ConfigRelease

// Reset reset initialization
func (m *ConfigReleaseArr) Reset() { *m = ConfigReleaseArr{} }

// String return string
func (m *ConfigReleaseArr) String() string { return proto.CompactTextString(m) }

// ProtoMessage return proto message
func (*ConfigReleaseArr) ProtoMessage() {}

// ServiceAliasArr 服务实例数组定义
type ServiceAliasArr []*apiservice.ServiceAlias

// Reset reset initialization
func (m *ServiceAliasArr) Reset() { *m = ServiceAliasArr{} }

// String return string
func (m *ServiceAliasArr) String() string { return proto.CompactTextString(m) }

// ProtoMessage proto message
func (*ServiceAliasArr) ProtoMessage() {}

// RouterArr 路由规则数组定义
type RouterArr []*apitraffic.RouteRule

// Reset reset initialization
func (m *RouterArr) Reset() { *m = RouterArr{} }

// String return string
func (m *RouterArr) String() string { return proto.CompactTextString(m) }

// ProtoMessage return proto message
func (*RouterArr) ProtoMessage() {}

// CircuitBreakerRuleAttr circuitbreaker rule array define
type CircuitBreakerRuleAttr []*apifault.CircuitBreakerRule

// Reset reset initialization
func (m *CircuitBreakerRuleAttr) Reset() { *m = CircuitBreakerRuleAttr{} }

// String return string
func (m *CircuitBreakerRuleAttr) String() string { return proto.CompactTextString(m) }

// ProtoMessage return proto message
func (*CircuitBreakerRuleAttr) ProtoMessage() {}

// FaultDetectRuleAttr fault detect rule array define
type FaultDetectRuleAttr []*apifault.FaultDetectRule

// Reset reset initialization
func (m *FaultDetectRuleAttr) Reset() { *m = FaultDetectRuleAttr{} }

// String return string
func (m *FaultDetectRuleAttr) String() string { return proto.CompactTextString(m) }

// ProtoMessage return proto message
func (*FaultDetectRuleAttr) ProtoMessage() {}
