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
	"context"
	"io"
	"strconv"
	"strings"
	"time"

	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	endpoint "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	healthservice "github.com/envoyproxy/go-control-plane/envoy/service/health/v3"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/wrappers"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	"github.com/polarismesh/specification/source/go/api/v1/service_manage"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/polarismesh/polaris/apiserver/xdsserverv3/resource"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/namespace"
	"github.com/polarismesh/polaris/service"
)

//  1. Envoy starts up and if its can_healthcheck option in the static
//     bootstrap config is enabled, sends HealthCheckRequest to the management
//     server. It supplies its capabilities (which protocol it can health check
//     with, what zone it resides in, etc.).
//  2. In response to (1), the management server designates this Envoy as a
//     healthchecker to health check a subset of all upstream hosts for a given
//     cluster (for example upstream Host 1 and Host 2). It streams
//     HealthCheckSpecifier messages with cluster related configuration for all
//     clusters this Envoy is designated to health check. Subsequent
//     HealthCheckSpecifier message will be sent on changes to:
//     a. Endpoints to health checks
//     b. Per cluster configuration change
//  3. Envoy creates a health probe based on the HealthCheck config and sends
//     it to endpoint(ip:port) of Host 1 and 2. Based on the HealthCheck
//     configuration Envoy waits upon the arrival of the probe response and
//     looks at the content of the response to decide whether the endpoint is
//     healthy or not. If a response hasn't been received within the timeout
//     interval, the endpoint health status is considered TIMEOUT.
//  4. Envoy reports results back in an EndpointHealthResponse message.
//     Envoy streams responses as often as the interval configured by the
//     management server in HealthCheckSpecifier.
//  5. The management Server collects health statuses for all endpoints in the
//     cluster (for all clusters) and uses this information to construct
//     EndpointDiscoveryResponse messages.
//  6. Once Envoy has a list of upstream endpoints to send traffic to, it load
//     balances traffic to them without additional health checking. It may
//     use inline healthcheck (i.e. consider endpoint UNHEALTHY if connection
//     failed to a particular endpoint to account for health status propagation
//     delay between HDS and EDS).
//
// By default, can_healthcheck is true. If can_healthcheck is false, Cluster
// configuration may not contain HealthCheck message.
// TODO(htuch): How is can_healthcheck communicated to CDS to ensure the above
// invariant?
// TODO(htuch): Add @amb67's diagram.
func (x *XDSServer) StreamHealthCheck(checksvr healthservice.HealthDiscoveryService_StreamHealthCheckServer) error {
	ctx := utils.ConvertGRPCContext(checksvr.Context())
	clientIP, _ := ctx.Value(utils.StringContext("client-ip")).(string)
	clientAddress, _ := ctx.Value(utils.StringContext("client-address")).(string)
	userAgent, _ := ctx.Value(utils.StringContext("user-agent")).(string)

	log.Info("[XDSV3] receive envoy node stream healthcheck",
		zap.String("client-ip", clientIP),
		zap.String("client-address", clientAddress),
		zap.String("user-agent", userAgent),
	)

	for {
		req, err := checksvr.Recv()
		if err != nil {
			if io.EOF == err {
				return nil
			}
			return err
		}

		var client *resource.XDSClient
		checkReq := req.GetHealthCheckRequest()
		if checkReq != nil {
			client = resource.ParseXDSClient(checkReq.Node)
			code := x.registerService(context.Background(), client)
			if code != apimodel.Code_ExecuteSuccess {
				return status.Errorf(codes.Unavailable, "fail to register services, code is %v", code)
			}
			resp := buildHealthCheckSpecifier(client)
			if log.DebugEnabled() {
				hdsRespStr := toJsonStr(resp)
				log.Debugf("[XDSV3] send hds register resp to channel %s, value is \n%s",
					clientAddress, hdsRespStr)
			}
			if err = checksvr.Send(resp); nil != err {
				log.Errorf("[XDSV3] fail to send hds register resp to channel %s, err is %v",
					clientAddress, err)
				return err
			}
		}
		if endpointHealthResponse := req.GetEndpointHealthResponse(); nil != endpointHealthResponse {
			// 处理心跳上报
			if client == nil {
				return status.Errorf(codes.NotFound, "xds node info not found")
			}
			err := x.processEndpointHealthResponse(client, endpointHealthResponse)
			if nil != err {
				return err
			}
		}
	}
}

