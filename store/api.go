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
	"time"

	"github.com/polarismesh/polaris-server/common/model"
)

// Store 通用存储接口
type Store interface {
	// Name 存储层的名字
	Name() string

	// Initialize 存储的初始化函数
	Initialize(c *Config) error

	// Destroy 存储的析构函数
	Destroy() error

	// CreateTransaction 创建事务对象
	CreateTransaction() (Transaction, error)

	// NamespaceStore 服务命名空间接口
	NamespaceStore

	// BusinessStore 服务业务集接口
	BusinessStore

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

	// RateLimitStore 熔断规则接口
	CircuitBreakerStore

	// PlatformStore 平台信息接口
	PlatformStore

	// ToolStore 函数及工具接口
	ToolStore

	// UserStore 用户接口
	UserStore

	// GroupStore 用户组接口
	GroupStore

	// StrategyStore 鉴权策略接口
	StrategyStore
}

// NamespaceStore 命名空间存储接口
type NamespaceStore interface {
	// AddNamespace 保存一个命名空间
	AddNamespace(namespace *model.Namespace) error

	// UpdateNamespace 更新命名空间
	UpdateNamespace(namespace *model.Namespace) error

	// UpdateNamespaceToken 更新命名空间token
	UpdateNamespaceToken(name string, token string) error

	// ListNamespaces 查询owner下所有的命名空间
	ListNamespaces(owner string) ([]*model.Namespace, error)

	// GetNamespace 根据name获取命名空间的详情
	GetNamespace(name string) (*model.Namespace, error)

	// GetNamespaces 从数据库查询命名空间
	GetNamespaces(filter map[string][]string, offset, limit int) ([]*model.Namespace, uint32, error)

	// GetMoreNamespaces 获取增量数据
	GetMoreNamespaces(mtime time.Time) ([]*model.Namespace, error)
}

// BusinessStore 业务集存储接口
type BusinessStore interface {
	// AddBusiness 增加一个业务集
	AddBusiness(business *model.Business) error

	// DeleteBusiness 删除一个业务集
	DeleteBusiness(bid string) error

	// UpdateBusiness 更新业务集
	UpdateBusiness(business *model.Business) error

	// UpdateBusinessToken 更新业务集token
	UpdateBusinessToken(bid string, token string) error

	// ListBusiness 查询owner下业务集
	ListBusiness(owner string) ([]*model.Business, error)

	// GetBusinessByID 根据业务集ID获取业务集详情
	GetBusinessByID(id string) (*model.Business, error)

	// GetMoreBusiness 根据mtime获取增量数据
	GetMoreBusiness(mtime time.Time) ([]*model.Business, error)
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

	// GetInstancesMainByService 根据服务和Host获取实例（不包括metadata）
	GetInstancesMainByService(serviceID, host string) ([]*model.Instance, error)

	// GetExpandInstances 根据过滤条件查看实例详情及对应数目
	GetExpandInstances(
		filter, metaFilter map[string]string, offset uint32, limit uint32) (uint32, []*model.Instance, error)

	// GetMoreInstances 根据mtime获取增量instances，返回所有store的变更信息
	GetMoreInstances(mtime time.Time, firstUpdate, needMeta bool, serviceID []string) (map[string]*model.Instance, error)

	// SetInstanceHealthStatus 设置实例的健康状态
	SetInstanceHealthStatus(instanceID string, flag int, revision string) error

	// BatchSetInstanceHealthStatus 批量设置实例的健康状态
	BatchSetInstanceHealthStatus(ids []interface{}, healthy int, revision string) error

	// BatchSetInstanceIsolate 批量修改实例的隔离状态
	BatchSetInstanceIsolate(ids []interface{}, isolate int, revision string) error
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

	// GetRoutingConfigsForCache 通过mtime拉取增量的路由配置信息
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

	// DeleteRateLimit 删除限流规则
	DeleteRateLimit(limiting *model.RateLimit) error

	// GetExtendRateLimits 根据过滤条件拉取限流规则
	GetExtendRateLimits(query map[string]string, offset uint32, limit uint32) (uint32, []*model.ExtendRateLimit, error)

	// GetRateLimitWithID 根据限流ID拉取限流规则
	GetRateLimitWithID(id string) (*model.RateLimit, error)

	// GetRateLimitsForCache 根据修改时间拉取增量限流规则及最新版本号
	GetRateLimitsForCache(mtime time.Time, firstUpdate bool) ([]*model.RateLimit, []*model.RateLimitRevision, error)
}

