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

package token

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
)

// Config 限流配置类
type Config struct {
	// 是否启用远程配置，默认false。TODO 暂时无远程配置，后续版本补全
	RemoteConf bool `mapstructure:"remote-conf"`

	// IP限流相关配置
	IPLimitConf *ResourceLimitConfig `mapstructure:"ip-limit"`

	// 接口限流相关配置
	APILimitConf *APILimitConfig `mapstructure:"api-limit"`

	// 基于实例的限流配置
	InstanceLimitConf *ResourceLimitConfig `mapstructure:"instance-limit"`
}

// BucketRatelimit 针对令牌桶的具体配置
type BucketRatelimit struct {
	// 是否开启限流
	Open bool `mapstructure:"open"`

	// 令牌桶大小
	Bucket int `mapstructure:"bucket"`

	// 每秒加入的令牌数
	Rate int `mapstructure:"rate"`
}

// ResourceLimitConfig 基于资源的限流配置
// 资源可以是：IP，实例，服务等
type ResourceLimitConfig struct {
	// 是否开启instance限流
	Open bool `mapstructure:"open"`

	// 全局限制规则，只有一条规则
	Global *BucketRatelimit `mapstructure:"global"`

	// 本地缓存最大多少个instance的限制器
	MaxResourceCacheAmount int `mapstructure:"resource-cache-amount"`

	// 白名单
	WhiteList []string `mapstructure:"white-list"`
}

// APILimitConfig api限流配置
type APILimitConfig struct {
	// 系统是否开启API限流
	Open bool `mapstructure:"open"`

	// 配置规则集合
	Rules []*RateLimitRule `mapstructure:"rules"`

	// 每个接口的单独配置
	Apis []*APILimitInfo `mapstructure:"apis"`
}

// RateLimitRule 限流规则
type RateLimitRule struct {
	// 规则名
	Name string `mapstructure:"name"`

	// 规则的限制
	Limit *BucketRatelimit `mapstructure:"limit"`
}

// APILimitInfo 每个接口的单独配置信息
type APILimitInfo struct {
	// 接口名，比如对于HTTP，就是：方法+URL
	Name string `mapstructure:"name"`

	// 限制规则名
	Rule string `mapstructure:"rule"`
}

// decodeConfig 把map解码为Config对象
func decodeConfig(data map[string]interface{}) (*Config, error) {
	if data == nil {
		return nil, fmt.Errorf("plugin(%s) option is empty", PluginName)
	}
	var config Config
	if err := mapstructure.Decode(data, &config); err != nil {
		log.Errorf("[Plugin][%s] decode config err: %s", PluginName, err.Error())
		return nil, err
	}

	/*log.Infof("%+v", config)
	  log.Infof("%+v", config.IPLimitConf.Global)
	  log.Infof("%+v", config.APILimitConf)
	  for _, entry := range config.APILimitConf.Rules {
	  	log.Infof("%s->%+v", entry.Name, entry.Limit)
	  }
	  for _, entry := range config.APILimitConf.Apis {
	  	log.Infof("%+v", entry)
	  }*/

	return &config, nil
}