func buildHealthCheckSpecifier(proxy *resource.XDSClient) *healthservice.HealthCheckSpecifier {
	var minTtl = defaultTTl
	specifier := &healthservice.HealthCheckSpecifier{}
	if len(proxy.GetRegisterServices()) == 0 {
		return specifier
	}
	for _, serviceInfo := range proxy.GetRegisterServices() {
		if len(serviceInfo.Ports) == 0 {
			continue
		}
		if serviceInfo.HealthCheckTtl > 0 && serviceInfo.HealthCheckTtl < minTtl {
			minTtl = serviceInfo.HealthCheckTtl
		}
		for _, ports := range serviceInfo.Ports {
			for _, port := range ports {
				chc := &healthservice.ClusterHealthCheck{}
				specifier.ClusterHealthChecks = append(specifier.ClusterHealthChecks, chc)
				chc.ClusterName = buildSubsetName(core.TrafficDirection_INBOUND, port, serviceInfo.Name)
				chc.HealthChecks = []*corev3.HealthCheck{
					buildHealthCheck(serviceInfo.HealthCheckPath),
				}
				localityEndpoints := &healthservice.LocalityEndpoints{}
				theEndpoint := &endpoint.Endpoint{
					Address: &corev3.Address{
						Address: &core.Address_SocketAddress{
							SocketAddress: &core.SocketAddress{
								Address:       proxy.IPAddr,
								PortSpecifier: &core.SocketAddress_PortValue{PortValue: uint32(port)},
							}},
					},
				}
				localityEndpoints.Endpoints = append(localityEndpoints.Endpoints, theEndpoint)
				chc.LocalityEndpoints = append(chc.LocalityEndpoints, localityEndpoints)
			}
		}
	}
	targetTtl := time.Second * time.Duration(minTtl)
	specifier.Interval = durationpb.New(targetTtl)
	return specifier
}

const (
	SubsetSep = "|"
)

// buildSubsetName to combine the direction and hostname into unique name
func buildSubsetName(direction corev3.TrafficDirection, port int, svcName string) string {
	builder := &strings.Builder{}
	builder.WriteString(direction.String())
	builder.WriteString(SubsetSep)
	builder.WriteString(strconv.Itoa(port))
	if len(svcName) > 0 {
		builder.WriteString(SubsetSep)
		builder.WriteString(svcName)
	}
	return builder.String()
}

func buildHealthCheck(path string) *core.HealthCheck {
	timeout := 1 * time.Second
	interval := 5 * time.Second
	var unhealthyThreshold uint32 = 2
	var healthyThreshold uint32 = 1
	healthCheck := &core.HealthCheck{
		Timeout:            durationpb.New(timeout),
		Interval:           durationpb.New(interval),
		UnhealthyThreshold: wrapperspb.UInt32(unhealthyThreshold),
		HealthyThreshold:   wrapperspb.UInt32(healthyThreshold),
	}
	if len(path) > 0 {
		healthCheck.HealthChecker = &core.HealthCheck_HttpHealthCheck_{
			HttpHealthCheck: &core.HealthCheck_HttpHealthCheck{
				Path: path,
			},
		}
	} else {
		healthCheck.HealthChecker = &core.HealthCheck_TcpHealthCheck_{
			TcpHealthCheck: &core.HealthCheck_TcpHealthCheck{},
		}
	}
	return healthCheck
}

func toJsonStr(msg proto.Message) string {
	marshaler := &jsonpb.Marshaler{}
	jsonStr, _ := marshaler.MarshalToString(msg)
	return jsonStr
}

