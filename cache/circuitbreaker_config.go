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
	"sync"
	"time"

	"github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/store"
)

const (
	// CircuitBreakerName circuit breaker config name
	CircuitBreakerName = "circuitBreakerConfig"
)

var _ CircuitBreakerCache = (*circuitBreakerCache)(nil)

// CircuitBreakerCache  circuitBreaker配置的cache接口
type CircuitBreakerCache interface {
	Cache

	// GetCircuitBreakerConfig 根据ServiceID获取熔断配置
	GetCircuitBreakerConfig(id string) *model.ServiceWithCircuitBreaker
}

// circuitBreaker的实现
type circuitBreakerCache struct {
	*baseCache

	storage         store.Store
	circuitBreakers map[string]*model.ServiceWithCircuitBreaker
	lock            sync.RWMutex
	lastTime        time.Time
	firstUpdate     bool
}

// init 自注册到缓存列表
func init() {
	RegisterCache(CircuitBreakerName, CacheCircuitBreaker)
}

// newCircuitBreakerCache 返回一个操作CircuitBreakerCache的对象
func newCircuitBreakerCache(s store.Store) *circuitBreakerCache {
	return &circuitBreakerCache{
		baseCache:       newBaseCache(),
		storage:         s,
		circuitBreakers: map[string]*model.ServiceWithCircuitBreaker{},
	}
}

// initialize 实现Cache接口的函数
func (c *circuitBreakerCache) initialize(_ map[string]interface{}) error {
	c.lastTime = time.Unix(0, 0)
	c.firstUpdate = true

	return nil
}

// update 实现Cache接口的函数
func (c *circuitBreakerCache) update(storeRollbackSec time.Duration) error {
	lastTime := c.lastTime.Add(storeRollbackSec)
	out, err := c.storage.GetCircuitBreakerForCache(lastTime, c.firstUpdate)
	if err != nil {
		log.CacheScope().Errorf("[Cache] circuit breaker config cache update err:%s", err.Error())
		return err
	}

	c.firstUpdate = false
	return c.setCircuitBreaker(out)
}

// clear 实现Cache接口的函数
func (c *circuitBreakerCache) clear() error {
	c.lock.Lock()
	c.circuitBreakers = map[string]*model.ServiceWithCircuitBreaker{}
	c.lock.Unlock()

	c.lastTime = time.Unix(0, 0)
	return nil
}

// name 实现资源名称
func (c *circuitBreakerCache) name() string {
	return CircuitBreakerName
}

// GetCircuitBreakerConfig 根据serviceID获取熔断规则
func (c *circuitBreakerCache) GetCircuitBreakerConfig(id string) *model.ServiceWithCircuitBreaker {
	if id == "" {
		return nil
	}

	c.lock.RLock()
	defer c.lock.RUnlock()
	value, ok := c.circuitBreakers[id]
	if !ok {
		return nil
	}

	return value
}

func (c *circuitBreakerCache) deleteCircuitBreaker(id string) {
	c.lock.Lock()
	delete(c.circuitBreakers, id)
	c.lock.Unlock()
}

func (c *circuitBreakerCache) storeCircuitBreaker(entry *model.ServiceWithCircuitBreaker) {
	c.lock.Lock()
	c.circuitBreakers[entry.ServiceID] = entry
	c.lock.Unlock()
}

// setCircuitBreaker 更新store的数据到cache中
func (c *circuitBreakerCache) setCircuitBreaker(cbs []*model.ServiceWithCircuitBreaker) error {
	if len(cbs) == 0 {
		return nil
	}

	lastTime := c.lastTime.Unix()

	// Here is a slice pointer type, do not use for _,entry := range mode
	// avoid pointer copy during processing
	for k := range cbs {
		if cbs[k].ServiceID == "" {
			continue
		}

		if cbs[k].ModifyTime.Unix() > lastTime {
			lastTime = cbs[k].ModifyTime.Unix()
		}

		if !cbs[k].Valid {
			c.deleteCircuitBreaker(cbs[k].ServiceID)
			continue
		}

		c.storeCircuitBreaker(cbs[k])
	}

	if c.lastTime.Unix() < lastTime {
		c.lastTime = time.Unix(lastTime, 0)
	}

	return nil
}

// GetCircuitBreakerCount 获取熔断规则总数
func (c *circuitBreakerCache) GetCircuitBreakerCount(f func(k, v interface{}) bool) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	for k, v := range c.circuitBreakers {
		if !f(k, v) {
			break
		}
	}
}
