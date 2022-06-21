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
	"math"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/polarismesh/polaris-server/common/log"
	commontime "github.com/polarismesh/polaris-server/common/time"
	"github.com/polarismesh/polaris-server/plugin"
)

// APICall 接口调用
type APICall struct {
	api       string
	code      int
	duration  int64
	protocol  string
	component string
}

// APICallStatisItem 接口调用统计条目
type APICallStatisItem struct {
	api          string
	code         int
	count        int64
	accTime      int64
	minTime      int64
	maxTime      int64
	protocol     string
	zeroDuration int64 // 没有请求持续的时间，持续时间长超过阈值从 prometheus 中移除掉
}

// ComponentStatics statics components
type ComponentStatics struct {
	mutex *sync.Mutex

	statis map[string]*APICallStatisItem

	logger *zap.Logger

	apiCallStatis *APICallStatis
}

func newComponentStatics(outputPath string, component string, statis *APICallStatis) *ComponentStatics {
	fileName := "apicall.log"
	fileName = fmt.Sprintf("%s-%s", component, fileName)
	return &ComponentStatics{
		mutex:         &sync.Mutex{},
		statis:        make(map[string]*APICallStatisItem),
		logger:        newLogger(outputPath + "/" + fileName),
		apiCallStatis: statis,
	}
}

func (c *ComponentStatics) add(ac *APICall) {
	index := fmt.Sprintf("%v-%v", ac.api, ac.code)
	item, exist := c.statis[index]
	if exist {
		item.count++

		item.accTime += ac.duration
		if ac.duration < item.minTime {
			item.minTime = ac.duration
		}
		if ac.duration > item.maxTime {
			item.maxTime = ac.duration
		}
	} else {
		c.statis[index] = &APICallStatisItem{
			api:      ac.api,
			code:     ac.code,
			count:    1,
			accTime:  ac.duration,
			minTime:  ac.duration,
			maxTime:  ac.duration,
			protocol: ac.protocol,
		}
	}
}

func (c *ComponentStatics) printStatics(staticsSlice []*APICallStatisItem, startStr string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	msg := fmt.Sprintf("Statis %s:\n", startStr)

	msg += fmt.Sprintf(
		"%-48v|%12v|%12v|%12v|%12v|%12v|\n", "", "Code", "Count", "Min(ms)", "Max(ms)", "Avg(ms)")

	for _, item := range staticsSlice {
		if item.count == 0 {
			continue
		}
		msg += fmt.Sprintf("%-48v|%12v|%12v|%12.3f|%12.3f|%12.3f|\n",
			item.api, item.code, item.count,
			float32(item.minTime)/1e6,
			float32(item.maxTime)/1e6,
			float32(item.accTime)/float32(item.count)/1e6,
		)
	}
	c.logger.Info(msg)
}

// log and print the statics messages
func (c *ComponentStatics) log() {
	startTime := time.Now()
	startStr := commontime.Time2String(startTime)
	if len(c.statis) == 0 {
		c.logger.Info(fmt.Sprintf("Statis %s: No API Call\n", startStr))
		return
	}
	defer func() {
		passDuration := time.Since(startTime)
		if passDuration >= maxLogWaitDuration {
			log.Warnf("[APICall]api static log duration %s, pass max %s", passDuration, maxLogWaitDuration)
		}
	}()

	duplicateStatis := make([]*APICallStatisItem, 0, len(c.statis))
	for _, item := range c.statis {
		duplicateStatis = append(duplicateStatis, item)
	}
	c.statis = make(map[string]*APICallStatisItem)

	go c.printStatics(duplicateStatis, startStr)
}

// collect log and print the statics messages
func (c *ComponentStatics) collect() {
	startTime := time.Now()
	startStr := commontime.Time2String(startTime)
	if len(c.statis) == 0 {
		c.logger.Info(fmt.Sprintf("Statis %s: No API Call\n", startStr))
		return
	}
	defer func() {
		passDuration := time.Since(startTime)
		if passDuration >= maxLogWaitDuration {
			log.Warnf("[APICall]api static log duration %s, pass max %s", passDuration, maxLogWaitDuration)
		}
	}()

	duplicateStatis := make([]*APICallStatisItem, 0, len(c.statis))
	currStatis := c.statis
	c.statis = make(map[string]*APICallStatisItem)
	for key, item := range currStatis {
		duplicateStatis = append(duplicateStatis, item)
		if item.count == 0 {
			item.zeroDuration++
			item.minTime = 0
		} else {
			item.zeroDuration = 0
		}

		if item.zeroDuration <= maxZeroDuration {
			c.statis[key] = &APICallStatisItem{
				api:          item.api,
				code:         item.code,
				count:        0,
				protocol:     item.protocol,
				accTime:      0,
				minTime:      math.MaxInt64,
				maxTime:      0,
				zeroDuration: item.zeroDuration,
			}
		}
	}

	go c.apiCallStatis.collectMetricData(duplicateStatis)
	go c.printStatics(duplicateStatis, startStr)
}

// APICallStatis 接口调用统计
type APICallStatis struct {
	components       map[string]*ComponentStatics
	prometheusStatis *PrometheusStatis
}

func newAPICallStatis(outputPath string, prometheusStatis *PrometheusStatis) (*APICallStatis, error) {
	value := &APICallStatis{
		components: make(map[string]*ComponentStatics),
	}
	componentNames := []string{plugin.ComponentServer, plugin.ComponentRedis}
	for _, componentName := range componentNames {
		value.components[componentName] = newComponentStatics(outputPath, componentName, value)
	}

	value.prometheusStatis = prometheusStatis

	return value, nil
}

// add 添加接口调用数据
func (a *APICallStatis) add(ac *APICall) {
	a.components[ac.component].add(ac)
}

const (
	maxLogWaitDuration = 800 * time.Millisecond
	maxZeroDuration    = 3
	metricsNumber      = 5
)

// log 打印接口调用统计
func (a *APICallStatis) log() {
	for name, component := range a.components {
		// server 的请求指标，同时输出到日志和 prometheus
		if name == plugin.ComponentServer {
			component.collect()
		} else {
			component.log()
		}
	}
}
