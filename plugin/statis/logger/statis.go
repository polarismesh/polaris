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

package logger

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"time"

	commonlog "github.com/polarismesh/polaris/common/log"
	"github.com/polarismesh/polaris/common/metrics"
	commontime "github.com/polarismesh/polaris/common/time"
	"github.com/polarismesh/polaris/plugin"
	"github.com/polarismesh/polaris/plugin/statis/base"
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
	*base.BaseWorker
	cancel context.CancelFunc
}

// Name 获取统计插件名称
func (s *StatisWorker) Name() string {
	return PluginName
}

// Initialize 初始化统计插件
func (s *StatisWorker) Initialize(conf *plugin.ConfigEntry) error {
	ctx, cancel := context.WithCancel(context.Background())
	baseWorker, err := base.NewBaseWorker(ctx, s.metricsHandle)
	if err != nil {
		cancel()
		return err
	}

	s.cancel = cancel
	s.BaseWorker = baseWorker

	// 设置统计打印周期
	interval, _ := conf.Option["interval"].(int)
	if interval == 0 {
		interval = 60
	}

	go s.Run(ctx, time.Duration(interval)*time.Second)
	return nil
}

// Destroy 销毁统计插件
func (s *StatisWorker) Destroy() error {
	if s.cancel != nil {
		s.cancel()
	}
	return nil
}

// ReportCallMetrics report call metrics info
func (s *StatisWorker) ReportCallMetrics(metric metrics.CallMetric) {
	s.BaseWorker.ReportCallMetrics(metric)
}

// ReportDiscoveryMetrics report discovery metrics
func (s *StatisWorker) ReportDiscoveryMetrics(metric ...metrics.DiscoveryMetric) {
}

// ReportConfigMetrics report config_center metrics
func (s *StatisWorker) ReportConfigMetrics(metric ...metrics.ConfigMetrics) {
}

// ReportDiscoverCall report discover service times
func (s *StatisWorker) ReportDiscoverCall(metric metrics.ClientDiscoverMetric) {
	discoverlog.Infof(metric.String())
}

func (a *StatisWorker) metricsHandle(mt metrics.CallMetricType, start time.Time,
	statics []*base.APICallStatisItem) {
	startStr := commontime.Time2String(start)
	if len(statics) == 0 {
		log.Info(fmt.Sprintf("Statis %s: No API Call\n", startStr))
		return
	}

	scope := commonlog.GetScopeOrDefaultByName(string(mt))
	if scope.Name() == commonlog.DefaultLoggerName {
		scope = log
	}

	var msg string
	var prefixMax int
	for i := range statics {
		prefixMax = int(math.Max(float64(prefixMax), float64(len(statics[i].API))))
	}
	for i := range statics {
		msg += formatAPICallStatisItem(prefixMax, mt, statics[i])
	}
	if len(msg) == 0 {
		log.Info(fmt.Sprintf("Statis %s: No API Call\n", startStr))
		return
	}

	header := fmt.Sprintf("Statis %s:\n", startStr)

	header += fmt.Sprintf(
		"%-"+strconv.Itoa(prefixMax)+"v|%12v|%17v|%12v|%12v|%12v|%12v|%12v|\n", "", "Protocol", "TrafficDirection", "Code", "Count",
		"Min(ms)", "Max(ms)", "Avg(ms)")

	log.Info(header + msg)
}

func formatAPICallStatisItem(prefixMax int, mt metrics.CallMetricType, item *base.APICallStatisItem) string {
	if item.Count == 0 {
		return ""
	}
	return fmt.Sprintf("%-"+strconv.Itoa(prefixMax)+"v|%12v|%17v|%12v|%12v|%12.3f|%12.3f|%12.3f|\n",
		item.API, item.Protocol, item.TrafficDirection, item.Code, item.Count,
		float64(item.MinTime)/1e6,
		float64(item.MaxTime)/1e6,
		float64(item.AccTime)/float64(item.Count)/1e6,
	)

}
