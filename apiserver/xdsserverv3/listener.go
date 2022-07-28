package xdsserverv3

import (
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	hcm "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	tcp "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/tcp_proxy/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/golang/protobuf/ptypes"
)

func makeListeners() []types.Resource {
	manager := &hcm.HttpConnectionManager{
		CodecType:  hcm.HttpConnectionManager_AUTO,
		StatPrefix: "http",
		RouteSpecifier: &hcm.HttpConnectionManager_Rds{
			Rds: &hcm.Rds{
				ConfigSource: &core.ConfigSource{
					ResourceApiVersion: resource.DefaultAPIVersion,
					ConfigSourceSpecifier: &core.ConfigSource_Ads{
						Ads: &core.AggregatedConfigSource{},
					},
				},
				RouteConfigName: "polaris-router",
			},
		},
		HttpFilters: []*hcm.HttpFilter{{
			Name: wellknown.Router,
		}},
	}

	pbst, err := ptypes.MarshalAny(manager)
	if err != nil {
		panic(err)
	}

	tcpConfig := &tcp.TcpProxy{
		StatPrefix: "PassthroughCluster",
		ClusterSpecifier: &tcp.TcpProxy_Cluster{
			Cluster: "PassthroughCluster",
		},
	}

	tcpC, err := ptypes.MarshalAny(tcpConfig)
	if err != nil {
		panic(err)
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
					Name: "envoy.filters.listener.original_dst",
				},
			},
		},
	}
}

func makePermissiveListeners() []types.Resource {
	resources := makeListeners()
	resources = append(resources, inboundListener())
	return resources
}

func makeStrictListeners() []types.Resource {
	resources := makeListeners()
	resources = append(resources, inboundStrictListener())
	return resources
}
