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

func registerClientMetrics() {
	// discoveryConnTotal 服务发现客户端链接数量
	discoveryConnTotal = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "discovery_conn_total",
		Help: "polaris discovery client connection total",
		ConstLabels: map[string]string{
			LabelServerNode: utils.LocalHost,
		},
	})

	// configurationConnTotal 配置中心客户端链接数量
	configurationConnTotal = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "config_conn_total",
		Help: "polaris configuration client connection total",
		ConstLabels: map[string]string{
			LabelServerNode: utils.LocalHost,
		},
	})

	// sdkClientTotal 客户端链接数量
	sdkClientTotal = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "sdk_client_total",
		Help: "polaris client connection total",
		ConstLabels: map[string]string{
			LabelServerNode: utils.LocalHost,
		},
	})

	_ = GetRegistry().Register(discoveryConnTotal)
	_ = GetRegistry().Register(configurationConnTotal)
	_ = GetRegistry().Register(sdkClientTotal)
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

// AddSDKClientConn add client connection number
func AddSDKClientConn() {
	sdkClientTotal.Inc()
}

// RemoveSDKClientConn remove client connection number
func RemoveSDKClientConn() {
	sdkClientTotal.Dec()
}

// Conn reset client connection number
func ResetSDKClientConn() {
	sdkClientTotal.Set(0)
}
