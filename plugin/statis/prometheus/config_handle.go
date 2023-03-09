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
	"github.com/prometheus/client_golang/prometheus"

	"github.com/polarismesh/polaris/common/metrics"
	"github.com/polarismesh/polaris/common/utils"
)

func init() {
	metrics.GetRegistry().MustRegister(
		configGroupTotal,
		configFileTotal,
		releaseConfigFileTotal,
	)
}

var (
	configGroupTotal = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "config_group_count",
		Help: "polaris config group total number",
		ConstLabels: map[string]string{
			metrics.LabelServerNode: utils.LocalHost,
		},
	}, []string{metrics.LabelNamespace})

	configFileTotal = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "config_file_count",
		Help: "total number of config_file each config group",
		ConstLabels: map[string]string{
			metrics.LabelServerNode: utils.LocalHost,
		},
	}, []string{metrics.LabelNamespace, metrics.LabelGroup})

	releaseConfigFileTotal = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "config_release_file_count",
		Help: "total number of config_release_file each config group",
		ConstLabels: map[string]string{
			metrics.LabelServerNode: utils.LocalHost,
		},
	}, []string{metrics.LabelNamespace, metrics.LabelGroup})
)

func newConfigMetricHandle() *configMetricHandle {
	return &configMetricHandle{}
}

type configMetricHandle struct {
}

func (h *configMetricHandle) handle(ms []metrics.ConfigMetrics) {
	configGroupTotal.Reset()
	configFileTotal.Reset()
	releaseConfigFileTotal.Reset()
	for i := range ms {
		m := ms[i]
		switch m.Type {
		case metrics.ConfigGroupMetric:
			configGroupTotal.With(m.Labels).Set(float64(m.Total))
		case metrics.FileMetric:
			configFileTotal.With(m.Labels).Set(float64(m.Total))
		case metrics.ReleaseFileMetric:
			releaseConfigFileTotal.With(m.Labels).Set(float64(m.Total))
		}
	}
}
