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

package plugin

import (
	"os"
	"sync"

	"github.com/polarismesh/polaris/common/metrics"
)

var (
	statisOnce sync.Once
)

type ComponentType string

const (
	ComponentServer        ComponentType = "server"
	ComponentRedis         ComponentType = "redis"
	ComponentDB            ComponentType = "db"
	ComponentProtobufCache ComponentType = "protobuf"
)

// Statis Statistical plugin interface
type Statis interface {
	Plugin
	// ReportCallMetrics report call metrics info
	ReportCallMetrics(metric metrics.CallMetric)
	// ReportDiscoveryMetrics report discovery metrics
	ReportDiscoveryMetrics(metric ...metrics.DiscoveryMetric)
	// ReportConfigMetrics report config_center metrics
	ReportConfigMetrics(metric ...metrics.ConfigMetrics)
}

type noopStatis struct {
}

func (n *noopStatis) Name() string {
	return "noopStatis"
}

func (n *noopStatis) Initialize(c *ConfigEntry) error {
	return nil
}

func (n *noopStatis) Destroy() error {
	return nil
}

// ReportCallMetrics report call metrics info
func (n *noopStatis) ReportCallMetrics(metric metrics.CallMetric) {}

// ReportDiscoveryMetrics report discovery metrics
func (n *noopStatis) ReportDiscoveryMetrics(metric ...metrics.DiscoveryMetric) {}

// ReportConfigMetrics report config_center metrics
func (n *noopStatis) ReportConfigMetrics(metric ...metrics.ConfigMetrics) {}

// GetStatis Get statistical plugin
func GetStatis() Statis {
	c := &config.Statis

	plugin, exist := pluginSet[c.Name]
	if !exist {
		return &noopStatis{}
	}

	statisOnce.Do(func() {
		if err := plugin.Initialize(c); err != nil {
			log.Errorf("Statis plugin init err: %s", err.Error())
			os.Exit(-1)
		}
	})

	return plugin.(Statis)
}
