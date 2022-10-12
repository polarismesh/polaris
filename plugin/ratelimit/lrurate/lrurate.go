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

	"github.com/polarismesh/polaris/plugin"
)

const (
	// PluginName lru rate plugin
	PluginName = "lrurate"
)

var (
	rateLimitIPLruSize      int
	rateLimitIPRate         int
	rateLimitIPBurst        int
	rateLimitServiceLruSize int
	rateLimitServiceRate    int
	rateLimitServiceBurst   int
)

// init 自注册到插件列表
func init() {
	plugin.RegisterPlugin(PluginName, &LRURate{})
}

// LRURate Ratelimit
type LRURate struct{}

// Name 返回插件名
func (m *LRURate) Name() string {
	return PluginName
}

// Initialize 初始化函数
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

// parseRateLimitIPOption 获取IP相关的参数
func parseRateLimitIPOption(opt map[string]interface{}) error {
	var ok bool
	var val interface{}

	if val = opt["rateLimitIPLruSize"]; val == nil {
		return errors.New("not found ratelimit::lrurate::rateLimitIPLruSize")
	}

	if rateLimitIPLruSize, ok = val.(int); !ok || rateLimitIPLruSize <= 0 {
		return errors.New("invalid ratelimit::lrurate::rateLimitIPLruSize, must be int and > 0")
	}

	if val = opt["rateLimitIPRate"]; val == nil {
		return errors.New("not found ratelimit::lrurate::rateLimitIPRate")
	}

	if rateLimitIPRate, ok = val.(int); !ok || rateLimitIPRate <= 0 {
		return errors.New("invalid ratelimit::lrurate::rateLimitIPRate, must be int and > 0")
	}

	if val = opt["rateLimitIPBurst"]; val == nil {
		return errors.New("not found ratelimit::lrurate::rateLimitIPBurst")
	}

	if rateLimitIPBurst, ok = val.(int); !ok || rateLimitIPBurst <= 0 {
		return errors.New("invalid ratelimit::lrurate::rateLimitIPBurst, must be int and > 0")
	}

	return nil
}

// parseRateLimitServiceOption 获取service相关的参数
func parseRateLimitServiceOption(opt map[string]interface{}) error {
	var ok bool
	var val interface{}

	if val = opt["rateLimitServiceLruSize"]; val == nil {
		return errors.New("not found ratelimit::lrurate::rateLimitServiceLruSize")
	}

	if rateLimitServiceLruSize, ok = val.(int); !ok || rateLimitServiceLruSize <= 0 {
		return errors.New("invalid ratelimit::lrurate::rateLimitServiceLruSize, must be int and > 0")
	}

	if val = opt["rateLimitServiceRate"]; val == nil {
		return errors.New("not found ratelimit::lrurate::rateLimitServiceRate")
	}

	if rateLimitServiceRate, ok = val.(int); !ok || rateLimitServiceRate <= 0 {
		return errors.New("invalid ratelimit::lrurate::rateLimitServiceRate, must be int and > 0")
	}

	if val = opt["rateLimitServiceBurst"]; val == nil {
		return errors.New("not found ratelimit::lrurate::rateLimitServiceBurst")
	}

	if rateLimitServiceBurst, ok = val.(int); !ok || rateLimitServiceBurst <= 0 {
		return errors.New("invalid ratelimit::lrurate::rateLimitServiceBurst, must be int and > 0")
	}

	return nil
}

// Destroy 销毁函数
func (m *LRURate) Destroy() error {
	return nil
}

// Allow 实现CMDB插件接口
func (m *LRURate) Allow(rateType plugin.RatelimitType, id string) bool {
	switch rateType {
	case plugin.IPRatelimit:
		return allowIP(id)
	case plugin.ServiceRatelimit:
		return allowService(id)
	}

	// 默认允许访问
	return true
}
