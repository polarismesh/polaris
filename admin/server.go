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
	"sync"

	"github.com/polarismesh/polaris/auth"
	"github.com/polarismesh/polaris/cache"
	"github.com/polarismesh/polaris/service"
	"github.com/polarismesh/polaris/service/healthcheck"
	"github.com/polarismesh/polaris/store"
)

var _ AdminOperateServer = (*Server)(nil)

type Server struct {
	mu                sync.Mutex
	namingServer      service.DiscoverServer
	healthCheckServer *healthcheck.Server
	cacheMgr          *cache.CacheManager
	storage           store.Store
	userSvr           auth.UserServer
	policySvr         auth.StrategyServer
}

func GetChainOrder() []string {
	return []string{
		"auth",
	}
}
