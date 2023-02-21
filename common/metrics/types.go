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
	"github.com/prometheus/client_golang/prometheus"

	"github.com/polarismesh/polaris/common/utils"
)

const (
	labelServerNode       = "polaris_server_instance"
	labelNamespace        = "namespace"
	labelService          = "service"
	labelServiceStatus    = "status"
	labelVersion          = "version"
	labelApi              = "api"
	labelApiType          = "api_type"
	labelProtocol         = "protocol"
	labelErrCode          = "err_code"
	labelCacheType        = "cache_type"
	labelCacheUpdateCount = "cache_update_count"
)

var (
	metricsPort int32
)

func SetMetricsPort(port int32) {
	metricsPort = port
}

func GetMetricsPort() int32 {
	return metricsPort
}

// instance astbc registry metrics
var (
	// instanceAsyncRegisCost 实例异步注册任务耗费时间
	instanceAsyncRegisCost = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name: "instance_regis_cost_time",
		Help: "instance regis cost time",
		ConstLabels: map[string]string{
			labelServerNode: utils.LocalHost,
		},
	})

	// instanceRegisTaskExpire 实例异步注册任务超时无效事件
	instanceRegisTaskExpire = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "instance_regis_task_expire",
		Help: "instance regis task expire that server drop it",
		ConstLabels: map[string]string{
			labelServerNode: utils.LocalHost,
		},
	})
)

// redis metrics
var (
	redisReadFailure = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "redis_read_failure",
		Help: "polaris exec redis read operation failure",
		ConstLabels: map[string]string{
			labelServerNode: utils.LocalHost,
		},
	})

	redisWriteFailure = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "redis_write_failure",
		Help: "polaris exec redis write operation failure",
		ConstLabels: map[string]string{
			labelServerNode: utils.LocalHost,
		},
	})

	redisAliveStatus = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "redis_alive_status",
		Help: "polaris redis alive status",
		ConstLabels: map[string]string{
			"polaris_server_instance": utils.LocalHost,
		},
	})
)

// client tcp connection metrics
var (
	// discoveryConnTotal 服务发现客户端链接数量
	discoveryConnTotal = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "discovery_conn_total",
		Help: "polaris discovery client connection total",
		ConstLabels: map[string]string{
			labelServerNode: utils.LocalHost,
		},
	})

	// configurationConnTotal 配置中心客户端链接数量
	configurationConnTotal = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "config_conn_total",
		Help: "polaris configuration client connection total",
		ConstLabels: map[string]string{
			labelServerNode: utils.LocalHost,
		},
	})

	// sdkClientTotal 客户端链接数量
	sdkClientTotal = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "sdk_client_total",
		Help: "polaris client connection total",
		ConstLabels: map[string]string{
			labelServerNode: utils.LocalHost,
		},
	})
)

// sdk instance metrics
var (
	clientInstanceTotal = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "client_total",
		Help: "polaris client instance total number",
		ConstLabels: map[string]string{
			labelServerNode: utils.LocalHost,
		},
	})
)

// service metrics
var (
	serviceCount = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "service_count",
		Help: "service total number",
		ConstLabels: map[string]string{
			labelServerNode: utils.LocalHost,
		},
	}, []string{labelNamespace})

	serviceOnlineCount = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "service_online_count",
		Help: "total number of service status is online",
		ConstLabels: map[string]string{
			labelServerNode: utils.LocalHost,
		},
	}, []string{labelNamespace})

	serviceAbnormalCount = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "service_abnormal_count",
		Help: "total number of service status is abnormal",
		ConstLabels: map[string]string{
			labelServerNode: utils.LocalHost,
		},
	}, []string{labelNamespace})

	serviceOfflineCount = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "service_offline_count",
		Help: "total number of service status is offline",
		ConstLabels: map[string]string{
			labelServerNode: utils.LocalHost,
		},
	}, []string{labelNamespace})
)

// instance metrics
var (
	instanceCount = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "instance_count",
		Help: "instance total number",
		ConstLabels: map[string]string{
			labelServerNode: utils.LocalHost,
		},
	}, []string{labelNamespace, labelService})

	instanceOnlineCount = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "instance_online_count",
		Help: "total number of instance status is health",
		ConstLabels: map[string]string{
			labelServerNode: utils.LocalHost,
		},
	}, []string{labelNamespace, labelService})

	instanceAbnormalCount = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "instance_abnormal_count",
		Help: "total number of instance status is unhealth",
		ConstLabels: map[string]string{
			labelServerNode: utils.LocalHost,
		},
	}, []string{labelNamespace, labelService})

	instanceIsolateCount = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "instance_isolate_count",
		Help: "total number of instance status is isolate",
		ConstLabels: map[string]string{
			labelServerNode: utils.LocalHost,
		},
	}, []string{labelNamespace, labelService})
)

var (
	MetricForClientRqTotal         string = "client_rq_total"
	MetricForClientRqFailure       string = "client_rq_failure"
	MetricForClientRqTimeout       string = "client_rq_timeout"
	MetricForClientRqIntervalCount string = "client_rq_interval_count"
	MetricForClientRqTimeoutMin    string = "client_rq_timeout_min"
	MetricForClientRqTimeoutAvg    string = "client_rq_timeout_avg"
	MetricForClientRqTimeoutMax    string = "client_rq_timeout_max"
	MetricForClientRqTimeoutP99    string = "client_rq_timeout_p99"

	clientRequestCount = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "client_rq_total",
		Help: "total number of client request",
		ConstLabels: map[string]string{
			labelServerNode: utils.LocalHost,
		},
	}, []string{labelApi, labelApiType, labelErrCode, labelProtocol})

	clientRequestFailureCount = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "client_rq_failure",
		Help: "total number of client request status is failure",
		ConstLabels: map[string]string{
			labelServerNode: utils.LocalHost,
		},
	}, []string{labelApi, labelApiType, labelErrCode, labelProtocol})

	clientRequestTime = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "client_rq_time",
		Help: "cost time of per client request",
		ConstLabels: map[string]string{
			labelServerNode: utils.LocalHost,
		},
	}, []string{labelApi, labelApiType, labelErrCode, labelProtocol})
)

var (
	cacheUpdateCost = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "cache_update_cost",
		Help: "cache update cost per resource cache",
		ConstLabels: map[string]string{
			"polaris_server_instance": utils.LocalHost,
		},
	}, []string{labelCacheType, labelCacheUpdateCount})
)
