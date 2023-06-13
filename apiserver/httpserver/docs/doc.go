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
	// Client               *Client                         `protobuf:"bytes,3,opt,name=client,proto3" json:"client,omitempty"`
	// Namespace            *model.Namespace                `protobuf:"bytes,4,opt,name=namespace,proto3" json:"namespace,omitempty"`
	// Service              *Service                        `protobuf:"bytes,5,opt,name=service,proto3" json:"service,omitempty"`
	// Instance             *Instance                       `protobuf:"bytes,6,opt,name=instance,proto3" json:"instance,omitempty"`
	// Routing              *traffic_manage.Routing         `protobuf:"bytes,7,opt,name=routing,proto3" json:"routing,omitempty"`
	// Alias                *ServiceAlias                   `protobuf:"bytes,8,opt,name=alias,proto3" json:"alias,omitempty"`
	// RateLimit            *traffic_manage.Rule            `protobuf:"bytes,9,opt,name=rateLimit,proto3" json:"rateLimit,omitempty"`
	// CircuitBreaker       *fault_tolerance.CircuitBreaker `protobuf:"bytes,10,opt,name=circuitBreaker,proto3" json:"circuitBreaker,omitempty"`
	// ConfigRelease        *ConfigRelease                  `protobuf:"bytes,11,opt,name=configRelease,proto3" json:"configRelease,omitempty"`
	// User                 *security.User                  `protobuf:"bytes,19,opt,name=user,proto3" json:"user,omitempty"`
	// UserGroup            *security.UserGroup             `protobuf:"bytes,20,opt,name=userGroup,proto3" json:"userGroup,omitempty"`
	// AuthStrategy         *security.AuthStrategy          `protobuf:"bytes,21,opt,name=authStrategy,proto3" json:"authStrategy,omitempty"`
	// Relation             *security.UserGroupRelation     `protobuf:"bytes,22,opt,name=relation,proto3" json:"relation,omitempty"`
	// LoginResponse        *security.LoginResponse         `protobuf:"bytes,23,opt,name=loginResponse,proto3" json:"loginResponse,omitempty"`
	// ModifyAuthStrategy   *security.ModifyAuthStrategy    `protobuf:"bytes,24,opt,name=modifyAuthStrategy,proto3" json:"modifyAuthStrategy,omitempty"`
	// ModifyUserGroup      *security.ModifyUserGroup       `protobuf:"bytes,25,opt,name=modifyUserGroup,proto3" json:"modifyUserGroup,omitempty"`
	// Resources            *security.StrategyResources     `protobuf:"bytes,26,opt,name=resources,proto3" json:"resources,omitempty"`
	// OptionSwitch         *OptionSwitch                   `protobuf:"bytes,27,opt,name=optionSwitch,proto3" json:"optionSwitch,omitempty"`
	// InstanceLabels       *InstanceLabels                 `protobuf:"bytes,28,opt,name=instanceLabels,proto3" json:"instanceLabels,omitempty"`
	// Data                 *anypb.Any                      `protobuf:"bytes,29,opt,name=data,proto3" json:"data,omitempty"`
	// ConfigFileGroup          *ConfigFileGroup          `protobuf:"bytes,3,opt,name=configFileGroup,proto3" json:"configFileGroup,omitempty"`
	// ConfigFile               *ConfigFile               `protobuf:"bytes,4,opt,name=configFile,proto3" json:"configFile,omitempty"`
	// ConfigFileRelease        *ConfigFileRelease        `protobuf:"bytes,5,opt,name=configFileRelease,proto3" json:"configFileRelease,omitempty"`
	// ConfigFileReleaseHistory *ConfigFileReleaseHistory `protobuf:"bytes,6,opt,name=configFileReleaseHistory,proto3" json:"configFileReleaseHistory,omitempty"`
	// ConfigFileTemplate       *ConfigFileTemplate       `protobuf:"bytes,7,opt,name=configFileTemplate,proto3" json:"configFileTemplate,omitempty"`
}

