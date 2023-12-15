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
	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	rawbuffer "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/raw_buffer/v3"
	tlstrans "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/polarismesh/polaris/apiserver/xdsserverv3/resource"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/service"
)

// CDSBuilder .
type CDSBuilder struct {
	svr service.DiscoverServer
}

func (cds *CDSBuilder) Init(svr service.DiscoverServer) {
	cds.svr = svr
}

const (
	SniTemp = "outbound_.default_.%s.%s.svc.cluster.local"
)

func (cds *CDSBuilder) Generate(option *resource.BuildOption) (interface{}, error) {
	var clusters []types.Resource

	// 默认 passthrough cluster
	clusters = append(clusters, resource.PassthroughCluster)

	switch option.TrafficDirection {
	case core.TrafficDirection_INBOUND:
		inBoundClusters, err := cds.GenerateByDirection(option, corev3.TrafficDirection_INBOUND)
		if err != nil {
			return nil, err
		}
		clusters = append(clusters, inBoundClusters...)
	case core.TrafficDirection_OUTBOUND:
		outBoundClusters, err := cds.GenerateByDirection(option, corev3.TrafficDirection_OUTBOUND)
		if err != nil {
			return nil, err
		}
		clusters = append(clusters, outBoundClusters...)
	}
	return clusters, nil
}

func (cds *CDSBuilder) GenerateByDirection(option *resource.BuildOption,
	direction corev3.TrafficDirection) ([]types.Resource, error) {
	var clusters []types.Resource

	selfServiceKey := option.SelfService

	ignore := func(svcKey model.ServiceKey) bool {
		// 如果是 INBOUND 场景，只需要下发 XDS Sidecar Node 所归属的服务 INBOUND Cluster 规则
		isGateway := option.RunType == resource.RunTypeGateway
		if direction == core.TrafficDirection_INBOUND {
			if isGateway {
				return true
			}
			if !isGateway && !selfServiceKey.Equal(&svcKey) {
				return true
			}
		}
		// 如果是网关，则自己的数据不会下发
		if isGateway && selfServiceKey.Equal(&svcKey) {
			return true
		}
		return false
	}

	services := option.Services
	// 每一个 polaris service 对应一个 envoy cluster
	for svcKey, svc := range services {
		if ignore(svcKey) {
			continue
		}
		c := cds.makeCluster(svc, direction, option)
		switch option.TLSMode {
		case resource.TLSModePermissive:
			// In permissive mode, we should use `TLSTransportSocket` to connect to mtls enabled endpoints.
			// Or we use rawbuffer transport for those endpoints which not enabled mtls.
			c.TransportSocketMatches = []*cluster.Cluster_TransportSocketMatch{
				{
					Name:  "tls-mode",
					Match: resource.MTLSTransportSocketMatch,
					TransportSocket: resource.MakeTLSTransportSocket(&tlstrans.UpstreamTlsContext{
						CommonTlsContext: resource.OutboundCommonTLSContext,
						Sni:              fmt.Sprintf(SniTemp, svc.Name, svc.Namespace),
					}),
				},
				{
					Name:  "rawbuffer",
					Match: &structpb.Struct{},
					TransportSocket: &core.TransportSocket{
						Name: wellknown.TransportSocketRawBuffer,
						ConfigType: &core.TransportSocket_TypedConfig{
							TypedConfig: resource.MustNewAny(&rawbuffer.RawBuffer{}),
						},
					},
				},
			}
		case resource.TLSModeStrict:
			// In strict mode, we should only use `TLSTransportSocket` to connect to mtls enabled endpoints.
			c.TransportSocketMatches = []*cluster.Cluster_TransportSocketMatch{
				{
					Name: "tls-mode",
					TransportSocket: resource.MakeTLSTransportSocket(&tlstrans.UpstreamTlsContext{
						CommonTlsContext: resource.OutboundCommonTLSContext,
						Sni:              fmt.Sprintf(SniTemp, svc.Name, svc.Namespace),
					}),
				},
			}
		}
		clusters = append(clusters, c)
	}
	return clusters, nil
}

func (cds *CDSBuilder) makeCluster(svcInfo *resource.ServiceInfo,
	trafficDirection corev3.TrafficDirection, opt *resource.BuildOption) *cluster.Cluster {

	name := resource.MakeServiceName(svcInfo.ServiceKey, trafficDirection, opt)

	return &cluster.Cluster{
		Name:                 name,
		ConnectTimeout:       durationpb.New(5 * time.Second),
		ClusterDiscoveryType: &cluster.Cluster_Type{Type: cluster.Cluster_EDS},
		EdsClusterConfig: &cluster.Cluster_EdsClusterConfig{
			ServiceName: name,
			EdsConfig: &core.ConfigSource{
				ConfigSourceSpecifier: &core.ConfigSource_Ads{
					Ads: &core.AggregatedConfigSource{},
				},
			},
		},
		LbSubsetConfig:   resource.MakeLbSubsetConfig(svcInfo),
		OutlierDetection: resource.MakeOutlierDetection(svcInfo),
		HealthChecks:     resource.MakeHealthCheck(svcInfo),
	}
}
