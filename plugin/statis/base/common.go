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
	"fmt"
	"time"

	"github.com/polarismesh/polaris/common/metrics"
)

const (
	MaxLogWaitDuration = 800 * time.Millisecond
	MaxZeroDuration    = 3
	MetricsNumber      = 5
	MaxAddDuration     = 800 * time.Millisecond
)

type MetricsHandler func(mt metrics.CallMetricType, start time.Time, staticsSlice []*APICallStatisItem)

// MetricData metric 结构体
type MetricData struct {
	Name       string
	Data       float64
	Labels     map[string]string
	DeleteFlag bool
}

type metricDesc struct {
	Name       string
	Help       string
	MetricType string
	LabelNames []string
}

const (
	// metric name
	// MetricForClientRqTimeout time consumed per interface call
	MetricForClientRqTimeout string = "client_rq_timeout"
	// MetricForClientRqIntervalCount total number of client request in current interval
	MetricForClientRqIntervalCount string = "client_rq_interval_count"
	// MetricForClientRqTimeoutMin max latency of client requests
	MetricForClientRqTimeoutMin string = "client_rq_timeout_min"
	// MetricForClientRqTimeoutAvg min latency of client requests
	MetricForClientRqTimeoutAvg string = "client_rq_timeout_avg"
	// MetricForClientRqTimeoutMax average latency of client requests
	MetricForClientRqTimeoutMax string = "client_rq_timeout_max"

	// metric type
	TypeForGaugeVec string = "gauge_vec"
)

var (
	// metricDescList Metrics Description Defines the list
	MetricDescList = []metricDesc{
		{
			Name:       MetricForClientRqTimeout,
			Help:       "time consumed per interface call",
			MetricType: TypeForGaugeVec,
			LabelNames: []string{
				metrics.LabelApi,
				metrics.LabelProtocol,
				metrics.LabelErrCode,
			},
		},
		{
			Name:       MetricForClientRqIntervalCount,
			Help:       "total number of client request in current interval",
			MetricType: TypeForGaugeVec,
			LabelNames: []string{
				metrics.LabelApi,
				metrics.LabelProtocol,
				metrics.LabelErrCode,
			},
		},
		{
			Name:       MetricForClientRqTimeoutMax,
			Help:       "max latency of client requests",
			MetricType: TypeForGaugeVec,
			LabelNames: []string{
				metrics.LabelApi,
				metrics.LabelProtocol,
				metrics.LabelErrCode,
			},
		},
		{
			Name:       MetricForClientRqTimeoutMin,
			Help:       "min latency of client requests",
			MetricType: TypeForGaugeVec,
			LabelNames: []string{
				metrics.LabelApi,
				metrics.LabelProtocol,
				metrics.LabelErrCode,
			},
		},
		{
			Name:       MetricForClientRqTimeoutAvg,
			Help:       "average latency of client requests",
			MetricType: TypeForGaugeVec,
			LabelNames: []string{
				metrics.LabelApi,
				metrics.LabelProtocol,
				metrics.LabelErrCode,
			},
		},
	}
)

// BuildMetricLabels build metric label from APICall or APICallStatisItem
func BuildMetricLabels(item *APICallStatisItem) map[string]string {
	return map[string]string{
		metrics.LabelErrCode:  fmt.Sprintf("%d", item.Code),
		metrics.LabelApi:      item.API,
		metrics.LabelProtocol: item.Protocol,
	}
}
