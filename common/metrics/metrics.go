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
	"net/http"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	registry = prometheus.NewRegistry()

	lastRedisReadFailureReport  atomic.Value
	lastRedisWriteFailureReport atomic.Value
)

// GetRegistry 获取 metrics 的 registry
func GetRegistry() *prometheus.Registry {
	return registry
}

// GetHttpHandler 获取 handler
func GetHttpHandler() http.Handler {
	// 添加 Golang runtime metrics
	registry.MustRegister(collectors.NewGoCollector(
		collectors.WithGoCollections(collectors.GoRuntimeMetricsCollection),
	))
	return promhttp.HandlerFor(registry, promhttp.HandlerOpts{EnableOpenMetrics: true})
}

// InitMetrics 初始化 metrics 的所有指标
func InitMetrics() {
	_ = registry.Register(instanceAsyncRegisCost)
	_ = registry.Register(instanceRegisTaskExpire)

	_ = registry.Register(redisReadFailure)
	_ = registry.Register(redisWriteFailure)
	_ = registry.Register(redisAliveStatus)

	_ = registry.Register(discoveryConnTotal)
	_ = registry.Register(configurationConnTotal)
	_ = registry.Register(sdkClientTotal)

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

// AddDiscoveryClientConn add discovery client connection number
func AddDiscoveryClientConn() {
	discoveryConnTotal.Inc()
}

// RemoveDiscoveryClientConn remove discovery client connection number
func RemoveDiscoveryClientConn() {
	discoveryConnTotal.Dec()
}

// ResetDiscoveryClientConn reset discovery client connection number
func ResetDiscoveryClientConn() {
	discoveryConnTotal.Set(0)
}

// AddConfigurationClientConn add configuration client connection number
func AddConfigurationClientConn() {
	configurationConnTotal.Inc()
}

// RemoveConfigurationClientConn remove configuration client connection number
func RemoveConfigurationClientConn() {
	configurationConnTotal.Dec()
}

// ResetConfigurationClientConn reset configuration client connection number
func ResetConfigurationClientConn() {
	configurationConnTotal.Set(0)
}

// AddSDKClient add client connection number
func AddSDKClient() {
	sdkClientTotal.Inc()
}

// RemoveSDKClient remove client connection number
func RemoveSDKClient() {
	sdkClientTotal.Dec()
}

// ResetSDKClient reset client connection number
func ResetSDKClient() {
	sdkClientTotal.Set(0)
}

// RecordCacheUpdateCost record per cache update cost time
func RecordCacheUpdateCost(cost time.Duration, cacheTye string, total int64) {
	cacheUpdateCost.With(map[string]string{
		labelCacheType: cacheTye,
		labelCacheUpdateCount: strconv.FormatInt(total, 10), 
	})
}
