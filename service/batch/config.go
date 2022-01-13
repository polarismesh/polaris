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

package batch

import (
	"errors"

	"github.com/mitchellh/mapstructure"
	"github.com/polarismesh/polaris-server/common/log"
)

// Config 批量配置，控制最大的条目，批量等待时间等
type Config struct {
	Register   *CtrlConfig `mapstructure:"register"`
	Deregister *CtrlConfig `mapstructure:"deregister"`
	Heartbeat  *CtrlConfig `mapstructure:"heartbeat"`
}

// CtrlConfig batch控制配置项
type CtrlConfig struct {
	// 是否开启Batch工作模式
	Open bool `mapstructure:"open"`
	// 注册请求队列的长度
	QueueSize int `mapstructure:"queueSize"`
	// 最长多久一次批量操作
	WaitTime string `mapstructure:"waitTime"`
	// 每次操作最大的批量数
	MaxBatchCount int `mapstructure:"maxBatchCount"`
	// 写store的并发协程数
	Concurrency int `mapstructure:"concurrency"`
}

// ParseBatchConfig 解析配置文件为config
func ParseBatchConfig(opt map[string]interface{}) (*Config, error) {
	if opt == nil {
		return nil, nil
	}

	var config Config
	if err := mapstructure.Decode(opt, &config); err != nil {
		log.Errorf("[Batch] parse config(%+v) err: %s", opt, err.Error())
		return nil, err
	}

	// 对配置文件做校验
	if !checkCtrlConfig(config.Register) {
		log.Errorf("[Controller] batch register config is invalid: %+v", config)
		return nil, errors.New("batch register config is invalid")
	}
	if !checkCtrlConfig(config.Deregister) {
		log.Errorf("[Controller] batch deregister config is invalid: %+v", config)
		return nil, errors.New("batch deregister config is invalid")
	}
	if !checkCtrlConfig(config.Heartbeat) {
		log.Errorf("[Controller] batch heartbeat config is invalid: %+v", config)
		return nil, errors.New("batch deregister config is invalid")
	}
	return &config, nil
}

// 配置文件校验
func checkCtrlConfig(ctrl *CtrlConfig) bool {
	if ctrl == nil {
		return true
	}

	if ctrl.QueueSize <= 0 || ctrl.MaxBatchCount <= 0 || ctrl.Concurrency <= 0 {
		return false
	}

	return true
}