// TODO(htuch): Unlike the gRPC version, there is no stream-based binding of
// request/response. Should we add an identifier to the HealthCheckSpecifier
// to bind with the response?
func (x *XDSServer) FetchHealthCheck(ctx context.Context,
	req *healthservice.HealthCheckRequestOrEndpointHealthResponse) (*healthservice.HealthCheckSpecifier, error) {

	// TODO: implement
	return nil, status.Errorf(codes.Unimplemented, "FetchHealthCheck unimplemented")
}

const (
	// 默认是5s的心跳间隔
	defaultTTl = 10
)

func convertInstances(client *resource.XDSClient, registerFrom string) map[model.ServiceKey][]*service_manage.Instance {
	if len(client.GetRegisterServices()) == 0 {
		return nil
	}
	var local *apimodel.Location
	if client.Node.Locality != nil {
		local = &apimodel.Location{
			Region: wrapperspb.String(client.Node.Locality.Region),
			Zone:   wrapperspb.String(client.Node.Locality.Zone),
			Campus: wrapperspb.String(client.Node.Locality.SubZone),
		}
	}

	svcInstances := make(map[model.ServiceKey][]*service_manage.Instance, len(client.GetRegisterServices()))
	for _, svc := range client.GetRegisterServices() {
		if len(svc.Ports) == 0 {
			continue
		}
		for protocol, ports := range svc.Ports {
			for _, port := range ports {
				instance := &service_manage.Instance{}
				instance.Location = local
				instance.Namespace = &wrappers.StringValue{Value: client.GetSelfNamespace()}
				instance.Service = &wrappers.StringValue{Value: svc.Name}
				instance.Host = &wrappers.StringValue{Value: client.IPAddr}
				instance.Port = &wrappers.UInt32Value{Value: uint32(port)}
				if len(client.Version) > 0 {
					instance.Version = &wrappers.StringValue{Value: client.Version}
				}
				if len(protocol) > 0 {
					instance.Protocol = &wrappers.StringValue{Value: protocol}
				}
				instance.EnableHealthCheck = &wrappers.BoolValue{Value: true}
				ttl := svc.HealthCheckTtl
				if ttl == 0 {
					ttl = defaultTTl
				}
				instance.HealthCheck = &service_manage.HealthCheck{
					Heartbeat: &service_manage.HeartbeatHealthCheck{
						Ttl: &wrappers.UInt32Value{Value: uint32(ttl)},
					},
					Type: service_manage.HealthCheck_HEARTBEAT,
				}
				instance.Metadata = make(map[string]string)
				instance.Metadata[model.MetadataRegisterFrom] = registerFrom
				if len(svc.HealthCheckPath) > 0 {
					instance.Metadata[model.MetadataInternalMetaHealthCheckPath] = svc.HealthCheckPath
				}
				if svc.TracingSampling > 0 {
					instance.Metadata[model.MetadataInternalMetaTraceSampling] = strconv.Itoa(int(svc.TracingSampling))
				}
				metadata := client.Metadata
				if len(metadata) > 0 {
					for k, v := range metadata {
						instance.Metadata[k] = v
					}
				}

				svcKey := model.ServiceKey{
					Namespace: client.GetSelfNamespace(),
					Name:      svc.Name,
				}

				if existInstances, ok := svcInstances[svcKey]; ok {
					existInstances = append(existInstances, instance)
					svcInstances[svcKey] = existInstances
				} else {
					svcInstances[svcKey] = []*apiservice.Instance{instance}
				}
			}
		}
	}
	return svcInstances
}

