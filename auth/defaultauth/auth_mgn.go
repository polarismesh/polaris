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

package defaultauth

import (
	"encoding/json"

	"github.com/polarismesh/polaris/auth"
	"github.com/polarismesh/polaris/cache"
	"github.com/polarismesh/polaris/store"
)

// defaultAuthChecker 北极星自带的默认鉴权中心
type defaultAuthChecker struct {
	cacheMgn *cache.CacheManager
}

// Initialize 执行初始化动作
func (d *defaultAuthChecker) Initialize(options *auth.Config, s store.Store, cacheMgn *cache.CacheManager) error {
	contentBytes, err := json.Marshal(options.Option)
	if err != nil {
		return err
	}

	cfg := DefaultAuthConfig()
	if err := json.Unmarshal(contentBytes, cfg); err != nil {
		return err
	}

	if err := cfg.Verify(); err != nil {
		return err
	}

	AuthOption = cfg
	d.cacheMgn = cacheMgn

	return nil
}

// Cache 获取缓存统一管理
func (d *defaultAuthChecker) Cache() *cache.CacheManager {
	return d.cacheMgn
}
