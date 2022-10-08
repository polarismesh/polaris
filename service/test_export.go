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

package service

import (
	"context"

	"golang.org/x/sync/singleflight"

	"github.com/polarismesh/polaris/auth"
	"github.com/polarismesh/polaris/cache"
	"github.com/polarismesh/polaris/namespace"
	"github.com/polarismesh/polaris/service/batch"
	"github.com/polarismesh/polaris/service/healthcheck"
	"github.com/polarismesh/polaris/store"
)

// TestInitialize 初始化
func TestInitialize(ctx context.Context, namingOpt *Config, cacheOpt *cache.Config, bc *batch.Controller,
	cacheMgr *cache.CacheManager, storage store.Store, namespaceSvr namespace.NamespaceOperateServer,
	healthSvr *healthcheck.Server, authSvr auth.AuthServer) (DiscoverServer, error) {

	namingServer.healthServer = healthSvr

	namingServer.storage = storage

	// 注入命名空间管理模块
	namingServer.namespaceSvr = namespaceSvr

	// cache模块，可以不开启
	// 对于控制台集群，只访问控制台接口的，可以不开启cache
	if cacheOpt.Open {
		log.Infof("[Naming][Server] cache is open, can access the client api function")
		namingServer.caches = cacheMgr
	}

	namingServer.bc = bc

	// l5service
	namingServer.l5service = &l5service{}

	namingServer.createServiceSingle = &singleflight.Group{}

	// 插件初始化
	pluginInitialize()

	return newServerAuthAbility(namingServer, authSvr), nil
}
