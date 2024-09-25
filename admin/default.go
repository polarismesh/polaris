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

	"github.com/polarismesh/polaris/admin/job"
	"github.com/polarismesh/polaris/auth"
	"github.com/polarismesh/polaris/cache"
	"github.com/polarismesh/polaris/service"
	"github.com/polarismesh/polaris/service/healthcheck"
	"github.com/polarismesh/polaris/store"
)

var (
	server         AdminOperateServer
	maintainServer = &Server{}
	finishInit     bool
)

// Initialize 初始化
func Initialize(ctx context.Context, cfg *Config, namingService service.DiscoverServer,
	healthCheckServer *healthcheck.Server, cacheMgn *cache.CacheManager, storage store.Store) error {

	if finishInit {
		return nil
	}

	err := initialize(ctx, cfg, namingService, healthCheckServer, cacheMgn, storage)
	if err != nil {
		return err
	}

	finishInit = true
	return nil
}

func initialize(_ context.Context, cfg *Config, namingService service.DiscoverServer,
	healthCheckServer *healthcheck.Server, cacheMgn *cache.CacheManager, storage store.Store) error {

	userMgn, err := auth.GetUserServer()
	if err != nil {
		return err
	}

	strategyMgn, err := auth.GetStrategyServer()
	if err != nil {
		return err
	}

	maintainServer.namingServer = namingService
	maintainServer.healthCheckServer = healthCheckServer
	maintainServer.cacheMgn = cacheMgn
	maintainServer.storage = storage

	maintainJobs := job.NewMaintainJobs(namingService, cacheMgn, storage)
	if err := maintainJobs.StartMaintianJobs(cfg.Jobs); err != nil {
		return err
	}

	server = newServerAuthAbility(maintainServer, userMgn, strategyMgn)
	return nil
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
