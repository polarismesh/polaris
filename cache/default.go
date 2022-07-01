/**
 * Tencent is pleased to support the open source community by making CL5 available.
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

	"github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/store"
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
func initialize(_ context.Context, cacheOpt *Config, storage store.Store) error {
	if !cacheOpt.Open {
		return nil
	}

	SetCacheConfig(cacheOpt)
	cacheMgn = &CacheManager{
		storage:       storage,
		caches:        make([]Cache, CacheLast),
		comRevisionCh: make(chan *revisionNotify, RevisionChanCount),
		revisions:     map[string]string{},
	}

	ic := newInstanceCache(storage, cacheMgn.comRevisionCh)
	sc := newServiceCache(storage, cacheMgn.comRevisionCh, ic)
	cacheMgn.caches[CacheService] = sc
	cacheMgn.caches[CacheInstance] = ic
	cacheMgn.caches[CacheRoutingConfig] = newRoutingConfigCache(storage)
	cacheMgn.caches[CacheCL5] = &l5Cache{
		storage: storage,
		ic:      ic,
		sc:      sc,
	}
	cacheMgn.caches[CacheRateLimit] = newRateLimitCache(storage)
	cacheMgn.caches[CacheCircuitBreaker] = newCircuitBreakerCache(storage)

	notify := make(chan interface{}, 8)

	cacheMgn.caches[CacheUser] = newUserCache(storage, notify)
	cacheMgn.caches[CacheAuthStrategy] = newStrategyCache(storage, notify, cacheMgn.caches[CacheUser].(UserCache))
	cacheMgn.caches[CacheNamespace] = newNamespaceCache(storage)
	cacheMgn.caches[CacheClient] = newClientCache(storage)

	if len(cacheMgn.caches) != CacheLast {
		return errors.New("some Cache implement not loaded into CacheManager")
	}

	// call cache.addlistener here, need ensure that all of cache impl has been instantiated and loaded
	cacheMgn.AddListener(CacheNameInstance, []Listener{
		&WatchInstanceReload{
			Handler: func(val interface{}) {
				if svcIds, ok := val.(map[string]bool); ok {
					cacheMgn.caches[CacheService].(*serviceCache).notifyServiceCountReload(svcIds)
				}
			},
		},
	})

	if err := cacheMgn.initialize(); err != nil {
		return err
	}

	return nil
}

func Run(ctx context.Context) error {
	if startErr := cacheMgn.Start(ctx); startErr != nil {
		log.CacheScope().Errorf("[Cache][Server] start cache err: %s", startErr.Error())
		return startErr
	}

	return nil
}

// GetCacheManager
//  @return *CacheManager
//  @return error
func GetCacheManager() (*CacheManager, error) {
	if !finishInit {
		return nil, errors.New("cache has not done Initialize")
	}

	return cacheMgn, nil
}
