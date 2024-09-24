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

	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"
	"golang.org/x/sync/singleflight"

	"github.com/polarismesh/polaris/auth"
	"github.com/polarismesh/polaris/cache"
	cachetypes "github.com/polarismesh/polaris/cache/api"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/namespace"
	"github.com/polarismesh/polaris/service/batch"
	"github.com/polarismesh/polaris/service/healthcheck"
	"github.com/polarismesh/polaris/store"
)

// GetBatchController .
func (s *Server) GetBatchController() *batch.Controller {
	return s.bc
}

// MockBatchController .
func (s *Server) MockBatchController(bc *batch.Controller) {
	s.bc = bc
}

func TestNewServer(mockStore store.Store, nsSvr namespace.NamespaceOperateServer,
	cacheMgr *cache.CacheManager) *Server {
	return &Server{
		storage:             mockStore,
		namespaceSvr:        nsSvr,
		caches:              cacheMgr,
		createServiceSingle: &singleflight.Group{},
		hooks:               []ResourceHook{},
	}
}

// TestInitialize 初始化
func TestInitialize(ctx context.Context, namingOpt *Config, cacheOpt *cache.Config,
	cacheEntries []cachetypes.ConfigEntry, bc *batch.Controller, cacheMgr *cache.CacheManager,
	storage store.Store, namespaceSvr namespace.NamespaceOperateServer,
	healthSvr *healthcheck.Server,
	userMgn auth.UserServer, strategyMgn auth.StrategyServer) (DiscoverServer, DiscoverServer, error) {
	entrites := []cachetypes.ConfigEntry{}
	if len(cacheEntries) != 0 {
		entrites = cacheEntries
	} else {
		entrites = GetAllCaches()
	}

	actualSvr, proxySvr, err := InitServer(ctx, namingOpt,
		WithBatchController(bc),
		WithCacheManager(cacheOpt, cacheMgr, entrites...),
		WithHealthCheckSvr(healthSvr),
		WithNamespaceSvr(namespaceSvr),
		WithStorage(storage),
	)
	namingServer = actualSvr
	return proxySvr, namingServer, err
}

// TestSerialCreateInstance .
func (s *Server) TestSerialCreateInstance(
	ctx context.Context, svcId string, req *apiservice.Instance, ins *apiservice.Instance) (
	*model.Instance, *apiservice.Response) {
	return s.serialCreateInstance(ctx, svcId, req, ins)
}

// TestSetStore .
func (s *Server) TestSetStore(storage store.Store) {
	s.storage = storage
}

// TestIsEmptyLocation .
func TestIsEmptyLocation(loc *apimodel.Location) bool {
	return isEmptyLocation(loc)
}
