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

package admin

import (
	"context"
	"errors"
	"fmt"

	"github.com/polarismesh/polaris/admin/job"
	"github.com/polarismesh/polaris/auth"
	"github.com/polarismesh/polaris/cache"
	"github.com/polarismesh/polaris/service"
	"github.com/polarismesh/polaris/service/healthcheck"
	"github.com/polarismesh/polaris/store"
)

var (
	server               AdminOperateServer
	maintainServer       = &Server{}
	finishInit           bool
	serverProxyFactories = map[string]ServerProxyFactory{}
)

type ServerProxyFactory func(ctx context.Context, pre AdminOperateServer) (AdminOperateServer, error)

func RegisterServerProxy(name string, factor ServerProxyFactory) error {
	if _, ok := serverProxyFactories[name]; ok {
		return fmt.Errorf("duplicate ServerProxyFactory, name(%s)", name)
	}
	serverProxyFactories[name] = factor
	return nil
}

// Initialize 初始化
func Initialize(ctx context.Context, cfg *Config, namingService service.DiscoverServer,
	healthCheckServer *healthcheck.Server, cacheMgn *cache.CacheManager, storage store.Store,
	userSvr auth.UserServer, policySvr auth.StrategyServer) error {

	if finishInit {
		return nil
	}

	proxySvr, actualSvr, err := InitServer(ctx, cfg, namingService, healthCheckServer, cacheMgn,
		storage, userSvr, policySvr)
	if err != nil {
		return err
	}

	server = proxySvr
	maintainServer = actualSvr
	finishInit = true
	return nil
}

func InitServer(ctx context.Context, cfg *Config, namingService service.DiscoverServer,
	healthCheckServer *healthcheck.Server, cacheMgn *cache.CacheManager, storage store.Store,
	userSvr auth.UserServer, policySvr auth.StrategyServer) (AdminOperateServer, *Server, error) {

	actualSvr := new(Server)

	actualSvr.userSvr = userSvr
	actualSvr.policySvr = policySvr
	actualSvr.namingServer = namingService
	actualSvr.healthCheckServer = healthCheckServer
	actualSvr.cacheMgr = cacheMgn
	actualSvr.storage = storage

	maintainJobs := job.NewMaintainJobs(namingService, cacheMgn, storage)
	if err := maintainJobs.StartMaintianJobs(cfg.Jobs); err != nil {
		return nil, nil, err
	}

	var proxySvr AdminOperateServer
	proxySvr = actualSvr
	order := GetChainOrder()
	for i := range order {
		factory, exist := serverProxyFactories[order[i]]
		if !exist {
			return nil, nil, fmt.Errorf("name(%s) not exist in serverProxyFactories", order[i])
		}

		afterSvr, err := factory(ctx, proxySvr)
		if err != nil {
			return nil, nil, err
		}
		proxySvr = afterSvr
	}

	return proxySvr, actualSvr, nil
}

// GetServer 获取已经初始化好的Server
func GetServer() (AdminOperateServer, error) {
	if !finishInit {
		return nil, errors.New("AdminOperateServer has not done Initialize")
	}

	return server, nil
}

// GetOriginServer 获取已经初始化好的Server
func GetOriginServer() (*Server, error) {
	if !finishInit {
		return nil, errors.New("AdminOperateServer has not done Initialize")
	}

	return maintainServer, nil
}
