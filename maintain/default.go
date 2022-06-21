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

package maintain

import (
	"context"
	"errors"

	"github.com/polarismesh/polaris-server/auth"
	"github.com/polarismesh/polaris-server/service"
	"github.com/polarismesh/polaris-server/service/healthcheck"
)

var (
	server         MaintainOperateServer
	maintainServer = &Server{}
	finishInit     bool
)

// Initialize 初始化
func Initialize(ctx context.Context, namingService service.DiscoverServer, healthCheckServer *healthcheck.Server) error {
	if finishInit {
		return nil
	}

	err := initialize(ctx, namingService, healthCheckServer)
	if err != nil {
		return err
	}

	finishInit = true
	return nil
}

func initialize(_ context.Context, namingService service.DiscoverServer, healthCheckServer *healthcheck.Server) error {
	authServer, err := auth.GetAuthServer()
	if err != nil {
		return err
	}

	maintainServer.namingServer = namingService
	maintainServer.healthCheckServer = healthCheckServer

	server = newServerAuthAbility(maintainServer, authServer)
	return nil
}

// GetServer 获取已经初始化好的Server
func GetServer() (MaintainOperateServer, error) {
	if !finishInit {
		return nil, errors.New("MaintainOperateServer has not done Initialize")
	}

	return server, nil
}

// GetOriginServer 获取已经初始化好的Server
func GetOriginServer() (*Server, error) {
	if !finishInit {
		return nil, errors.New("MaintainOperateServer has not done Initialize")
	}

	return maintainServer, nil
}
