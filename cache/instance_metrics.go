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

package cache

import (
	"github.com/polarismesh/polaris/common/metrics"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/plugin"
	"go.uber.org/zap"
)

func (ic *instanceCache) reportMetricsInfo() {
	cacheMgr, err := GetCacheManager()
	if err != nil {
		log.Warn("[Cache][Instance] report metrics get cache manager, but impossible", zap.Error(err))
		return
	}
	serviceCache := cacheMgr.Service()

	metricValues := make([]metrics.DiscoveryMetric, 0, 32)

	// instance count metrics
	ic.instanceCounts.Range(func(key, value any) bool {
		serviceID := key.(string)
		countInfo := value.(*model.InstanceCount)

		svc := serviceCache.GetServiceByID(serviceID)
		if svc == nil {
			log.Debug("[Cache][Instance] report metrics get service not found", zap.String("svc-id", serviceID))
			return true
		}

		metricValues = append(metricValues, metrics.DiscoveryMetric{
			Type:     metrics.InstanceMetrics,
			Total:    int64(countInfo.TotalInstanceCount),
			Abnormal: int64(countInfo.TotalInstanceCount - countInfo.HealthyInstanceCount),
			Online:   int64(countInfo.HealthyInstanceCount),
			Isolate:  int64(countInfo.IsolateInstanceCount),
			Labels: map[string]string{
				metrics.LabelNamespace: svc.Namespace,
				metrics.LabelService:   svc.Name,
			},
		})

		return true
	})

	plugin.GetStatis().ReportDiscoveryMetrics(metricValues...)
}
