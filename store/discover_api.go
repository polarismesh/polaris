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

package store

import (
	"context"
	"time"

	"github.com/polarismesh/polaris/common/model"
)

// NamingModuleStore Service discovery, governance center module storage interface
type NamingModuleStore interface {
	// ServiceStore 服务接口
	ServiceStore
	// InstanceStore 实例接口
	InstanceStore
	// RoutingConfigStore 路由配置接口
	RoutingConfigStore
	// L5Store L5扩展接口
	L5Store
	// RateLimitStore 限流规则接口
	RateLimitStore
	// CircuitBreakerStore 熔断规则接口
	CircuitBreakerStore
	// ToolStore 函数及工具接口
	ToolStore
	// RoutingConfigStoreV2 路由策略 v2 接口
	RoutingConfigStoreV2
	// FaultDetectRuleStore fault detect rule interface
	FaultDetectRuleStore
	// ServiceContractStore 服务契约操作接口
	ServiceContractStore
	// LaneStore 泳道规则存储操作接口
	LaneStore
}

// ServiceStore 服务存储接口
type ServiceStore interface {
	// AddService 保存一个服务
	AddService(service *model.Service) error
	// DeleteService 删除服务
	DeleteService(id, serviceName, namespaceName string) error
	// DeleteServiceAlias 删除服务别名
	DeleteServiceAlias(name string, namespace string) error
	// UpdateServiceAlias 修改服务别名
	UpdateServiceAlias(alias *model.Service, needUpdateOwner bool) error
	// UpdateService 更新服务
	UpdateService(service *model.Service, needUpdateOwner bool) error
	// UpdateServiceToken 更新服务token
	UpdateServiceToken(serviceID string, token string, revision string) error
	// GetSourceServiceToken 获取源服务的token信息
	GetSourceServiceToken(name string, namespace string) (*model.Service, error)
	// GetService 根据服务名和命名空间获取服务的详情
	GetService(name string, namespace string) (*model.Service, error)
	// GetServiceByID 根据服务ID查询服务详情
	GetServiceByID(id string) (*model.Service, error)
	// GetServices 根据相关条件查询对应服务及数目
	GetServices(serviceFilters, serviceMetas map[string]string, instanceFilters *InstanceArgs, offset, limit uint32) (
		uint32, []*model.Service, error)
	// GetServicesCount 获取所有服务总数
	GetServicesCount() (uint32, error)
	// GetMoreServices 获取增量services
	// 此方法用于 cache 增量更新，需要注意 mtime 应为数据库时间戳
	GetMoreServices(mtime time.Time, firstUpdate, disableBusiness, needMeta bool) (map[string]*model.Service, error)
	// GetServiceAliases 获取服务别名列表
	GetServiceAliases(filter map[string]string, offset uint32, limit uint32) (uint32, []*model.ServiceAlias, error)
	// GetSystemServices 获取系统服务
	GetSystemServices() ([]*model.Service, error)
	// GetServicesBatch 批量获取服务id、负责人等信息
	GetServicesBatch(services []*model.Service) ([]*model.Service, error)
}

