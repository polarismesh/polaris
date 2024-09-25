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

package auth

import (
	apisecurity "github.com/polarismesh/specification/source/go/api/v1/security"
)

type ServerFunctionName string

// SDK 接口
const (
	// 注册发现接口
	RegisterInstance      ServerFunctionName = "RegisterInstance"
	DeregisterInstance    ServerFunctionName = "DeregisterInstance"
	ReportServiceContract ServerFunctionName = "ReportServiceContract"
	DiscoverServices      ServerFunctionName = "DiscoverServices"
	DiscoverInstances     ServerFunctionName = "DiscoverInstances"
	UpdateInstance        ServerFunctionName = "UpdateInstance"

	// 服务治理接口
	DiscoverRouterRule         ServerFunctionName = "DiscoverRouterRule"
	DiscoverRateLimitRule      ServerFunctionName = "DiscoverRateLimitRule"
	DiscoverCircuitBreakerRule ServerFunctionName = "DiscoverCircuitBreakerRule"
	DiscoverFaultDetectRule    ServerFunctionName = "DiscoverFaultDetectRule"
	DiscoverServiceContract    ServerFunctionName = "DiscoverServiceContract"
	DiscoverLaneRule           ServerFunctionName = "DiscoverLaneRule"

	// 配置接口
	DiscoverConfigFile      ServerFunctionName = "DiscoverConfigFile"
	WatchConfigFile         ServerFunctionName = "WatchConfigFile"
	DiscoverConfigFileNames ServerFunctionName = "DiscoverConfigFileNames"
	DiscoverConfigGroups    ServerFunctionName = "DiscoverConfigGroups"
)

// 命名空间
const (
	CreateNamespace        ServerFunctionName = "CreateNamespace"
	CreateNamespaces       ServerFunctionName = "CreateNamespaces"
	DeleteNamespace        ServerFunctionName = "DeleteNamespace"
	DeleteNamespaces       ServerFunctionName = "DeleteNamespaces"
	UpdateNamespaces       ServerFunctionName = "UpdateNamespaces"
	UpdateNamespaceToken   ServerFunctionName = "UpdateNamespaceToken"
	DescribeNamespaces     ServerFunctionName = "DescribeNamespaces"
	DescribeNamespaceToken ServerFunctionName = "DescribeNamespaceToken"
)

// 服务/服务别名
const (
	CreateServices        ServerFunctionName = "CreateServices"
	DeleteServices        ServerFunctionName = "DeleteServices"
	UpdateServices        ServerFunctionName = "UpdateServices"
	UpdateServiceToken    ServerFunctionName = "UpdateServiceToken"
	DescribeAllServices   ServerFunctionName = "DescribeAllServices"
	DescribeServices      ServerFunctionName = "DescribeServices"
	DescribeServicesCount ServerFunctionName = "DescribeServicesCount"
	DescribeServiceToken  ServerFunctionName = "DescribeServiceToken"
	DescribeServiceOwner  ServerFunctionName = "DescribeServiceOwner"

	CreateServiceAlias     ServerFunctionName = "CreateServiceAlias"
	DeleteServiceAliases   ServerFunctionName = "DeleteServiceAliases"
	UpdateServiceAlias     ServerFunctionName = "UpdateServiceAlias"
	DescribeServiceAliases ServerFunctionName = "DescribeServiceAliases"
)

// 服务接口定义
const (
	CreateServiceContracts          ServerFunctionName = "CreateServiceContracts"
	DescribeServiceContracts        ServerFunctionName = "DescribeServiceContracts"
	DescribeServiceContractVersions ServerFunctionName = "DescribeServiceContractVersions"
	DeleteServiceContracts          ServerFunctionName = "DeleteServiceContracts"

	CreateServiceContractInterfaces ServerFunctionName = "CreateServiceContractInterfaces"
	AppendServiceContractInterfaces ServerFunctionName = "AppendServiceContractInterfaces"
	DeleteServiceContractInterfaces ServerFunctionName = "DeleteServiceContractInterfaces"
)

