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

package resource

import (
	"strings"

	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	resourcev3 "github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/polarismesh/specification/source/go/api/v1/fault_tolerance"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"
	"github.com/polarismesh/specification/source/go/api/v1/traffic_manage"

	"github.com/polarismesh/polaris/common/model"
)

const (
	EnvoyHttpFilter_OnDemand = "envoy.filters.http.on_demand"
)

const (
	PassthroughClusterName  = "PassthroughCluster"
	RouteConfigName         = "polaris-router"
	OutBoundRouteConfigName = "polaris-outbound-router"
	InBoundRouteConfigName  = "polaris-inbound-cluster"
	OdcdsRouteConfigName    = "polaris-odcds-router"
	InternalOdcdsHeader     = "internal-service-cluster"
)

const (
	// LocalRateLimitStage envoy local ratelimit stage
	LocalRateLimitStage = 0
	// DistributedRateLimitStage envoy remote ratelimit stage
	DistributedRateLimitStage = 1
)

var (
	defaultOdcdsLuaScriptFile string = "./conf/xds/envoy_lua/odcds.lua"
)

var (
	odcdsLuaCode string
)

func Init() {
	// if val := os.Getenv("ENVOY_ODCDS_LUA_SCRIPT"); val != "" {
	// 	defaultOdcdsLuaScriptFile = val
	// }
	// code, _ := os.ReadFile(defaultOdcdsLuaScriptFile)
	// odcdsLuaCode = string(code)
	// log.Infof("[XDSV3][ODCDS] lua script path :%s content\n%s\n", defaultOdcdsLuaScriptFile, odcdsLuaCode)
}

var (
	TrafficBoundRoute = map[corev3.TrafficDirection]string{
		corev3.TrafficDirection_INBOUND:  InBoundRouteConfigName,
		corev3.TrafficDirection_OUTBOUND: OutBoundRouteConfigName,
	}
)

type XDSType int16

const (
	_ XDSType = iota
	LDS
	RDS
	EDS
	CDS
	RLS
	SDS
	VHDS
	UnknownXDS
)

func FromSimpleXDS(s string) XDSType {
	s = strings.ToLower(s)
	switch s {
	case "cds":
		return CDS
	case "eds":
		return EDS
	case "rds":
		return RDS
	case "lds":
		return LDS
	case "rls":
		return RLS
	case "vhds":
		return VHDS
	default:
		return UnknownXDS
	}
}

func FormatTypeUrl(typeUrl string) XDSType {
	switch typeUrl {
	case resourcev3.ListenerType:
		return LDS
	case resourcev3.RouteType:
		return RDS
	case resourcev3.EndpointType:
		return EDS
	case resourcev3.ClusterType:
		return CDS
	case resourcev3.RateLimitConfigType:
		return RLS
	case resourcev3.VirtualHostType:
		return VHDS
	default:
		return UnknownXDS
	}
}

func (x XDSType) ResourceType() resourcev3.Type {
	if x == LDS {
		return resourcev3.ListenerType
	}
	if x == RDS {
		return resourcev3.RouteType
	}
	if x == EDS {
		return resourcev3.EndpointType
	}
	if x == CDS {
		return resourcev3.ClusterType
	}
	if x == RLS {
		return resourcev3.RateLimitConfigType
	}
	if x == VHDS {
		return resourcev3.VirtualHostType
	}
	return resourcev3.AnyType
}

func (x XDSType) String() string {
	if x == LDS {
		return resourcev3.ListenerType
	}
	if x == RDS {
		return resourcev3.RouteType
	}
	if x == EDS {
		return resourcev3.EndpointType
	}
	if x == CDS {
		return resourcev3.ClusterType
	}
	if x == RLS {
		return resourcev3.RateLimitConfigType
	}
	if x == VHDS {
		return resourcev3.VirtualHostType
	}
	return resourcev3.AnyType
}

const (
	K8sDnsResolveSuffixSvc             = ".svc"
	K8sDnsResolveSuffixSvcCluster      = ".svc.cluster"
	K8sDnsResolveSuffixSvcClusterLocal = ".svc.cluster.local"
)

type TLSMode string

const (
	TLSModeTag                = "polarismesh.cn/tls-mode"
	TLSModeNone       TLSMode = "none"
	TLSModeStrict     TLSMode = "strict"
	TLSModePermissive TLSMode = "permissive"
)

func EnableTLS(t TLSMode) bool {
	return t == TLSModePermissive || t == TLSModeStrict
}

const (
	// 这个是特殊指定的 prefix
	MatchString_Prefix = apimodel.MatchString_MatchStringType(-1)
)

// ServiceInfo 北极星服务结构体
type ServiceInfo struct {
	ID                     string
	Name                   string
	Namespace              string
	ServiceKey             model.ServiceKey
	AliasFor               *model.Service
	Instances              []*apiservice.Instance
	SvcInsRevision         string
	Routing                *traffic_manage.Routing
	SvcRoutingRevision     string
	Ports                  []*model.ServicePort
	RateLimit              *traffic_manage.RateLimit
	SvcRateLimitRevision   string
	CircuitBreaker         *fault_tolerance.CircuitBreaker
	CircuitBreakerRevision string
	FaultDetect            *fault_tolerance.FaultDetector
	FaultDetectRevision    string
}

func (s *ServiceInfo) Equal(o *ServiceInfo) bool {
	// 通过 revision 判断
	if s.SvcInsRevision != o.SvcInsRevision {
		return false
	}
	if s.SvcRoutingRevision != o.SvcRoutingRevision {
		return false
	}
	if s.SvcRateLimitRevision != o.SvcRateLimitRevision {
		return false
	}
	if s.CircuitBreakerRevision != o.CircuitBreakerRevision {
		return false
	}
	if s.FaultDetectRevision != o.FaultDetectRevision {
		return false
	}
	return true
}

func (s *ServiceInfo) MatchService(ns, name string) bool {
	if s.Namespace == ns && s.Name == name {
		return true
	}

	if s.AliasFor != nil {
		if s.AliasFor.Namespace == ns && s.AliasFor.Name == name {
			return true
		}
	}
	return false
}
