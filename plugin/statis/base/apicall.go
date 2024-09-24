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
	"math"
	"sync"
	"time"

	"github.com/polarismesh/polaris/common/log"
	"github.com/polarismesh/polaris/common/metrics"
)

// APICall 接口调用
type APICall struct {
	Count            int
	Api              string
	Code             int
	Duration         int64
	Protocol         string
	TrafficDirection string
	Component        metrics.CallMetricType
}

// APICallStatisItem 接口调用统计条目
type APICallStatisItem struct {
	API              string
	TrafficDirection string
	Code             int
	Count            int64
	AccTime          int64
	MinTime          int64
	MaxTime          int64
	Protocol         string
	ZeroDuration     int64 // 没有请求持续的时间，持续时间长超过阈值从 prometheus 中移除掉
}

// ComponentStatics statics components
type ComponentStatics struct {
	t       metrics.CallMetricType
	acc     chan *APICall
	mutex   sync.Mutex
	statis  map[string]*APICallStatisItem
	handler MetricsHandler
}

func NewComponentStatics(ctx context.Context, t metrics.CallMetricType, handler MetricsHandler) *ComponentStatics {
	c := &ComponentStatics{
		t:       t,
		acc:     make(chan *APICall, 1024),
		statis:  make(map[string]*APICallStatisItem),
		handler: handler,
	}
	go c.run(ctx)
	return c
}

// add 添加接口调用数据
func (a *ComponentStatics) Add(ac *APICall) {
	startTime := time.Now()
	select {
	case a.acc <- ac:
		passDuration := time.Since(startTime)
		if passDuration >= MaxAddDuration {
			log.Warnf("[APICall]add api call cost %s, exceed max %s", passDuration, MaxAddDuration)
		}
	default:
		// quick return.
	}
}

// add 添加接口调用数据
func (a *ComponentStatics) run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case item := <-a.acc:
			a.add(item)
		}
	}
}

func (c *ComponentStatics) add(ac *APICall) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	index := fmt.Sprintf("%v-%v", ac.Api, ac.Code)
	item, exist := c.statis[index]
	if exist {
		item.Count++

		item.AccTime += ac.Duration
		if ac.Duration < item.MinTime {
			item.MinTime = ac.Duration
		}
		if ac.Duration > item.MaxTime {
			item.MaxTime = ac.Duration
		}
	} else {
		c.statis[index] = &APICallStatisItem{
			API:              ac.Api,
			Code:             ac.Code,
			Count:            int64(ac.Count),
			AccTime:          ac.Duration,
			MinTime:          ac.Duration,
			MaxTime:          ac.Duration,
			Protocol:         ac.Protocol,
			TrafficDirection: ac.TrafficDirection,
		}
	}
}

// collect log and print the statics messages
func (c *ComponentStatics) deal() {
	startTime := time.Now()
	if len(c.statis) == 0 {
		c.mutex.Lock()
		defer c.mutex.Unlock()
		c.handler(c.t, startTime, nil)
		return
	}

	c.mutex.Lock()
	defer func() {
		c.mutex.Unlock()
		passDuration := time.Since(startTime)
		if passDuration >= MaxLogWaitDuration {
			log.Warnf("[APICall]api static log duration %s, pass max %s", passDuration, MaxLogWaitDuration)
		}
	}()

	duplicateStatis := make([]*APICallStatisItem, 0, len(c.statis))
	currStatis := c.statis
	c.statis = make(map[string]*APICallStatisItem)

	for key, item := range currStatis {
		duplicateStatis = append(duplicateStatis, item)
		if item.Count == 0 {
			item.ZeroDuration++
			item.MinTime = 0
		} else {
			item.ZeroDuration = 0
		}

		if item.ZeroDuration <= MaxZeroDuration {
			c.statis[key] = &APICallStatisItem{
				API:              item.API,
				Code:             item.Code,
				Count:            0,
				Protocol:         item.Protocol,
				AccTime:          0,
				MinTime:          math.MaxInt64,
				MaxTime:          0,
				ZeroDuration:     item.ZeroDuration,
				TrafficDirection: item.TrafficDirection,
			}
		}
	}

	go func() {
		c.mutex.Lock()
		defer c.mutex.Unlock()
		c.handler(c.t, startTime, duplicateStatis)
	}()
}
