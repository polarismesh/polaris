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
	"time"

	"github.com/golang/protobuf/ptypes/wrappers"
	v1 "github.com/polarismesh/polaris-server/common/api/v1"
)

/**
 * Namespace 命名空间结构体
 */
type Namespace struct {
	Name       string
	Comment    string
	Token      string
	Owner      string
	Valid      bool
	CreateTime time.Time
	ModifyTime time.Time
}

/**
 * Business 业务集
 */
type Business struct {
	ID         string
	Name       string
	Token      string
	Owner      string
	Valid      bool
	CreateTime time.Time
	ModifyTime time.Time
}

/**
 * Service 服务数据
 */
type Service struct {
	ID          string
	Name        string
	Namespace   string
	Business    string
	Ports       string
	Meta        map[string]string
	Comment     string
	Department  string
	CmdbMod1    string
	CmdbMod2    string
	CmdbMod3    string
	Token       string
	Owner       string
	Revision    string
	Reference   string
	ReferFilter string
	PlatformID  string
	Valid       bool
	CreateTime  time.Time
	ModifyTime  time.Time
	Mtime       int64
	Ctime       int64
}

// EnhancedService 服务增强数据
type EnhancedService struct {
	*Service
	TotalInstanceCount   uint32
	HealthyInstanceCount uint32
}

/**
 * ServiceKey 服务名
 */
type ServiceKey struct {
	Namespace string
	Name      string
}

// IsAlias 便捷函数封装
func (s *Service) IsAlias() bool {
	if s.Reference != "" {
		return true
	}

	return false
}

// ServiceAlias 服务别名结构体
type ServiceAlias struct {
	ID             string
	Alias          string
	AliasNamespace string
	ServiceID      string
	Service        string
	Namespace      string
	Owner          string
	Comment        string
	CreateTime     time.Time
	ModifyTime     time.Time
}

/**
 * WeightType 服务下实例的权重类型
 */
type WeightType uint32

const (
	// WEIGHTDYNAMIC 动态权重
	WEIGHTDYNAMIC WeightType = iota

	// WEIGHTSTATIC 静态权重
	WEIGHTSTATIC
)

// WeightString weight string map
var WeightString = map[WeightType]string{
	WEIGHTDYNAMIC: "dynamic",
	WEIGHTSTATIC:  "static",
}

// WeightEnum weight enum map
var WeightEnum = map[string]WeightType{
	"dynamic": WEIGHTDYNAMIC,
	"static":  WEIGHTSTATIC,
}

/**
 * LocationStore 地域信息，对应数据库字段
 */
type LocationStore struct {
	IP         string
	Region     string
	Zone       string
	Campus     string
	RegionID   uint32
	ZoneID     uint32
	CampusID   uint32
	Flag       int
	ModifyTime int64
}

// Location cmdb信息，对应内存结构体
type Location struct {
	Proto    *v1.Location
	RegionID uint32
	ZoneID   uint32
	CampusID uint32
	Valid    bool
}

// Store2Location 转成内存数据结构
func Store2Location(s *LocationStore) *Location {
	return &Location{
		Proto: &v1.Location{
			Region: &wrappers.StringValue{Value: s.Region},
			Zone:   &wrappers.StringValue{Value: s.Zone},
			Campus: &wrappers.StringValue{Value: s.Campus},
		},
		RegionID: s.RegionID,
		ZoneID:   s.ZoneID,
		CampusID: s.CampusID,
		Valid:    flag2valid(s.Flag),
	}
}

/**
 * Client 客户端上报信息表
 */
type Client struct {
	VpcID      string
	Host       string
	Typ        int
	Version    string
	CreateTime time.Time
	ModifyTime time.Time
}

/**
 * RoutingConfig 路由配置
 */
type RoutingConfig struct {
	ID         string
	InBounds   string
	OutBounds  string
	Revision   string
	Valid      bool
	CreateTime time.Time
	ModifyTime time.Time
}

/**
 * ExtendRoutingConfig 路由配置的扩展结构体
 */
type ExtendRoutingConfig struct {
	ServiceName   string
	NamespaceName string
	Config        *RoutingConfig
}

/**
 * RateLimit 限流规则
 */
type RateLimit struct {
	ID         string
	ServiceID  string
	ClusterID  string
	Labels     string
	Priority   uint32
	Rule       string
	Revision   string
	Valid      bool
	CreateTime time.Time
	ModifyTime time.Time
}

/**
 * ExtendRateLimit 包含服务信息的限流规则
 */
type ExtendRateLimit struct {
	ServiceName   string
	NamespaceName string
	RateLimit     *RateLimit
}

/**
 * RateLimitRevision 包含最新版本号的限流规则
 */
type RateLimitRevision struct {
	ServiceID    string
	LastRevision string
	ModifyTime   time.Time
}

/**
 * CircuitBreaker 熔断规则
 */
type CircuitBreaker struct {
	ID         string
	Version    string
	Name       string
	Namespace  string
	Business   string
	Department string
	Comment    string
	Inbounds   string
	Outbounds  string
	Token      string
	Owner      string
	Revision   string
	Valid      bool
	CreateTime time.Time
	ModifyTime time.Time
}

/**
 * ServiceWithCircuitBreaker 与服务关系绑定的熔断规则
 */
