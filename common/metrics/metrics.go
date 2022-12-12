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
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	registry = prometheus.NewRegistry()
)

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

func InitMetrics() {
	_ = registry.Register(instanceAsyncRegisCost)
	_ = registry.Register(instanceRegisTaskExpire)

	_ = registry.Register(redisReadFailure)
	_ = registry.Register(redisWriteFailure)
	_ = registry.Register(redisAliveStatus)
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
	redisReadFailure.Inc()
}

// ReportRedisWriteFailure report redis exec write operatio failure
func ReportRedisWriteFailure() {
	redisWriteFailure.Inc()
}

// ReportRedisIsDead report redis alive status is dead
func ReportRedisIsDead() {
	redisAliveStatus.Set(0)
}

// ReportRedisIsDead report redis alive status is health
func ReportRedisIsAlive() {
	redisAliveStatus.Set(1)
}
