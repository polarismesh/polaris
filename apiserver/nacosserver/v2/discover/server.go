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

package discover

import (
	"github.com/polarismesh/polaris/apiserver/nacosserver/core"
	nacospb "github.com/polarismesh/polaris/apiserver/nacosserver/v2/pb"
	"github.com/polarismesh/polaris/apiserver/nacosserver/v2/remote"
	"github.com/polarismesh/polaris/auth"
	"github.com/polarismesh/polaris/namespace"
	"github.com/polarismesh/polaris/service"
	"github.com/polarismesh/polaris/service/healthcheck"
)

type ServerOption struct {
	// nacos
	ConnectionManager *remote.ConnectionManager
	Store             *core.NacosDataStorage

	// polaris
	UserSvr           auth.UserServer
	NamespaceSvr      namespace.NamespaceOperateServer
	DiscoverSvr       service.DiscoverServer
	OriginDiscoverSvr service.DiscoverServer
	HealthSvr         *healthcheck.Server
}

type DiscoverServer struct {
	clientManager *ConnectionClientManager
	connMgr       *remote.ConnectionManager

	pushCenter     core.PushCenter
	store          *core.NacosDataStorage
	handleRegistry map[string]*remote.RequestHandlerWarrper
	checker        *Checker

	namespaceSvr      namespace.NamespaceOperateServer
	discoverSvr       service.DiscoverServer
	originDiscoverSvr service.DiscoverServer
	healthSvr         *healthcheck.Server
}

func (h *DiscoverServer) Initialize(option *ServerOption) error {
	var err error
	h.namespaceSvr = option.NamespaceSvr
	h.discoverSvr = option.DiscoverSvr
	h.originDiscoverSvr = option.OriginDiscoverSvr
	h.healthSvr = option.HealthSvr
	h.store = option.Store
	h.connMgr = option.ConnectionManager

	h.clientManager, err = NewConnectionClientManager([]ClientConnectionInterceptor{h})
	if err != nil {
		return err
	}
	h.checker, err = newChecker(h.originDiscoverSvr, h.healthSvr, h.connMgr, h.clientManager)
	if err != nil {
		return err
	}
	grpcPush, err := NewGrpcPushCenter(h.store, h.sendPushData)
	if err != nil {
		return err
	}
	h.pushCenter = grpcPush
	h.initGRPCHandlers()
	return nil
}

func (h *DiscoverServer) initGRPCHandlers() {
	h.handleRegistry = map[string]*remote.RequestHandlerWarrper{
		// Request
		nacospb.TypeInstanceRequest: {
			Handler: h.handleInstanceRequest,
			PayloadBuilder: func() nacospb.CustomerPayload {
				return nacospb.NewInstanceRequest()
			},
		},
		// Request
		nacospb.TypePersistentInstanceRequest: {
			Handler: h.handleInstanceRequest,
			PayloadBuilder: func() nacospb.CustomerPayload {
				return nacospb.NewPersistentInstanceRequest()
			},
		},
		nacospb.TypeBatchInstanceRequest: {
			Handler: h.handleBatchInstanceRequest,
			PayloadBuilder: func() nacospb.CustomerPayload {
				return nacospb.NewBatchInstanceRequest()
			},
		},
		nacospb.TypeSubscribeServiceRequest: {
			Handler: h.handleSubscribeServiceReques,
			PayloadBuilder: func() nacospb.CustomerPayload {
				return nacospb.NewSubscribeServiceRequest()
			},
		},
		nacospb.TypeServiceListRequest: {
			Handler: h.handleServiceListRequest,
			PayloadBuilder: func() nacospb.CustomerPayload {
				return nacospb.NewServiceListRequest()
			},
		},
		nacospb.TypeServiceQueryRequest: {
			Handler: h.handleServiceQueryRequest,
			PayloadBuilder: func() nacospb.CustomerPayload {
				return nacospb.NewServiceQueryRequest()
			},
		},
		// RequestBiStream
		nacospb.TypeConnectionSetupRequest: {
			PayloadBuilder: func() nacospb.CustomerPayload {
				return nacospb.NewConnectionSetupRequest()
			},
		},
		nacospb.TypeSubscribeServiceResponse: {
			PayloadBuilder: func() nacospb.CustomerPayload {
				return &nacospb.SubscribeServiceResponse{}
			},
		},
		nacospb.TypeNotifySubscriberResponse: {
			PayloadBuilder: func() nacospb.CustomerPayload {
				return &nacospb.NotifySubscriberResponse{}
			},
		},
	}
}

func (h *DiscoverServer) ListGRPCHandlers() map[string]*remote.RequestHandlerWarrper {
	return h.handleRegistry
}
