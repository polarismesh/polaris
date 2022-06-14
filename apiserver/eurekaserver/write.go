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

package eurekaserver

import (
	"context"
	"math"
	"strconv"

	"github.com/golang/protobuf/ptypes/wrappers"

	api "github.com/polarismesh/polaris-server/common/api/v1"
)

func buildBaseInstance(instance *InstanceInfo, namespace string, appId string) *api.Instance {
	targetInstance := &api.Instance{}
	eurekaMetadata := make(map[string]string)

	eurekaMetadata[MetadataRegisterFrom] = ServerEureka
	if len(instance.AppGroupName) > 0 {
		eurekaMetadata[MetadataAppGroupName] = instance.AppGroupName
	}
	countryIdStr := ObjectToString(instance.CountryId)
	if DefaultCountryId != countryIdStr {
		eurekaMetadata[MetadataCountryId] = countryIdStr
	}
	if instance.DataCenterInfo != nil {
		if DefaultDciClazz != instance.DataCenterInfo.Clazz {
			eurekaMetadata[MetadataDataCenterInfoClazz] = instance.DataCenterInfo.Clazz
		}
		if DefaultDciName != instance.DataCenterInfo.Name {
			eurekaMetadata[MetadataDataCenterInfoName] = instance.DataCenterInfo.Name
		}
	}
	if len(instance.HostName) > 0 {
		eurekaMetadata[MetadataHostName] = instance.HostName
	}
	if len(instance.HomePageUrl) > 0 {
		eurekaMetadata[MetadataHomePageUrl] = instance.HomePageUrl
	}
	if len(instance.StatusPageUrl) > 0 {
		eurekaMetadata[MetadataStatusPageUrl] = instance.StatusPageUrl
	}
	if len(instance.HealthCheckUrl) > 0 {
		eurekaMetadata[MetadataHealthCheckUrl] = instance.HealthCheckUrl
	}
	if len(instance.VipAddress) > 0 {
		eurekaMetadata[MetadataVipAddress] = instance.VipAddress
	}
	if len(instance.SecureVipAddress) > 0 {
		eurekaMetadata[MetadataSecureVipAddress] = instance.SecureVipAddress
	}
	targetInstance.Id = &wrappers.StringValue{Value: instance.InstanceId}
	targetInstance.Metadata = eurekaMetadata
	targetInstance.Service = &wrappers.StringValue{Value: appId}
	targetInstance.Namespace = &wrappers.StringValue{Value: namespace}
	targetInstance.Host = &wrappers.StringValue{Value: instance.IpAddr}
	if instance.Metadata != nil && len(instance.Metadata.Meta) > 0 {
		targetInstance.Location = &api.Location{}
		for k, v := range instance.Metadata.Meta {
			strValue := ObjectToString(v)
			switch k {
			case KeyRegion:
				targetInstance.Location.Region = &wrappers.StringValue{Value: strValue}
			case keyZone:
				targetInstance.Location.Zone = &wrappers.StringValue{Value: strValue}
			case keyCampus:
				targetInstance.Location.Campus = &wrappers.StringValue{Value: strValue}
			}
			targetInstance.Metadata[k] = strValue
		}
	}
	targetInstance.Weight = &wrappers.UInt32Value{Value: 100}
	buildHealthCheck(instance, targetInstance, eurekaMetadata)
	buildStatus(instance, targetInstance)
	return targetInstance
}

func buildHealthCheck(instance *InstanceInfo, targetInstance *api.Instance, eurekaMetadata map[string]string) {
	leaseInfo := instance.LeaseInfo
	var durationInSecs int
	var renewalIntervalInSecs int
	if leaseInfo != nil {
		renewalIntervalInSecs = leaseInfo.RenewalIntervalInSecs
		durationInSecs = leaseInfo.DurationInSecs
		if renewalIntervalInSecs == 0 {
			renewalIntervalInSecs = DefaultRenewInterval
		}
		if durationInSecs == 0 {
			durationInSecs = DefaultDuration
		}
		if renewalIntervalInSecs != DefaultRenewInterval {
			eurekaMetadata[MetadataRenewalInterval] = strconv.Itoa(renewalIntervalInSecs)
		}
		if durationInSecs != DefaultDuration {
			eurekaMetadata[MetadataDuration] = strconv.Itoa(durationInSecs)
		}
	}
	durationMin := math.Ceil(float64(durationInSecs) / 3)
	ttl := uint32(math.Min(durationMin, float64(renewalIntervalInSecs)))

	targetInstance.EnableHealthCheck = &wrappers.BoolValue{Value: true}
	targetInstance.HealthCheck = &api.HealthCheck{
		Type:      api.HealthCheck_HEARTBEAT,
		Heartbeat: &api.HeartbeatHealthCheck{Ttl: &wrappers.UInt32Value{Value: ttl}},
	}
}

