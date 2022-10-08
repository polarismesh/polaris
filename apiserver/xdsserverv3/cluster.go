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
	"fmt"
	"time"

	cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	tlstrans "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/golang/protobuf/ptypes"

	"github.com/polarismesh/polaris/common/model"
)

type CircuitBreakerConfigGetter func(id string) *model.ServiceWithCircuitBreaker

func (x *XDSServer) makeCluster(service *ServiceInfo) *cluster.Cluster {
	var circuitBreakerConf *model.ServiceWithCircuitBreaker

	if x.CircuitBreakerConfigGetter != nil {
		circuitBreakerConf = x.CircuitBreakerConfigGetter(service.ID)
	} else {
		circuitBreakerConf = x.namingServer.Cache().CircuitBreaker().GetCircuitBreakerConfig(service.ID)
	}

	return &cluster.Cluster{
		Name:                 service.Name,
		ConnectTimeout:       ptypes.DurationProto(5 * time.Second),
		ClusterDiscoveryType: &cluster.Cluster_Type{Type: cluster.Cluster_EDS},
		EdsClusterConfig: &cluster.Cluster_EdsClusterConfig{
			ServiceName: service.Name,
			EdsConfig: &core.ConfigSource{
				ResourceApiVersion: resource.DefaultAPIVersion,
				ConfigSourceSpecifier: &core.ConfigSource_Ads{
					Ads: &core.AggregatedConfigSource{},
				},
			},
		},

		LbSubsetConfig:   makeLbSubsetConfig(service),
		OutlierDetection: makeOutlierDetection(circuitBreakerConf),
	}
}

func (x *XDSServer) makeClusters(services []*ServiceInfo) []types.Resource {
	var clusters []types.Resource
	// 默认 passthrough cluster

	clusters = append(clusters, passthroughCluster)

	// 每一个 polaris service 对应一个 envoy cluster
	for _, service := range services {
		clusters = append(clusters, x.makeCluster(service))
	}

	return clusters
}

func (x *XDSServer) makePermissiveClusters(services []*ServiceInfo) []types.Resource {
	var clusters []types.Resource
	// 默认 passthrough cluster & inbound cluster

	clusters = append(clusters, passthroughCluster, inboundCluster)

	// 每一个 polaris service 对应一个 envoy cluster
	for _, service := range services {
		c := x.makeCluster(service)
		// In permissive mode, we should use `TLSTransportSocket` to connect to mtls enabled endpoints.
		// Or we use rawbuffer transport for those endpoints which not enabled mtls.
		c.TransportSocketMatches = []*cluster.Cluster_TransportSocketMatch{
			{
				Name:  "tls-mode",
				Match: mtlsTransportSocketMatch,
				TransportSocket: makeTLSTransportSocket(&tlstrans.UpstreamTlsContext{
					CommonTlsContext: outboundCommonTLSContext,
					Sni:              fmt.Sprintf("outbound_.default_.%s.%s.svc.cluster.local", service.Name, service.Namespace),
				}),
			},
			rawBufferTransportSocketMatch,
		}

		clusters = append(clusters, c)
	}

	return clusters
}

func (x *XDSServer) makeStrictClusters(services []*ServiceInfo) []types.Resource {
	var clusters []types.Resource
	// 默认 passthrough cluster & inbound cluster

	clusters = append(clusters, passthroughCluster, inboundCluster)

	// 每一个 polaris service 对应一个 envoy cluster
	for _, service := range services {
		c := x.makeCluster(service)

		// In strict mode, we should only use `TLSTransportSocket` to connect to mtls enabled endpoints.
		c.TransportSocketMatches = []*cluster.Cluster_TransportSocketMatch{
			{
				Name: "tls-mode",
				TransportSocket: makeTLSTransportSocket(&tlstrans.UpstreamTlsContext{
					CommonTlsContext: outboundCommonTLSContext,
					Sni:              fmt.Sprintf("outbound_.default_.%s.%s.svc.cluster.local", service.Name, service.Namespace),
				}),
			},
		}

		clusters = append(clusters, c)
	}

	return clusters
}
