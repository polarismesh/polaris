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
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/polarismesh/polaris/common/log"
	"github.com/polarismesh/polaris/common/metrics"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/plugin"
	"github.com/polarismesh/polaris/plugin/statis/base"
)

const (
	PluginName = "prometheus"
)

func init() {
	s := &StatisWorker{}
	plugin.RegisterPlugin(s.Name(), s)
}

// PrometheusStatis is a struct for prometheus statistics
type StatisWorker struct {
	*base.BaseWorker
	cancel           context.CancelFunc
	discoveryHandler *discoveryMetricHandle
	configHandler    *configMetricHandle
	metricVecCaches  map[string]*prometheus.GaugeVec
}

// Name 获取统计插件名称
func (s *StatisWorker) Name() string {
	return PluginName
}

// Initialize 初始化统计插件
func (s *StatisWorker) Initialize(conf *plugin.ConfigEntry) error {
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.metricVecCaches = make(map[string]*prometheus.GaugeVec)
	s.discoveryHandler = &discoveryMetricHandle{}
	s.configHandler = &configMetricHandle{}
	if err := s.registerMetrics(); err != nil {
		return err
	}

	// 设置统计打印周期
	interval, _ := conf.Option["interval"].(int)
	if interval == 0 {
		interval = 60
	}

	baseWorker, err := base.NewBaseWorker(ctx, s.metricsHandle)
	if err != nil {
		cancel()
		return err
	}
	s.BaseWorker = baseWorker

	go s.Run(ctx, time.Duration(interval)*time.Second)
	return nil
}

// Destroy 销毁统计插件
func (s *StatisWorker) Destroy() error {
	return nil
}

// registerMetrics Registers the interface invocation-related observability metrics
func (s *StatisWorker) registerMetrics() error {
	for _, desc := range base.MetricDescList {
		var collector prometheus.Collector
		switch desc.MetricType {
		case base.TypeForGaugeVec:
			collector = prometheus.NewGaugeVec(prometheus.GaugeOpts{
				Name: desc.Name,
				Help: desc.Help,
				ConstLabels: prometheus.Labels{
					metrics.LabelServerNode: utils.LocalHost,
				},
			}, desc.LabelNames)
			s.metricVecCaches[desc.Name] = collector.(*prometheus.GaugeVec)
		default:
			continue
		}

		if err := metrics.GetRegistry().Register(collector); err != nil {
			log.Errorf("[APICall] register prometheus collector error, %v", err)
			return err
		}
	}
	return nil
}

// ReportCallMetrics report call metrics info
func (s *StatisWorker) ReportCallMetrics(metric metrics.CallMetric) {
	// 只上报服务端接受客户端请求调用的结果
	if metric.Type != metrics.ServerCallMetric {
		return
	}
	s.BaseWorker.ReportCallMetrics(metric)
}

// ReportDiscoveryMetrics report discovery metrics
func (s *StatisWorker) ReportDiscoveryMetrics(metric ...metrics.DiscoveryMetric) {
	s.discoveryHandler.handle(metric)
}

// ReportConfigMetrics report config_center metrics
func (s *StatisWorker) ReportConfigMetrics(metric ...metrics.ConfigMetrics) {
	s.configHandler.handle(metric)
}

// ReportDiscoverCall report discover service times
func (s *StatisWorker) ReportDiscoverCall(metric metrics.ClientDiscoverMetric) {
	// ignore not support this
}

func (a *StatisWorker) metricsHandle(mt metrics.CallMetricType, start time.Time,
	staticsSlice []*base.APICallStatisItem) {
	if mt != metrics.ServerCallMetric {
		return
	}
	if len(staticsSlice) == 0 {
		return
	}

	// 每一个接口，共 metricsNumber 个指标，下面的指标的个数调整时，这里的 metricsNumber 也需要调整
	statInfos := make([]*base.MetricData, 0, len(staticsSlice)*base.MetricsNumber)

	for _, item := range staticsSlice {
		var (
			maxTime, avgTime, rqTime, rqCount, minTime float64
			deleteFlag                                 bool
		)

		if item.Count == 0 && item.ZeroDuration > base.MaxZeroDuration {
			deleteFlag = true
		} else {
			maxTime = float64(item.MaxTime) / 1e6
			minTime = float64(item.MinTime) / 1e6
			rqTime = float64(item.AccTime) / 1e6
			rqCount = float64(item.Count)
			if item.Count > 0 {
				avgTime = float64(item.AccTime) / float64(item.Count) / 1e6
			}
			deleteFlag = false
		}

		statInfos = append(statInfos, &base.MetricData{
			Name:       base.MetricForClientRqTimeoutMax,
			Data:       maxTime,
			Labels:     base.BuildMetricLabels(item),
			DeleteFlag: deleteFlag,
		}, &base.MetricData{
			Name:       base.MetricForClientRqTimeoutAvg,
			Data:       avgTime,
			Labels:     base.BuildMetricLabels(item),
			DeleteFlag: deleteFlag,
		}, &base.MetricData{
			Name:       base.MetricForClientRqTimeout,
			Data:       rqTime,
			Labels:     base.BuildMetricLabels(item),
			DeleteFlag: deleteFlag,
		}, &base.MetricData{
			Name:       base.MetricForClientRqIntervalCount,
			Data:       rqCount,
			Labels:     base.BuildMetricLabels(item),
			DeleteFlag: deleteFlag,
		}, &base.MetricData{
			Name:       base.MetricForClientRqTimeoutMin,
			Data:       minTime,
			Labels:     base.BuildMetricLabels(item),
			DeleteFlag: deleteFlag,
		},
		)
	}

	a.reportToPrometheus(statInfos)
}

// collectMetricData
func (s *StatisWorker) reportToPrometheus(statInfos []*base.MetricData) {
	if len(statInfos) == 0 {
		return
	}

	// prometheus-sdk 本身做了pull时数据一致性的保证，这里不需要自己在进行额外的保护动作
	// 清理掉当前的存量数据
	for _, statInfo := range statInfos {
		if len(statInfo.Labels) == 0 {
			statInfo.Labels = make(map[string]string)
		}

		metricVec, ok := s.metricVecCaches[statInfo.Name]
		if !ok {
			continue
		}

		if statInfo.DeleteFlag {
			metricVec.Delete(statInfo.Labels)
		} else {
			metricVec.With(statInfo.Labels).Set(statInfo.Data)
		}
	}
}
