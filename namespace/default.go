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
	"errors"
	"sync"

	"golang.org/x/sync/singleflight"

	"github.com/polarismesh/polaris/auth"
	"github.com/polarismesh/polaris/cache"
	cachetypes "github.com/polarismesh/polaris/cache/api"
	"github.com/polarismesh/polaris/plugin"
	"github.com/polarismesh/polaris/store"
)

var (
	server          NamespaceOperateServer
	namespaceServer = &Server{}
	once            sync.Once
	finishInit      bool
)

type Config struct {
	AutoCreate bool `yaml:"autoCreate"`
}

// Initialize 初始化
func Initialize(ctx context.Context, nsOpt *Config, storage store.Store, cacheMgn *cache.CacheManager) error {
	var err error
	once.Do(func() {
		err = initialize(ctx, nsOpt, storage, cacheMgn)
	})

	if err != nil {
		return err
	}

	finishInit = true
	return nil
}

func initialize(_ context.Context, nsOpt *Config, storage store.Store, cacheMgn *cache.CacheManager) error {
	if err := cacheMgn.OpenResourceCache(cachetypes.ConfigEntry{
		Name: cachetypes.NamespaceName,
	}); err != nil {
		return err
	}
	namespaceServer.caches = cacheMgn
	namespaceServer.storage = storage
	namespaceServer.cfg = *nsOpt
	namespaceServer.createNamespaceSingle = &singleflight.Group{}

	// 获取History插件，注意：插件的配置在bootstrap已经设置好
	namespaceServer.history = plugin.GetHistory()
	if namespaceServer.history == nil {
		log.Warn("Not Found History Log Plugin")
	}

	userMgn, err := auth.GetUserServer()
	if err != nil {
		return err
	}

	strategyMgn, err := auth.GetStrategyServer()
	if err != nil {
		return err
	}

	server = newServerAuthAbility(namespaceServer, userMgn, strategyMgn)
	return nil
}

// GetServer 获取已经初始化好的Server
func GetServer() (NamespaceOperateServer, error) {
	if !finishInit {
		return nil, errors.New("NamespaceOperateServer has not done Initialize")
	}

	return server, nil
}

// GetOriginServer 获取已经初始化好的Server
func GetOriginServer() (*Server, error) {
	if !finishInit {
		return nil, errors.New("NamespaceOperateServer has not done Initialize")
	}

	return namespaceServer, nil
}