func (x *XDSServer) registerService(ctx context.Context, client *resource.XDSClient) apimodel.Code {
	svcInstances := convertInstances(client, x.GetProtocol())
	if len(svcInstances) == 0 {
		return apimodel.Code_ExecuteSuccess
	}

	// 允许自动创建命名空间以及服务
	ctx = namespace.AllowAutoCreate(ctx)
	ctx = service.AllowAutoCreate(ctx)

	for _, instances := range svcInstances {
		// 1. 注册实例
		resp := x.namingServer.CreateInstances(ctx, instances)
		code := apimodel.Code(resp.GetCode().GetValue())
		// 2. 注册成功，则返回
		if code == apimodel.Code_ExecuteSuccess || code == apimodel.Code_ExistedResource {
			continue
		}
		// 3. 如果报服务不存在，对服务进行注册
		if code == apimodel.Code_NotFoundResource {
			// 4. 继续注册实例
			resp = x.namingServer.CreateInstances(ctx, instances)
			code = apimodel.Code(resp.GetCode().GetValue())
			if code == apimodel.Code_ExecuteSuccess || code == apimodel.Code_ExistedResource {
				continue
			}
		}
		return code
	}
	return apimodel.Code_ExecuteSuccess
}

func (x *XDSServer) processEndpointHealthResponse(
	client *resource.XDSClient, endpointHealthResponse *healthservice.EndpointHealthResponse) error {
	// 处理心跳上报
	endpoints := endpointHealthResponse.GetClusterEndpointsHealth()
	if len(endpoints) == 0 {
		return nil
	}
	log.Debugf("heartbeat for service(%s) namespace(%s)", client.GetSelfService(), client.GetSelfNamespace())
	namespaceName := client.GetSelfNamespace()
	serviceName := client.GetSelfService()
	ctx := context.Background()
	for _, targetEndpoint := range endpoints {
		var host string
		var port uint32
		localityEndpoints := targetEndpoint.GetLocalityEndpointsHealth()
		for _, localityEndpoint := range localityEndpoints {
			for _, endopint := range localityEndpoint.EndpointsHealth {
				address := endopint.GetEndpoint().GetAddress().GetAddress()
				if socketAddr, ok := address.(*corev3.Address_SocketAddress); ok {
					host = socketAddr.SocketAddress.Address
					portSpecifier := socketAddr.SocketAddress.PortSpecifier
					if portValue, ok := portSpecifier.(*corev3.SocketAddress_PortValue); ok {
						port = portValue.PortValue
					}
				}
				if len(namespaceName) == 0 || len(host) == 0 || port == 0 {
					log.Errorf("tuple arguments is invalid, namespace %s, host %s, port %d", namespaceName, host, port)
					return status.Errorf(codes.InvalidArgument,
						"tuple arguments is invalid, namespace %s, host %s, port %d",
						namespaceName, host, port)
				}

				ins := &service_manage.Instance{
					Namespace: utils.NewStringValue(client.GetSelfNamespace()),
					Service:   utils.NewStringValue(client.GetSelfService()),
					Host:      utils.NewStringValue(host),
					Port:      utils.NewUInt32Value(port),
				}

				switch endopint.GetHealthStatus() {
				case corev3.HealthStatus_HEALTHY:
					resp := x.healthSvr.Report(ctx, ins)
					code := apimodel.Code(resp.GetCode().GetValue())
					if code != apimodel.Code_ExecuteSuccess && code != apimodel.Code_HeartbeatExceedLimit {
						log.Errorf("[XdsV2Server] fail to do heartbeat, namespace %s, service %s, host %s, port %d, err is %v",
							namespaceName, serviceName, host, port, resp.GetInfo().GetValue())
						return status.Errorf(codes.InvalidArgument,
							"fail to do heartbeat, code is %d, namespace %s, service %s, host %s, port %d",
							code, namespaceName, serviceName, host, port)
					}
				case corev3.HealthStatus_DRAINING:
					// 进行反注册
					resp := x.namingServer.DeregisterInstance(ctx, ins)
					code := apimodel.Code(resp.GetCode().GetValue())
					if code != apimodel.Code_ExecuteSuccess && code != apimodel.Code_NotFoundResource {
						log.Errorf("[XdsV2Server] fail to process endpoint health, err is %v", resp.GetInfo().GetValue())
						return status.Errorf(codes.InvalidArgument,
							"fail to do deregister, code is %d, namespace %s, service %s, host %s, port %d",
							code, namespaceName, serviceName, host, port)
					}
				default:
				}
			}
		}

	}
	return nil
}
