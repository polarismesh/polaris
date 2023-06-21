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

package v2

import (
	"strconv"
	"strings"

	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	endpoint "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	_struct "github.com/golang/protobuf/ptypes/struct"
	"github.com/polarismesh/polaris/apiserver/xdsserverv3/resource"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/service"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"
)

// EDSBuilder .
type EDSBuilder struct {
	client *resource.XDSClient
	svr    service.DiscoverServer
}

func (eds *EDSBuilder) Init(client *resource.XDSClient, svr service.DiscoverServer) {
	eds.client = client
	eds.svr = svr
}

func (eds *EDSBuilder) Generate(option *resource.BuildOption) (interface{}, error) {
	var resources []types.Resource
	switch eds.client.RunType {
	case resource.RunTypeGateway:
		resources = append(resources, eds.makeBoundEndpoints(option)...)
	case resource.RunTypeSidecar:
		// sidecar 场景，如果流量方向是 envoy -> sidecar 的话，那么 endpoint 只能是 本地 127.0.0.1
		if option.TrafficDirection == corev3.TrafficDirection_INBOUND {
			resources = append(resources, eds.makeSelfEndpoint(option)...)
		} else {
			resources = append(resources, eds.makeBoundEndpoints(option)...)
		}
	}
	return resources, nil
}

func (eds *EDSBuilder) makeBoundEndpoints(option *resource.BuildOption) []types.Resource {
	services := option.Services

	var clusterLoads []types.Resource
	for svcKey, serviceInfo := range services {
		var lbEndpoints []*endpoint.LbEndpoint
		for _, instance := range serviceInfo.Instances {
			// 只加入健康的实例
			if !isNormalEndpoint(instance) {
				continue
			}
			ep := &endpoint.LbEndpoint{
				HostIdentifier: &endpoint.LbEndpoint_Endpoint{
					Endpoint: &endpoint.Endpoint{
						Address: &core.Address{
							Address: &core.Address_SocketAddress{
								SocketAddress: &core.SocketAddress{
									Protocol: core.SocketAddress_TCP,
									Address:  instance.Host.Value,
									PortSpecifier: &core.SocketAddress_PortValue{
										PortValue: instance.Port.Value,
									},
								},
							},
						},
					},
				},
				HealthStatus:        formatEndpointHealth(instance),
				LoadBalancingWeight: utils.NewUInt32Value(instance.GetWeight().GetValue()),
				Metadata:            getEndpointMetaFromPolarisIns(instance),
			}
			lbEndpoints = append(lbEndpoints, ep)
		}

		cla := &endpoint.ClusterLoadAssignment{
			ClusterName: resource.MakeServiceName(svcKey, option.TrafficDirection),
			Endpoints: []*endpoint.LocalityLbEndpoints{
				{
					LbEndpoints: lbEndpoints,
				},
			},
		}
		clusterLoads = append(clusterLoads, cla)
	}
	return clusterLoads
}

func (eds *EDSBuilder) makeSelfEndpoint(option *resource.BuildOption) []types.Resource {
	var clusterLoads []types.Resource
	var lbEndpoints []*endpoint.LbEndpoint

	selfServiceKey := model.ServiceKey{
		Namespace: eds.client.GetSelfNamespace(),
		Name:      eds.client.GetSelfService(),
	}

	var portsStr = ""
	selfServiceInfo, ok := option.Services[selfServiceKey]
	if ok {
		portsStr = selfServiceInfo.Ports
	} else {
		// sidecar 的服务没有注册，那就看下 envoy metadata 上有没有设置 sidecar_bindports 标签
		portsStr = eds.client.Metadata[resource.SidecarBindPort]
	}

	ports := strings.Split(portsStr, ",")
	for _, port := range ports {
		portVal, _ := strconv.ParseUint(port, 10, 64)
		ep := &endpoint.LbEndpoint{
			HostIdentifier: &endpoint.LbEndpoint_Endpoint{
				Endpoint: &endpoint.Endpoint{
					Address: &core.Address{
						Address: &core.Address_SocketAddress{
							SocketAddress: &core.SocketAddress{
								Protocol: core.SocketAddress_TCP,
								Address:  "127.0.0.1",
								PortSpecifier: &core.SocketAddress_PortValue{
									PortValue: uint32(portVal),
								},
							},
						},
					},
				},
			},
			HealthStatus: core.HealthStatus_HEALTHY,
		}
		lbEndpoints = append(lbEndpoints, ep)
	}
	cla := &endpoint.ClusterLoadAssignment{
		ClusterName: resource.MakeServiceName(selfServiceKey, option.TrafficDirection),
		Endpoints: []*endpoint.LocalityLbEndpoints{
			{
				LbEndpoints: lbEndpoints,
			},
		},
	}
	clusterLoads = append(clusterLoads, cla)
	return clusterLoads
}

func getEndpointMetaFromPolarisIns(ins *apiservice.Instance) *core.Metadata {
	meta := &core.Metadata{}
	fields := make(map[string]*_struct.Value)
	for k, v := range ins.Metadata {
		fields[k] = &_struct.Value{
			Kind: &_struct.Value_StringValue{
				StringValue: v,
			},
		}
	}

	meta.FilterMetadata = make(map[string]*_struct.Struct)
	meta.FilterMetadata["envoy.lb"] = &_struct.Struct{
		Fields: fields,
	}
	if ins.Metadata != nil && ins.Metadata[resource.TLSModeTag] != "" {
		meta.FilterMetadata["envoy.transport_socket_match"] = resource.MTLSTransportSocketMatch
	}
	return meta
}

func isNormalEndpoint(ins *apiservice.Instance) bool {
	if ins.GetIsolate().GetValue() {
		return false
	}
	if ins.GetWeight().GetValue() == 0 {
		return false
	}
	return true
}

func formatEndpointHealth(ins *apiservice.Instance) core.HealthStatus {
	if ins.GetHealthy().GetValue() {
		return core.HealthStatus_HEALTHY
	}
	return core.HealthStatus_UNHEALTHY
}
