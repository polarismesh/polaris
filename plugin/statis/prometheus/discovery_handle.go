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
	"github.com/polarismesh/polaris/common/metrics"
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
			metrics.GetServiceCount().With(m.Labels).Set(float64(m.Total))
			metrics.GetServiceAbnormalCountl().With(m.Labels).Set(float64(m.Abnormal))
			metrics.GetServiceOfflineCountl().With(m.Labels).Set(float64(m.Offline))
			metrics.GetServiceOnlineCountl().With(m.Labels).Set(float64(m.Online))
		case metrics.InstanceMetrics:
			metrics.GetInstanceCount().With(m.Labels).Set(float64(m.Total))
			metrics.GetInstanceAbnormalCountl().With(m.Labels).Set(float64(m.Abnormal))
			metrics.GetInstanceIsolateCountl().With(m.Labels).Set(float64(m.Isolate))
			metrics.GetInstanceOnlineCountl().With(m.Labels).Set(float64(m.Online))
		case metrics.ClientMetrics:
			metrics.GetClientInstanceTotal().Set(float64(m.Total))
		}
	}
}
