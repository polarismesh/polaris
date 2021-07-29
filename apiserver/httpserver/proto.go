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
	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/golang/protobuf/proto"
)

/**
 * @brief 命名空间数组定义
 */
type NamespaceArr []*api.Namespace

//
func (m *NamespaceArr) Reset() { *m = NamespaceArr{} }

//
func (m *NamespaceArr) String() string { return proto.CompactTextString(m) }

//
func (*NamespaceArr) ProtoMessage() {}

/**
 * @brief 服务数组定义
 */
type ServiceArr []*api.Service

//
func (m *ServiceArr) Reset() { *m = ServiceArr{} }

//
func (m *ServiceArr) String() string { return proto.CompactTextString(m) }

//
func (*ServiceArr) ProtoMessage() {}

/**
 * @brief 服务实例数组定义
 */
type InstanceArr []*api.Instance

//
func (m *InstanceArr) Reset() { *m = InstanceArr{} }

//
func (m *InstanceArr) String() string { return proto.CompactTextString(m) }

//
func (*InstanceArr) ProtoMessage() {}

/**
 * @brief 路由规则数组定义
 */
type RoutingArr []*api.Routing

//
func (m *RoutingArr) Reset() { *m = RoutingArr{} }

//
func (m *RoutingArr) String() string { return proto.CompactTextString(m) }

//
func (*RoutingArr) ProtoMessage() {}

/**
 * @brief 限流规则数组定义
 */
type RateLimitArr []*api.Rule

//
func (m *RateLimitArr) Reset() { *m = RateLimitArr{} }

//
func (m *RateLimitArr) String() string { return proto.CompactTextString(m) }

//
func (*RateLimitArr) ProtoMessage() {}

/**
 * @brief 熔断规则数组定义
 */
type CircuitBreakerArr []*api.CircuitBreaker

//
func (m *CircuitBreakerArr) Reset() { *m = CircuitBreakerArr{} }

//
func (m *CircuitBreakerArr) String() string { return proto.CompactTextString(m) }

//
func (*CircuitBreakerArr) ProtoMessage() {}

/**
 * @brief 发布规则数组定义
 */
type ConfigReleaseArr []*api.ConfigRelease

//
func (m *ConfigReleaseArr) Reset() { *m = ConfigReleaseArr{} }

//
func (m *ConfigReleaseArr) String() string { return proto.CompactTextString(m) }

//
func (*ConfigReleaseArr) ProtoMessage() {}

/*
 * @brief 平台数组定义
 */
type PlatformArr []*api.Platform

// proto reset
func (m *PlatformArr) Reset() { *m = PlatformArr{} }

// proto string
func (m *PlatformArr) String() string { return proto.CompactTextString(m) }

// proto message
func (m *PlatformArr) ProtoMessage() {}
