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

// PrometheusStatis is a struct for prometheus statistics
type PrometheusStatis struct {
	registry        *prometheus.Registry
	metricVecCaches map[string]interface{}
}

// NewPrometheusStatis 初始化 PrometheusStatis
func NewPrometheusStatis() (*PrometheusStatis, error) {
	statis := &PrometheusStatis{}
	statis.metricVecCaches = make(map[string]interface{})
	statis.registry = metrics.GetRegistry()

	err := statis.registerMetrics()
	if err != nil {
		return nil, err
	}

	return statis, nil
}

// registerMetrics Registers the interface invocation-related observability metrics
func (statis *PrometheusStatis) registerMetrics() error {
	for _, desc := range metricDescList {
		var collector prometheus.Collector
		switch desc.MetricType {
		case TypeForGaugeVec:
			collector = prometheus.NewGaugeVec(prometheus.GaugeOpts{
				Name: desc.Name,
				Help: desc.Help,
			}, desc.LabelNames)
		default:
			continue
		}

		err := statis.registry.Register(collector)
		if err != nil {
			log.Errorf("[APICall] register prometheus collector error, %v", err)
			return err
		}
		statis.metricVecCaches[desc.Name] = collector
	}
	return nil
}

func (a *APICallStatis) collectMetricData(staticsSlice []*APICallStatisItem) {
	if len(staticsSlice) == 0 {
		return
	}

	// 每一个接口，共 metricsNumber 个指标，下面的指标的个数调整时，这里的 metricsNumber 也需要调整
	statInfos := make([]*MetricData, 0, len(staticsSlice)*metricsNumber)

	for _, item := range staticsSlice {

		var maxTime, avgTime, rqTime, rqCount, minTime float64
		var deleteFlag bool

		if item.count == 0 && item.zeroDuration > maxZeroDuration {
			deleteFlag = true
		} else {
			maxTime = float64(item.maxTime) / 1e6
			minTime = float64(item.minTime) / 1e6
			rqTime = float64(item.accTime) / 1e6
			rqCount = float64(item.count)
			if item.count > 0 {
				avgTime = float64(item.accTime) / float64(item.count) / 1e6
			}
			deleteFlag = false
		}

		statInfos = append(statInfos, &MetricData{
			Name:       MetricForClientRqTimeoutMax,
			Data:       maxTime,
			Labels:     buildMetricLabels(nil, item),
			DeleteFlag: deleteFlag,
		}, &MetricData{
			Name:       MetricForClientRqTimeoutAvg,
			Data:       avgTime,
			Labels:     buildMetricLabels(nil, item),
			DeleteFlag: deleteFlag,
		}, &MetricData{
			Name:       MetricForClientRqTimeout,
			Data:       rqTime,
			Labels:     buildMetricLabels(nil, item),
			DeleteFlag: deleteFlag,
		}, &MetricData{
			Name:       MetricForClientRqIntervalCount,
			Data:       rqCount,
			Labels:     buildMetricLabels(nil, item),
			DeleteFlag: deleteFlag,
		}, &MetricData{
			Name:       MetricForClientRqTimeoutMin,
			Data:       minTime,
			Labels:     buildMetricLabels(nil, item),
			DeleteFlag: deleteFlag,
		},
		)
	}

	// 将指标收集到 prometheus
	a.prometheusStatis.collectMetricData(statInfos)
}

// collectMetricData
func (statis *PrometheusStatis) collectMetricData(statInfos []*MetricData) {

	if len(statInfos) == 0 {
		return
	}

	// prometheus-sdk 本身做了pull时数据一致性的保证，这里不需要自己在进行额外的保护动作

	// 清理掉当前的存量数据

	for _, statInfo := range statInfos {

		if len(statInfo.Labels) == 0 {
			statInfo.Labels = make(map[string]string)
		}

		// Set a public label information: Polaris node information
		statInfo.Labels[LabelForPolarisServerInstance] = utils.LocalHost
		metricVec := statis.metricVecCaches[statInfo.Name]

		if metricVec == nil {
			continue
		}

		switch metric := metricVec.(type) {
		case *prometheus.GaugeVec:
			if statInfo.DeleteFlag {
				metric.Delete(statInfo.Labels)
			} else {
				metric.With(statInfo.Labels).Set(statInfo.Data)
			}
		}
	}

}

// GetRegistry return prometheus.Registry instance
func (statis *PrometheusStatis) GetRegistry() *prometheus.Registry {
	return statis.registry
}