// InstanceStore 实例存储接口
type InstanceStore interface {
	// AddInstance 增加一个实例
	AddInstance(instance *model.Instance) error
	// BatchAddInstances 增加多个实例
	BatchAddInstances(instances []*model.Instance) error
	// UpdateInstance 更新实例
	UpdateInstance(instance *model.Instance) error
	// DeleteInstance 删除一个实例，实际是把valid置为false
	DeleteInstance(instanceID string) error
	// BatchDeleteInstances 批量删除实例，flag=1
	BatchDeleteInstances(ids []interface{}) error
	// CleanInstance 清空一个实例，真正删除
	CleanInstance(instanceID string) error
	// BatchGetInstanceIsolate 检查ID是否存在，并且返回存在的ID，以及ID的隔离状态
	BatchGetInstanceIsolate(ids map[string]bool) (map[string]bool, error)
	// GetInstancesBrief 获取实例关联的token
	GetInstancesBrief(ids map[string]bool) (map[string]*model.Instance, error)
	// GetInstance 查询一个实例的详情，只返回有效的数据
	GetInstance(instanceID string) (*model.Instance, error)
	// GetInstancesCount 获取有效的实例总数
	GetInstancesCount() (uint32, error)
	// GetInstancesCountTx 获取有效的实例总数
	GetInstancesCountTx(tx Tx) (uint32, error)
	// GetInstancesMainByService 根据服务和Host获取实例（不包括metadata）
	GetInstancesMainByService(serviceID, host string) ([]*model.Instance, error)
	// GetExpandInstances 根据过滤条件查看实例详情及对应数目
	GetExpandInstances(
		filter, metaFilter map[string]string, offset uint32, limit uint32) (uint32, []*model.Instance, error)
	// GetMoreInstances 根据mtime获取增量instances，返回所有store的变更信息
	// 此方法用于 cache 增量更新，需要注意 mtime 应为数据库时间戳
	GetMoreInstances(tx Tx, mtime time.Time, firstUpdate, needMeta bool, serviceID []string) (map[string]*model.Instance, error)
	// SetInstanceHealthStatus 设置实例的健康状态
	SetInstanceHealthStatus(instanceID string, flag int, revision string) error
	// BatchSetInstanceHealthStatus 批量设置实例的健康状态
	BatchSetInstanceHealthStatus(ids []interface{}, healthy int, revision string) error
	// BatchSetInstanceIsolate 批量修改实例的隔离状态
	BatchSetInstanceIsolate(ids []interface{}, isolate int, revision string) error
	// AppendInstanceMetadata 追加实例 metadata
	BatchAppendInstanceMetadata(requests []*InstanceMetadataRequest) error
	// RemoveInstanceMetadata 删除实例指定的 metadata
	BatchRemoveInstanceMetadata(requests []*InstanceMetadataRequest) error
}

// L5Store L5扩展存储接口
type L5Store interface {
	// GetL5Extend 获取扩展数据
	GetL5Extend(serviceID string) (map[string]interface{}, error)
	// SetL5Extend 设置meta里保存的扩展数据，并返回剩余的meta
	SetL5Extend(serviceID string, meta map[string]interface{}) (map[string]interface{}, error)
	// GenNextL5Sid 获取module
	GenNextL5Sid(layoutID uint32) (string, error)
	// GetMoreL5Extend 获取增量数据
	GetMoreL5Extend(mtime time.Time) (map[string]map[string]interface{}, error)
	// GetMoreL5Routes 获取Route增量数据
	GetMoreL5Routes(flow uint32) ([]*model.Route, error)
	// GetMoreL5Policies 获取Policy增量数据
	GetMoreL5Policies(flow uint32) ([]*model.Policy, error)
	// GetMoreL5Sections 获取Section增量数据
	GetMoreL5Sections(flow uint32) ([]*model.Section, error)
	// GetMoreL5IPConfigs 获取IP Config增量数据
	GetMoreL5IPConfigs(flow uint32) ([]*model.IPConfig, error)
}

// RoutingConfigStore 路由配置表的存储接口
type RoutingConfigStore interface {
	// CreateRoutingConfig 新增一个路由配置
	CreateRoutingConfig(conf *model.RoutingConfig) error
	// UpdateRoutingConfig 更新一个路由配置
	UpdateRoutingConfig(conf *model.RoutingConfig) error
	// DeleteRoutingConfig 删除一个路由配置
	DeleteRoutingConfig(serviceID string) error
	// DeleteRoutingConfigTx 删除一个路由配置
	DeleteRoutingConfigTx(tx Tx, serviceID string) error
	// GetRoutingConfigsForCache 通过mtime拉取增量的路由配置信息
	// 此方法用于 cache 增量更新，需要注意 mtime 应为数据库时间戳
	GetRoutingConfigsForCache(mtime time.Time, firstUpdate bool) ([]*model.RoutingConfig, error)
	// GetRoutingConfigWithService 根据服务名和命名空间拉取路由配置
	GetRoutingConfigWithService(name string, namespace string) (*model.RoutingConfig, error)
	// GetRoutingConfigWithID 根据服务ID拉取路由配置
	GetRoutingConfigWithID(id string) (*model.RoutingConfig, error)
	// GetRoutingConfigs 查询路由配置列表
	GetRoutingConfigs(filter map[string]string, offset uint32, limit uint32) (uint32, []*model.ExtendRoutingConfig, error)
}

