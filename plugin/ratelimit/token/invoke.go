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

// tokenBucket 实现Plugin接口
type tokenBucket struct {
	config   *Config
	limiters map[plugin.RatelimitType]limiter
}

// Name 实现Plugin接口，Name方法
func (tb *tokenBucket) Name() string {
	return PluginName
}

// Initialize 实现Plugin接口，Initialize方法
func (tb *tokenBucket) Initialize(c *plugin.ConfigEntry) error {
	return tb.initialize(c)
}

// Destroy 实现Plugin接口，Destroy方法
func (tb *tokenBucket) Destroy() error {
	return nil
}

// Allow 限流接口实现
func (tb *tokenBucket) Allow(typ plugin.RatelimitType, key string) bool {
	if !tb.config.Enable {
		return true
	}
	return tb.allow(typ, key)
}
