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
	"github.com/polarismesh/polaris/plugin"
)

// initialize 插件初始化函数
func (tb *tokenBucket) initialize(c *plugin.ConfigEntry) error {
	config, err := decodeConfig(c.Option)
	if err != nil {
		log.Errorf("[Plugin][%s] initialize err: %s", PluginName, err.Error())
		return err
	}
	if !config.Enable {
		tb.config = config
		return nil
	}
	// 加载本地配置
	if config.RuleFile != "" {
		config, err = loadLocalConfig(config.RuleFile)
		if err != nil {
			log.Errorf("[Plugin][%s] load local rule fail err: %s", PluginName, err.Error())
			return err
		}
	}
	// 注册一个配置中心的 Change Event
	if config.RemoteConf {
		// TODO 监听规则
	}

	tb.config = config
	tb.limiters = make(map[plugin.RatelimitType]limiter)

	// IP限流
	irt, err := newResourceRatelimit(plugin.IPRatelimit, config.IPLimitConf)
	if err != nil {
		return err
	}
	tb.limiters[plugin.IPRatelimit] = irt

	// 接口限流
	art, err := newAPIRatelimit(config.APILimitConf)
	if err != nil {
		return err
	}
	tb.limiters[plugin.APIRatelimit] = art

	// 操作实例限流
	instance, err := newResourceRatelimit(plugin.InstanceRatelimit, config.InstanceLimitConf)
	if err != nil {
		return err
	}
	tb.limiters[plugin.InstanceRatelimit] = instance

	return nil
}

// allow 插件的限流实现函数
func (tb *tokenBucket) allow(typ plugin.RatelimitType, key string) bool {
	// key为空，则不作限制
	if key == "" {
		return true
	}
	l, ok := tb.limiters[typ]
	if !ok {
		return true
	}

	return l.allow(key)
}
