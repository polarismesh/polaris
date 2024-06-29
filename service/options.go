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
	cachetypes "github.com/polarismesh/polaris/cache/api"
	"github.com/polarismesh/polaris/namespace"
	"github.com/polarismesh/polaris/service/batch"
	"github.com/polarismesh/polaris/service/healthcheck"
	"github.com/polarismesh/polaris/store"
)

/*
   - name: users # Load user and user group data
   - name: strategyRule # Loading the rules of appraisal
   - name: namespace # Load the naming space data
   - name: client # Load Client-SDK instance data
*/

func GetRegisterCaches() []cachetypes.ConfigEntry {
	ret := []cachetypes.ConfigEntry{}
	// ret = append(ret, l5CacheEntry)
	ret = append(ret, namingCacheEntries...)
	return ret
}

func GetAllCaches() []cachetypes.ConfigEntry {
	ret := []cachetypes.ConfigEntry{}
	ret = append(ret, l5CacheEntry)
	ret = append(ret, namingCacheEntries...)
	ret = append(ret, governanceCacheEntries...)
	return ret
}

var (
	l5CacheEntry = cachetypes.ConfigEntry{
		Name: cachetypes.L5Name,
	}
	namingCacheEntries = []cachetypes.ConfigEntry{
		{
			Name: cachetypes.ServiceName,
			Option: map[string]interface{}{
				"disableBusiness": false,
				"needMeta":        true,
			},
		},
		{
			Name: cachetypes.InstanceName,
			Option: map[string]interface{}{
				"disableBusiness": false,
				"needMeta":        true,
			},
		},
		{
			Name: cachetypes.ServiceContractName,
		},
		{
			Name: cachetypes.ClientName,
		},
	}
	governanceCacheEntries = []cachetypes.ConfigEntry{
		{
			Name: cachetypes.RoutingConfigName,
		},
		{
			Name: cachetypes.RateLimitConfigName,
		},
		{
			Name: cachetypes.CircuitBreakerName,
		},
		{
			Name: cachetypes.FaultDetectRuleName,
		},
		{
			Name: cachetypes.LaneRuleName,
		},
	}
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

func WithCacheManager(cacheOpt *cache.Config, c cachetypes.CacheManager, entries ...cachetypes.ConfigEntry) InitOption {
	return func(s *Server) {
		log.Infof("[Naming][Server] cache is open, can access the client api function")
		if len(entries) != 0 {
			_ = c.OpenResourceCache(entries...)
		} else {
			_ = c.OpenResourceCache(namingCacheEntries...)
			_ = c.OpenResourceCache(governanceCacheEntries...)
			if s.isSupportL5() {
				_ = c.OpenResourceCache(l5CacheEntry)
			}
		}
		s.caches = c
	}
}

func WithBatchController(c *batch.Controller) InitOption {
	return func(s *Server) {
		s.bc = c
	}
}
