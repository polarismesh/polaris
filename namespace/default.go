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
	"fmt"
	"sync"

	"golang.org/x/sync/singleflight"

	"github.com/polarismesh/polaris/cache"
	cachetypes "github.com/polarismesh/polaris/cache/api"
	"github.com/polarismesh/polaris/store"
)

var (
	server          NamespaceOperateServer
	namespaceServer = &Server{}
	once            sync.Once
	finishInit      bool
	// serverProxyFactories Service Server API 代理工厂
	serverProxyFactories = map[string]ServerProxyFactory{}
)

type ServerProxyFactory func(context.Context, NamespaceOperateServer, cachetypes.CacheManager) (NamespaceOperateServer, error)

func RegisterServerProxy(name string, factor ServerProxyFactory) error {
	if _, ok := serverProxyFactories[name]; ok {
		return fmt.Errorf("duplicate ServerProxyFactory, name(%s)", name)
	}
	serverProxyFactories[name] = factor
	return nil
}

type Config struct {
	AutoCreate   bool     `yaml:"autoCreate"`
	Interceptors []string `yaml:"-"`
}

// Initialize 初始化
func Initialize(ctx context.Context, nsOpt *Config, storage store.Store, cacheMgr *cache.CacheManager) error {
	var err error
	once.Do(func() {
		actualSvr, proxySvr, err := InitServer(ctx, nsOpt, storage, cacheMgr)
		if err != nil {
			return
		}
		namespaceServer = actualSvr
		server = proxySvr
		return
	})

	if err != nil {
		return err
	}

	finishInit = true
	return nil
}

func InitServer(ctx context.Context, nsOpt *Config, storage store.Store,
	cacheMgr *cache.CacheManager) (*Server, NamespaceOperateServer, error) {
	if err := cacheMgr.OpenResourceCache(cachetypes.ConfigEntry{
		Name: cachetypes.NamespaceName,
	}); err != nil {
		return nil, nil, err
	}

	actualSvr := new(Server)
	actualSvr.caches = cacheMgr
	actualSvr.storage = storage
	actualSvr.cfg = *nsOpt
	actualSvr.createNamespaceSingle = &singleflight.Group{}

	var proxySvr NamespaceOperateServer
	proxySvr = actualSvr
	order := GetChainOrder()
	for i := range order {
		factory, exist := serverProxyFactories[order[i]]
		if !exist {
			return nil, nil, fmt.Errorf("name(%s) not exist in serverProxyFactories", order[i])
		}

		afterSvr, err := factory(ctx, proxySvr, cacheMgr)
		if err != nil {
			return nil, nil, err
		}
		proxySvr = afterSvr
	}
	return actualSvr, proxySvr, nil
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

func GetChainOrder() []string {
	return []string{
		"auth",
	}
}
