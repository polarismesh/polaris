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
	DataNoChange                    uint32 = 200001
	NoNeedUpdate                    uint32 = 200002
	BadRequest                      uint32 = 400000
	ParseException                  uint32 = 400001
	EmptyRequest                    uint32 = 400002
	BatchSizeOverLimit              uint32 = 400003
	InvalidDiscoverResource         uint32 = 400004
	InvalidRequestID                uint32 = 400100
	InvalidUserName                 uint32 = 400101
	InvalidUserToken                uint32 = 400102
	InvalidParameter                uint32 = 400103
	EmptyQueryParameter             uint32 = 400104
	InvalidQueryInsParameter        uint32 = 400105
	InvalidNamespaceName            uint32 = 400110
	InvalidNamespaceOwners          uint32 = 400111
	InvalidNamespaceToken           uint32 = 400112
	InvalidServiceName              uint32 = 400120
	InvalidServiceOwners            uint32 = 400121
	InvalidServiceToken             uint32 = 400122
	InvalidServiceMetadata          uint32 = 400123
	InvalidServicePorts             uint32 = 400124
	InvalidServiceBusiness          uint32 = 400125
	InvalidServiceDepartment        uint32 = 400126
	InvalidServiceCMDB              uint32 = 400127
	InvalidServiceComment           uint32 = 400128
	InvalidServiceAliasComment      uint32 = 400129
	InvalidInstanceID               uint32 = 400130
	InvalidInstanceHost             uint32 = 400131
	InvalidInstancePort             uint32 = 400132
	InvalidServiceAlias             uint32 = 400133
	InvalidNamespaceWithAlias       uint32 = 400134
	InvalidServiceAliasOwners       uint32 = 400135
	InvalidInstanceProtocol         uint32 = 400136
	InvalidInstanceVersion          uint32 = 400137
	InvalidInstanceLogicSet         uint32 = 400138
	InvalidInstanceIsolate          uint32 = 400139
	HealthCheckNotOpen              uint32 = 400140
	HeartbeatOnDisabledIns          uint32 = 400141
	HeartbeatExceedLimit            uint32 = 400142
	HeartbeatTypeNotFound           uint32 = 400143
	InvalidMetadata                 uint32 = 400150
	InvalidRateLimitID              uint32 = 400151
	InvalidRateLimitLabels          uint32 = 400152
	InvalidRateLimitAmounts         uint32 = 400153
	InvalidRateLimitName            uint32 = 400154
	InvalidCircuitBreakerID         uint32 = 400160
	InvalidCircuitBreakerVersion    uint32 = 400161
	InvalidCircuitBreakerName       uint32 = 400162
	InvalidCircuitBreakerNamespace  uint32 = 400163
	InvalidCircuitBreakerOwners     uint32 = 400164
	InvalidCircuitBreakerToken      uint32 = 400165
	InvalidCircuitBreakerBusiness   uint32 = 400166
	InvalidCircuitBreakerDepartment uint32 = 400167
	InvalidCircuitBreakerComment    uint32 = 400168
	InvalidRoutingID                uint32 = 400700
	InvalidRoutingPolicy            uint32 = 400701
	InvalidRoutingName              uint32 = 400702
	InvalidRoutingPriority          uint32 = 400703

	// 网格相关错误码
	ServicesExistedMesh  uint32 = 400170
	ResourcesExistedMesh uint32 = 400171
	InvalidMeshParameter uint32 = 400172

	// 平台信息相关错误码
	InvalidPlatformID         uint32 = 400180
	InvalidPlatformName       uint32 = 400181
	InvalidPlatformDomain     uint32 = 400182
	InvalidPlatformQPS        uint32 = 400183
	InvalidPlatformToken      uint32 = 400184
	InvalidPlatformOwner      uint32 = 400185
	InvalidPlatformDepartment uint32 = 400186
	InvalidPlatformComment    uint32 = 400187
	NotFoundPlatform          uint32 = 400188

	// flux相关错误码
	InvalidFluxRateLimitId     uint32 = 400190
	InvalidFluxRateLimitQps    uint32 = 400191
	InvalidFluxRateLimitSetKey uint32 = 400192

	ExistedResource                 uint32 = 400201
	NotFoundResource                uint32 = 400202
	NamespaceExistedServices        uint32 = 400203
	ServiceExistedInstances         uint32 = 400204
	ServiceExistedRoutings          uint32 = 400205
	ServiceExistedRateLimits        uint32 = 400206
	ExistReleasedConfig             uint32 = 400207
	SameInstanceRequest             uint32 = 400208
	ServiceExistedCircuitBreakers   uint32 = 400209
	ServiceExistedAlias             uint32 = 400210
	NamespaceExistedMeshResources   uint32 = 400211
	NamespaceExistedCircuitBreakers uint32 = 400212
	ServiceSubscribedByMeshes       uint32 = 400213
	ServiceExistedFluxRateLimits    uint32 = 400214
	NamespaceExistedConfigGroups    uint32 = 400219

	NotFoundService                    uint32 = 400301
	NotFoundRouting                    uint32 = 400302
	NotFoundInstance                   uint32 = 400303
	NotFoundServiceAlias               uint32 = 400304
	NotFoundNamespace                  uint32 = 400305
	NotFoundSourceService              uint32 = 400306
	NotFoundRateLimit                  uint32 = 400307
	NotFoundCircuitBreaker             uint32 = 400308
	NotFoundMasterConfig               uint32 = 400309
	NotFoundTagConfig                  uint32 = 400310
	NotFoundTagConfigOrService         uint32 = 400311
	ClientAPINotOpen                   uint32 = 400401
	NotAllowBusinessService            uint32 = 400402
	NotAllowAliasUpdate                uint32 = 400501
	NotAllowAliasCreateInstance        uint32 = 400502
	NotAllowAliasCreateRouting         uint32 = 400503
	NotAllowCreateAliasForAlias        uint32 = 400504
	NotAllowAliasCreateRateLimit       uint32 = 400505
	NotAllowAliasBindRule              uint32 = 400506
	NotAllowDifferentNamespaceBindRule uint32 = 400507
	Unauthorized                       uint32 = 401000
	NotAllowedAccess                   uint32 = 401001
	IPRateLimit                        uint32 = 403001
	APIRateLimit                       uint32 = 403002
	CMDBNotFindHost                    uint32 = 404001
	DataConflict                       uint32 = 409000
	InstanceTooManyRequests            uint32 = 429001
	ExecuteException                   uint32 = 500000
	StoreLayerException                uint32 = 500001
	CMDBPluginException                uint32 = 500002
	ParseRoutingException              uint32 = 500004
	ParseRateLimitException            uint32 = 500005
	ParseCircuitBreakerException       uint32 = 500006
	HeartbeatException                 uint32 = 500007
	InstanceRegisTimeout               uint32 = 500008

	// 配置中心模块的错误码

	InvalidConfigFileGroupName     uint32 = 400801
	InvalidConfigFileName          uint32 = 400802
	InvalidConfigFileContentLength uint32 = 400803
	InvalidConfigFileFormat        uint32 = 400804
	InvalidConfigFileTags          uint32 = 400805
	InvalidWatchConfigFileFormat   uint32 = 400806
	NotFoundResourceConfigFile     uint32 = 400807
	InvalidConfigFileTemplateName  uint32 = 400808

	// 鉴权相关错误码
	InvalidUserOwners         uint32 = 400410
	InvalidUserID             uint32 = 400411
	InvalidUserPassword       uint32 = 400412
	InvalidUserMobile         uint32 = 400413
	InvalidUserEmail          uint32 = 400414
	InvalidUserGroupOwners    uint32 = 400420
	InvalidUserGroupID        uint32 = 400421
	InvalidAuthStrategyOwners uint32 = 400430
	InvalidAuthStrategyName   uint32 = 400431
	InvalidAuthStrategyID     uint32 = 400432
	InvalidPrincipalType      uint32 = 400440

	UserExisted                            uint32 = 400215
	UserGroupExisted                       uint32 = 400216
	AuthStrategyRuleExisted                uint32 = 400217
	SubAccountExisted                      uint32 = 400218
	NotFoundUser                           uint32 = 400312
	NotFoundOwnerUser                      uint32 = 400313
	NotFoundUserGroup                      uint32 = 400314
	NotFoundAuthStrategyRule               uint32 = 400315
	NotAllowModifyDefaultStrategyPrincipal uint32 = 400508
	NotAllowModifyOwnerDefaultStrategy     uint32 = 400509

	EmptyAutToken   uint32 = 401002
	TokenDisabled   uint32 = 401003
	TokenNotExisted uint32 = 401004

	AuthTokenVerifyException uint32 = 500100
	OperationRoleException   uint32 = 500101
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
	InstanceRegisTimeout:               "instance async regist timeout",

	// 配置中心的错误信息
	InvalidConfigFileGroupName:     "invalid config file group name",
	InvalidConfigFileName:          "invalid config file name",
	InvalidConfigFileContentLength: "config file content too long",
	InvalidConfigFileFormat:        "invalid config file format, support json,xml,html,properties,text,yaml",
	InvalidConfigFileTags:          "invalid config file tags, tags should be pair, like key1,value1,key2,value2. and key,value should not blank",
	InvalidWatchConfigFileFormat:   "invalid watch config file format",
	NotFoundResourceConfigFile:     "config file not existed",
	InvalidConfigFileTemplateName:  "invalid config file template name",

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
