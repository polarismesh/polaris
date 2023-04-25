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

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	registry = prometheus.NewRegistry()
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

var (
	metricsPort int32
)

func SetMetricsPort(port int32) {
	metricsPort = port
}

func GetMetricsPort() int32 {
	return metricsPort
}

// InitMetrics 初始化 metrics 的所有指标
func InitMetrics() {
	registerSysMetrics()
	registerClientMetrics()
	registerConfigFileMetrics()
	registerDiscoveryMetrics()
}
