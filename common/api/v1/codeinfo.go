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

// 北极星错误码
// 六位构成，前面三位参照HTTP Status的标准
// 后面三位，依据内部的具体错误自定义
const (
	ExecuteSuccess                  uint32 = 200000
	DataNoChange                           = 200001
	NoNeedUpdate                           = 200002
	BadRequest                             = 400000
	ParseException                         = 400001
	EmptyRequest                           = 400002
	BatchSizeOverLimit                     = 400003
	InvalidDiscoverResource                = 400004
	InvalidRequestID                       = 400100
	InvalidUserName                        = 400101
	InvalidUserToken                       = 400102
	InvalidParameter                       = 400103
	EmptyQueryParameter                    = 400104
	InvalidQueryInsParameter               = 400105
	InvalidNamespaceName                   = 400110
	InvalidNamespaceOwners                 = 400111
	InvalidNamespaceToken                  = 400112
	InvalidServiceName                     = 400120
	InvalidServiceOwners                   = 400121
	InvalidServiceToken                    = 400122
	InvalidServiceMetadata                 = 400123
	InvalidServicePorts                    = 400124
	InvalidServiceBusiness                 = 400125
	InvalidServiceDepartment               = 400126
	InvalidServiceCMDB                     = 400127
	InvalidServiceComment                  = 400128
	InvalidServiceAliasComment             = 400129
	InvalidInstanceID                      = 400130
	InvalidInstanceHost                    = 400131
	InvalidInstancePort                    = 400132
	InvalidServiceAlias                    = 400133
	InvalidNamespaceWithAlias              = 400134
	InvalidServiceAliasOwners              = 400135
	InvalidInstanceProtocol                = 400136
	InvalidInstanceVersion                 = 400137
	InvalidInstanceLogicSet                = 400138
	InvalidInstanceIsolate                 = 400139
	HealthCheckNotOpen                     = 400140
	HeartbeatOnDisabledIns                 = 400141
	HeartbeatExceedLimit                   = 400142
	HeartbeatTypeNotFound                  = 400143
	InvalidMetadata                        = 400150
	InvalidRateLimitID                     = 400151
	InvalidRateLimitLabels                 = 400152
	InvalidRateLimitAmounts                = 400153
	InvalidCircuitBreakerID                = 400160
	InvalidCircuitBreakerVersion           = 400161
	InvalidCircuitBreakerName              = 400162
	InvalidCircuitBreakerNamespace         = 400163
	InvalidCircuitBreakerOwners            = 400164
	InvalidCircuitBreakerToken             = 400165
	InvalidCircuitBreakerBusiness          = 400166
	InvalidCircuitBreakerDepartment        = 400167
	InvalidCircuitBreakerComment           = 400168

	// 网格相关错误码
	ServicesExistedMesh  = 400170
	ResourcesExistedMesh = 400171
	InvalidMeshParameter = 400172

	// 平台信息相关错误码
	InvalidPlatformID         = 400180
	InvalidPlatformName       = 400181
	InvalidPlatformDomain     = 400182
	InvalidPlatformQPS        = 400183
	InvalidPlatformToken      = 400184
	InvalidPlatformOwner      = 400185
	InvalidPlatformDepartment = 400186
	InvalidPlatformComment    = 400187
	NotFoundPlatform          = 400188

	// flux相关错误码
	InvalidFluxRateLimitId     = 400190
	InvalidFluxRateLimitQps    = 400191
	InvalidFluxRateLimitSetKey = 400192

	ExistedResource                 = 400201
	NotFoundResource                = 400202
	NamespaceExistedServices        = 400203
	ServiceExistedInstances         = 400204
	ServiceExistedRoutings          = 400205
	ServiceExistedRateLimits        = 400206
	ExistReleasedConfig             = 400207
	SameInstanceRequest             = 400208
	ServiceExistedCircuitBreakers   = 400209
	ServiceExistedAlias             = 400210
	NamespaceExistedMeshResources   = 400211
	NamespaceExistedCircuitBreakers = 400212
	ServiceSubscribedByMeshes       = 400213
	ServiceExistedFluxRateLimits    = 400214

	NotFoundService                    = 400301
	NotFoundRouting                    = 400302
	NotFoundInstance                   = 400303
	NotFoundServiceAlias               = 400304
	NotFoundNamespace                  = 400305
	NotFoundSourceService              = 400306
	NotFoundRateLimit                  = 400307
	NotFoundCircuitBreaker             = 400308
	NotFoundMasterConfig               = 400309
	NotFoundTagConfig                  = 400310
	NotFoundTagConfigOrService         = 400311
	ClientAPINotOpen                   = 400401
	NotAllowBusinessService            = 400402
	NotAllowAliasUpdate                = 400501
	NotAllowAliasCreateInstance        = 400502
	NotAllowAliasCreateRouting         = 400503
	NotAllowCreateAliasForAlias        = 400504
	NotAllowAliasCreateRateLimit       = 400505
	NotAllowAliasBindRule              = 400506
	NotAllowDifferentNamespaceBindRule = 400507
	Unauthorized                       = 401000
	NotAllowedAccess                   = 401001
	IPRateLimit                        = 403001
	APIRateLimit                       = 403002
	CMDBNotFindHost                    = 404001
	DataConflict                       = 409000
	InstanceTooManyRequests            = 429001
	ExecuteException                   = 500000
	StoreLayerException                = 500001
	CMDBPluginException                = 500002
	ParseRoutingException              = 500004
	ParseRateLimitException            = 500005
	ParseCircuitBreakerException       = 500006
	HeartbeatException                 = 500007

	// 配置中心模块的错误码

	InvalidConfigFileGroupName     = 400801
	InvalidConfigFileName          = 400802
	InvalidConfigFileContentLength = 400803
	InvalidConfigFileFormat        = 400804
	InvalidConfigFileTags          = 400805
	InvalidWatchConfigFileFormat   = 400806
	NotFoundResourceConfigFile     = 400807

	// 鉴权相关错误码
	InvalidUserOwners         = 400410
	InvalidUserID             = 400411
	InvalidUserPassword       = 400412
	InvalidUserMobile         = 400413
	InvalidUserEmail          = 400414
	InvalidUserGroupOwners    = 400420
	InvalidUserGroupID        = 400421
	InvalidAuthStrategyOwners = 400430
	InvalidAuthStrategyName   = 400431
	InvalidAuthStrategyID     = 400432
	InvalidPrincipalType      = 400440

	UserExisted                            = 400215
	UserGroupExisted                       = 400216
	AuthStrategyRuleExisted                = 400217
	SubAccountExisted                      = 400218
	NotFoundUser                           = 400312
	NotFoundOwnerUser                      = 400313
	NotFoundUserGroup                      = 400314
	NotFoundAuthStrategyRule               = 400315
	NotAllowModifyDefaultStrategyPrincipal = 400508

	EmptyAutToken   = 401002
	TokenDisabled   = 401003
	TokenNotExisted = 401004

	AuthTokenVerifyException = 500100
	OperationRoleException   = 500101
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
	InvalidQueryInsParameter:           "query instance, (service,namespace) or host is required",
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
	// 配置中心的错误信息
	InvalidConfigFileGroupName:     "invalid config file group name",
	InvalidConfigFileName:          "invalid config file name",
	InvalidConfigFileContentLength: "config file content too long",
	InvalidConfigFileFormat:        "invalid config file format, support json,xml,html,properties,text,yaml",
	InvalidConfigFileTags:          "invalid config file tags, tags should be pair, like key1,value1,key2,value2. and key,value should not blank",
	InvalidWatchConfigFileFormat:   "invalid watch config file format",
	NotFoundResourceConfigFile:     "config file not existed",

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
}

// code to info
func Code2Info(code uint32) string {
	info, ok := code2info[code]
	if ok {
		return info
	}

	return ""
}
