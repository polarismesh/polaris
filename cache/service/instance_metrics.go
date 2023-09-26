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

package service

import (
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"

	types "github.com/polarismesh/polaris/cache/api"
	"github.com/polarismesh/polaris/common/metrics"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/plugin"
)

var preServiceInfos = map[string]map[string]struct{}{}

func (ic *instanceCache) reportMetricsInfo() {
	cacheMgr := ic.BaseCache.CacheMgr
	tmpServiceInfos := map[string]map[string]struct{}{}

	allServices := map[string]map[string]struct{}{}
	onlineService := map[string]map[string]struct{}{}
	offlineService := map[string]map[string]struct{}{}
	abnormalService := map[string]map[string]struct{}{}
	serviceCache := cacheMgr.GetCacher(types.CacheService).(*serviceCache)
	metricValues := make([]metrics.DiscoveryMetric, 0, 32)

	_ = serviceCache.IteratorServices(func(key string, svc *model.Service) (bool, error) {
		if _, ok := tmpServiceInfos[svc.Namespace]; !ok {
			tmpServiceInfos[svc.Namespace] = map[string]struct{}{}
		}
		tmpServiceInfos[svc.Namespace][svc.Name] = struct{}{}

		if _, ok := allServices[svc.Namespace]; !ok {
			allServices[svc.Namespace] = map[string]struct{}{}
		}
		allServices[svc.Namespace][svc.Name] = struct{}{}

		if _, ok := offlineService[svc.Namespace]; !ok {
			offlineService[svc.Namespace] = map[string]struct{}{}
		}
		countInfo := ic.GetInstancesCountByServiceID(svc.ID)
		if countInfo.TotalInstanceCount == 0 {
			offlineService[svc.Namespace][svc.Name] = struct{}{}
		}
		return true, nil
	})

	// instance count metrics
	ic.instanceCounts.ReadRange(func(serviceID string, countInfo *model.InstanceCount) {
		svc := serviceCache.GetServiceByID(serviceID)
		if svc == nil {
			log.Debug("[Cache][Instance] report metrics get service not found", zap.String("svc-id", serviceID))
			return
		}

		if _, ok := onlineService[svc.Namespace]; !ok {
			onlineService[svc.Namespace] = map[string]struct{}{}
		}
		if _, ok := abnormalService[svc.Namespace]; !ok {
			abnormalService[svc.Namespace] = map[string]struct{}{}
		}

		if countInfo.TotalInstanceCount != 0 && countInfo.HealthyInstanceCount == 0 {
			abnormalService[svc.Namespace][svc.Name] = struct{}{}
		}
		if countInfo.TotalInstanceCount != 0 && countInfo.HealthyInstanceCount > 0 {
			onlineService[svc.Namespace][svc.Name] = struct{}{}
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

		return
	})

	for ns := range allServices {
		metricValues = append(metricValues, metrics.DiscoveryMetric{
			Type:     metrics.ServiceMetrics,
			Total:    int64(len(allServices[ns])),
			Abnormal: int64(len(abnormalService[ns])),
			Offline:  int64(len(offlineService[ns])),
			Online:   int64(len(onlineService[ns])),
			Labels: map[string]string{
				metrics.LabelNamespace: ns,
			},
		})
	}

	cleanExpireServiceMetricLabel(preServiceInfos, tmpServiceInfos)
	preServiceInfos = tmpServiceInfos
	plugin.GetStatis().ReportDiscoveryMetrics(metricValues...)
}

func cleanExpireServiceMetricLabel(pre, curr map[string]map[string]struct{}) {
	if len(pre) == 0 {
		return
	}

	var (
		removeNs = map[string]struct{}{}
		remove   = map[string]map[string]struct{}{}
	)

	for ns, services := range pre {
		if _, ok := curr[ns]; !ok {
			removeNs[ns] = struct{}{}
		}
		if _, ok := remove[ns]; !ok {
			remove[ns] = map[string]struct{}{}
		}
		for service := range services {
			if _, ok := curr[ns][service]; !ok {
				remove[ns][service] = struct{}{}
			}
		}
	}

	for ns := range removeNs {
		metrics.GetServiceCount().Delete(prometheus.Labels{
			metrics.LabelNamespace: ns,
		})
		metrics.GetServiceOfflineCountl().Delete(prometheus.Labels{
			metrics.LabelNamespace: ns,
		})
		metrics.GetServiceOnlineCountl().Delete(prometheus.Labels{
			metrics.LabelNamespace: ns,
		})
		metrics.GetServiceAbnormalCountl().Delete(prometheus.Labels{
			metrics.LabelNamespace: ns,
		})
	}

	for ns, services := range remove {
		for service := range services {
			metrics.GetInstanceCount().Delete(prometheus.Labels{
				metrics.LabelNamespace: ns,
				metrics.LabelService:   service,
			})
			metrics.GetInstanceAbnormalCountl().Delete(prometheus.Labels{
				metrics.LabelNamespace: ns,
				metrics.LabelService:   service,
			})
			metrics.GetInstanceIsolateCountl().Delete(prometheus.Labels{
				metrics.LabelNamespace: ns,
				metrics.LabelService:   service,
			})
			metrics.GetInstanceOnlineCountl().Delete(prometheus.Labels{
				metrics.LabelNamespace: ns,
				metrics.LabelService:   service,
			})
		}
	}

}
