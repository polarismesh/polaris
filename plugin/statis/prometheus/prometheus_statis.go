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
//
//@Author: springliao
//@Description:
//@Time: 2021/10/26 15:40

package prometheus

import (
	"math"
	"time"

	v1 "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/plugin"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	PluginName      string        = "prometheus"
	DefalutInterval time.Duration = time.Duration(30) * time.Second
)

/**
 * @brief 初始化注册函数
 */
func init() {
	plugin.RegisterPlugin(PluginName, &PrometheusStatis{})
}

// PrometheusStatis
type PrometheusStatis struct {
	registry        *prometheus.Registry
	metricVecCaches map[string]interface{}
	callCh          chan *APICall
	apiCallStatis   *APICallStatis
	interval        time.Duration
}

func (statis *PrometheusStatis) Name() string {
	return PluginName
}

// Initialize do PrometheusStatis initialize
func (statis *PrometheusStatis) Initialize(conf *plugin.ConfigEntry) error {

	interval := conf.Option["interval"].(int)
	customerInterval := time.Duration(interval) * time.Second

	statis.interval = time.Duration(int64(math.Min(float64(customerInterval), float64(DefalutInterval))))
	statis.metricVecCaches = make(map[string]interface{})
	statis.callCh = make(chan *APICall, 1024)
	statis.registry = prometheus.NewRegistry()
	statis.apiCallStatis = &APICallStatis{
		statis: statis,
		items:  make(map[string]*APICallStatisItem),
	}

	statis.registerMetrics()
	go statis.Run()
	return nil
}

// AddAPICall Description Interface invocation data is reported
// api: Information about the invoked access layer interface
// protocol: layer interface protocol
// code: response code
// duration: time spent per request
func (statis *PrometheusStatis) AddAPICall(api, protocol string, code int, duration int64) error {
	statis.callCh <- &APICall{
		api:      api,
		protocol: protocol,
		code:     code,
		duration: duration,
	}
	return nil
}

// Destroy
func (statis *PrometheusStatis) Destroy() error {
	return nil
}

// Run Main loop, trigger interface indicator call calculation and report to Prometheus
func (statis *PrometheusStatis) Run() {
	metricTicker := time.NewTicker(statis.interval)
	defer func() {
		metricTicker.Stop()
	}()

	for {
		select {
		case <-metricTicker.C:
			statis.apiCallStatis.collectMetricData()
		case call := <-statis.callCh:
			statis.apiCallStatis.add(call)

			// Time and number of interface invocations reported last time
			statInfos := []*MetricData{
				{
					Name:   MetricForClientRqTimeout,
					Data:   float64(call.duration) / 1e6,
					Labels: buildMetricLabels(call, nil),
				},
				{
					Name:   MetricForClientRqTotal,
					Labels: buildMetricLabels(call, nil),
				},
			}

			if call.code > v1.NoNeedUpdate {
				statInfos = append(statInfos, &MetricData{
					Name:   MetricForClientRqFailure,
					Labels: buildMetricLabels(call, nil),
				})
			}

			statis.collectMetricData(statInfos...)
		}
	}
}

// registerMetrics Registers the interface invocation-related observability metrics
func (statis *PrometheusStatis) registerMetrics() {
	for _, desc := range metricDescList {
		var collector prometheus.Collector
		switch desc.MetricType {
		case TypeForCounterVec:
			collector = prometheus.NewCounterVec(prometheus.CounterOpts{
				Name: desc.Name,
				Help: desc.Help,
			}, desc.LabelNames)
		case TypeForGaugeVec:
			collector = prometheus.NewGaugeVec(prometheus.GaugeOpts{
				Name: desc.Name,
				Help: desc.Help,
			}, desc.LabelNames)
		case TypeForHistogramVec:
			collector = prometheus.NewHistogramVec(prometheus.HistogramOpts{
				Name: desc.Name,
				Help: desc.Help,
			}, desc.LabelNames)
		default:
			continue
		}

		statis.registry.Register(collector)
		statis.metricVecCaches[desc.Name] = collector
	}

}

// collectMetricData
func (statis *PrometheusStatis) collectMetricData(statInfos ...*MetricData) {

	if len(statInfos) == 0 {
		return
	}

	for i := range statInfos {
		statInfo := statInfos[i]

		if len(statInfo.Labels) == 0 {
			statInfo.Labels = make(map[string]string)
		}

		// Set a public label information: Polaris node information
		statInfo.Labels[LabelForPolarisServerInstance] = plugin.LocalHost
		metricVec := statis.metricVecCaches[statInfo.Name]

		if metricVec == nil {
			continue
		}

		switch metric := metricVec.(type) {
		case *prometheus.CounterVec:
			metric.With(statInfo.Labels).Inc()
		case *prometheus.GaugeVec:
			metric.With(statInfo.Labels).Set(statInfo.Data)
		case *prometheus.HistogramVec:
			metric.With(statInfo.Labels).Observe(statInfo.Data)
		}
	}

}

// GetRegistry return prometheus.Registry instance
func (statis *PrometheusStatis) GetRegistry() *prometheus.Registry {
	return statis.registry
}
