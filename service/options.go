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
	"github.com/polarismesh/polaris/cache"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/namespace"
	"github.com/polarismesh/polaris/service/batch"
	"github.com/polarismesh/polaris/service/healthcheck"
	"github.com/polarismesh/polaris/store"
)

type InitOption func(s *Server)

func WithNamespaceSvr(svr namespace.NamespaceOperateServer) InitOption {
	return func(s *Server) {
		s.namespaceSvr = svr
	}
}

func WithHealthCheckSvr(svr *healthcheck.Server) InitOption {
	return func(s *Server) {
		s.healthServer = svr
	}
}

func WithStorage(storage store.Store) InitOption {
	return func(s *Server) {
		s.storage = storage
	}
}

func WithCacheManager(cacheOpt *cache.Config, c *cache.CacheManager) InitOption {
	return func(s *Server) {
		if cacheOpt.Open {
			log.Infof("[Naming][Server] cache is open, can access the client api function")
			s.caches = c
		}
	}
}

func WithBatchController(c *batch.Controller) InitOption {
	return func(s *Server) {
		s.bc = c
	}
}

func WithHiddenService(c map[model.ServiceKey]struct{}) InitOption {
	return func(s *Server) {
		s.polarisServiceSet = c
	}
}
