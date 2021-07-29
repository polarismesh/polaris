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

package lrurate

import (
	"errors"
	"github.com/polarismesh/polaris-server/plugin"
)

const (
	PluginName = "lrurate"
)

var (
	ratelimitIPLruSize      int
	ratelimitIPRate         int
	ratelimitIPBurst        int
	ratelimitServiceLruSize int
	ratelimitServiceRate    int
	ratelimitServiceBurst   int
)

// 自注册到插件列表
func init() {
	plugin.RegisterPlugin(PluginName, &LRURate{})
}

// LRURate Ratelimit
type LRURate struct {
}

// 返回插件名
func (m *LRURate) Name() string {
	return PluginName
}

// 初始化函数
func (m *LRURate) Initialize(c *plugin.ConfigEntry) error {
	if err := parseRateLimitIPOption(c.Option); err != nil {
		return err
	}
	if err := parseRateLimitServiceOption(c.Option); err != nil {
		return err
	}

	if err := initEnv(); err != nil {
		return err
	}

	return nil
}

// 获取IP相关的参数
func parseRateLimitIPOption(opt map[string]interface{}) error {
	var ok bool
	var val interface{}

	if val = opt["ratelimitIPLruSize"]; val == nil {
		return errors.New("not found ratelimit::lrurate::ratelimitIPLruSize")
	}

	if ratelimitIPLruSize, ok = val.(int); !ok || ratelimitIPLruSize <= 0 {
		return errors.New("invalid ratelimit::lrurate::ratelimitIPLruSize, must be int and > 0")
	}

	if val = opt["ratelimitIPRate"]; val == nil {
		return errors.New("not found ratelimit::lrurate::ratelimitIPRate")
	}

	if ratelimitIPRate, ok = val.(int); !ok || ratelimitIPRate <= 0 {
		return errors.New("invalid ratelimit::lrurate::ratelimitIPRate, must be int and > 0")
	}

	if val = opt["ratelimitIPBurst"]; val == nil {
		return errors.New("not found ratelimit::lrurate::ratelimitIPBurst")
	}

	if ratelimitIPBurst, ok = val.(int); !ok || ratelimitIPBurst <= 0 {
		return errors.New("invalid ratelimit::lrurate::ratelimitIPBurst, must be int and > 0")
	}

	return nil
}

// 获取service相关的参数
func parseRateLimitServiceOption(opt map[string]interface{}) error {
	var ok bool
	var val interface{}

	if val = opt["ratelimitServiceLruSize"]; val == nil {
		return errors.New("not found ratelimit::lrurate::ratelimitServiceLruSize")
	}

	if ratelimitServiceLruSize, ok = val.(int); !ok || ratelimitServiceLruSize <= 0 {
		return errors.New("invalid ratelimit::lrurate::ratelimitServiceLruSize, must be int and > 0")
	}

	if val = opt["ratelimitServiceRate"]; val == nil {
		return errors.New("not found ratelimit::lrurate::ratelimitServiceRate")
	}

	if ratelimitServiceRate, ok = val.(int); !ok || ratelimitServiceRate <= 0 {
		return errors.New("invalid ratelimit::lrurate::ratelimitServiceRate, must be int and > 0")
	}

	if val = opt["ratelimitServiceBurst"]; val == nil {
		return errors.New("not found ratelimit::lrurate::ratelimitServiceBurst")
	}

	if ratelimitServiceBurst, ok = val.(int); !ok || ratelimitServiceBurst <= 0 {
		return errors.New("invalid ratelimit::lrurate::ratelimitServiceBurst, must be int and > 0")
	}

	return nil
}

// 销毁函数
func (m *LRURate) Destroy() error {
	return nil
}

// 实现CMDB插件接口
func (m *LRURate) Allow(rateType plugin.RatelimitType, id string) bool {
	switch plugin.RatelimitType(rateType) {
	case plugin.IPRatelimit:
		return allowIP(id)
	case plugin.ServiceRatelimit:
		return allowService(id)
	}

	// 默认允许访问
	return true
}
