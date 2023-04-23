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

func registerDiscoveryMetrics() {
	clientInstanceTotal = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "client_total",
		Help: "polaris client instance total number",
		ConstLabels: map[string]string{
			LabelServerNode: utils.LocalHost,
		},
	})

	serviceCount = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "service_count",
		Help: "service total number",
		ConstLabels: map[string]string{
			LabelServerNode: utils.LocalHost,
		},
	}, []string{LabelNamespace})

	serviceOnlineCount = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "service_online_count",
		Help: "total number of service status is online",
		ConstLabels: map[string]string{
			LabelServerNode: utils.LocalHost,
		},
	}, []string{LabelNamespace})

	serviceAbnormalCount = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "service_abnormal_count",
		Help: "total number of service status is abnormal",
		ConstLabels: map[string]string{
			LabelServerNode: utils.LocalHost,
		},
	}, []string{LabelNamespace})

	serviceOfflineCount = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "service_offline_count",
		Help: "total number of service status is offline",
		ConstLabels: map[string]string{
			LabelServerNode: utils.LocalHost,
		},
	}, []string{LabelNamespace})

	instanceCount = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "instance_count",
		Help: "instance total number",
		ConstLabels: map[string]string{
			LabelServerNode: utils.LocalHost,
		},
	}, []string{LabelNamespace, LabelService})

	instanceOnlineCount = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "instance_online_count",
		Help: "total number of instance status is health",
		ConstLabels: map[string]string{
			LabelServerNode: utils.LocalHost,
		},
	}, []string{LabelNamespace, LabelService})

	instanceAbnormalCount = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "instance_abnormal_count",
		Help: "total number of instance status is unhealth",
		ConstLabels: map[string]string{
			LabelServerNode: utils.LocalHost,
		},
	}, []string{LabelNamespace, LabelService})

	instanceIsolateCount = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "instance_isolate_count",
		Help: "total number of instance status is isolate",
		ConstLabels: map[string]string{
			LabelServerNode: utils.LocalHost,
		},
	}, []string{LabelNamespace, LabelService})

	_ = GetRegistry().Register(serviceCount)
	_ = GetRegistry().Register(serviceOnlineCount)
	_ = GetRegistry().Register(serviceAbnormalCount)
	_ = GetRegistry().Register(serviceOfflineCount)
	_ = GetRegistry().Register(instanceCount)
	_ = GetRegistry().Register(instanceOnlineCount)
	_ = GetRegistry().Register(instanceAbnormalCount)
	_ = GetRegistry().Register(instanceIsolateCount)
	_ = GetRegistry().Register(clientInstanceTotal)
}

func GetClientInstanceTotal() prometheus.Gauge {
	return clientInstanceTotal
}

func GetServiceCount() *prometheus.GaugeVec {
	return serviceCount
}

func GetServiceOnlineCountl() *prometheus.GaugeVec {
	return serviceOnlineCount
}

func GetServiceOfflineCountl() *prometheus.GaugeVec {
	return serviceOfflineCount
}

func GetServiceAbnormalCountl() *prometheus.GaugeVec {
	return serviceAbnormalCount
}

func GetInstanceCount() *prometheus.GaugeVec {
	return instanceCount
}

func GetInstanceOnlineCountl() *prometheus.GaugeVec {
	return instanceOnlineCount
}

func GetInstanceIsolateCountl() *prometheus.GaugeVec {
	return instanceIsolateCount
}

func GetInstanceAbnormalCountl() *prometheus.GaugeVec {
	return instanceAbnormalCount
}
