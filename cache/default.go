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

	"github.com/polarismesh/polaris/store"
)

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
	if !cacheOpt.Open {
		return nil
	}

	var err error

	cacheMgn, err = newCacheManager(ctx, cacheOpt, storage)

	return err
}

func newCacheManager(ctx context.Context, cacheOpt *Config, storage store.Store) (*CacheManager, error) {
	SetCacheConfig(cacheOpt)
	mgr := &CacheManager{
		storage:       storage,
		caches:        make([]Cache, CacheLast),
		comRevisionCh: make(chan *revisionNotify, RevisionChanCount),
		revisions:     map[string]string{},
	}

	ic := newInstanceCache(storage, mgr.comRevisionCh)
	sc := newServiceCache(storage, mgr.comRevisionCh, ic)
	mgr.caches[CacheService] = sc
	mgr.caches[CacheInstance] = ic
	mgr.caches[CacheRoutingConfig] = newRoutingConfigCache(storage, sc)
	mgr.caches[CacheCL5] = &l5Cache{
		storage: storage,
		ic:      ic,
		sc:      sc,
	}
	mgr.caches[CacheRateLimit] = newRateLimitCache(storage)
	mgr.caches[CacheCircuitBreaker] = newCircuitBreakerCache(storage)

	notify := make(chan interface{}, 8)

	mgr.caches[CacheUser] = newUserCache(storage, notify)
	mgr.caches[CacheAuthStrategy] = newStrategyCache(storage, notify, mgr.caches[CacheUser].(UserCache))
	mgr.caches[CacheNamespace] = newNamespaceCache(storage)
	mgr.caches[CacheClient] = newClientCache(storage)
	mgr.caches[CacheConfigFile] = newFileCache(ctx, storage)

	if len(mgr.caches) != CacheLast {
		return nil, errors.New("some Cache implement not loaded into CacheManager")
	}

	// call cache.addlistener here, need ensure that all of cache impl has been instantiated and loaded
	mgr.AddListener(CacheNameInstance, []Listener{
		&WatchInstanceReload{
			Handler: func(val interface{}) {
				if svcIds, ok := val.(map[string]bool); ok {
					mgr.caches[CacheService].(*serviceCache).notifyServiceCountReload(svcIds)
				}
			},
		},
	})

	if err := mgr.initialize(); err != nil {
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
//
//	@return *CacheManager
//	@return error
func GetCacheManager() (*CacheManager, error) {
	if !finishInit {
		return nil, errors.New("cache has not done Initialize")
	}

	return cacheMgn, nil
}
