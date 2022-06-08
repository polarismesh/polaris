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

package local

import (
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	commontime "github.com/polarismesh/polaris-server/common/time"
	"github.com/polarismesh/polaris-server/plugin"
)

// CacheCall 接口调用
type CacheCall struct {
	component string
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
type ComponentCacheStatics struct {
	mutex *sync.Mutex

	statis map[string]*CacheCallStatisItem

	logger *zap.Logger

	CacheCallStatis *CacheCallStatis
}

func newComponentCacheStatics(outputPath string, component string, statis *CacheCallStatis) *ComponentCacheStatics {
	fileName := "cachecall.log"
	fileName = fmt.Sprintf("%s-%s", component, fileName)
	return &ComponentCacheStatics{
		mutex:           &sync.Mutex{},
		statis:          make(map[string]*CacheCallStatisItem),
		logger:          newLogger(outputPath + "/" + fileName),
		CacheCallStatis: statis,
	}
}

func (c *ComponentCacheStatics) add(ac *CacheCall) {
	index := fmt.Sprintf("%v", ac.cacheType)
	item, exist := c.statis[index]
	if !exist {
		c.statis[index] = &CacheCallStatisItem{
			cacheType: ac.cacheType,
		}
	}

	item, _ = c.statis[index]
	if ac.miss {
		item.missCount += int64(ac.count)
	} else {
		item.hitCount += int64(ac.count)
	}
}

func (c *ComponentCacheStatics) printStatics(staticsSlice []*CacheCallStatisItem, startStr string) {
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
	c.logger.Info(msg)
}

// log and print the statics messages
func (c *ComponentCacheStatics) log() {

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
	components       map[string]*ComponentCacheStatics
	prometheusStatis *PrometheusStatis
}

func newCacheCallStatis(outputPath string, prometheusStatis *PrometheusStatis) (*CacheCallStatis, error) {
	value := &CacheCallStatis{
		components: make(map[string]*ComponentCacheStatics),
	}

	componentNames := []string{plugin.ComponentProtobufCache}
	for _, componentName := range componentNames {
		value.components[componentName] = newComponentCacheStatics(outputPath, componentName, value)
	}

	value.prometheusStatis = prometheusStatis
	return value, nil
}

// add 添加接口调用数据
func (a *CacheCallStatis) add(ac *CacheCall) {
	a.components[ac.component].add(ac)
}

// log 打印接口调用统计
func (a *CacheCallStatis) log() {
	for _, component := range a.components {
		component.log()
	}
}
