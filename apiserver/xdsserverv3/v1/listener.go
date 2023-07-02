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

package v1

import (
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	routerv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/router/v3"
	original_dstv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/listener/original_dst/v3"
	hcm "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	tcp "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/tcp_proxy/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	resourcev3 "github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/golang/protobuf/ptypes"
	"github.com/polarismesh/polaris/apiserver/xdsserverv3/resource"
)

func makeListeners() ([]types.Resource, error) {
	manager := &hcm.HttpConnectionManager{
		CodecType:  hcm.HttpConnectionManager_AUTO,
		StatPrefix: "http",
		RouteSpecifier: &hcm.HttpConnectionManager_Rds{
			Rds: &hcm.Rds{
				ConfigSource: &core.ConfigSource{
					ResourceApiVersion: resourcev3.DefaultAPIVersion,
					ConfigSourceSpecifier: &core.ConfigSource_Ads{
						Ads: &core.AggregatedConfigSource{},
					},
				},
				RouteConfigName: "polaris-router",
			},
		},
		HttpFilters: []*hcm.HttpFilter{},
	}
	manager.HttpFilters = append(manager.HttpFilters, &hcm.HttpFilter{
		Name: wellknown.Router,
		ConfigType: &hcm.HttpFilter_TypedConfig{
			TypedConfig: resource.MustNewAny(&routerv3.Router{}),
		},
	})

	pbst, err := ptypes.MarshalAny(manager)
	if err != nil {
		return nil, err
	}

	tcpConfig := &tcp.TcpProxy{
		StatPrefix: "PassthroughCluster",
		ClusterSpecifier: &tcp.TcpProxy_Cluster{
			Cluster: "PassthroughCluster",
		},
	}

	tcpC, err := ptypes.MarshalAny(tcpConfig)
	if err != nil {
		return nil, err
	}

	return []types.Resource{
		&listener.Listener{
			Name: "listener_15001",
			Address: &core.Address{
				Address: &core.Address_SocketAddress{
					SocketAddress: &core.SocketAddress{
						Protocol: core.SocketAddress_TCP,
						Address:  "0.0.0.0",
						PortSpecifier: &core.SocketAddress_PortValue{
							PortValue: 15001,
						},
					},
				},
			},
			FilterChains: []*listener.FilterChain{
				{
					Filters: []*listener.Filter{
						{
							Name: wellknown.HTTPConnectionManager,
							ConfigType: &listener.Filter_TypedConfig{
								TypedConfig: pbst,
							},
						},
					},
				},
			},
			DefaultFilterChain: &listener.FilterChain{
				Name: "PassthroughFilterChain",
				Filters: []*listener.Filter{
					{
						Name: wellknown.TCPProxy,
						ConfigType: &listener.Filter_TypedConfig{
							TypedConfig: tcpC,
						},
					},
				},
			},
			ListenerFilters: []*listener.ListenerFilter{
				{
					// type.googleapis.com/envoy.extensions.filters.listener.original_dst.v3.OriginalDst
					Name: wellknown.OriginalDestination,
					ConfigType: &listener.ListenerFilter_TypedConfig{
						TypedConfig: resource.MustNewAny(&original_dstv3.OriginalDst{}),
					},
				},
			},
		},
	}, nil
}

func makePermissiveListeners() ([]types.Resource, error) {
	resources, err := makeListeners()
	if err != nil {
		return nil, err
	}
	resources = append(resources, inboundListener())
	return resources, nil
}

func makeStrictListeners() ([]types.Resource, error) {
	resources, err := makeListeners()
	if err != nil {
		return nil, err
	}
	resources = append(resources, inboundStrictListener())
	return resources, nil
}