func buildStatus(instance *InstanceInfo, targetInstance *api.Instance) {
	// 由于eureka的实例都会自动报心跳，心跳由北极星接管，因此客户端报上来的人工状态OUT_OF_SERVICE，通过isolate来进行代替
	status := instance.Status
	if status == "OUT_OF_SERVICE" {
		targetInstance.Isolate = &wrappers.BoolValue{Value: true}
	} else if status == "UP" {
		targetInstance.Healthy = &wrappers.BoolValue{Value: true}
	} else {
		targetInstance.Healthy = &wrappers.BoolValue{Value: false}
	}
}

func convertEurekaInstance(instance *InstanceInfo, namespace string, appId string) *api.Instance {
	var secureEnable bool
	var securePort int
	var insecureEnable bool
	var insecurePort int

	securePortWrap := instance.SecurePort
	if securePortWrap != nil {
		secureEnable = securePortWrap.RealEnable
		securePort = securePortWrap.RealPort
	} else {
		secureEnable = false
		securePort = DefaultSSLPort
	}
	insecurePortWrap := instance.Port
	if insecurePortWrap != nil {
		insecureEnable = insecurePortWrap.RealEnable
		insecurePort = insecurePortWrap.RealPort
	} else {
		insecureEnable = true
		insecurePort = DefaultInsecurePort
	}

	targetInstance := buildBaseInstance(instance, namespace, appId)

	// 同时打开2个端口，通过medata保存http端口
	targetInstance.Protocol = &wrappers.StringValue{Value: InsecureProtocol}
	targetInstance.Port = &wrappers.UInt32Value{Value: uint32(insecurePort)}
	targetInstance.Metadata[MetadataInsecurePort] = strconv.Itoa(insecurePort)
	targetInstance.Metadata[MetadataInsecurePortEnabled] = strconv.FormatBool(insecureEnable)
	targetInstance.Metadata[MetadataSecurePort] = strconv.Itoa(securePort)
	targetInstance.Metadata[MetadataSecurePortEnabled] = strconv.FormatBool(secureEnable)
	return targetInstance
}

func (h *EurekaServer) registerInstances(ctx context.Context, appId string, instance *InstanceInfo) uint32 {
	// 1. 先转换数据结构
	totalInstance := convertEurekaInstance(instance, h.namespace, appId)
	// 3. 注册实例
	resp := h.namingServer.RegisterInstance(ctx, totalInstance)
	// 4. 注册成功，则返回
	if resp.GetCode().GetValue() == api.ExecuteSuccess || resp.GetCode().GetValue() == api.ExistedResource {
		return api.ExecuteSuccess
	}
	// 5. 如果报服务不存在，对服务进行注册
	if resp.Code.Value == api.NotFoundResource {
		svc := &api.Service{}
		svc.Namespace = &wrappers.StringValue{Value: h.namespace}
		svc.Name = &wrappers.StringValue{Value: appId}
		svcResp := h.namingServer.CreateServices(ctx, []*api.Service{svc})
		svcCreateCode := svcResp.GetCode().GetValue()
		if svcCreateCode != api.ExecuteSuccess && svcCreateCode != api.ExistedResource {
			return svcCreateCode
		}
		// 6. 再重试注册实例列表
		resp = h.namingServer.RegisterInstance(ctx, totalInstance)
		return resp.GetCode().GetValue()
	}
	return resp.GetCode().GetValue()
}

func (h *EurekaServer) deregisterInstance(ctx context.Context, appId string, instanceId string) uint32 {
	resp := h.namingServer.DeregisterInstance(ctx, &api.Instance{Id: &wrappers.StringValue{Value: instanceId}})
	return resp.GetCode().GetValue()
}

func (h *EurekaServer) updateStatus(ctx context.Context, appId string, instanceId string, status string) uint32 {
	var isolated = false
	if status != StatusUp {
		isolated = true
	}
	resp := h.namingServer.UpdateInstances(ctx,
		[]*api.Instance{{Id: &wrappers.StringValue{Value: instanceId}, Isolate: &wrappers.BoolValue{Value: isolated}}})
	return resp.GetCode().GetValue()
}

func (h *EurekaServer) renew(ctx context.Context, appId string, instanceId string) uint32 {
	resp := h.healthCheckServer.Report(ctx, &api.Instance{Id: &wrappers.StringValue{Value: instanceId}})
	return resp.GetCode().GetValue()
}

func (h *EurekaServer) updateMetadata(ctx context.Context, instanceId string, metadata map[string]string) uint32 {
	resp := h.namingServer.UpdateInstances(ctx,
		[]*api.Instance{{Id: &wrappers.StringValue{Value: instanceId}, Metadata: metadata}})
	return resp.GetCode().GetValue()
}
