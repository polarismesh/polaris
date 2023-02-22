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
	"github.com/prometheus/client_golang/prometheus"

	"github.com/polarismesh/polaris/common/metrics"
	"github.com/polarismesh/polaris/common/utils"
)

func init() {
	metrics.GetRegistry().MustRegister([]prometheus.Collector{
		serviceCount,
		serviceOnlineCount,
		serviceAbnormalCount,
		serviceOfflineCount,

		instanceCount,
		instanceOnlineCount,
		instanceAbnormalCount,
		instanceIsolateCount,

		clientInstanceTotal,
	}...)
}

var (
	clientInstanceTotal = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "client_total",
		Help: "polaris client instance total number",
		ConstLabels: map[string]string{
			metrics.LabelServerNode: utils.LocalHost,
		},
	})
)

// service metrics
var (
	serviceCount = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "service_count",
		Help: "service total number",
		ConstLabels: map[string]string{
			metrics.LabelServerNode: utils.LocalHost,
		},
	}, []string{metrics.LabelNamespace})

	serviceOnlineCount = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "service_online_count",
		Help: "total number of service status is online",
		ConstLabels: map[string]string{
			metrics.LabelServerNode: utils.LocalHost,
		},
	}, []string{metrics.LabelNamespace})

	serviceAbnormalCount = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "service_abnormal_count",
		Help: "total number of service status is abnormal",
		ConstLabels: map[string]string{
			metrics.LabelServerNode: utils.LocalHost,
		},
	}, []string{metrics.LabelNamespace})

	serviceOfflineCount = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "service_offline_count",
		Help: "total number of service status is offline",
		ConstLabels: map[string]string{
			metrics.LabelServerNode: utils.LocalHost,
		},
	}, []string{metrics.LabelNamespace})
)

// instance metrics
var (
	instanceCount = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "instance_count",
		Help: "instance total number",
		ConstLabels: map[string]string{
			metrics.LabelServerNode: utils.LocalHost,
		},
	}, []string{metrics.LabelNamespace, metrics.LabelService})

	instanceOnlineCount = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "instance_online_count",
		Help: "total number of instance status is health",
		ConstLabels: map[string]string{
			metrics.LabelServerNode: utils.LocalHost,
		},
	}, []string{metrics.LabelNamespace, metrics.LabelService})

	instanceAbnormalCount = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "instance_abnormal_count",
		Help: "total number of instance status is unhealth",
		ConstLabels: map[string]string{
			metrics.LabelServerNode: utils.LocalHost,
		},
	}, []string{metrics.LabelNamespace, metrics.LabelService})

	instanceIsolateCount = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "instance_isolate_count",
		Help: "total number of instance status is isolate",
		ConstLabels: map[string]string{
			metrics.LabelServerNode: utils.LocalHost,
		},
	}, []string{metrics.LabelNamespace, metrics.LabelService})
)

func newDiscoveryMetricHandle() *discoveryMetricHandle {
	return &discoveryMetricHandle{}
}

type discoveryMetricHandle struct {
}

func (h *discoveryMetricHandle) handle(ms []metrics.DiscoveryMetric) {
	for i := range ms {
		m := ms[i]
		switch m.Type {
		case metrics.ServiceMetrics:
			serviceCount.With(m.Labels).Set(float64(m.Total))
			serviceAbnormalCount.With(m.Labels).Set(float64(m.Abnormal))
			serviceOfflineCount.With(m.Labels).Set(float64(m.Offline))
			serviceOnlineCount.With(m.Labels).Set(float64(m.Online))
		case metrics.InstanceMetrics:
			instanceCount.With(m.Labels).Set(float64(m.Total))
			instanceAbnormalCount.With(m.Labels).Set(float64(m.Abnormal))
			instanceIsolateCount.With(m.Labels).Set(float64(m.Isolate))
			instanceOnlineCount.With(m.Labels).Set(float64(m.Online))
		case metrics.ClientMetrics:
			clientInstanceTotal.Set(float64(m.Total))
		}
	}
}