// RateLimitStore 限流规则的存储接口
type RateLimitStore interface {
	// CreateRateLimit 新增限流规则
	CreateRateLimit(limiting *model.RateLimit) error
	// UpdateRateLimit 更新限流规则
	UpdateRateLimit(limiting *model.RateLimit) error
	// EnableRateLimit 启用限流规则
	EnableRateLimit(limit *model.RateLimit) error
	// DeleteRateLimit 删除限流规则
	DeleteRateLimit(limiting *model.RateLimit) error
	// GetExtendRateLimits 根据过滤条件拉取限流规则
	GetExtendRateLimits(query map[string]string, offset uint32, limit uint32) (uint32, []*model.ExtendRateLimit, error)
	// GetRateLimitWithID 根据限流ID拉取限流规则
	GetRateLimitWithID(id string) (*model.RateLimit, error)
	// GetRateLimitsForCache 根据修改时间拉取增量限流规则及最新版本号
	// 此方法用于 cache 增量更新，需要注意 mtime 应为数据库时间戳
	GetRateLimitsForCache(mtime time.Time, firstUpdate bool) ([]*model.RateLimit, error)
}

// CircuitBreakerStore 熔断规则的存储接口
type CircuitBreakerStore interface {
	// CreateCircuitBreakerRule create general circuitbreaker rule
	CreateCircuitBreakerRule(cbRule *model.CircuitBreakerRule) error
	// UpdateCircuitBreakerRule update general circuitbreaker rule
	UpdateCircuitBreakerRule(cbRule *model.CircuitBreakerRule) error
	// DeleteCircuitBreakerRule delete general circuitbreaker rule
	DeleteCircuitBreakerRule(id string) error
	// HasCircuitBreakerRule check circuitbreaker rule exists
	HasCircuitBreakerRule(id string) (bool, error)
	// HasCircuitBreakerRuleByName check circuitbreaker rule exists for name
	HasCircuitBreakerRuleByName(name string, namespace string) (bool, error)
	// HasCircuitBreakerRuleByNameExcludeId check circuitbreaker rule exists for name not this id
	HasCircuitBreakerRuleByNameExcludeId(name string, namespace string, id string) (bool, error)
	// GetCircuitBreakerRules get all circuitbreaker rules by query and limit
	GetCircuitBreakerRules(
		filter map[string]string, offset uint32, limit uint32) (uint32, []*model.CircuitBreakerRule, error)
	// GetCircuitBreakerRulesForCache get increment circuitbreaker rules
	GetCircuitBreakerRulesForCache(mtime time.Time, firstUpdate bool) ([]*model.CircuitBreakerRule, error)
	// EnableCircuitBreakerRule enable specific circuitbreaker rule
	EnableCircuitBreakerRule(cbRule *model.CircuitBreakerRule) error
}

// ClientStore store interface for client info
type ClientStore interface {
	// BatchAddClients insert the client info
	BatchAddClients(clients []*model.Client) error
	// BatchDeleteClients delete the client info
	BatchDeleteClients(ids []string) error
	// GetMoreClients 根据mtime获取增量clients，返回所有store的变更信息
	// 此方法用于 cache 增量更新，需要注意 mtime 应为数据库时间戳
	GetMoreClients(mtime time.Time, firstUpdate bool) (map[string]*model.Client, error)
}

// RoutingConfigStoreV2 路由配置表的存储接口
type RoutingConfigStoreV2 interface {
	// EnableRouting 设置路由规则是否启用
	EnableRouting(conf *model.RouterConfig) error
	// CreateRoutingConfigV2 新增一个路由配置
	CreateRoutingConfigV2(conf *model.RouterConfig) error
	// CreateRoutingConfigV2Tx 新增一个路由配置
	CreateRoutingConfigV2Tx(tx Tx, conf *model.RouterConfig) error
	// UpdateRoutingConfigV2 更新一个路由配置
	UpdateRoutingConfigV2(conf *model.RouterConfig) error
	// UpdateRoutingConfigV2Tx 更新一个路由配置
	UpdateRoutingConfigV2Tx(tx Tx, conf *model.RouterConfig) error
	// DeleteRoutingConfigV2 删除一个路由配置
	DeleteRoutingConfigV2(serviceID string) error
	// GetRoutingConfigsV2ForCache 通过mtime拉取增量的路由配置信息
	// 此方法用于 cache 增量更新，需要注意 mtime 应为数据库时间戳
	GetRoutingConfigsV2ForCache(mtime time.Time, firstUpdate bool) ([]*model.RouterConfig, error)
	// GetRoutingConfigV2WithID 根据服务ID拉取路由配置
	GetRoutingConfigV2WithID(id string) (*model.RouterConfig, error)
	// GetRoutingConfigV2WithIDTx 根据服务ID拉取路由配置
	GetRoutingConfigV2WithIDTx(tx Tx, id string) (*model.RouterConfig, error)
}

