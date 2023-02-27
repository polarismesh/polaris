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
	"context"
	"time"

	"github.com/polarismesh/polaris/common/metrics"
	"github.com/polarismesh/polaris/plugin"
	"github.com/polarismesh/polaris/store"
)

const (
	PluginName = "local"
)

// init 注册统计插件
func init() {
	s := &StatisWorker{}
	plugin.RegisterPlugin(s.Name(), s)
}

// StatisWorker 本地统计插件
type StatisWorker struct {
	interval time.Duration
	cancel   context.CancelFunc

	acc chan *APICall
	acs *APICallStatis

	cacheStatis *CacheCallStatis

	discoveryHandle *discoveryMetricHandle
	configHandle    *configMetricHandle
}

// Name 获取统计插件名称
func (s *StatisWorker) Name() string {
	return PluginName
}

// Initialize 初始化统计插件
func (s *StatisWorker) Initialize(conf *plugin.ConfigEntry) error {
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel

	// 设置统计打印周期
	var err error
	interval := conf.Option["interval"].(int)
	s.interval = time.Duration(interval) * time.Second

	// 初始化 prometheus 输出
	prometheusStatis, err := NewPrometheusStatis()
	if err != nil {
		return err
	}

	// 初始化接口调用统计
	s.acc = make(chan *APICall, 1024)
	s.acs, err = newAPICallStatis(prometheusStatis)
	if err != nil {
		return err
	}
	s.cacheStatis, err = newCacheCallStatis(ctx)
	s.discoveryHandle = newDiscoveryMetricHandle()
	s.configHandle = newConfigMetricHandle()

	go s.Run()
	return nil
}

// Destroy 销毁统计插件
func (s *StatisWorker) Destroy() error {
	return nil
}

const maxAddDuration = 800 * time.Millisecond

// ReportCallMetrics report call metrics info
func (s *StatisWorker) ReportCallMetrics(metric metrics.CallMetric) {
	switch metric.Type {
	case metrics.ServerCallMetric:
		clientRequestTimeout.With(metric.GetLabels()).Observe(float64(metric.Duration.Milliseconds()))
		startTime := time.Now()
		s.acc <- &APICall{
			api:       metric.API,
			protocol:  metric.Protocol,
			code:      metric.Code,
			duration:  int64(metric.Duration.Nanoseconds()),
			component: plugin.ComponentServer,
		}
		passDuration := time.Since(startTime)
		if passDuration >= maxAddDuration {
			log.Warnf("[APICall]add api call cost %s, exceed max %s", passDuration, maxAddDuration)
		}
	case metrics.SystemCallMetric:
		s.acc <- &APICall{
			api:       metric.API,
			protocol:  metric.Protocol,
			code:      metric.Code,
			duration:  int64(metric.Duration.Nanoseconds()),
			component: plugin.ComponentInner,
		}
	case metrics.RedisCallMetric:
		s.acc <- &APICall{
			api:       metric.API,
			protocol:  metric.Protocol,
			code:      metric.Code,
			duration:  int64(metric.Duration.Nanoseconds()),
			component: plugin.ComponentRedis,
		}
	case metrics.ProtobufCacheCallMetric:
		s.cacheStatis.add(metric)
	}
}

// ReportDiscoveryMetrics report discovery metrics
func (s *StatisWorker) ReportDiscoveryMetrics(metric ...metrics.DiscoveryMetric) {
	s.discoveryHandle.handle(metric)
}

// ReportConfigMetrics report config_center metrics
func (s *StatisWorker) ReportConfigMetrics(metric ...metrics.ConfigMetrics) {
	s.configHandle.handle(metric)
}

// Run 主流程
func (s *StatisWorker) Run() {
	getStore, err := store.GetStore()
	if err != nil {
		log.Errorf("[APICall] get store error, %v", err)
		return
	}

	nowSeconds, err := getStore.GetUnixSecond()
	if err != nil {
		log.Errorf("[APICall] get now second from store error, %v", err)
		return
	}
	if nowSeconds == 0 {
		nowSeconds = time.Now().Unix()
	}
	dest := nowSeconds
	dest += 60
	dest = dest - (dest % 60)
	diff := dest - nowSeconds

	log.Infof("[APICall] prometheus stats need sleep %ds", diff)

	time.Sleep(time.Duration(diff) * time.Second)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.acs.log()
		case ac := <-s.acc:
			s.acs.add(ac)
		}
	}
}