// CircuitBreakerStore 熔断规则的存储接口
type CircuitBreakerStore interface {
	// CreateCircuitBreaker 新增熔断规则
	CreateCircuitBreaker(circuitBreaker *model.CircuitBreaker) error

	// TagCircuitBreaker 标记熔断规则
	TagCircuitBreaker(circuitBreaker *model.CircuitBreaker) error

	// ReleaseCircuitBreaker 发布熔断规则
	ReleaseCircuitBreaker(circuitBreakerRelation *model.CircuitBreakerRelation) error

	// UnbindCircuitBreaker 解绑熔断规则
	UnbindCircuitBreaker(serviceID, ruleID, ruleVersion string) error

	// DeleteTagCircuitBreaker 删除已标记熔断规则
	DeleteTagCircuitBreaker(id string, version string) error

	// DeleteMasterCircuitBreaker 删除master熔断规则
	DeleteMasterCircuitBreaker(id string) error

	// UpdateCircuitBreaker 修改熔断规则
	UpdateCircuitBreaker(circuitBraker *model.CircuitBreaker) error

	// GetCircuitBreaker 获取熔断规则
	GetCircuitBreaker(id, version string) (*model.CircuitBreaker, error)

	// GetCircuitBreakerVersions 获取熔断规则的所有版本
	GetCircuitBreakerVersions(id string) ([]string, error)

	// GetCircuitBreakerMasterRelation 获取熔断规则master版本的绑定关系
	GetCircuitBreakerMasterRelation(ruleID string) ([]*model.CircuitBreakerRelation, error)

	// GetCircuitBreakerRelation 获取已标记熔断规则的绑定关系
	GetCircuitBreakerRelation(ruleID, ruleVersion string) ([]*model.CircuitBreakerRelation, error)

	// GetCircuitBreakerForCache 根据修改时间拉取增量熔断规则
	GetCircuitBreakerForCache(mtime time.Time, firstUpdate bool) ([]*model.ServiceWithCircuitBreaker, error)

	// ListMasterCircuitBreakers 获取master熔断规则
	ListMasterCircuitBreakers(filters map[string]string, offset uint32, limit uint32) (
		*model.CircuitBreakerDetail, error)

	// ListReleaseCircuitBreakers 获取已发布规则
	ListReleaseCircuitBreakers(filters map[string]string, offset, limit uint32) (
		*model.CircuitBreakerDetail, error)

	// GetCircuitBreakersByService 根据服务获取熔断规则
	GetCircuitBreakersByService(name string, namespace string) (*model.CircuitBreaker, error)
}

// PlatformStore 平台信息的存储接口
type PlatformStore interface {
	// CreatePlatform 新增平台信息
	CreatePlatform(platform *model.Platform) error

	// UpdatePlatform 更新平台信息
	UpdatePlatform(platform *model.Platform) error

	// DeletePlatform 删除平台信息
	DeletePlatform(id string) error

	// GetPlatformById 查询平台信息
	GetPlatformById(id string) (*model.Platform, error)

	// GetPlatforms 根据过滤条件查询平台信息
	GetPlatforms(query map[string]string, offset uint32, limit uint32) (uint32, []*model.Platform, error)
}

// UserStore
type UserStore interface {

	// AddUser
	AddUser(user *model.User) error

	// UpdateUser
	UpdateUser(user *model.User) error

	// DeleteUser
	DeleteUser(id string) error

	// GetUser
	GetUser(id string) (*model.User, error)

	// GetUserByName
	GetUserByName(name, ownerId string) (*model.User, error)

	// GetUserByIDS
	GetUserByIDS(ids []string) ([]*model.User, error)

	// GetUsers
	GetUsers(filters map[string]string, offset uint32, limit uint32) (uint32, []*model.User, error)

	// GetUsersForCache
	GetUsersForCache(mtime time.Time, firstUpdate bool) ([]*model.User, error)
}

type GroupStore interface {

	// AddGroup
	AddGroup(group *model.UserGroupDetail) error

	// UpdateGroup
	UpdateGroup(group *model.ModifyUserGroup) error

	// DeleteGroup
	DeleteGroup(id string) error

	// GetGroup
	GetGroup(id string) (*model.UserGroupDetail, error)

	// GetGroupByName
	GetGroupByName(name, owner string) (*model.UserGroup, error)

	// GetGroups
	GetGroups(filters map[string]string, offset uint32, limit uint32) (uint32, []*model.UserGroup, error)

	// GetUserGroupsForCache
	GetGroupsForCache(mtime time.Time, firstUpdate bool) ([]*model.UserGroupDetail, error)
}

// StrategyStore
type StrategyStore interface {

	// AddStrategy
	AddStrategy(strategy *model.StrategyDetail) error

	// UpdateStrategy
	UpdateStrategy(strategy *model.ModifyStrategyDetail) error

	// DeleteStrategy
	DeleteStrategy(id string) error

	// LooseAddStrategyResources 松要求的添加鉴权策略的资源，允许忽略主键冲突的问题
	LooseAddStrategyResources(resources []model.StrategyResource) error

	// RemoveStrategyResources 松要求的添加鉴权策略的资源，允许忽略主键冲突的问题
	RemoveStrategyResources(resources []model.StrategyResource) error

	// GetStrategyDetail
	GetStrategyDetail(id string) (*model.StrategyDetail, error)

	// GetStrategyDetailByName
	GetStrategyDetailByName(owner, name string) (*model.StrategyDetail, error)

	// GetStrategySimpleByName
	GetStrategySimpleByName(owner, name string) (*model.Strategy, error)

	// GetSimpleStrategies
	GetSimpleStrategies(filters map[string]string, offset uint32, limit uint32) (uint32, []*model.StrategyDetail, error)

	// GetStrategyDetailsForCache
	GetStrategyDetailsForCache(mtime time.Time, firstUpdate bool) ([]*model.StrategyDetail, error)
}

// Transaction 事务接口，不支持多协程并发操作，当前只支持单个协程串行操作
type Transaction interface {
	// Commit 提交事务
	Commit() error

	// LockBootstrap 启动锁，限制Server启动的并发数
	LockBootstrap(key string, server string) error

	// LockNamespace 排它锁namespace
	LockNamespace(name string) (*model.Namespace, error)

	// DeleteNamespace 删除namespace
	DeleteNamespace(name string) error

	// LockService 排它锁service
	LockService(name string, namespace string) (*model.Service, error)

	// RLockService 共享锁service
	RLockService(name string, namespace string) (*model.Service, error)
}

// ToolStore 存储相关的函数及工具接口
type ToolStore interface {
	// GetNow 获取当前时间
	GetNow() (int64, error)
}
