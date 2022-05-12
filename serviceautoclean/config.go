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
package serviceautoclean

import (
	"time"

	"github.com/polarismesh/polaris-server/common/utils"
)

// Config 服务自动清理配置
type Config struct {
	Open                  bool          `yaml:"open"`
	Namespace             string        `yaml:"namespace"`
	Service               string        `yaml:"service"`
	LocalHost             string        `yaml:"localHost"`
	CheckInterval         time.Duration `yaml:"checkInterval"`
	ExpireTime            time.Duration `yaml:"expireTime"`
	CheckCountBeforeClean int           `yaml:"checkCountBeforeClean"`
	IgnoredNamespaces     []string      `yaml:"ignoredNamespaces"`
}

const (
	checkInterval         = 60 * time.Second
	expireTime            = 600 * time.Second
	checkCountBeforeClean = 3
	selfServiceName       = "polaris.serviceAutoCleaner"
)

// SetDefault 设置默认值
func (c *Config) SetDefault(polarisNamespace string) {
	if len(c.Service) == 0 {
		c.Service = selfServiceName
	}
	if len(c.Namespace) == 0 {
		c.Namespace = polarisNamespace
	}
	if len(c.IgnoredNamespaces) == 0 {
		c.IgnoredNamespaces = append(c.IgnoredNamespaces, polarisNamespace)
	}
	if len(c.LocalHost) == 0 {
		c.LocalHost = utils.LocalHost
	}
	if c.CheckInterval == 0 {
		c.CheckInterval = checkInterval
	}
	if c.ExpireTime == 0 {
		c.ExpireTime = expireTime
	}
	if c.CheckCountBeforeClean == 0 {
		c.CheckCountBeforeClean = checkCountBeforeClean
	}
}