// FaultDetectRuleStore store api for the fault detector config
type FaultDetectRuleStore interface {
	// CreateFaultDetectRule create fault detect rule
	CreateFaultDetectRule(conf *model.FaultDetectRule) error
	// UpdateFaultDetectRule update fault detect rule
	UpdateFaultDetectRule(conf *model.FaultDetectRule) error
	// DeleteFaultDetectRule delete fault detect rule
	DeleteFaultDetectRule(id string) error
	// HasFaultDetectRule check fault detect rule exists
	HasFaultDetectRule(id string) (bool, error)
	// HasFaultDetectRuleByName check fault detect rule exists by name
	HasFaultDetectRuleByName(name string, namespace string) (bool, error)
	// HasFaultDetectRuleByNameExcludeId check fault detect rule exists by name not this id
	HasFaultDetectRuleByNameExcludeId(name string, namespace string, id string) (bool, error)
	// GetFaultDetectRules get all fault detect rules by query and limit
	GetFaultDetectRules(filter map[string]string, offset uint32, limit uint32) (uint32, []*model.FaultDetectRule, error)
	// GetFaultDetectRulesForCache get increment fault detect rules
	GetFaultDetectRulesForCache(mtime time.Time, firstUpdate bool) ([]*model.FaultDetectRule, error)
}

type ServiceContractStore interface {
	// CreateServiceContract 创建服务契约
	CreateServiceContract(contract *model.ServiceContract) error
	// UpdateServiceContract 更新服务契约
	UpdateServiceContract(contract *model.ServiceContract) error
	// DeleteServiceContract 删除服务契约
	DeleteServiceContract(contract *model.ServiceContract) error
	// GetServiceContract 查询服务契约数据
	GetServiceContract(id string) (data *model.EnrichServiceContract, err error)
	// GetServiceContracts 查询服务契约公共属性列表
	GetServiceContracts(ctx context.Context, filter map[string]string, offset, limit uint32) (uint32, []*model.EnrichServiceContract, error)
	// AddServiceContractInterfaces 创建服务契约API接口
	AddServiceContractInterfaces(contract *model.EnrichServiceContract) error
	// AppendServiceContractInterfaces 追加服务契约API接口
	AppendServiceContractInterfaces(contract *model.EnrichServiceContract) error
	// DeleteServiceContractInterfaces 批量删除服务契约API接口
	DeleteServiceContractInterfaces(contract *model.EnrichServiceContract) error
	// GetInterfaceDescriptors 查询服务接口列表
	GetInterfaceDescriptors(ctx context.Context, filter map[string]string, offset, limit uint32) (uint32, []*model.InterfaceDescriptor, error)
	// ListVersions .
	ListVersions(ctx context.Context, service, namespace string) ([]*model.ServiceContract, error)
	// GetMoreServiceContracts 查询服务契约公共属性列表
	GetMoreServiceContracts(firstUpdate bool, mtime time.Time) ([]*model.EnrichServiceContract, error)
}

// LaneStore 泳道资源存储操作
type LaneStore interface {
	// AddLaneGroup 添加泳道组
	AddLaneGroup(tx Tx, item *model.LaneGroup) error
	// UpdateLaneGroup 更新泳道组
	UpdateLaneGroup(tx Tx, item *model.LaneGroup) error
	// GetLaneGroup 按照名称查询泳道组
	GetLaneGroup(name string) (*model.LaneGroup, error)
	// GetLaneGroupByID 按照名称查询泳道组
	GetLaneGroupByID(id string) (*model.LaneGroup, error)
	// GetLaneGroups 查询泳道组
	GetLaneGroups(filter map[string]string, offset, limit uint32) (uint32, []*model.LaneGroup, error)
	// LockLaneGroup 锁住一个泳道分组
	LockLaneGroup(tx Tx, name string) (*model.LaneGroup, error)
	// GetMoreLaneGroups 获取泳道规则列表到缓存层
	GetMoreLaneGroups(mtime time.Time, firstUpdate bool) (map[string]*model.LaneGroup, error)
	// DeleteLaneGroup 删除泳道组
	DeleteLaneGroup(id string) error
	// GetLaneRuleMaxPriority 获取泳道规则中当前最大的泳道规则优先级信息
	GetLaneRuleMaxPriority() (int32, error)
}
