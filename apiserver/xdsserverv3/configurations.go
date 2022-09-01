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

package xdsserverv3

import (
	"time"

	accesslog "github.com/envoyproxy/go-control-plane/envoy/config/accesslog/v3"
	cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	filev3 "github.com/envoyproxy/go-control-plane/envoy/extensions/access_loggers/file/v3"
	httpinspector "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/listener/http_inspector/v3"
	tlsinspector "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/listener/tls_inspector/v3"
	hcm "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	rawbuffer "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/raw_buffer/v3"
	tlstrans "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	upstreams_http "github.com/envoyproxy/go-control-plane/envoy/extensions/upstreams/http/v3"
	matcherv3 "github.com/envoyproxy/go-control-plane/envoy/type/matcher/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/golang/protobuf/ptypes/duration"
	"github.com/golang/protobuf/ptypes/wrappers"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/structpb"
)

func mustNewAny(src proto.Message) *anypb.Any {
	a, _ := anypb.New(src)
	return a
}

var rawBufferTransportSocket = &core.TransportSocket{
	Name: wellknown.TransportSocketRawBuffer,
	ConfigType: &core.TransportSocket_TypedConfig{
		TypedConfig: mustNewAny(&rawbuffer.RawBuffer{}),
	},
}

var rawBufferTransportSocketMatch = &cluster.Cluster_TransportSocketMatch{
	Name:            "rawbuffer",
	Match:           &structpb.Struct{},
	TransportSocket: rawBufferTransportSocket,
}

var defaultSdsConfig = &core.ConfigSource{
	ConfigSourceSpecifier: &core.ConfigSource_ApiConfigSource{
		ApiConfigSource: &core.ApiConfigSource{
			ApiType:             core.ApiConfigSource_GRPC,
			TransportApiVersion: core.ApiVersion_V3,
			GrpcServices: []*core.GrpcService{
				{
					TargetSpecifier: &core.GrpcService_EnvoyGrpc_{
						EnvoyGrpc: &core.GrpcService_EnvoyGrpc{
							ClusterName: "sds-grpc",
						},
					},
				},
			},
			SetNodeOnFirstMessageOnly: true,
		},
	},
	InitialFetchTimeout: &duration.Duration{},
	ResourceApiVersion:  core.ApiVersion_V3,
}

var mtlsTransportSocketMatch = &structpb.Struct{
	Fields: map[string]*structpb.Value{
		"acceptMTLS": {Kind: &structpb.Value_StringValue{StringValue: "true"}},
	},
}

var outboundCommonTLSContext = &tlstrans.CommonTlsContext{
	TlsCertificateSdsSecretConfigs: []*tlstrans.SdsSecretConfig{
		{
			Name:      "default",
			SdsConfig: defaultSdsConfig,
		},
	},
	ValidationContextType: &tlstrans.CommonTlsContext_CombinedValidationContext{
		CombinedValidationContext: &tlstrans.CommonTlsContext_CombinedCertificateValidationContext{
			DefaultValidationContext: &tlstrans.CertificateValidationContext{},
			ValidationContextSdsSecretConfig: &tlstrans.SdsSecretConfig{
				Name:      "ROOTCA",
				SdsConfig: defaultSdsConfig,
			},
		},
	},
}

var passthroughCluster = &cluster.Cluster{
	Name:                 "PassthroughCluster",
	ConnectTimeout:       durationpb.New(5 * time.Second),
	ClusterDiscoveryType: &cluster.Cluster_Type{Type: cluster.Cluster_ORIGINAL_DST},
	LbPolicy:             cluster.Cluster_CLUSTER_PROVIDED,
	CircuitBreakers: &cluster.CircuitBreakers{
		Thresholds: []*cluster.CircuitBreakers_Thresholds{
			{
				MaxConnections:     &wrappers.UInt32Value{Value: 4294967295},
				MaxPendingRequests: &wrappers.UInt32Value{Value: 4294967295},
				MaxRequests:        &wrappers.UInt32Value{Value: 4294967295},
				MaxRetries:         &wrappers.UInt32Value{Value: 4294967295},
			},
		},
	},
}

var inboundCommonTLSContext = &tlstrans.CommonTlsContext{
	TlsParams: &tlstrans.TlsParameters{
		TlsMinimumProtocolVersion: tlstrans.TlsParameters_TLSv1_2,
		CipherSuites: []string{
			"ECDHE-ECDSA-AES256-GCM-SHA384",
			"ECDHE-RSA-AES256-GCM-SHA384",
			"ECDHE-ECDSA-AES128-GCM-SHA256",
			"ECDHE-RSA-AES128-GCM-SHA256",
			"AES256-GCM-SHA384",
			"AES128-GCM-SHA256",
		},
	},
	TlsCertificateSdsSecretConfigs: []*tlstrans.SdsSecretConfig{
		{
			Name:      "default",
			SdsConfig: defaultSdsConfig,
		},
	},
	ValidationContextType: &tlstrans.CommonTlsContext_CombinedValidationContext{
		CombinedValidationContext: &tlstrans.CommonTlsContext_CombinedCertificateValidationContext{
			DefaultValidationContext: &tlstrans.CertificateValidationContext{
				MatchSubjectAltNames: []*matcherv3.StringMatcher{
					{
						MatchPattern: &matcherv3.StringMatcher_Prefix{
							Prefix: "spiffe://cluster.local/",
						},
					},
				},
			},
			ValidationContextSdsSecretConfig: &tlstrans.SdsSecretConfig{
				Name:      "ROOTCA",
				SdsConfig: defaultSdsConfig,
			},
		},
	},
}

