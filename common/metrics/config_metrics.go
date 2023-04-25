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

func registerConfigFileMetrics() {
	configGroupTotal = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "config_group_count",
		Help: "polaris config group total number",
		ConstLabels: map[string]string{
			LabelServerNode: utils.LocalHost,
		},
	}, []string{LabelNamespace})

	configFileTotal = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "config_file_count",
		Help: "total number of config_file each config group",
		ConstLabels: map[string]string{
			LabelServerNode: utils.LocalHost,
		},
	}, []string{LabelNamespace, LabelGroup})

	releaseConfigFileTotal = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "config_release_file_count",
		Help: "total number of config_release_file each config group",
		ConstLabels: map[string]string{
			LabelServerNode: utils.LocalHost,
		},
	}, []string{LabelNamespace, LabelGroup})

	_ = GetRegistry().Register(configGroupTotal)
	_ = GetRegistry().Register(configFileTotal)
	_ = GetRegistry().Register(releaseConfigFileTotal)
}

func GetConfigGroupTotal() *prometheus.GaugeVec {
	return configGroupTotal
}

func GetConfigFileTotal() *prometheus.GaugeVec {
	return configFileTotal
}

func GetReleaseConfigFileTotal() *prometheus.GaugeVec {
	return releaseConfigFileTotal
}
