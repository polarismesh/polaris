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

	commonlog "github.com/polarismesh/polaris/common/log"
	"github.com/polarismesh/polaris/common/metrics"
	commontime "github.com/polarismesh/polaris/common/time"
)

// CacheCall 接口调用
type CacheCall struct {
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

func NewCacheStatics(statis *CacheCallStatis) *CacheStatics {
	return &CacheStatics{
		mutex:           &sync.Mutex{},
		statis:          make(map[string]*CacheCallStatisItem),
		CacheCallStatis: statis,
	}
}

func (c *CacheStatics) Add(ac metrics.CallMetric) {
	index := fmt.Sprintf("%v", ac.Protocol)
	item, exist := c.statis[index]
	if !exist {
		item = &CacheCallStatisItem{
			cacheType: ac.Protocol,
		}
		c.statis[index] = item
	}

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

func NewCacheCallStatis(ctx context.Context) (*CacheCallStatis, error) {
	value := &CacheCallStatis{
		cacheCall: make(chan metrics.CallMetric, 1024),
	}
	value.cacheStatics = NewCacheStatics(value)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case ac := <-value.cacheCall:
				value.cacheStatics.Add(ac)
			}
		}
	}()

	return value, nil
}

// add 添加接口调用数据
func (a *CacheCallStatis) Add(ac metrics.CallMetric) {
	select {
	case a.cacheCall <- ac:
	}
}

// log 打印接口调用统计
func (a *CacheCallStatis) deal() {
	a.cacheStatics.log()
}
