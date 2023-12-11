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

import apimodel "github.com/polarismesh/specification/source/go/api/v1/model"

// 北极星错误码
// 六位构成，前面三位参照HTTP Status的标准
// 后面三位，依据内部的具体错误自定义
const (
	ExecuteSuccess                  = uint32(apimodel.Code_ExecuteSuccess)
	DataNoChange                    = uint32(apimodel.Code_DataNoChange)
	NoNeedUpdate                    = uint32(apimodel.Code_NoNeedUpdate)
	BadRequest                      = uint32(apimodel.Code_BadRequest)
	ParseException                  = uint32(apimodel.Code_ParseException)
	EmptyRequest                    = uint32(apimodel.Code_EmptyRequest)
	BatchSizeOverLimit              = uint32(apimodel.Code_BatchSizeOverLimit)
	InvalidDiscoverResource         = uint32(apimodel.Code_InvalidDiscoverResource)
	InvalidRequestID                = uint32(apimodel.Code_InvalidRequestID)
	InvalidUserName                 = uint32(apimodel.Code_InvalidUserName)
	InvalidUserToken                = uint32(apimodel.Code_InvalidUserToken)
	InvalidParameter                = uint32(apimodel.Code_InvalidParameter)
	EmptyQueryParameter             = uint32(apimodel.Code_EmptyQueryParameter)
	InvalidQueryInsParameter        = uint32(apimodel.Code_InvalidQueryInsParameter)
	InvalidNamespaceName            = uint32(apimodel.Code_InvalidNamespaceName)
	InvalidNamespaceOwners          = uint32(apimodel.Code_InvalidNamespaceOwners)
	InvalidNamespaceToken           = uint32(apimodel.Code_InvalidNamespaceToken)
	InvalidServiceName              = uint32(apimodel.Code_InvalidServiceName)
	InvalidServiceOwners            = uint32(apimodel.Code_InvalidServiceOwners)
	InvalidServiceToken             = uint32(apimodel.Code_InvalidServiceToken)
	InvalidServiceMetadata          = uint32(apimodel.Code_InvalidServiceMetadata)
	InvalidServicePorts             = uint32(apimodel.Code_InvalidServicePorts)
	InvalidServiceBusiness          = uint32(apimodel.Code_InvalidServiceBusiness)
	InvalidServiceDepartment        = uint32(apimodel.Code_InvalidServiceDepartment)
	InvalidServiceCMDB              = uint32(apimodel.Code_InvalidServiceCMDB)
	InvalidServiceComment           = uint32(apimodel.Code_InvalidServiceComment)
	InvalidServiceAliasComment      = uint32(apimodel.Code_InvalidServiceAliasComment)
	InvalidInstanceID               = uint32(apimodel.Code_InvalidInstanceID)
	InvalidInstanceHost             = uint32(apimodel.Code_InvalidInstanceHost)
	InvalidInstancePort             = uint32(apimodel.Code_InvalidInstancePort)
	InvalidServiceAlias             = uint32(apimodel.Code_InvalidServiceAlias)
	InvalidNamespaceWithAlias       = uint32(apimodel.Code_InvalidNamespaceWithAlias)
	InvalidServiceAliasOwners       = uint32(apimodel.Code_InvalidServiceAliasOwners)
	InvalidInstanceProtocol         = uint32(apimodel.Code_InvalidInstanceProtocol)
	InvalidInstanceVersion          = uint32(apimodel.Code_InvalidInstanceVersion)
	InvalidInstanceLogicSet         = uint32(apimodel.Code_InvalidInstanceLogicSet)
	InvalidInstanceIsolate          = uint32(apimodel.Code_InvalidInstanceIsolate)
	HealthCheckNotOpen              = uint32(apimodel.Code_HealthCheckNotOpen)
	HeartbeatOnDisabledIns          = uint32(apimodel.Code_HeartbeatOnDisabledIns)
	HeartbeatExceedLimit            = uint32(apimodel.Code_HeartbeatExceedLimit)
	HeartbeatTypeNotFound           = uint32(apimodel.Code_HeartbeatTypeNotFound)
	InvalidMetadata                 = uint32(apimodel.Code_InvalidMetadata)
	InvalidRateLimitID              = uint32(apimodel.Code_InvalidRateLimitID)
	InvalidRateLimitLabels          = uint32(apimodel.Code_InvalidRateLimitLabels)
	InvalidRateLimitAmounts         = uint32(apimodel.Code_InvalidRateLimitAmounts)
	InvalidRateLimitName            = uint32(apimodel.Code_InvalidRateLimitName)
	InvalidCircuitBreakerID         = uint32(apimodel.Code_InvalidCircuitBreakerID)
	InvalidCircuitBreakerVersion    = uint32(apimodel.Code_InvalidCircuitBreakerVersion)
	InvalidCircuitBreakerName       = uint32(apimodel.Code_InvalidCircuitBreakerName)
	InvalidCircuitBreakerNamespace  = uint32(apimodel.Code_InvalidCircuitBreakerNamespace)
	InvalidCircuitBreakerOwners     = uint32(apimodel.Code_InvalidCircuitBreakerOwners)
	InvalidCircuitBreakerToken      = uint32(apimodel.Code_InvalidCircuitBreakerToken)
	InvalidCircuitBreakerBusiness   = uint32(apimodel.Code_InvalidCircuitBreakerBusiness)
	InvalidCircuitBreakerDepartment = uint32(apimodel.Code_InvalidCircuitBreakerDepartment)
	InvalidCircuitBreakerComment    = uint32(apimodel.Code_InvalidCircuitBreakerComment)
	InvalidRoutingID                = uint32(apimodel.Code_InvalidRoutingID)
	InvalidRoutingPolicy            = uint32(apimodel.Code_InvalidRoutingPolicy)
	InvalidRoutingName              = uint32(apimodel.Code_InvalidRoutingName)
	InvalidRoutingPriority          = uint32(apimodel.Code_InvalidRoutingPriority)

	// 网格相关错误码
	ServicesExistedMesh  = uint32(apimodel.Code_ServicesExistedMesh)
	ResourcesExistedMesh = uint32(apimodel.Code_ResourcesExistedMesh)
	InvalidMeshParameter = uint32(apimodel.Code_InvalidMeshParameter)

	// 平台信息相关错误码
	InvalidPlatformID         = uint32(apimodel.Code_InvalidPlatformID)
	InvalidPlatformName       = uint32(apimodel.Code_InvalidPlatformName)
	InvalidPlatformDomain     = uint32(apimodel.Code_InvalidPlatformDomain)
	InvalidPlatformQPS        = uint32(apimodel.Code_InvalidPlatformQPS)
	InvalidPlatformToken      = uint32(apimodel.Code_InvalidPlatformToken)
	InvalidPlatformOwner      = uint32(apimodel.Code_InvalidPlatformOwner)
	InvalidPlatformDepartment = uint32(apimodel.Code_InvalidPlatformDepartment)
	InvalidPlatformComment    = uint32(apimodel.Code_InvalidPlatformComment)
	NotFoundPlatform          = uint32(apimodel.Code_NotFoundPlatform)

	// flux相关错误码
	InvalidFluxRateLimitId     = uint32(apimodel.Code_InvalidFluxRateLimitId)
	InvalidFluxRateLimitQps    = uint32(apimodel.Code_InvalidFluxRateLimitQps)
	InvalidFluxRateLimitSetKey = uint32(apimodel.Code_InvalidFluxRateLimitSetKey)

	ExistedResource                 = uint32(apimodel.Code_ExistedResource)
	NotFoundResource                = uint32(apimodel.Code_NotFoundResource)
	NamespaceExistedServices        = uint32(apimodel.Code_NamespaceExistedServices)
	ServiceExistedInstances         = uint32(apimodel.Code_ServiceExistedInstances)
	ServiceExistedRoutings          = uint32(apimodel.Code_ServiceExistedRoutings)
	ServiceExistedRateLimits        = uint32(apimodel.Code_ServiceExistedRateLimits)
	ExistReleasedConfig             = uint32(apimodel.Code_ExistReleasedConfig)
	SameInstanceRequest             = uint32(apimodel.Code_SameInstanceRequest)
	ServiceExistedCircuitBreakers   = uint32(apimodel.Code_ServiceExistedCircuitBreakers)
	ServiceExistedAlias             = uint32(apimodel.Code_ServiceExistedAlias)
	NamespaceExistedMeshResources   = uint32(apimodel.Code_NamespaceExistedMeshResources)
	NamespaceExistedCircuitBreakers = uint32(apimodel.Code_NamespaceExistedCircuitBreakers)
	ServiceSubscribedByMeshes       = uint32(apimodel.Code_ServiceSubscribedByMeshes)
	ServiceExistedFluxRateLimits    = uint32(apimodel.Code_ServiceExistedFluxRateLimits)
	NamespaceExistedConfigGroups    = uint32(apimodel.Code_NamespaceExistedConfigGroups)

	NotFoundService                    = uint32(apimodel.Code_NotFoundService)
	NotFoundRouting                    = uint32(apimodel.Code_NotFoundRouting)
	NotFoundInstance                   = uint32(apimodel.Code_NotFoundInstance)
	NotFoundServiceAlias               = uint32(apimodel.Code_NotFoundServiceAlias)
	NotFoundNamespace                  = uint32(apimodel.Code_NotFoundNamespace)
	NotFoundSourceService              = uint32(apimodel.Code_NotFoundSourceService)
	NotFoundRateLimit                  = uint32(apimodel.Code_NotFoundRateLimit)
	NotFoundCircuitBreaker             = uint32(apimodel.Code_NotFoundCircuitBreaker)
	NotFoundMasterConfig               = uint32(apimodel.Code_NotFoundMasterConfig)
	NotFoundTagConfig                  = uint32(apimodel.Code_NotFoundTagConfig)
	NotFoundTagConfigOrService         = uint32(apimodel.Code_NotFoundTagConfigOrService)
	ClientAPINotOpen                   = uint32(apimodel.Code_ClientAPINotOpen)
	NotAllowBusinessService            = uint32(apimodel.Code_NotAllowBusinessService)
	NotAllowAliasUpdate                = uint32(apimodel.Code_NotAllowAliasUpdate)
	NotAllowAliasCreateInstance        = uint32(apimodel.Code_NotAllowAliasCreateInstance)
	NotAllowAliasCreateRouting         = uint32(apimodel.Code_NotAllowAliasCreateRouting)
	NotAllowCreateAliasForAlias        = uint32(apimodel.Code_NotAllowCreateAliasForAlias)
	NotAllowAliasCreateRateLimit       = uint32(apimodel.Code_NotAllowAliasCreateRateLimit)
	NotAllowAliasBindRule              = uint32(apimodel.Code_NotAllowAliasBindRule)
	NotAllowDifferentNamespaceBindRule = uint32(apimodel.Code_NotAllowDifferentNamespaceBindRule)
	Unauthorized                       = uint32(apimodel.Code_Unauthorized)
	NotAllowedAccess                   = uint32(apimodel.Code_NotAllowedAccess)
	IPRateLimit                        = uint32(apimodel.Code_IPRateLimit)
	APIRateLimit                       = uint32(apimodel.Code_APIRateLimit)
	CMDBNotFindHost                    = uint32(apimodel.Code_CMDBNotFindHost)
	DataConflict                       = uint32(apimodel.Code_DataConflict)
	InstanceTooManyRequests            = uint32(apimodel.Code_InstanceTooManyRequests)
	ExecuteException                   = uint32(apimodel.Code_ExecuteException)
	StoreLayerException                = uint32(apimodel.Code_StoreLayerException)
	CMDBPluginException                = uint32(apimodel.Code_CMDBPluginException)
	ParseRoutingException              = uint32(apimodel.Code_ParseRoutingException)
	ParseRateLimitException            = uint32(apimodel.Code_ParseRateLimitException)
	ParseCircuitBreakerException       = uint32(apimodel.Code_ParseCircuitBreakerException)
	HeartbeatException                 = uint32(apimodel.Code_HeartbeatException)
	InstanceRegisTimeout               = uint32(apimodel.Code_InstanceRegisTimeout)

	// 配置中心模块的错误码

	InvalidConfigFileGroupName     = uint32(apimodel.Code_InvalidConfigFileGroupName)
	InvalidConfigFileName          = uint32(apimodel.Code_InvalidConfigFileName)
	InvalidConfigFileContentLength = uint32(apimodel.Code_InvalidConfigFileContentLength)
	InvalidConfigFileFormat        = uint32(apimodel.Code_InvalidConfigFileFormat)
	InvalidConfigFileTags          = uint32(apimodel.Code_InvalidConfigFileTags)
	InvalidWatchConfigFileFormat   = uint32(apimodel.Code_InvalidWatchConfigFileFormat)
	NotFoundResourceConfigFile     = uint32(apimodel.Code_NotFoundResourceConfigFile)
	InvalidConfigFileTemplateName  = uint32(apimodel.Code_InvalidConfigFileTemplateName)
	InvalidMatchRule               = uint32(apimodel.Code_InvalidMatchRule)

	// 鉴权相关错误码
	InvalidUserOwners         = uint32(apimodel.Code_InvalidUserOwners)
	InvalidUserID             = uint32(apimodel.Code_InvalidUserID)
	InvalidUserPassword       = uint32(apimodel.Code_InvalidUserPassword)
	InvalidUserMobile         = uint32(apimodel.Code_InvalidUserMobile)
	InvalidUserEmail          = uint32(apimodel.Code_InvalidUserEmail)
	InvalidUserGroupOwners    = uint32(apimodel.Code_InvalidUserGroupOwners)
	InvalidUserGroupID        = uint32(apimodel.Code_InvalidUserGroupID)
	InvalidAuthStrategyOwners = uint32(apimodel.Code_InvalidAuthStrategyOwners)
	InvalidAuthStrategyName   = uint32(apimodel.Code_InvalidAuthStrategyName)
	InvalidAuthStrategyID     = uint32(apimodel.Code_InvalidAuthStrategyID)
	InvalidPrincipalType      = uint32(apimodel.Code_InvalidPrincipalType)

	UserExisted                            = uint32(apimodel.Code_UserExisted)
	UserGroupExisted                       = uint32(apimodel.Code_UserGroupExisted)
	AuthStrategyRuleExisted                = uint32(apimodel.Code_AuthStrategyRuleExisted)
	SubAccountExisted                      = uint32(apimodel.Code_SubAccountExisted)
	NotFoundUser                           = uint32(apimodel.Code_NotFoundUser)
	NotFoundOwnerUser                      = uint32(apimodel.Code_NotFoundOwnerUser)
	NotFoundUserGroup                      = uint32(apimodel.Code_NotFoundUserGroup)
	NotFoundAuthStrategyRule               = uint32(apimodel.Code_NotFoundAuthStrategyRule)
	NotAllowModifyDefaultStrategyPrincipal = uint32(apimodel.Code_NotAllowModifyDefaultStrategyPrincipal)
	NotAllowModifyOwnerDefaultStrategy     = uint32(apimodel.Code_NotAllowModifyOwnerDefaultStrategy)

	EmptyAutToken   = uint32(apimodel.Code_EmptyAutToken)
	TokenDisabled   = uint32(apimodel.Code_TokenDisabled)
	TokenNotExisted = uint32(apimodel.Code_TokenNotExisted)

	AuthTokenVerifyException = uint32(apimodel.Code_AuthTokenForbidden)
	OperationRoleException   = uint32(apimodel.Code_OperationRoleForbidden)
)