// 服务实例
const (
	CreateInstances               ServerFunctionName = "CreateInstances"
	DeleteInstances               ServerFunctionName = "DeleteInstances"
	DeleteInstancesByHost         ServerFunctionName = "DeleteInstancesByHost"
	UpdateInstances               ServerFunctionName = "UpdateInstances"
	UpdateInstancesIsolate        ServerFunctionName = "UpdateInstancesIsolate"
	DescribeInstances             ServerFunctionName = "DescribeInstances"
	DescribeInstancesCount        ServerFunctionName = "DescribeInstancesCount"
	DescribeInstanceLabels        ServerFunctionName = "DescribeInstanceLabels"
	CleanInstance                 ServerFunctionName = "CleanInstance"
	BatchCleanInstances           ServerFunctionName = "BatchCleanInstances"
	DescribeInstanceLastHeartbeat ServerFunctionName = "DescribeInstanceLastHeartbeat"
)

// 配置
const (
	// 配置分组
	CreateConfigFileGroup    ServerFunctionName = "CreateConfigFileGroup"
	DeleteConfigFileGroup    ServerFunctionName = "DeleteConfigFileGroup"
	UpdateConfigFileGroup    ServerFunctionName = "UpdateConfigFileGroup"
	DescribeConfigFileGroups ServerFunctionName = "DescribeConfigFileGroups"

	// 配置文件
	PublishConfigFile          ServerFunctionName = "PublishConfigFile"
	CreateConfigFile           ServerFunctionName = "CreateConfigFile"
	UpdateConfigFile           ServerFunctionName = "UpdateConfigFile"
	DeleteConfigFile           ServerFunctionName = "DeleteConfigFile"
	DescribeConfigFileRichInfo ServerFunctionName = "DescribeConfigFileRichInfo"
	DescribeConfigFiles        ServerFunctionName = "DescribeConfigFiles"
	BatchDeleteConfigFiles     ServerFunctionName = "BatchDeleteConfigFiles"
	ExportConfigFiles          ServerFunctionName = "ExportConfigFiles"
	ImportConfigFiles          ServerFunctionName = "ImportConfigFiles"

	// 配置发布历史
	DescribeConfigFileReleaseHistories ServerFunctionName = "DescribeConfigFileReleaseHistories"

	// 配置发布
	RollbackConfigFileReleases        ServerFunctionName = "RollbackConfigFileReleases"
	DeleteConfigFileReleases          ServerFunctionName = "DeleteConfigFileReleases"
	StopGrayConfigFileReleases        ServerFunctionName = "StopGrayConfigFileReleases"
	DescribeConfigFileRelease         ServerFunctionName = "DescribeConfigFileRelease"
	DescribeConfigFileReleases        ServerFunctionName = "DescribeConfigFileReleases"
	DescribeConfigFileReleaseVersions ServerFunctionName = "DescribeConfigFileReleaseVersions"
	UpsertAndReleaseConfigFile        ServerFunctionName = "UpsertAndReleaseConfigFile"

	// 配置模板
	DescribeAllConfigFileTemplates ServerFunctionName = "DescribeAllConfigFileTemplates"
	DescribeConfigFileTemplate     ServerFunctionName = "DescribeConfigFileTemplate"
	CreateConfigFileTemplate       ServerFunctionName = "CreateConfigFileTemplate"
)

// 路由
const (
	CreateRouteRules   ServerFunctionName = "CreateRouteRules"
	DeleteRouteRules   ServerFunctionName = "DeleteRouteRules"
	UpdateRouteRules   ServerFunctionName = "UpdateRouteRules"
	EnableRouteRules   ServerFunctionName = "EnableRouteRules"
	DescribeRouteRules ServerFunctionName = "DescribeRouteRules"
)

// 限流
const (
	CreateRateLimitRules   ServerFunctionName = "CreateRateLimitRules"
	DeleteRateLimitRules   ServerFunctionName = "DeleteRateLimitRules"
	UpdateRateLimitRules   ServerFunctionName = "UpdateRateLimitRules"
	EnableRateLimitRules   ServerFunctionName = "EnableRateLimitRules"
	DescribeRateLimitRules ServerFunctionName = "DescribeRateLimitRules"
)

