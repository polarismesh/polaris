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
	"encoding/json"
	"sync"
	"time"

	apitraffic "github.com/polarismesh/specification/source/go/api/v1/traffic_manage"
	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"

	"github.com/polarismesh/polaris/common/metrics"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/store"
)

var (
	_ RateLimitCache = (*rateLimitCache)(nil)
)

const (
	// RateLimitConfigName rate limit config name
	RateLimitConfigName = "rateLimitConfig"
)

// RateLimitIterProc rate limit iter func
type RateLimitIterProc func(id string, rateLimit *model.RateLimit) (bool, error)

// RateLimitCache rateLimit的cache接口
type RateLimitCache interface {
	Cache

	// GetRateLimit 根据serviceID进行迭代回调
	GetRateLimit(serviceID string, rateLimitIterProc RateLimitIterProc) error

	// GetLastRevision 根据serviceID获取最新revision
	GetLastRevision(serviceID string) string

	// GetRateLimitByServiceID 根据serviceID获取限流数据
	GetRateLimitByServiceID(serviceID string) []*model.RateLimit

	// GetRevisionsCount 获取revision总数
	GetRevisionsCount() int

	// GetRateLimitsCount 获取限流规则总数
	GetRateLimitsCount() int
}

// rateLimitCache的实现
type rateLimitCache struct {
	*baseCache

	storage     store.Store
	ids         *sync.Map
	revisions   *sync.Map
	lastTime    int64
	lastMtime   time.Time
	firstUpdate bool

	singleFlight singleflight.Group
}

// init 自注册到缓存列表
func init() {
	RegisterCache(RateLimitConfigName, CacheRateLimit)
}

// newRateLimitCache 返回一个操作RateLimitCache的对象
func newRateLimitCache(s store.Store) *rateLimitCache {
	return &rateLimitCache{
		baseCache: newBaseCache(),
		storage:   s,
	}
}

// initialize 实现Cache接口的initialize函数
func (rlc *rateLimitCache) initialize(_ map[string]interface{}) error {
	rlc.ids = new(sync.Map)
	rlc.revisions = new(sync.Map)
	rlc.lastTime = 0
	rlc.firstUpdate = true
	return nil
}

// update 实现Cache接口的update函数
func (rlc *rateLimitCache) update() error {
	// 多个线程竞争，只有一个线程进行更新
	_, err, _ := rlc.singleFlight.Do(rlc.name(), func() (interface{}, error) {
		curStoreTime, err := rlc.storage.GetUnixSecond()
		if err != nil {
			curStoreTime = rlc.lastTime
			log.Warn("[Cache][RateLimit] get store timestamp fail, skip update lastMtime", zap.Error(err))
		}
		defer func() {
			rlc.lastTime = curStoreTime
		}()
		return nil, rlc.realUpdate()
	})

	return err
}

func (rlc *rateLimitCache) realUpdate() error {
	start := time.Now()
	lastTime := time.Unix(rlc.lastTime, 0).Add(DefaultTimeDiff)
	rateLimits, revisions, err := rlc.storage.GetRateLimitsForCache(lastTime, rlc.firstUpdate)
	if err != nil {
		log.Errorf("[Cache] rate limit cache update err: %s", err.Error())
		return err
	}
	metrics.RecordCacheUpdateCost(time.Since(start), rlc.name(), int64(len(rateLimits)))
	rlc.firstUpdate = false
	return rlc.setRateLimit(rateLimits, revisions)
}

// name 获取资源名称
func (rlc *rateLimitCache) name() string {
	return RateLimitConfigName
}

// clear 实现Cache接口的clear函数
func (rlc *rateLimitCache) clear() error {
	rlc.ids = new(sync.Map)
	rlc.revisions = new(sync.Map)
	rlc.lastTime = 0
	rlc.firstUpdate = true
	return nil
}

