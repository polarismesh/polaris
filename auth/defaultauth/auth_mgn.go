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
	"errors"

	"github.com/polarismesh/polaris-server/auth"
	"github.com/polarismesh/polaris-server/cache"
	"github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/plugin"
	"github.com/polarismesh/polaris-server/store"
)

// defaultAuthChecker 北极星自带的默认鉴权中心
type defaultAuthChecker struct {
	cacheMgn   *cache.CacheManager
	authPlugin plugin.Auth
}

// Initialize 执行初始化动作
func (d *defaultAuthChecker) Initialize(options *auth.Config, cacheMgn *cache.CacheManager) error {
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

	// 获取存储层对象
	s, err := store.GetStore()
	if err != nil {
		log.AuthScope().Errorf("[Auth][Server] can not get store, err: %s", err.Error())
		return errors.New("auth-checker can not get store")
	}
	if s == nil {
		log.AuthScope().Errorf("[Auth][Server] store is null")
		return errors.New("store is null")
	}

	authPlugin := plugin.GetAuth()
	if authPlugin == nil {
		return errors.New("AuthChecker needs to configure plugin.Auth plugin for permission check")
	}

	d.cacheMgn = cacheMgn
	d.authPlugin = authPlugin

	return nil
}

// Cache 获取缓存统一管理
func (d *defaultAuthChecker) Cache() *cache.CacheManager {
	return d.cacheMgn
}