// code to string
// code的字符串描述信息
var code2info = map[uint32]string{
	ExecuteSuccess:                     "execute success",
	DataNoChange:                       "discover data is no change",
	NoNeedUpdate:                       "update data is no change, no need to update",
	BadRequest:                         "bad request",
	ParseException:                     "request decode failed",
	EmptyRequest:                       "empty request",
	BatchSizeOverLimit:                 "batch size over the limit",
	InvalidDiscoverResource:            "invalid discover resource",
	InvalidRequestID:                   "invalid request id",
	InvalidUserName:                    "invalid user name",
	InvalidUserToken:                   "invalid user token",
	InvalidParameter:                   "invalid parameter",
	EmptyQueryParameter:                "query instance parameter is empty",
	InvalidQueryInsParameter:           "query instance, service or namespace or host is required",
	InvalidNamespaceName:               "invalid namespace name",
	InvalidNamespaceOwners:             "invalid namespace owners",
	InvalidNamespaceToken:              "invalid namespace token",
	InvalidServiceName:                 "invalid service name",
	InvalidServiceOwners:               "invalid service owners",
	InvalidServiceToken:                "invalid service token",
	InvalidServiceMetadata:             "invalid service metadata",
	InvalidServicePorts:                "invalid service ports",
	InvalidServiceBusiness:             "invalid service business",
	InvalidServiceDepartment:           "invalid service department",
	InvalidServiceCMDB:                 "invalid service CMDB",
	InvalidServiceComment:              "invalid service comment",
	InvalidServiceAliasComment:         "invalid service alias comment",
	InvalidInstanceID:                  "invalid instance id",
	InvalidInstanceHost:                "invalid instance host",
	InvalidInstancePort:                "invalid instance port",
	InvalidInstanceProtocol:            "invalid instance protocol",
	InvalidInstanceVersion:             "invalid instance version",
	InvalidInstanceLogicSet:            "invalid instance logic set",
	InvalidInstanceIsolate:             "invalid instance isolate",
	InvalidServiceAlias:                "invalid service alias",
	InvalidNamespaceWithAlias:          "request namespace is not allow to create sid type alias",
	InvalidServiceAliasOwners:          "invalid service alias owners",
	HealthCheckNotOpen:                 "server not open health check",
	HeartbeatOnDisabledIns:             "heartbeat on disabled instance",
	HeartbeatExceedLimit:               "instance can only heartbeat 1 time per second",
	InvalidMetadata:                    "the length of metadata is too long or metadata contains invalid characters",
	InvalidRateLimitID:                 "invalid rate limit id",
	InvalidRateLimitLabels:             "invalid rate limit labels",
	InvalidRateLimitAmounts:            "invalid rate limit amounts",
	InvalidCircuitBreakerID:            "invalid circuit breaker id",
	InvalidCircuitBreakerVersion:       "invalid circuit breaker version",
	InvalidCircuitBreakerName:          "invalid circuit breaker name",
	InvalidCircuitBreakerNamespace:     "invalid circuit breaker namespace",
	InvalidCircuitBreakerOwners:        "invalid circuit breaker owners",
	InvalidCircuitBreakerToken:         "invalid circuit breaker token",
	InvalidCircuitBreakerBusiness:      "invalid circuit breaker business",
	InvalidCircuitBreakerDepartment:    "invalid circuit breaker department",
	InvalidCircuitBreakerComment:       "invalid circuit breaker comment",
	ExistedResource:                    "existed resource",
	SameInstanceRequest:                "the same instance request",
	NotFoundResource:                   "not found resource",
	ClientAPINotOpen:                   "client api is not open",
	NotAllowBusinessService:            "not allow requesting business service",
	NotAllowAliasUpdate:                "not allow service alias updating",
	NotAllowAliasCreateInstance:        "not allow service alias creating instance",
	NotAllowAliasCreateRouting:         "not allow service alias creating routing config",
	NotAllowCreateAliasForAlias:        "only source service can create alias",
	NotAllowAliasCreateRateLimit:       "not allow service alias creating rate limit",
	NotAllowAliasBindRule:              "not allow service alias binding rule",
	NotAllowDifferentNamespaceBindRule: "not allow different namespace binding rule",
	NamespaceExistedServices:           "some services existed in namespace",
	ServiceExistedInstances:            "some instances existed in service",
	ServiceExistedRoutings:             "some routings existed in service",
	ServiceExistedRateLimits:           "some rate limits existed in service",
	ServiceExistedCircuitBreakers:      "some circuit breakers existed in service",
	ServiceExistedAlias:                "some aliases existed in service",
	NamespaceExistedMeshResources:      "some mesh resources existed in namespace",
	NamespaceExistedCircuitBreakers:    "some circuit breakers existed in namespace",
	ExistReleasedConfig:                "exist released config",
	NotFoundService:                    "not found service",
	NotFoundRouting:                    "not found routing",
	NotFoundInstance:                   "not found instances",
	NotFoundServiceAlias:               "not found service alias",
	NotFoundNamespace:                  "not found namespace",
	NotFoundSourceService:              "not found the source service link with the alias",
	NotFoundRateLimit:                  "not found rate limit",
	NotFoundCircuitBreaker:             "not found circuit breaker",
	NotFoundTagConfig:                  "not found tag config",
	NotFoundMasterConfig:               "not found master config",
	NotFoundTagConfigOrService:         "not found tag config or service, or relation already exists",
	Unauthorized:                       "unauthorized",
	NotAllowedAccess:                   "access is not approved",
	IPRateLimit:                        "server limit the ip access",
	APIRateLimit:                       "server limit the api access",
	CMDBNotFindHost:                    "not found the host cmdb",
	DataConflict:                       "data is conflict, please try again",
	InstanceTooManyRequests:            "your instance has too many requests",
	ExecuteException:                   "execute exception",
	StoreLayerException:                "store layer exception",
	CMDBPluginException:                "cmdb plugin exception",
	ParseRoutingException:              "parsing routing failed",
	ParseRateLimitException:            "parse rate limit failed",
	ParseCircuitBreakerException:       "parse circuit breaker failed",
	HeartbeatException:                 "heartbeat execute exception",
	InvalidPlatformID:                  "invalid platform id",
	InvalidPlatformName:                "invalid platform name",
	InvalidPlatformDomain:              "invalid platform domain",
	InvalidPlatformQPS:                 "invalid platform qps",
	InvalidPlatformToken:               "invalid platform token",
	InvalidPlatformOwner:               "invalid platform owner",
	InvalidPlatformDepartment:          "invalid platform department",
	InvalidPlatformComment:             "invalid platform comment",
	NotFoundPlatform:                   "not found platform",
	ServicesExistedMesh:                "services existed mesh",
	ResourcesExistedMesh:               "resources existed mesh",
	ServiceSubscribedByMeshes:          "service subscribed by some mesh",
	InvalidMeshParameter:               "invalid mesh parameter",
	InvalidFluxRateLimitId:             "invalid flux ratelimit id",
	InvalidFluxRateLimitQps:            "invalid flux ratelimit qps",
	InvalidFluxRateLimitSetKey:         "invalid flux ratelimit key",
	InstanceRegisTimeout:               "instance async regist timeout",

	// 配置中心的错误信息
	InvalidConfigFileGroupName:     "invalid config file group name",
	InvalidConfigFileName:          "invalid config file name",
	InvalidConfigFileContentLength: "config file content too long",
	InvalidConfigFileFormat:        "invalid config file format, support json,xml,html,properties,text,yaml",
	InvalidConfigFileTags: "invalid config file tags, tags should be pair, like key1,value1,key2,value2, " +
		"both key and value should not blank",
	InvalidWatchConfigFileFormat:  "invalid watch config file format",
	NotFoundResourceConfigFile:    "config file not existed",
	InvalidConfigFileTemplateName: "invalid config file template name",
	InvalidMatchRule:              "invalid gray config beta labels",

	// 鉴权错误
	NotFoundUser:             "not found user",
	NotFoundOwnerUser:        "not found owner user",
	NotFoundUserGroup:        "not found usergroup",
	NotFoundAuthStrategyRule: "not found auth strategy rule",

	UserExisted:               "exist user",
	UserGroupExisted:          "exist usergroup",
	AuthStrategyRuleExisted:   "exist auth strategy rule",
	InvalidUserGroupOwners:    "invalid usergroup owner attribute",
	InvalidAuthStrategyName:   "invalid auth strategy rule name",
	InvalidAuthStrategyOwners: "invalid auth strategy rule owner",
	InvalidUserPassword:       "invalid user password",
	InvalidPrincipalType:      "invalid principal type",
	TokenDisabled:             "token already disabled",
	AuthTokenVerifyException:  "token verify exception",
	OperationRoleException:    "operation role exception",
	EmptyAutToken:             "auth token empty",
	SubAccountExisted:         "some sub-account existed in owner",
	InvalidUserID:             "invalid user-id",
	TokenNotExisted:           "token not existed",

	NotAllowModifyDefaultStrategyPrincipal: "not allow modify default strategy principal",
	NotAllowModifyOwnerDefaultStrategy:     "not allow modify main account default strategy",

	InvalidRoutingID:     "invalid routing id",
	InvalidRoutingPolicy: "invalid routing policy, only support (RulePolicy,MetadataPolicy)",
	InvalidRoutingName:   "invalid routing name",

	NamespaceExistedConfigGroups: "some config group existed in namespace",
}

// code to info
func Code2Info(code uint32) string {
	info, ok := code2info[code]
	if ok {
		return info
	}

	return ""
}
