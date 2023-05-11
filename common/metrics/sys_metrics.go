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

package metrics

import (
	"strconv"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/polarismesh/polaris/common/utils"
)

var (
	lastRedisReadFailureReport  atomic.Value
	lastRedisWriteFailureReport atomic.Value
)

func registerSysMetrics() {
	// instanceAsyncRegisCost 实例异步注册任务耗费时间
	instanceAsyncRegisCost = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name: "instance_regis_cost_time",
		Help: "instance regis cost time",
		ConstLabels: map[string]string{
			LabelServerNode: utils.LocalHost,
		},
	})

	// instanceRegisTaskExpire 实例异步注册任务超时无效事件
	instanceRegisTaskExpire = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "instance_regis_task_expire",
		Help: "instance regis task expire that server drop it",
		ConstLabels: map[string]string{
			LabelServerNode: utils.LocalHost,
		},
	})

	redisReadFailure = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "redis_read_failure",
		Help: "polaris exec redis read operation failure",
		ConstLabels: map[string]string{
			LabelServerNode: utils.LocalHost,
		},
	})

	redisWriteFailure = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "redis_write_failure",
		Help: "polaris exec redis write operation failure",
		ConstLabels: map[string]string{
			LabelServerNode: utils.LocalHost,
		},
	})

	redisAliveStatus = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "redis_alive_status",
		Help: "polaris redis alive status",
		ConstLabels: map[string]string{
			"polaris_server_instance": utils.LocalHost,
		},
	})

	cacheUpdateCost = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "cache_update_cost",
		Help: "cache update cost per resource cache",
		ConstLabels: map[string]string{
			"polaris_server_instance": utils.LocalHost,
		},
	}, []string{labelCacheType, labelCacheUpdateCount})

	batchJobUnFinishJobs = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "batch_job_unfinish",
		Help: "count unfinish batch job",
		ConstLabels: map[string]string{
			"polaris_server_instance": utils.LocalHost,
		},
	}, []string{
		labelBatchJobLabel,
	})

	_ = registry.Register(instanceAsyncRegisCost)
	_ = registry.Register(instanceRegisTaskExpire)
	_ = registry.Register(redisReadFailure)
	_ = registry.Register(redisWriteFailure)
	_ = registry.Register(redisAliveStatus)
	_ = registry.Register(cacheUpdateCost)
	_ = registry.Register(batchJobUnFinishJobs)

	go func() {
		lastRedisReadFailureReport.Store(time.Now())
		lastRedisWriteFailureReport.Store(time.Now())
		ticker := time.NewTicker(time.Minute)
		for range ticker.C {
			tn := time.Now()
			if tn.Sub(lastRedisReadFailureReport.Load().(time.Time)) > time.Minute {
				redisReadFailure.Set(0)
			}
			if tn.Sub(lastRedisWriteFailureReport.Load().(time.Time)) > time.Minute {
				redisWriteFailure.Set(0)
			}
		}
	}()
}

// ReportInstanceRegisCost Total time to report the short-term registered task of the reporting instance
func ReportInstanceRegisCost(cost time.Duration) {
	instanceAsyncRegisCost.Observe(float64(cost.Milliseconds()))
}

// ReportDropInstanceRegisTask Record the number of registered tasks discarded
func ReportDropInstanceRegisTask() {
	instanceRegisTaskExpire.Inc()
}

// ReportRedisReadFailure report redis exec read operatio failure
func ReportRedisReadFailure() {
	lastRedisReadFailureReport.Store(time.Now())
	redisReadFailure.Inc()
}

// ReportRedisWriteFailure report redis exec write operatio failure
func ReportRedisWriteFailure() {
	lastRedisWriteFailureReport.Store(time.Now())
	redisWriteFailure.Inc()
}

// ReportRedisIsDead report redis alive status is dead
func ReportRedisIsDead() {
	redisAliveStatus.Set(0)
}

// ReportRedisIsAlive report redis alive status is health
func ReportRedisIsAlive() {
	redisAliveStatus.Set(1)
}

// RecordCacheUpdateCost record per cache update cost time
func RecordCacheUpdateCost(cost time.Duration, cacheTye string, total int64) {
	if cacheUpdateCost == nil {
		return
	}
	cacheUpdateCost.With(map[string]string{
		labelCacheType:        cacheTye,
		labelCacheUpdateCount: strconv.FormatInt(total, 10),
	})
}

// ReportAddBatchJob .
func ReportAddBatchJob(label string, count int64) {
	if batchJobUnFinishJobs == nil {
		return
	}
	batchJobUnFinishJobs.With(map[string]string{
		labelBatchJobLabel: label,
	}).Add(float64(count))
}

// ReportFinishBatchJob .
func ReportFinishBatchJob(label string, count int64) {
	if batchJobUnFinishJobs == nil {
		return
	}
	batchJobUnFinishJobs.With(map[string]string{
		labelBatchJobLabel: label,
	}).Sub(float64(count))
}
