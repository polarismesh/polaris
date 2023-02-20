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

package base

import (
	"context"
	"fmt"
	"sync"
	"time"

<<<<<<< HEAD:plugin/statis/base/cachecall.go
	commonlog "github.com/polarismesh/polaris/common/log"
=======
>>>>>>> c97b0c1e... metrics:add cache update metrics:plugin/statis/local/cachecall_handle.go
	"github.com/polarismesh/polaris/common/metrics"
	commontime "github.com/polarismesh/polaris/common/time"
)

// CacheCall 接口调用
type CacheCall struct {
<<<<<<< HEAD:plugin/statis/base/cachecall.go
=======
	component plugin.ComponentType
>>>>>>> c97b0c1e... metrics:add cache update metrics:plugin/statis/local/cachecall_handle.go
	cacheType string
	miss      bool
	count     int32
}

// CacheCallStatisItem 接口调用统计条目
type CacheCallStatisItem struct {
	cacheType string
	hitCount  int64
	missCount int64
	hitRate   float64
}

// ComponentCacheStatics statics components
type CacheStatics struct {
	mutex           *sync.Mutex
	statis          map[string]*CacheCallStatisItem
	CacheCallStatis *CacheCallStatis
}

<<<<<<< HEAD:plugin/statis/base/cachecall.go
func NewCacheStatics(statis *CacheCallStatis) *CacheStatics {
=======
func newCacheStatics(statis *CacheCallStatis) *CacheStatics {
>>>>>>> c97b0c1e... metrics:add cache update metrics:plugin/statis/local/cachecall_handle.go
	return &CacheStatics{
		mutex:           &sync.Mutex{},
		statis:          make(map[string]*CacheCallStatisItem),
		CacheCallStatis: statis,
	}
}

<<<<<<< HEAD:plugin/statis/base/cachecall.go
func (c *CacheStatics) Add(ac metrics.CallMetric) {
=======
func (c *CacheStatics) add(ac metrics.CallMetric) {
>>>>>>> c97b0c1e... metrics:add cache update metrics:plugin/statis/local/cachecall_handle.go
	index := fmt.Sprintf("%v", ac.Protocol)
	item, exist := c.statis[index]
	if !exist {
		c.statis[index] = &CacheCallStatisItem{
			cacheType: ac.Protocol,
		}
	}

	item = c.statis[index]
	if ac.Success {
		item.hitCount += int64(ac.Times)
	} else {
		item.missCount += int64(ac.Times)

	}
}

func (c *CacheStatics) printStatics(staticsSlice []*CacheCallStatisItem, startStr string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	msg := fmt.Sprintf("Statis %s:\n", startStr)

	msg += fmt.Sprintf(
		"%-48v|%12v|%12v|%12v|\n", "", "HitCount", "MissCount", "HitRate")

	for _, item := range staticsSlice {
		if item.hitCount == 0 && item.missCount == 0 {
			continue
		}
		msg += fmt.Sprintf("%-48v|%12v|%12v|%12.3f|\n",
			item.cacheType, item.hitCount, item.missCount,
			float64(item.hitCount)/float64(item.hitCount+item.missCount),
		)
	}
	commonlog.Info(msg)
}

// log and print the statics messages
func (c *CacheStatics) log() {
	if len(c.statis) == 0 {
		return
	}

	duplicateStatis := make([]*CacheCallStatisItem, 0, len(c.statis))
	for _, item := range c.statis {
		duplicateStatis = append(duplicateStatis, item)
	}
	c.statis = make(map[string]*CacheCallStatisItem)

	go c.printStatics(duplicateStatis, commontime.Time2String(time.Now()))
}

// CacheCallStatis 接口调用统计
type CacheCallStatis struct {
	cacheCall    chan metrics.CallMetric
	cacheStatics *CacheStatics
}

<<<<<<< HEAD:plugin/statis/base/cachecall.go
func NewCacheCallStatis(ctx context.Context) (*CacheCallStatis, error) {
	value := &CacheCallStatis{
		cacheCall: make(chan metrics.CallMetric, 1024),
	}
	value.cacheStatics = NewCacheStatics(value)
=======
func newCacheCallStatis(ctx context.Context) (*CacheCallStatis, error) {
	value := &CacheCallStatis{
		cacheCall: make(chan metrics.CallMetric, 1024),
	}
	value.cacheStatics = newCacheStatics(value)
>>>>>>> c97b0c1e... metrics:add cache update metrics:plugin/statis/local/cachecall_handle.go

	go func() {
		for {
			select {
<<<<<<< HEAD:plugin/statis/base/cachecall.go
			case <-ctx.Done():
				return
			case ac := <-value.cacheCall:
				value.cacheStatics.Add(ac)
=======
			case ac := <-value.cacheCall:
				value.cacheStatics.add(ac)
>>>>>>> c97b0c1e... metrics:add cache update metrics:plugin/statis/local/cachecall_handle.go
			}
		}
	}()

	return value, nil
}

// add 添加接口调用数据
<<<<<<< HEAD:plugin/statis/base/cachecall.go
func (a *CacheCallStatis) Add(ac metrics.CallMetric) {
=======
func (a *CacheCallStatis) add(ac metrics.CallMetric) {
>>>>>>> c97b0c1e... metrics:add cache update metrics:plugin/statis/local/cachecall_handle.go
	select {
	case a.cacheCall <- ac:
	}
}

// log 打印接口调用统计
<<<<<<< HEAD:plugin/statis/base/cachecall.go
func (a *CacheCallStatis) deal() {
	a.cacheStatics.log()
=======
func (a *CacheCallStatis) log(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	for {
		select {
		case <-ctx.Done():
			ticker.Stop()
		case <-ticker.C:
			a.cacheStatics.log()
		}
	}
>>>>>>> c97b0c1e... metrics:add cache update metrics:plugin/statis/local/cachecall_handle.go
}
