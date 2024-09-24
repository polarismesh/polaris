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

package model

import (
	"strings"
)

const (
	MIME = "application/x-www-form-urlencoded"
)

const (
	DefaultListenPort = 8848
)

const (
	NacosClientAuthHeader = "accessToken"
)

const (
	ParamCode              = "code"
	ParamServiceName       = "serviceName"
	ParamClusterList       = "clusters"
	ParamCluster           = "cluster"
	ParamClusterName       = "clusterName"
	ParamNamespaceID       = "namespaceId"
	ParamGroupName         = "groupName"
	ParamInstanceIP        = "ip"
	ParamInstancePort      = "port"
	ParamInstanceHealthy   = "healthy"
	ParamInstanceEphemeral = "ephemeral"
	ParamInstanceWeight    = "weight"
	ParamInstanceEnabled   = "enabled"
	ParamInstanceEnable    = "enable"
	ParamInstanceMetadata  = "metadata"
	ParamInstanceBeat      = "beat"
	ParamPageNo            = "pageNo"
	ParamPageSize          = "pageSize"
	ParamSelector          = "selector"
	ParamTenant            = "tenant"
)

const (
	DefaultNacosNamespace       = "public"
	DefaultNacosConfigNamespace = ""
	DefaultServiceGroup         = "DEFAULT_GROUP"
	DefaultServiceClusterName   = "DEFAULT"
	DefaultNacosGroupConnectStr = "@@"
	ReplaceNacosGroupConnectStr = "__"
)

const (
	InstanceMaxWeight     = float64(1000)
	InstanceMinWeight     = float64(0)
	InstanceDefaultWeight = float64(1)

	ClientBeatIntervalMill = 5000
)

const (
	InternalNacosCluster                 = "internal-nacos-cluster"
	InternalNacosServiceName             = "internal-nacos-service"
	InternalNacosServiceProtectThreshold = "internal-nacos-protectThreshold"
	InternalNacosClientConnectionID      = "internal-nacos-clientconnId"
)

const (
	HeaderProbeModifyResponseNew = "Probe-Modify-Response-New"
	HeaderProbeModifyResponse    = "Probe-Modify-Response"
	HeaderContent                = "content"
	HeaderCacheControl           = "Cache-Control"
	HeaderPragma                 = "Pragma"
	HeaderExpires                = "Expires"
	HeaderLastModified           = "Last-Modified"
	HeaderContentMD5             = "Content-MD5"
)

// GetServiceName 获取 nacos 服务中的 service 名称
func GetServiceName(s string) string {
	s = ReplaceNacosService(s)
	if !strings.Contains(s, ReplaceNacosGroupConnectStr) {
		return s
	}
	ss := strings.Split(s, ReplaceNacosGroupConnectStr)
	return ss[1]
}

// GetGroupName 获取 nacos 服务中的 group 名称
func GetGroupName(s string) string {
	s = ReplaceNacosService(s)
	if !strings.Contains(s, ReplaceNacosGroupConnectStr) {
		return DefaultServiceGroup
	}
	ss := strings.Split(s, ReplaceNacosGroupConnectStr)
	return ss[0]
}

var ConvertPolarisNamespaceVal = "default"

// ToPolarisNamespace 替换 nacos namespace 为 polaris 的 namespace 信息，主要是针对默认命令空间转为 polaris 的 default
func ToPolarisNamespace(ns string) string {
	if ns == "" || ns == DefaultNacosNamespace {
		return ConvertPolarisNamespaceVal
	}
	return ns
}

// ToNacosNamespace 替换 polaris namespace 为 nacos 的 namespace 信息，恢复下发 nacos 的数据包信息
func ToNacosNamespace(ns string) string {
	if ns == ConvertPolarisNamespaceVal {
		return "public"
	}
	return ns
}

// ToNacosConfigNamespace 替换 polaris namespace 为 nacos 的 namespace 信息，恢复下发 nacos 的数据包信息
func ToNacosConfigNamespace(ns string) string {
	if ns == ConvertPolarisNamespaceVal {
		return ""
	}
	return ns
}

const (
	ActionGetConfigFile         = "NACOS_GET_CONFIG"
	ActionPublishConfigFile     = "NACOS_PUBLISH_CONFIG"
	ActionGrpcGetConfigFile     = "NACOS_GRPC_GET_CONFIG"
	ActionGrpcPublishConfigFile = "NACOS_GRPC_PUBLISH_CONFIG"
	ActionGrpcPushConfigFile    = "NACOS_GRPC_PUSH_CONFIG"
)