type ServiceWithCircuitBreaker struct {
	ServiceID      string
	CircuitBreaker *CircuitBreaker
	Valid          bool
	CreateTime     time.Time
	ModifyTime     time.Time
}

/**
 * CircuitBreakerRelation 熔断规则绑定关系
 */
type CircuitBreakerRelation struct {
	ServiceID   string
	RuleID      string
	RuleVersion string
	Valid       bool
	CreateTime  time.Time
	ModifyTime  time.Time
}

/**
 * CircuitBreakerDetail 返回给控制台的熔断规则及服务数据
 */
type CircuitBreakerDetail struct {
	Total               uint32
	CircuitBreakerInfos []*CircuitBreakerInfo
}

/**
 * CircuitBreakerInfo 熔断规则及绑定服务
 */
type CircuitBreakerInfo struct {
	CircuitBreaker *CircuitBreaker
	Services       []*Service
}

/**
 * Platform 平台信息
 */
type Platform struct {
	ID         string
	Name       string
	Domain     string
	QPS        uint32
	Token      string
	Owner      string
	Department string
	Comment    string
	Valid      bool
	CreateTime time.Time
	ModifyTime time.Time
}

// int2bool 整数转换为bool值
func int2bool(entry int) bool {
	if entry == 0 {
		return false
	}
	return true
}

// StatusBoolToInt 状态bool转int
func StatusBoolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

// store的flag转换为valid
// flag==1为无效，其他情况为有效
func flag2valid(flag int) bool {
	if flag == 1 {
		return false
	}
	return true

}

// OperationType 操作类型
type OperationType string

// 定义包含的操作类型
const (
	// OCreate 新建
	OCreate OperationType = "Create"

	// ODelete 删除
	ODelete OperationType = "Delete"

	// OUpdate 更新
	OUpdate OperationType = "Update"

	// OUpdateIsolate 更新隔离状态
	OUpdateIsolate OperationType = "UpdateIsolate"

	// OGetToken 查看token
	OGetToken OperationType = "GetToken" // nolint

	// OUpdateToken 更新token
	OUpdateToken OperationType = "UpdateToken" // nolint
)

// Resource 操作资源
type Resource string

// 定义包含的资源类型
const (
	RNamespace         Resource = "Namespace"
	RService           Resource = "Service"
	RRouting           Resource = "Routing"
	RInstance          Resource = "Instance"
	RRateLimit         Resource = "RateLimit"
	RMeshResource      Resource = "MeshResource"
	RMesh              Resource = "Mesh"
	RMeshService       Resource = "MeshService"
	RFluxRateLimit     Resource = "FluxRateLimit"
	RUser              Resource = "User"
	RUserGroup         Resource = "UserGroup"
	RUserGroupRelation Resource = "UserGroupRelation"
	RAuthStrategy      Resource = "AuthStrategy"
)

// ResourceType 资源类型
type ResourceType int

const (
	// MeshType 网格类型资源
	MeshType ResourceType = iota
	// ServiceType 北极星服务类型资源
	ServiceType
)

// ResourceTypeMap resource type map
var ResourceTypeMap = map[Resource]ResourceType{
	RNamespace:    ServiceType,
	RService:      ServiceType,
	RRouting:      ServiceType,
	RInstance:     ServiceType,
	RRateLimit:    ServiceType,
	RMesh:         MeshType,
	RMeshResource: MeshType,
	RMeshService:  MeshType,
}

// GetResourceType 获取资源的大类型
func GetResourceType(r Resource) ResourceType {
	return ResourceTypeMap[r]
}

// RecordEntry 操作记录entry
type RecordEntry struct {
	ResourceType  Resource
	OperationType OperationType
	Namespace     string
	Service       string
	MeshID        string
	MeshName      string
	Context       string
	Operator      string
	Revision      string
	Username      string
	UserGroup     string
	StrategyName  string
	CreateTime    time.Time
}

// DiscoverEventType
type DiscoverEventType string

const (
	// empty discover event
	EventDiscoverNone DiscoverEventType = "EventDiscoverNone"
	// Instance becomes unhealthy
	EventInstanceTurnUnHealth DiscoverEventType = "InstanceTurnUnHealth"
	// Instance becomes healthy
	EventInstanceTurnHealth DiscoverEventType = "InstanceTurnHealth"
	// Instance is in isolation
	EventInstanceOpenIsolate DiscoverEventType = "InstanceOpenIsolate"
	// Instance shutdown isolation state
	EventInstanceCloseIsolate DiscoverEventType = "InstanceCloseIsolate"
	// Instance offline
	EventInstanceOffline DiscoverEventType = "InstanceOffline"
)

// DiscoverEvent 服务发现事件
type DiscoverEvent struct {
	Namespace     string
	Service       string
	Host          string
	Port          int
	EType         DiscoverEventType
	CreateTimeSec int64
}

// InstancesDetail Service instance list and details
type InstanceCount struct {
	// HealthyInstanceCount 健康实例数
	HealthyInstanceCount uint32
	// TotalInstanceCount 总实例数
	TotalInstanceCount uint32
}

// NamespaceServiceCount Namespace service data
type NamespaceServiceCount struct {
	// ServiceCount 服务数量
	ServiceCount uint32
	// InstanceCnt 实例健康数/实例总数
	InstanceCnt *InstanceCount
}
