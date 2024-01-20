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

package resource

import (
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	tlstrans "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	matcherv3 "github.com/envoyproxy/go-control-plane/envoy/type/matcher/v3"
	"github.com/golang/protobuf/ptypes/duration"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

var DefaultSdsConfig = &core.ConfigSource{
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
}

var MTLSTransportSocketMatch = &structpb.Struct{
	Fields: map[string]*structpb.Value{
		"acceptMTLS": {Kind: &structpb.Value_StringValue{StringValue: "true"}},
	},
}

var OutboundCommonTLSContext = &tlstrans.CommonTlsContext{
	TlsCertificateSdsSecretConfigs: []*tlstrans.SdsSecretConfig{
		{
			Name:      "default",
			SdsConfig: DefaultSdsConfig,
		},
	},
	ValidationContextType: &tlstrans.CommonTlsContext_CombinedValidationContext{
		CombinedValidationContext: &tlstrans.CommonTlsContext_CombinedCertificateValidationContext{
			DefaultValidationContext: &tlstrans.CertificateValidationContext{},
			ValidationContextSdsSecretConfig: &tlstrans.SdsSecretConfig{
				Name:      "ROOTCA",
				SdsConfig: DefaultSdsConfig,
			},
		},
	},
}

var InboundCommonTLSContext = &tlstrans.CommonTlsContext{
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
			SdsConfig: DefaultSdsConfig,
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
				SdsConfig: DefaultSdsConfig,
			},
		},
	},
}

func MakeTLSTransportSocket(ctx proto.Message) *core.TransportSocket {
	tls := MustNewAny(ctx)
	return &core.TransportSocket{
		Name: "envoy.transport_sockets.tls",
		ConfigType: &core.TransportSocket_TypedConfig{
			TypedConfig: tls,
		},
	}
}
