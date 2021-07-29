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

/**
 * @brief 通用存储接口
 */
type Store interface {
	// 存储层的名字
	Name() string

	// 存储的初始化函数
	Initialize(c *Config) error

	// 存储的析构函数
	Destroy() error

	CreateTransaction() (Transaction, error)

	// 服务命名空间接口
	NamespaceStore

	// 服务业务集接口
	BusinessStore

	// 服务接口
	ServiceStore

	// 实例接口
	InstanceStore

	// 路由配置接口
	RoutingConfigStore

	// L5扩展接口
	L5Store

	// 限流规则接口
	RateLimitStore

	// 熔断规则接口
	CircuitBreakerStore

	// 平台信息接口
	PlatformStore
}

/**
 * @brief 命名空间存储接口
 */
type NamespaceStore interface {
	// 保存一个命名空间
	AddNamespace(namespace *model.Namespace) error

	// 更新命名空间
	UpdateNamespace(namespace *model.Namespace) error

	// 更新命名空间token
	UpdateNamespaceToken(name string, token string) error

	// 查询owner下所有的命名空间
	ListNamespaces(owner string) ([]*model.Namespace, error)

	// 根据name获取命名空间的详情
	GetNamespace(name string) (*model.Namespace, error)

	// 从数据库查询命名空间
	GetNamespaces(filter map[string][]string, offset, limit int) ([]*model.Namespace, uint32, error)

	// 获取增量数据
	GetMoreNamespaces(mtime time.Time) ([]*model.Namespace, error)
}

/**
 * @brief 业务集存储接口
 */
type BusinessStore interface {
	// 增加一个业务集
	AddBusiness(business *model.Business) error

	// 删除一个业务集
	DeleteBusiness(bid string) error

	// 更新业务集
	UpdateBusiness(business *model.Business) error

	// 更新业务集token
	UpdateBusinessToken(bid string, token string) error

	// 查询owner下业务集
	ListBusiness(owner string) ([]*model.Business, error)

	// 根据业务集ID获取业务集详情
	GetBusinessByID(id string) (*model.Business, error)

	// 根据mtime获取增量数据
	GetMoreBusiness(mtime time.Time) ([]*model.Business, error)
}

/**
 * @brief 服务存储接口
 */
type ServiceStore interface {
	// 保存一个服务
	AddService(service *model.Service) error

	// 删除服务
	DeleteService(id, serviceName, namespaceName string) error

	// 删除服务别名
	DeleteServiceAlias(name string, namespace string) error

	// 修改服务别名
	UpdateServiceAlias(alias *model.Service, needUpdateOwner bool) error

	// 更新服务
	UpdateService(service *model.Service, needUpdateOwner bool) error

	// 更新服务token
	UpdateServiceToken(serviceID string, token string, revision string) error

	// 获取源服务的token信息
	GetSourceServiceToken(name string, namespace string) (*model.Service, error)

	// 根据服务名和命名空间获取服务的详情
	GetService(name string, namespace string) (*model.Service, error)

	// 根据服务ID查询服务详情
	GetServiceByID(id string) (*model.Service, error)

	// 根据相关条件查询对应服务及数目
	GetServices(serviceFilters, serviceMetas map[string]string, instanceFilters *InstanceArgs, offset, limit uint32) (
		uint32, []*model.Service, error)

	// 获取所有服务总数
	GetServicesCount() (uint32, error)

	// 获取增量services
	GetMoreServices(mtime time.Time, firstUpdate, disableBusiness, needMeta bool) (map[string]*model.Service, error)

	// 获取服务别名列表
	GetServiceAliases(filter map[string]string, offset uint32, limit uint32) (uint32, []*model.ServiceAlias, error)

	// 获取系统服务
	GetSystemServices() ([]*model.Service, error)

	// 批量获取服务id、负责人等信息
	GetServicesBatch(services []*model.Service) ([]*model.Service, error)
}

/**
 * @brief 实例存储接口
 */
