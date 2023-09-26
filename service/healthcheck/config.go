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

package healthcheck

import (
	"time"

	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/plugin"
)

// Config 健康检查配置
type Config struct {
	Open                *bool                  `yaml:"open"`
	Service             string                 `yaml:"service"`
	SlotNum             int                    `yaml:"slotNum"`
	LocalHost           string                 `yaml:"localHost"`
	MinCheckInterval    time.Duration          `yaml:"minCheckInterval"`
	MaxCheckInterval    time.Duration          `yaml:"maxCheckInterval"`
	ClientCheckInterval time.Duration          `yaml:"clientCheckInterval"`
	ClientCheckTtl      time.Duration          `yaml:"clientCheckTtl"`
	Checkers            []plugin.ConfigEntry   `yaml:"checkers"`
	Batch               map[string]interface{} `yaml:"batch"`
}

const (
	defaultMinCheckInterval    = 1 * time.Second
	defaultMaxCheckInterval    = 30 * time.Second
	defaultSlotNum             = 30
	defaultClientReportTtl     = 120 * time.Second
	defaultClientCheckInterval = 120 * time.Second
)

func (c *Config) IsOpen() bool {
	if c.Open == nil {
		return true
	}
	return *c.Open
}

// SetDefault 设置默认值
func (c *Config) SetDefault() {
	if c.Open == nil {
		c.Open = utils.BoolPtr(true)
	}
	if len(c.Service) == 0 {
		c.Service = "polaris.checker"
	}
	if c.SlotNum == 0 {
		c.SlotNum = defaultSlotNum
	}
	if c.MinCheckInterval == 0 {
		c.MinCheckInterval = defaultMinCheckInterval
	}
	if c.MaxCheckInterval == 0 {
		c.MaxCheckInterval = defaultMaxCheckInterval
	}
	if c.MinCheckInterval > c.MaxCheckInterval {
		c.MinCheckInterval = defaultMinCheckInterval
		c.MaxCheckInterval = defaultMaxCheckInterval
	}
	if c.ClientCheckInterval == 0 {
		c.ClientCheckInterval = defaultClientCheckInterval
	}
	if c.ClientCheckTtl == 0 {
		c.ClientCheckTtl = defaultClientReportTtl
	}
}
