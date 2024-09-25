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

package cache

import (
	"context"
	"errors"
	"sync"

	types "github.com/polarismesh/polaris/cache/api"
	cacheauth "github.com/polarismesh/polaris/cache/auth"
	cacheclient "github.com/polarismesh/polaris/cache/client"
	cacheconfig "github.com/polarismesh/polaris/cache/config"
	cachegray "github.com/polarismesh/polaris/cache/gray"
	cachens "github.com/polarismesh/polaris/cache/namespace"
	cachesvc "github.com/polarismesh/polaris/cache/service"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/store"
)

func init() {
	RegisterCache(types.NamespaceName, types.CacheNamespace)
	RegisterCache(types.ServiceName, types.CacheService)
	RegisterCache(types.InstanceName, types.CacheInstance)
	RegisterCache(types.RoutingConfigName, types.CacheRoutingConfig)
	RegisterCache(types.RateLimitConfigName, types.CacheRateLimit)
	RegisterCache(types.FaultDetectRuleName, types.CacheFaultDetector)
	RegisterCache(types.CircuitBreakerName, types.CacheCircuitBreaker)
	RegisterCache(types.L5Name, types.CacheCL5)
	RegisterCache(types.ConfigFileCacheName, types.CacheConfigFile)
	RegisterCache(types.ConfigGroupCacheName, types.CacheConfigGroup)
	RegisterCache(types.UsersName, types.CacheUser)
	RegisterCache(types.StrategyRuleName, types.CacheAuthStrategy)
	RegisterCache(types.ClientName, types.CacheClient)
	RegisterCache(types.ServiceContractName, types.CacheServiceContract)
	RegisterCache(types.GrayName, types.CacheGray)
	RegisterCache(types.LaneRuleName, types.CacheLaneRule)
	RegisterCache(types.RolesName, types.CacheRole)
}

var (
	cacheMgn   *CacheManager
	once       sync.Once
	finishInit bool
)

// Initialize 初始化
func Initialize(ctx context.Context, cacheOpt *Config, storage store.Store) error {
	var err error
	once.Do(func() {
		err = initialize(ctx, cacheOpt, storage)
	})

	if err != nil {
		return err
	}

	finishInit = true
	return nil
}

// initialize cache 初始化
func initialize(ctx context.Context, cacheOpt *Config, storage store.Store) error {
	var err error
	cacheMgn, err = newCacheManager(ctx, cacheOpt, storage)
	return err
}

func newCacheManager(ctx context.Context, cacheOpt *Config, storage store.Store) (*CacheManager, error) {
	SetCacheConfig(cacheOpt)
	mgr := &CacheManager{
		storage:  storage,
		caches:   make([]types.Cache, types.CacheLast),
		needLoad: utils.NewSyncSet[string](),
	}

	// 命名空间缓存
	mgr.RegisterCacher(types.CacheNamespace, cachens.NewNamespaceCache(storage, mgr))
	// 注册发现 & 服务治理缓存
	mgr.RegisterCacher(types.CacheService, cachesvc.NewServiceCache(storage, mgr))
	mgr.RegisterCacher(types.CacheInstance, cachesvc.NewInstanceCache(storage, mgr))
	mgr.RegisterCacher(types.CacheRoutingConfig, cachesvc.NewRouteRuleCache(storage, mgr))
	mgr.RegisterCacher(types.CacheRateLimit, cachesvc.NewRateLimitCache(storage, mgr))
	mgr.RegisterCacher(types.CacheCircuitBreaker, cachesvc.NewCircuitBreakerCache(storage, mgr))
	mgr.RegisterCacher(types.CacheFaultDetector, cachesvc.NewFaultDetectCache(storage, mgr))
	mgr.RegisterCacher(types.CacheCL5, cachesvc.NewL5Cache(storage, mgr))
	mgr.RegisterCacher(types.CacheServiceContract, cachesvc.NewServiceContractCache(storage, mgr))
	mgr.RegisterCacher(types.CacheLaneRule, cachesvc.NewLaneCache(storage, mgr))
	// 配置分组 & 配置发布缓存
	mgr.RegisterCacher(types.CacheConfigFile, cacheconfig.NewConfigFileCache(storage, mgr))
	mgr.RegisterCacher(types.CacheConfigGroup, cacheconfig.NewConfigGroupCache(storage, mgr))
	// 用户/用户组 & 鉴权规则缓存
	mgr.RegisterCacher(types.CacheUser, cacheauth.NewUserCache(storage, mgr))
	mgr.RegisterCacher(types.CacheAuthStrategy, cacheauth.NewStrategyCache(storage, mgr))
	mgr.RegisterCacher(types.CacheRole, cacheauth.NewRoleCache(storage, mgr))
	// 北极星SDK Client
	mgr.RegisterCacher(types.CacheClient, cacheclient.NewClientCache(storage, mgr))
	mgr.RegisterCacher(types.CacheGray, cachegray.NewGrayCache(storage, mgr))

	if len(mgr.caches) != int(types.CacheLast) {
		return nil, errors.New("some Cache implement not loaded into CacheManager")
	}

	if err := mgr.Initialize(); err != nil {
		return nil, err
	}
	return mgr, nil
}

func Run(cacheMgr *CacheManager, ctx context.Context) error {
	if startErr := cacheMgr.Start(ctx); startErr != nil {
		log.Errorf("[Cache][Server] start cache err: %s", startErr.Error())
		return startErr
	}

	return nil
}

// GetCacheManager
func GetCacheManager() (*CacheManager, error) {
	if !finishInit {
		return nil, errors.New("cache has not done Initialize")
	}
	return cacheMgn, nil
}
