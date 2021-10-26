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

package prometheus

import (
	"fmt"

	"go.uber.org/zap"
)

var (
	// 统计p99，p999
	sqrtExps     = []float32{1, 2, 4, 8, 16, 32, 64, 128, 256, 512, 1024}
	sqrtTimeMs   = []int64{1, 2, 4, 8, 16, 32, 64, 128, 256, 512, 1024}
	enableBucket = true
)

/**
 * @brief 接口调用
 */
type APICall struct {
	api      string
	code     int
	duration int64
	protocol string
}

/**
 * @brief 接口调用统计条目
 */
type APICallStatisItem struct {
	api      string
	code     int
	count    int64
	protocol string
	accTime  int64
	minTime  int64
	maxTime  int64
	p99Time  int64
	buckets  []int64
}

/**
 * @brief 接口调用统计
 */
type APICallStatis struct {
	items  map[string]*APICallStatisItem
	logger *zap.Logger
	statis *PrometheusStatis
}

/**
 * @brief 添加接口调用数据
 */
func (a *APICallStatis) add(ac *APICall) {
	index := fmt.Sprintf("%v-%v", ac.api, ac.code)

	item, exist := a.items[index]
	if exist {
		item.count++

		item.accTime += ac.duration
		if ac.duration < item.minTime {
			item.minTime = ac.duration
		}
		if ac.duration > item.maxTime {
			item.maxTime = ac.duration
		}
		a.updateItemBuckets(item, ac.duration)
	} else {
		a.items[index] = &APICallStatisItem{
			api:      ac.api,
			code:     ac.code,
			count:    1,
			protocol: ac.protocol,
			accTime:  ac.duration,
			minTime:  ac.duration,
			maxTime:  ac.duration,
			buckets:  make([]int64, len(sqrtExps)+1),
		}
		a.updateItemBuckets(a.items[index], ac.duration)
	}
}

func (a *APICallStatis) updateItemBuckets(item *APICallStatisItem, dur int64) {
	if !enableBucket {
		return
	}
	durMs := float32(dur) / 1e6
	for i := 0; i < len(sqrtExps); i++ {
		if durMs <= sqrtExps[i] {
			item.buckets[i]++
			return
		}
	}
	item.buckets[len(sqrtExps)]++
}

func (a *APICallStatis) calcuItemBuckets(item *APICallStatisItem) {
	if !enableBucket || item.buckets == nil || len(item.buckets) < len(sqrtTimeMs) {
		return
	}
	var acc int64
	p99 := false
	for i, tms := range sqrtTimeMs {
		acc += item.buckets[i]
		if !p99 && float64(acc)/float64(item.count) >= 0.99 {
			p99 = true
			item.p99Time = tms
		}
	}
	if !p99 {
		item.p99Time = 2048
	}
}

/**
 * @brief 打印接口调用统计
 */
func (a *APICallStatis) log() {
	if len(a.items) == 0 {
		return
	}

	msg := "Statis:\n"

	msg += fmt.Sprintf("%-36v|%12v|%12v|%12v|%12v|%12v|%8v|\n", "",
		"Code", "Count", "Min(ms)", "Max(ms)", "Avg(ms)", "P99(ms)")

	for _, item := range a.items {
		a.calcuItemBuckets(item)
		msg += fmt.Sprintf("%-36v|%12v|%12v|%12.3f|%12.3f|%12.3f|%8.3f|\n",
			item.api, item.code, item.count,
			float32(item.minTime)/1e6,
			float32(item.maxTime)/1e6,
			float32(item.accTime)/float32(item.count)/1e6,
			float32(item.p99Time),
		)
	}

	a.logger.Info(msg)

	a.items = make(map[string]*APICallStatisItem)
}

/**
 * @brief 接口调用统计上报至 prometheus
 */
func (a *APICallStatis) collectMetricData() {
	if len(a.items) == 0 {
		return
	}

	statInfos := make([]*MetricData, 0)

	for _, item := range a.items {
		a.calcuItemBuckets(item)
		statInfos = append(statInfos, &MetricData{
			Name:   MetricForClientRqTimeoutMin,
			Data:   float64(item.minTime) / 1e6,
			Labels: buildMetricLabels(nil, item),
		}, &MetricData{
			Name:   MetricForClientRqTimeoutMax,
			Data:   float64(item.maxTime) / 1e6,
			Labels: buildMetricLabels(nil, item),
		}, &MetricData{
			Name:   MetricForClientRqTimeoutAvg,
			Data:   float64(item.accTime) / float64(item.count) / 1e6,
			Labels: buildMetricLabels(nil, item),
		}, &MetricData{
			Name:   MetricForClientRqTimeoutAvg,
			Data:   float64(item.p99Time),
			Labels: buildMetricLabels(nil, item),
		})
	}

	a.statis.collectMetricData(statInfos...)

	a.items = make(map[string]*APICallStatisItem)
}