type InstanceStore interface {
	// 增加一个实例
	AddInstance(instance *model.Instance) error

	// 增加多个实例
	BatchAddInstances(instances []*model.Instance) error

	// 更新实例
	UpdateInstance(instance *model.Instance) error

	// 删除一个实例，实际是把valid置为false
	DeleteInstance(instanceID string) error

	// 批量删除实例，flag=1
	BatchDeleteInstances(ids []interface{}) error

	// 清空一个实例，真正删除
	CleanInstance(instanceID string) error

	// 检查ID是否存在，并且返回所有ID的查询结果
	CheckInstancesExisted(ids map[string]bool) (map[string]bool, error)

	// 获取实例关联的token
	GetInstancesBrief(ids map[string]bool) (map[string]*model.Instance, error)

	// 查询一个实例的详情，只返回有效的数据
	GetInstance(instanceID string) (*model.Instance, error)

	// 获取有效的实例总数
	GetInstancesCount() (uint32, error)

	// 根据服务和Host获取实例（不包括metadata）
	GetInstancesMainByService(serviceID, host string) ([]*model.Instance, error)

	// 根据过滤条件查看实例详情及对应数目
	GetExpandInstances(
		filter, metaFilter map[string]string, offset uint32, limit uint32) (uint32, []*model.Instance, error)

	// 根据mtime获取增量instances，返回所有store的变更信息
	GetMoreInstances(mtime time.Time, firstUpdate, needMeta bool, serviceID []string) (map[string]*model.Instance, error)

	// 设置实例的健康状态
	SetInstanceHealthStatus(instanceID string, flag int, revision string) error

	// 批量修改实例的隔离状态
	BatchSetInstanceIsolate(ids []interface{}, isolate int, revision string) error
}

/**
 * @brief L5扩展存储接口
 */
type L5Store interface {
	// 获取扩展数据
	GetL5Extend(serviceID string) (map[string]interface{}, error)

	// 设置meta里保存的扩展数据，并返回剩余的meta
	SetL5Extend(serviceID string, meta map[string]interface{}) (map[string]interface{}, error)

	// 获取module
	GenNextL5Sid(layoutID uint32) (string, error)

	// 获取增量数据
	GetMoreL5Extend(mtime time.Time) (map[string]map[string]interface{}, error)

	// 获取Route增量数据
	GetMoreL5Routes(flow uint32) ([]*model.Route, error)

	// 获取Policy增量数据
	GetMoreL5Policies(flow uint32) ([]*model.Policy, error)

	//获取Section增量数据
	GetMoreL5Sections(flow uint32) ([]*model.Section, error)

	//获取IP Config增量数据
	GetMoreL5IPConfigs(flow uint32) ([]*model.IPConfig, error)
}

/**
 * @brief 路由配置表的存储接口
 */
type RoutingConfigStore interface {
	// 新增一个路由配置
	CreateRoutingConfig(conf *model.RoutingConfig) error

	// 更新一个路由配置
	UpdateRoutingConfig(conf *model.RoutingConfig) error

	// 删除一个路由配置
	DeleteRoutingConfig(serviceID string) error

	// 通过mtime拉取增量的路由配置信息
	GetRoutingConfigsForCache(mtime time.Time, firstUpdate bool) ([]*model.RoutingConfig, error)

	// 根据服务名和命名空间拉取路由配置
	GetRoutingConfigWithService(name string, namespace string) (*model.RoutingConfig, error)

	// 根据服务ID拉取路由配置
	GetRoutingConfigWithID(id string) (*model.RoutingConfig, error)

	// 查询路由配置列表
	GetRoutingConfigs(filter map[string]string, offset uint32, limit uint32) (uint32, []*model.ExtendRoutingConfig, error)
}

/**
 * @brief 限流规则的存储接口
 */