func rateLimitToProto(rateLimit *model.RateLimit) error {
	rateLimit.Proto = &apitraffic.Rule{}
	if len(rateLimit.Rule) == 0 {
		return nil
	}
	// 反序列化rule
	if err := json.Unmarshal([]byte(rateLimit.Rule), rateLimit.Proto); err != nil {
		return err
	}
	return rateLimit.AdaptArgumentsAndLabels()
}

// setRateLimit 更新限流规则到缓存中
func (rlc *rateLimitCache) setRateLimit(rateLimits []*model.RateLimit,
	revisions []*model.RateLimitRevision) error {
	if len(rateLimits) == 0 {
		return nil
	}

	lastMtime := rlc.lastMtime.Unix()
	for _, item := range rateLimits {
		err := rateLimitToProto(item)
		if nil != err {
			log.Errorf("[Cache]fail to unmarshal rule to proto, err: %v", err)
			continue
		}
		if item.ModifyTime.Unix() > lastMtime {
			lastMtime = item.ModifyTime.Unix()
		}

		// 待删除的rateLimit
		if !item.Valid {
			value, ok := rlc.ids.Load(item.ServiceID)
			if !ok {
				continue
			}
			value.(*sync.Map).Delete(item.ID)
			continue
		}

		value, ok := rlc.ids.Load(item.ServiceID)
		if !ok {
			value = new(sync.Map)
			rlc.ids.Store(item.ServiceID, value)
		}
		value.(*sync.Map).Store(item.ID, item)
	}

	// 更新last revision
	for _, item := range revisions {
		rlc.revisions.Store(item.ServiceID, item.LastRevision)
	}

	if rlc.lastMtime.Unix() < lastMtime {
		log.Infof("[Cache][RateLimit] RateLimit lastMtime update from %s to %s",
			rlc.lastMtime, time.Unix(lastMtime, 0))
		rlc.lastMtime = time.Unix(lastMtime, 0)
	}

	return nil
}

// GetRateLimit 根据serviceID进行迭代回调
func (rlc *rateLimitCache) GetRateLimit(serviceID string, rateLimitIterProc RateLimitIterProc) error {
	if serviceID == "" {
		return nil
	}
	value, ok := rlc.ids.Load(serviceID)
	if !ok {
		return nil
	}

	var (
		result bool
		err    error
	)
	f := func(k, v interface{}) bool {
		result, err = rateLimitIterProc(k.(string), v.(*model.RateLimit))
		if err != nil {
			return false
		}
		return result
	}

	value.(*sync.Map).Range(f)
	return err
}

// GetLastRevision 根据serviceID获取最新revision
func (rlc *rateLimitCache) GetLastRevision(serviceID string) string {
	if serviceID == "" {
		return ""
	}
	value, ok := rlc.revisions.Load(serviceID)
	if !ok {
		return ""
	}
	return value.(string)
}

// GetRateLimitByServiceID 根据serviceID获取限流数据
func (rlc *rateLimitCache) GetRateLimitByServiceID(serviceID string) []*model.RateLimit {
	if serviceID == "" {
		return nil
	}
	value, ok := rlc.ids.Load(serviceID)
	if !ok {
		return nil
	}

	var out []*model.RateLimit
	value.(*sync.Map).Range(func(k interface{}, v interface{}) bool {
		out = append(out, v.(*model.RateLimit))
		return true
	})

	return out
}

// GetRevisionsCount 获取revisions总数
func (rlc *rateLimitCache) GetRevisionsCount() int {
	count := 0
	rlc.revisions.Range(func(k interface{}, v interface{}) bool {
		count++
		return true
	})
	return count
}

// GetRateLimitsCount 获取限流规则总数
func (rlc *rateLimitCache) GetRateLimitsCount() int {
	count := 0

	rlc.ids.Range(func(k interface{}, v interface{}) bool {
		rateLimits := v.(*sync.Map)
		rateLimits.Range(func(k interface{}, v interface{}) bool {
			count++
			return true
		})
		return true
	})
	return count
}
