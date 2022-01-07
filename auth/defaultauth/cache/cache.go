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
	"sync"
	"time"

	"github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/store"
)

const (
	CacheForUser     string = "userCache"
	CacheForStrategy string = "strategyCache"

	// UpdateCacheInterval 缓存更新时间间隔
	UpdateCacheInterval = 1 * time.Second

	// DefaultTimeDiff default time diff
	DefaultTimeDiff = -1 * time.Second * 10
)

// Cache
type Cache interface {

	// initialize
	//  @return error
	initialize() error

	// update
	//  @return error
	update() error

	// clear
	//  @return error
	clear() error

	// name
	//  @return string
	name() string
}

// AuthCache
type AuthCache struct {
	caches map[string]Cache
}

// NewAuthCache
//  @param s
//  @return *AuthCache
//  @return error
func NewAuthCache(s store.Store) (*AuthCache, error) {

	authCache := &AuthCache{
		caches: make(map[string]Cache),
	}

	authCache.caches[CacheForUser] = newUserCache(s)
	authCache.caches[CacheForStrategy] = newStrategyCache(s)

	return authCache, authCache.initialize()
}

func (ac *AuthCache) initialize() error {
	for _, cache := range ac.caches {
		if err := cache.initialize(); err != nil {
			return err
		}
	}

	return nil
}

// Start 缓存对象启动协程，定时更新缓存
func (ac *AuthCache) Start(ctx context.Context) error {
	log.Infof("[AuthCache] cache goroutine start")

	// 启动的时候，先更新一版缓存
	log.Infof("[AuthCache] cache update now first time")
	if err := ac.update(); err != nil {
		return err
	}
	log.Infof("[AuthCache] cache update done")

	// 启动协程，开始定时更新缓存数据
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				_ = ac.update()
			case <-ctx.Done():
				return
			}
		}
	}()

	return nil
}

func (ac *AuthCache) update() error {
	var wg sync.WaitGroup
	for _, entry := range ac.caches {
		wg.Add(1)
		go func(c Cache) {
			defer wg.Done()
			_ = c.update()
		}(entry)
	}

	wg.Wait()
	return nil
}

func (ac *AuthCache) clear() error {
	for _, cache := range ac.caches {
		if err := cache.clear(); err != nil {
			return err
		}
	}

	return nil
}

func (ac *AuthCache) name() string {
	return "AuthCache"
}

func (ac *AuthCache) UserCache() UserCache {
	return ac.caches[CacheForUser].(UserCache)
}

func (ac *AuthCache) StrategyCache() StrategyCache {
	return ac.caches[CacheForStrategy].(StrategyCache)
}