type RateLimitStore interface {
	// 新增限流规则
	CreateRateLimit(limiting *model.RateLimit) error

	// 更新限流规则
	UpdateRateLimit(limiting *model.RateLimit) error

	// 删除限流规则
	DeleteRateLimit(limiting *model.RateLimit) error

	// 根据过滤条件拉取限流规则
	GetExtendRateLimits(query map[string]string, offset uint32, limit uint32) (uint32, []*model.ExtendRateLimit, error)

	// 根据限流ID拉取限流规则
	GetRateLimitWithID(id string) (*model.RateLimit, error)

	// 根据修改时间拉取增量限流规则及最新版本号
	GetRateLimitsForCache(mtime time.Time, firstUpdate bool) ([]*model.RateLimit, []*model.RateLimitRevision, error)
}

/**
 * @brief 熔断规则的存储接口
 */
type CircuitBreakerStore interface {
	// 新增熔断规则
	CreateCircuitBreaker(circuitBreaker *model.CircuitBreaker) error

	// 标记熔断规则
	TagCircuitBreaker(circuitBreaker *model.CircuitBreaker) error

	// 发布熔断规则
	ReleaseCircuitBreaker(circuitBreakerRelation *model.CircuitBreakerRelation) error

	// 解绑熔断规则
	UnbindCircuitBreaker(serviceID, ruleID, ruleVersion string) error

	// 删除已标记熔断规则
	DeleteTagCircuitBreaker(id string, version string) error

	// 删除master熔断规则
	DeleteMasterCircuitBreaker(id string) error

	// 修改熔断规则
	UpdateCircuitBreaker(circuitBraker *model.CircuitBreaker) error

	// 获取熔断规则
	GetCircuitBreaker(id, version string) (*model.CircuitBreaker, error)

	// 获取熔断规则的所有版本
	GetCircuitBreakerVersions(id string) ([]string, error)

	// 获取熔断规则master版本的绑定关系
	GetCircuitBreakerMasterRelation(ruleID string) ([]*model.CircuitBreakerRelation, error)

	// 获取已标记熔断规则的绑定关系
	GetCircuitBreakerRelation(ruleID, ruleVersion string) ([]*model.CircuitBreakerRelation, error)

	// 根据修改时间拉取增量熔断规则
	GetCircuitBreakerForCache(mtime time.Time, firstUpdate bool) ([]*model.ServiceWithCircuitBreaker, error)

	// 获取master熔断规则
	ListMasterCircuitBreakers(filters map[string]string, offset uint32, limit uint32) (
		*model.CircuitBreakerDetail, error)

	// 获取已发布规则
	ListReleaseCircuitBreakers(filters map[string]string, offset, limit uint32) (
		*model.CircuitBreakerDetail, error)

	// 根据服务获取熔断规则
	GetCircuitBreakersByService(name string, namespace string) (*model.CircuitBreaker, error)
}

/**
 * @brief 平台信息的存储接口
 */
type PlatformStore interface {
	// 新增平台信息
	CreatePlatform(platform *model.Platform) error

	// 更新平台信息
	UpdatePlatform(platform *model.Platform) error

	// 删除平台信息
	DeletePlatform(id string) error

	// 查询平台信息
	GetPlatformById(id string) (*model.Platform, error)

	// 根据过滤条件查询平台信息
	GetPlatforms(query map[string]string, offset uint32, limit uint32) (uint32, []*model.Platform, error)
}

/**
 * @brief 事务接口
 */
type Transaction interface {
	// 提交事务
	Commit() error

	// 启动锁，限制Server启动的并发数
	LockBootstrap(key string, server string) error

	// 排它锁namespace
	LockNamespace(name string) (*model.Namespace, error)

	// 共享锁namespace
	RLockNamespace(name string) (*model.Namespace, error)

	// 删除namespace
	DeleteNamespace(name string) error

	// 排它锁service
	LockService(name string, namespace string) (*model.Service, error)

	// 共享锁service
	RLockService(name string, namespace string) (*model.Service, error)

	// 批量锁住service，只需返回valid/bool，增加速度
	BatchRLockServices(ids map[string]bool) (map[string]bool, error)

	// 删除service
	DeleteService(name string, namespace string) error

	// 删除源服服务下的所有别名
	DeleteAliasWithSourceID(sourceServiceID string) error
}