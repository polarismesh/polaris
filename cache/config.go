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

package cache

import "time"

// Config 缓存配置
type Config struct {
	// DiffTime 设置拉取时间范围, [T1 - abs(DiffTime), T1]
	DiffTime time.Duration `yaml:"diffTime"`
	// ReportInterval 监控数据上报周期
	ReportInterval time.Duration `yaml:"reportInterval"`
}

var (
	config *Config
)

// SetCacheConfig 设置缓存配置
func SetCacheConfig(conf *Config) {
	config = conf
}
