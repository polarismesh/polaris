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

package connlimit

import (
	"time"

	"github.com/mitchellh/mapstructure"

	"github.com/polarismesh/polaris/common/log"
)

// Config 连接限制配置
type Config struct {
	// 开启连接限制
	OpenConnLimit bool `mapstructure:"openConnLimit"`

	// 单个host最大的连接数，必须 > 1
	MaxConnPerHost int `mapstructure:"maxConnPerHost"`

	// 当前协议监听端口的最大连接数
	// 兼容老版本，> 1，则开启listen的全局限制；< 1则不开启listen的全局限制
	MaxConnLimit int `mapstructure:"maxConnLimit"`

	// 白名单，不进行host连接数限制
	WhiteList string `mapstructure:"whiteList"`

	// 读超时
	ReadTimeout time.Duration `mapstructure:"readTimeout"`

	// 回收连接统计数据的周期
	PurgeCounterInterval time.Duration `mapstructure:"purgeCounterInterval"`

	// 回收连接的最大超时时间
	PurgeCounterExpire time.Duration `mapstructure:"purgeCounterExpire"`
}

// ParseConnLimitConfig 解析配置
func ParseConnLimitConfig(raw map[interface{}]interface{}) (*Config, error) {
	if raw == nil {
		return nil, nil
	}

	config := &Config{}
	decodeConfig := &mapstructure.DecoderConfig{
		DecodeHook: mapstructure.StringToTimeDurationHookFunc(),
		Result:     config,
	}
	decoder, err := mapstructure.NewDecoder(decodeConfig)
	if err != nil {
		log.Errorf("conn limit new decoder err: %s", err.Error())
		return nil, err
	}

	err = decoder.Decode(raw)
	if err != nil {
		log.Errorf("parse conn limit config(%+v) err: %s", raw, err.Error())
		return nil, err
	}

	return config, nil
}
