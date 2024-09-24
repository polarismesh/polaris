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

package paramcheck

import (
	cachetypes "github.com/polarismesh/polaris/cache/api"
	"github.com/polarismesh/polaris/common/log"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/plugin"
	"github.com/polarismesh/polaris/service"
	"github.com/polarismesh/polaris/store"
)

// Server 带有鉴权能力的 discoverServer
//
//	该层会对请求参数做一些调整，根据具体的请求发起人，设置为数据对应的 owner，不可为为别人进行创建资源
type Server struct {
	storage   store.Store
	nextSvr   service.DiscoverServer
	ratelimit plugin.Ratelimit
}

func NewServer(nextSvr service.DiscoverServer) service.DiscoverServer {
	proxy := &Server{
		nextSvr: nextSvr,
	}
	// 获取限流插件
	proxy.ratelimit = plugin.GetRatelimit()
	if proxy.ratelimit == nil {
		log.Warnf("Not found Ratelimit Plugin")
	}
	return proxy
}

// Cache Get cache management
func (svr *Server) Cache() cachetypes.CacheManager {
	return svr.nextSvr.Cache()
}

// GetServiceInstanceRevision 获取服务实例的版本号
func (svr *Server) GetServiceInstanceRevision(serviceID string,
	instances []*model.Instance) (string, error) {
	return svr.nextSvr.GetServiceInstanceRevision(serviceID, instances)
}