var inboundCluster = &cluster.Cluster{
	Name:           "Inbound",
	ConnectTimeout: durationpb.New(5 * time.Second),
	TypedExtensionProtocolOptions: map[string]*anypb.Any{
		"envoy.extensions.upstreams.http.v3.HttpProtocolOptions": mustNewAny(&upstreams_http.HttpProtocolOptions{
			UpstreamProtocolOptions: &upstreams_http.HttpProtocolOptions_UseDownstreamProtocolConfig{
				UseDownstreamProtocolConfig: &upstreams_http.HttpProtocolOptions_UseDownstreamHttpConfig{
					Http2ProtocolOptions: &core.Http2ProtocolOptions{
						MaxConcurrentStreams: &wrappers.UInt32Value{Value: 1073741824},
					},
					HttpProtocolOptions: &core.Http1ProtocolOptions{},
				},
			},
		}),
	},
	UpstreamBindConfig: &core.BindConfig{
		SourceAddress: &core.SocketAddress{
			Address: "127.0.0.6",
			PortSpecifier: &core.SocketAddress_PortValue{
				PortValue: 0,
			},
		},
	},
	ClusterDiscoveryType: &cluster.Cluster_Type{Type: cluster.Cluster_ORIGINAL_DST},
	LbPolicy:             cluster.Cluster_CLUSTER_PROVIDED,
	CircuitBreakers: &cluster.CircuitBreakers{
		Thresholds: []*cluster.CircuitBreakers_Thresholds{
			{
				MaxConnections:     &wrappers.UInt32Value{Value: 4294967295},
				MaxPendingRequests: &wrappers.UInt32Value{Value: 4294967295},
				MaxRequests:        &wrappers.UInt32Value{Value: 4294967295},
				MaxRetries:         &wrappers.UInt32Value{Value: 4294967295},
			},
		},
	},
}

func makeTLSTransportSocket(ctx proto.Message) *core.TransportSocket {
	tls := mustNewAny(ctx)
	return &core.TransportSocket{
		Name: "envoy.transport_sockets.tls",
		ConfigType: &core.TransportSocket_TypedConfig{
			TypedConfig: tls,
		},
	}
}

func inboundHCM() *hcm.HttpConnectionManager {
	return &hcm.HttpConnectionManager{
		StatPrefix: "Inbound",
		HttpFilters: []*hcm.HttpFilter{
			{
				Name: wellknown.Router,
			},
		},
		AccessLog: []*accesslog.AccessLog{
			{
				Name: wellknown.FileAccessLog,
				ConfigType: &accesslog.AccessLog_TypedConfig{
					TypedConfig: mustNewAny(&filev3.FileAccessLog{
						Path: "/dev/stdout",
					}),
				},
			},
		},
		HttpProtocolOptions: &core.Http1ProtocolOptions{AcceptHttp_10: true},
		RouteSpecifier: &hcm.HttpConnectionManager_RouteConfig{
			RouteConfig: &route.RouteConfiguration{
				Name:             "Inbound",
				ValidateClusters: &wrappers.BoolValue{Value: false},
				VirtualHosts: []*route.VirtualHost{
					{
						Name:    "inbound|http|0",
						Domains: []string{"*"},
						Routes: []*route.Route{
							{
								Name: "default",
								Match: &route.RouteMatch{
									PathSpecifier: &route.RouteMatch_Prefix{
										Prefix: "/",
									},
								},
								Action: &route.Route_Route{
									Route: &route.RouteAction{
										ClusterSpecifier: &route.RouteAction_Cluster{
											Cluster: "Inbound",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func inboundHCMF() *listener.Filter {
	return &listener.Filter{
		Name: "envoy.filters.network.http_connection_manager",
		ConfigType: &listener.Filter_TypedConfig{
			TypedConfig: mustNewAny(inboundHCM()),
		},
	}
}

func inboundStrictListener() *listener.Listener {
	l := inboundListener()
	l.DefaultFilterChain = nil
	return l
}

func inboundListener() *listener.Listener {
	return &listener.Listener{
		Name:             "virtualInbound",
		TrafficDirection: core.TrafficDirection_INBOUND,
		UseOriginalDst:   &wrappers.BoolValue{Value: true},
		Address: &core.Address{
			Address: &core.Address_SocketAddress{
				SocketAddress: &core.SocketAddress{
					Address: "0.0.0.0",
					PortSpecifier: &core.SocketAddress_PortValue{
						PortValue: 15006,
					},
				},
			},
		},
		DefaultFilterChain: &listener.FilterChain{
			Filters: []*listener.Filter{inboundHCMF()},
			Name:    "virtualInbound-catchall",
		},
		FilterChains: []*listener.FilterChain{
			{
				FilterChainMatch: &listener.FilterChainMatch{
					TransportProtocol: "tls",
				},
				TransportSocket: makeTLSTransportSocket(&tlstrans.DownstreamTlsContext{
					CommonTlsContext: inboundCommonTLSContext,
					RequireClientCertificate: &wrappers.BoolValue{
						Value: true,
					},
				}),
				Filters: []*listener.Filter{inboundHCMF()},
				Name:    "virtualInbound-catchall-tls",
			},
		},
		ListenerFilters: []*listener.ListenerFilter{
			{
				Name: "envoy.filters.listener.tls_inspector",
				ConfigType: &listener.ListenerFilter_TypedConfig{
					TypedConfig: mustNewAny(&tlsinspector.TlsInspector{}),
				},
			},
			{
				Name: "envoy.filters.listener.http_inspector",
				ConfigType: &listener.ListenerFilter_TypedConfig{
					TypedConfig: mustNewAny(&httpinspector.HttpInspector{}),
				},
			},
		},
	}
}