type BatchQueryResponse struct {
	Code   *wrapperspb.UInt32Value `protobuf:"bytes,1,opt,name=code,proto3" json:"code,omitempty"`
	Info   *wrapperspb.StringValue `protobuf:"bytes,2,opt,name=info,proto3" json:"info,omitempty"`
	Amount *wrapperspb.UInt32Value `protobuf:"bytes,3,opt,name=amount,proto3" json:"amount,omitempty"`
	Size   *wrapperspb.UInt32Value `protobuf:"bytes,4,opt,name=size,proto3" json:"size,omitempty"`
	Total  *wrapperspb.UInt32Value `protobuf:"bytes,3,opt,name=total,proto3" json:"total,omitempty"`
	// Namespaces           []*model.Namespace        `protobuf:"bytes,5,rep,name=namespaces,proto3" json:"namespaces,omitempty"`
	// Services             []*Service                `protobuf:"bytes,6,rep,name=services,proto3" json:"services,omitempty"`
	// Instances            []*Instance               `protobuf:"bytes,7,rep,name=instances,proto3" json:"instances,omitempty"`
	// Routings             []*traffic_manage.Routing `protobuf:"bytes,8,rep,name=routings,proto3" json:"routings,omitempty"`
	// Aliases              []*ServiceAlias           `protobuf:"bytes,9,rep,name=aliases,proto3" json:"aliases,omitempty"`
	// RateLimits           []*traffic_manage.Rule    `protobuf:"bytes,10,rep,name=rateLimits,proto3" json:"rateLimits,omitempty"`
	// ConfigWithServices   []*ConfigWithService      `protobuf:"bytes,11,rep,name=configWithServices,proto3" json:"configWithServices,omitempty"`
	// Users                []*security.User          `protobuf:"bytes,18,rep,name=users,proto3" json:"users,omitempty"`
	// UserGroups           []*security.UserGroup     `protobuf:"bytes,19,rep,name=userGroups,proto3" json:"userGroups,omitempty"`
	// AuthStrategies       []*security.AuthStrategy  `protobuf:"bytes,20,rep,name=authStrategies,proto3" json:"authStrategies,omitempty"`
	// Clients              []*Client                 `protobuf:"bytes,21,rep,name=clients,proto3" json:"clients,omitempty"`
	// Data                 []*anypb.Any              `protobuf:"bytes,22,rep,name=data,proto3" json:"data,omitempty"`
	// Summary              *model.Summary            `protobuf:"bytes,23,opt,name=summary,proto3" json:"summary,omitempty"`
	// ConfigFileGroups           []*ConfigFileGroup          `protobuf:"bytes,4,rep,name=configFileGroups,proto3" json:"configFileGroups,omitempty"`
	// ConfigFiles                []*ConfigFile               `protobuf:"bytes,5,rep,name=configFiles,proto3" json:"configFiles,omitempty"`
	// ConfigFileReleases         []*ConfigFileRelease        `protobuf:"bytes,6,rep,name=configFileReleases,proto3" json:"configFileReleases,omitempty"`
	// ConfigFileReleaseHistories []*ConfigFileReleaseHistory `protobuf:"bytes,7,rep,name=configFileReleaseHistories,proto3" json:"configFileReleaseHistories,omitempty"`
	// ConfigFileTemplates        []*ConfigFileTemplate       `protobuf:"bytes,8,rep,name=configFileTemplates,proto3" json:"configFileTemplates,omitempty"`
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
	RoutingPolicy traffic_manage.RoutingPolicy `protobuf:"varint,5,opt,name=routing_policy,proto3,enum=v1.RoutingPolicy" json:"routing_policy,omitempty"`
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
	ExtendInfo map[string]string `protobuf:"bytes,20,rep,name=extendInfo,proto3" json:"extendInfo,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
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