// 熔断
const (
	CreateCircuitBreakerRules   ServerFunctionName = "CreateCircuitBreakerRules"
	DeleteCircuitBreakerRules   ServerFunctionName = "DeleteCircuitBreakerRules"
	EnableCircuitBreakerRules   ServerFunctionName = "EnableCircuitBreakerRules"
	UpdateCircuitBreakerRules   ServerFunctionName = "UpdateCircuitBreakerRules"
	DescribeCircuitBreakerRules ServerFunctionName = "DescribeCircuitBreakerRules"
)

// 主动探测
const (
	CreateFaultDetectRules   ServerFunctionName = "CreateFaultDetectRules"
	DeleteFaultDetectRules   ServerFunctionName = "DeleteFaultDetectRules"
	EnableFaultDetectRules   ServerFunctionName = "EnableFaultDetectRules"
	UpdateFaultDetectRules   ServerFunctionName = "UpdateFaultDetectRules"
	DescribeFaultDetectRules ServerFunctionName = "DescribeFaultDetectRules"
)

// 全链路灰度
const ()

// 用户/用户组
const (
	// 用户
	CreateUsers        ServerFunctionName = "CreateUsers"
	DeleteUsers        ServerFunctionName = "DeleteUsers"
	DescribeUsers      ServerFunctionName = "DescribeUsers"
	DescribeUserToken  ServerFunctionName = "DescribeUserToken"
	EnableUserToken    ServerFunctionName = "EnableUserToken"
	ResetUserToken     ServerFunctionName = "ResetUserToken"
	UpdateUser         ServerFunctionName = "UpdateUser"
	UpdateUserPassword ServerFunctionName = "UpdateUserPassword"

	// 用户组
	CreateUserGroup         ServerFunctionName = "CreateUserGroup"
	UpdateUserGroups        ServerFunctionName = "UpdateUserGroups"
	DeleteUserGroups        ServerFunctionName = "DeleteUserGroups"
	DescribeUserGroups      ServerFunctionName = "DescribeUserGroups"
	DescribeUserGroupDetail ServerFunctionName = "DescribeUserGroupDetail"
	DescribeUserGroupToken  ServerFunctionName = "DescribeUserGroupToken"
	EnableUserGroupToken    ServerFunctionName = "EnableUserGroupToken"
	ResetUserGroupToken     ServerFunctionName = "ResetUserGroupToken"
)

// 策略/角色
const (
	// 策略
	CreateAuthPolicy           ServerFunctionName = "CreateAuthPolicy"
	UpdateAuthPolicies         ServerFunctionName = "UpdateAuthPolicies"
	DeleteAuthPolicies         ServerFunctionName = "DeleteAuthPolicies"
	DescribeAuthPolicies       ServerFunctionName = "DescribeAuthPolicies"
	DescribeAuthPolicyDetail   ServerFunctionName = "DescribeAuthPolicyDetail"
	DescribePrincipalResources ServerFunctionName = "DescribePrincipalResources"

	// 角色
	CreateAuthRoles        ServerFunctionName = "CreateAuthRoles"
	UpdateAuthRoles        ServerFunctionName = "UpdateAuthRoles"
	DeleteAuthRoles        ServerFunctionName = "DeleteAuthRoles"
	DescribeAuthRoles      ServerFunctionName = "DescribeAuthRoles"
	DescribeAuthRoleDetail ServerFunctionName = "DescribeAuthRoleDetail"
)

// 运维接口
const (
	DescribeServerConnections ServerFunctionName = "DescribeServerConnections"
	DescribeServerConnStats   ServerFunctionName = "DescribeServerConnStats"
	CloseConnections          ServerFunctionName = "CloseConnections"
	FreeOSMemory              ServerFunctionName = "FreeOSMemory"
	DescribeLeaderElections   ServerFunctionName = "DescribeLeaderElections"
	ReleaseLeaderElection     ServerFunctionName = "ReleaseLeaderElection"
	DescribeGetLogOutputLevel ServerFunctionName = "DescribeGetLogOutputLevel"
	UpdateLogOutputLevel      ServerFunctionName = "UpdateLogOutputLevel"
	DescribeCMDBInfo          ServerFunctionName = "DescribeCMDBInfo"
)

var (
	SearchTypeMapping = map[string]apisecurity.ResourceType{
		"0": apisecurity.ResourceType_Namespaces,
		"1": apisecurity.ResourceType_Services,
		"2": apisecurity.ResourceType_ConfigGroups,
	}
)
