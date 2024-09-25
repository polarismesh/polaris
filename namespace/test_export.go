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

package namespace

import (
	"context"

	"golang.org/x/sync/singleflight"

	"github.com/polarismesh/polaris/auth"
	"github.com/polarismesh/polaris/cache"
	cachetypes "github.com/polarismesh/polaris/cache/api"
	"github.com/polarismesh/polaris/plugin"
	"github.com/polarismesh/polaris/store"
)

func TestInitialize(_ context.Context, nsOpt *Config, storage store.Store, cacheMgn *cache.CacheManager,
	userMgn auth.UserServer, strategyMgn auth.StrategyServer) (NamespaceOperateServer, error) {
	_ = cacheMgn.OpenResourceCache(cachetypes.ConfigEntry{
		Name: cachetypes.NamespaceName,
	})
	nsOpt.AutoCreate = true
	namespaceServer := &Server{}
	namespaceServer.caches = cacheMgn
	namespaceServer.storage = storage
	namespaceServer.cfg = *nsOpt
	namespaceServer.createNamespaceSingle = &singleflight.Group{}

	// 获取History插件，注意：插件的配置在bootstrap已经设置好
	namespaceServer.history = plugin.GetHistory()
	if namespaceServer.history == nil {
		log.Warn("Not Found History Log Plugin")
	}

	return newServerAuthAbility(namespaceServer, userMgn, strategyMgn), nil
}
