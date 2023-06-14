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

package docs

import (
	"github.com/polarismesh/specification/source/go/api/v1/traffic_manage"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type BaseResponse struct {
	Code *wrapperspb.UInt32Value `json:"code"`
	Info *wrapperspb.StringValue `json:"info"`
}

type BatchQueryResponse struct {
	Code   *wrapperspb.UInt32Value `protobuf:"bytes,1,opt,name=code,proto3" json:"code,omitempty"`
	Info   *wrapperspb.StringValue `protobuf:"bytes,2,opt,name=info,proto3" json:"info,omitempty"`
	Amount *wrapperspb.UInt32Value `protobuf:"bytes,3,opt,name=amount,proto3" json:"amount,omitempty"`
	Size   *wrapperspb.UInt32Value `protobuf:"bytes,4,opt,name=size,proto3" json:"size,omitempty"`
	Total  *wrapperspb.UInt32Value `protobuf:"bytes,3,opt,name=total,proto3" json:"total,omitempty"`
}

type BatchWriteResponse struct {
	Code *wrapperspb.UInt32Value `protobuf:"bytes,1,opt,name=code,proto3" json:"code,omitempty"`
	Info *wrapperspb.StringValue `protobuf:"bytes,2,opt,name=info,proto3" json:"info,omitempty"`
}

// configuration root for route
type RouteRule struct {
	Id string `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	// route rule name
	Name string `protobuf:"bytes,2,opt,name=name,proto3" json:"name,omitempty"`
	// namespace namingspace of routing rules
	Namespace string `protobuf:"bytes,3,opt,name=namespace,proto3" json:"namespace,omitempty"`
	// Enable this router
	Enable bool `protobuf:"varint,4,opt,name=enable,proto3" json:"enable,omitempty"`
	// Router type
	RoutingPolicy traffic_manage.RoutingPolicy `json:"routing_policy,omitempty"`
	// Routing configuration for router
	RoutingConfig RuleRoutingConfig `protobuf:"bytes,6,opt,name=routing_config,proto3" json:"routing_config,omitempty"`
	// revision routing version
	Revision string `protobuf:"bytes,7,opt,name=revision,proto3" json:"revision,omitempty"`
	// ctime create time of the rules
	Ctime string `protobuf:"bytes,8,opt,name=ctime,proto3" json:"ctime,omitempty"`
	// mtime modify time of the rules
	Mtime string `protobuf:"bytes,9,opt,name=mtime,proto3" json:"mtime,omitempty"`
	// etime enable time of the rules
	Etime string `protobuf:"bytes,10,opt,name=etime,proto3" json:"etime,omitempty"`
	// priority rules priority
	Priority uint32 `protobuf:"varint,11,opt,name=priority,proto3" json:"priority,omitempty"`
	// description simple description rules
	Description string `protobuf:"bytes,12,opt,name=description,proto3" json:"description,omitempty"`
	// extendInfo 用于承载一些额外信息
	// case 1: 升级到 v2 版本时，记录对应到 v1 版本的 id 信息
	ExtendInfo map[string]string `json:"extendInfo,omitempty"`
}

// RuleRoutingConfig routing configuration
type RuleRoutingConfig struct {
	// rule route chain
	Rules []traffic_manage.SubRuleRouting `json:"rules,omitempty"`
}

type SimpleService struct {
	Name      *wrapperspb.StringValue `json:"name,omitempty"`
	Namespace *wrapperspb.StringValue `json:"namespace,omitempty"`
}

// DiscoverRequest
// 0:  "UNKNOWN",
// 1:  "INSTANCE",
// 2:  "CLUSTER",
// 3:  "ROUTING",
// 4:  "RATE_LIMIT",
// 5:  "CIRCUIT_BREAKER",
// 6:  "SERVICES",
// 12: "NAMESPACES",
// 13: "FAULT_DETECTOR",
type DiscoverRequest struct {
	Type    string        `json:"type,omitempty"`
	Service SimpleService `json:"service,omitempty"`
}
