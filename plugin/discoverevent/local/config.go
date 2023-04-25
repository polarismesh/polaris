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

import "errors"

// DiscoverEventConfig 服务实例事件插件配置
type DiscoverEventConfig struct {
	QueueSize          int    `json:"queueSize"`
	OutputPath         string `json:"outputPath"`
	RotationMaxSize    int    `json:"rotationMaxSize"`
	RotationMaxAge     int    `json:"rotationMaxAge"`
	RotationMaxBackups int    `json:"rotationMaxBackups"`
}

// Validate 检查配置是否正确配置
func (c *DiscoverEventConfig) Validate() error {
	if c.QueueSize <= 0 {
		return errors.New("QueueSize is <= 0")
	}
	if c.OutputPath == "" {
		return errors.New("OutputPath is empty")
	}
	if c.RotationMaxSize <= 0 {
		return errors.New("RotationMaxSize is <= 0")
	}
	if c.RotationMaxAge <= 0 {
		return errors.New("RotationMaxAge is <= 0")
	}
	if c.RotationMaxBackups <= 0 {
		return errors.New("RotationMaxBackups is <= 0")
	}
	return nil
}

// DefaultDiscoverEventConfig 创建一个默认的服务事件插件配置
func DefaultDiscoverEventConfig() *DiscoverEventConfig {
	return &DiscoverEventConfig{
		QueueSize:          128,
		OutputPath:         "./discover-event",
		RotationMaxSize:    50,
		RotationMaxAge:     7,
		RotationMaxBackups: 100,
	}
}
