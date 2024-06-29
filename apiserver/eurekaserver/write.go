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
	"strings"

	"github.com/golang/protobuf/ptypes/wrappers"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"
	"go.uber.org/zap"

	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/model"
	commonstore "github.com/polarismesh/polaris/common/store"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/service"
)

func checkOrBuildNewInstanceId(appId string, instId string, generateUniqueInstId bool) string {
	if !generateUniqueInstId {
		return instId
	}
	lowerAppId := strings.ToLower(appId)
	lowerInstIdId := strings.ToLower(instId)
	if strings.Contains(lowerInstIdId, lowerAppId) {
		return instId
	}
	return lowerAppId + ":" + lowerInstIdId
}

func checkOrBuildNewInstanceIdByNamespace(namespace string, defaultNamespace string, appId string,
	instId string, generateUniqueInstId bool) string {
	instId = checkOrBuildNewInstanceId(appId, instId, generateUniqueInstId)
	if namespace != defaultNamespace {
		return namespace + ":" + instId
	}
	return instId
}

func buildBaseInstance(
	instance *InstanceInfo, namespace string, defaultNamespace string,
	appId string, generateUniqueInstId bool) *apiservice.Instance {
	targetInstance := &apiservice.Instance{}
	eurekaMetadata := make(map[string]string)

	eurekaMetadata[MetadataRegisterFrom] = ServerEureka
	eurekaInstanceId := instance.InstanceId
	if len(eurekaInstanceId) == 0 {
		eurekaInstanceId = instance.HostName
	}
	eurekaMetadata[MetadataInstanceId] = eurekaInstanceId
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
	targetInstance.Id = &wrappers.StringValue{
		Value: checkOrBuildNewInstanceIdByNamespace(namespace, defaultNamespace,
			appId, eurekaInstanceId, generateUniqueInstId),
	}
	targetInstance.Metadata = eurekaMetadata
	targetInstance.Service = &wrappers.StringValue{Value: appId}
	targetInstance.Namespace = &wrappers.StringValue{Value: namespace}
	targetInstance.Host = &wrappers.StringValue{Value: instance.IpAddr}
	if instance.Metadata != nil && len(instance.Metadata.Meta) > 0 {
		targetInstance.Location = &apimodel.Location{}
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

func buildHealthCheck(instance *InstanceInfo, targetInstance *apiservice.Instance, eurekaMetadata map[string]string) {
	leaseInfo := instance.LeaseInfo
	durationInSecs := DefaultDuration
	renewalIntervalInSecs := DefaultRenewInterval
	if leaseInfo != nil {
		if leaseInfo.RenewalIntervalInSecs != 0 {
			renewalIntervalInSecs = leaseInfo.RenewalIntervalInSecs
		}
		if leaseInfo.DurationInSecs != 0 {
			durationInSecs = leaseInfo.DurationInSecs
		}
	}
	eurekaMetadata[MetadataRenewalInterval] = strconv.Itoa(renewalIntervalInSecs)
	eurekaMetadata[MetadataDuration] = strconv.Itoa(durationInSecs)
	durationMin := math.Ceil(float64(durationInSecs) / 3)
	ttl := uint32(math.Min(durationMin, float64(renewalIntervalInSecs)))

	targetInstance.EnableHealthCheck = &wrappers.BoolValue{Value: true}
	targetInstance.HealthCheck = &apiservice.HealthCheck{
		Type:      apiservice.HealthCheck_HEARTBEAT,
		Heartbeat: &apiservice.HeartbeatHealthCheck{Ttl: &wrappers.UInt32Value{Value: ttl}},
	}
}

func buildStatus(instance *InstanceInfo, targetInstance *apiservice.Instance) {
	// eureka注册的实例默认healthy为true，即使设置为false也会被心跳触发变更为true
	// eureka实例非UP状态设置isolate为true，进行流量隔离
	targetInstance.Healthy = &wrappers.BoolValue{Value: true}
	targetInstance.Isolate = &wrappers.BoolValue{Value: false}
	if instance.Status != StatusUp {
		targetInstance.Isolate = &wrappers.BoolValue{Value: true}
	}
}

func convertEurekaInstance(
	instance *InstanceInfo, namespace string, defaultNamespace string,
	appId string, generateUniqueInstId bool) *apiservice.Instance {
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

	targetInstance := buildBaseInstance(instance, namespace, defaultNamespace, appId, generateUniqueInstId)

	// 同时打开2个端口，通过medata保存http端口
	targetInstance.Protocol = &wrappers.StringValue{Value: InsecureProtocol}
	targetInstance.Port = &wrappers.UInt32Value{Value: uint32(insecurePort)}
	targetInstance.Metadata[MetadataInsecurePort] = strconv.Itoa(insecurePort)
	targetInstance.Metadata[MetadataInsecurePortEnabled] = strconv.FormatBool(insecureEnable)
	targetInstance.Metadata[MetadataSecurePort] = strconv.Itoa(securePort)
	targetInstance.Metadata[MetadataSecurePortEnabled] = strconv.FormatBool(secureEnable)
	// 保存客户端注册时设置的 status 信息，该信息不会随着心跳的变化而调整
	targetInstance.Metadata[InternalMetadataStatus] = instance.Status
	targetInstance.Metadata[InternalMetadataOverriddenStatus] = instance.OverriddenStatus
	return targetInstance
}

func (h *EurekaServer) registerInstances(
	ctx context.Context, namespace string, appId string, instance *InstanceInfo, replicated bool) uint32 {
	ctx = context.WithValue(
		ctx, model.CtxEventKeyMetadata, map[string]string{MetadataReplicate: strconv.FormatBool(replicated)})
	ctx = context.WithValue(ctx, utils.ContextOpenAsyncRegis, h.allowAsyncRegis)
	appId = formatWriteName(appId)
	// 1. 先转换数据结构
	totalInstance := convertEurekaInstance(instance, namespace, h.namespace, appId, h.generateUniqueInstId)
	// 3. 注册实例
	resp := h.namingServer.RegisterInstance(ctx, totalInstance)
	// 4. 注册成功，则返回
	if resp.GetCode().GetValue() == api.ExecuteSuccess || resp.GetCode().GetValue() == api.ExistedResource {
		return api.ExecuteSuccess
	}
	// 5. 如果报服务不存在，对服务进行注册
	if resp.Code.Value == api.NotFoundResource {
		svc := &apiservice.Service{}
		svc.Namespace = &wrappers.StringValue{Value: namespace}
		svc.Name = &wrappers.StringValue{Value: appId}
		svcResp := h.namingServer.CreateServices(ctx, []*apiservice.Service{svc})
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

func (h *EurekaServer) deregisterInstance(
	ctx context.Context, namespace string, appId string, instanceId string, replicated bool) uint32 {
	ctx = context.WithValue(
		ctx, model.CtxEventKeyMetadata, map[string]string{
			MetadataReplicate:  strconv.FormatBool(replicated),
			MetadataInstanceId: instanceId,
		})
	ctx = context.WithValue(ctx, utils.ContextOpenAsyncRegis, true)
	instanceId = checkOrBuildNewInstanceIdByNamespace(namespace, h.namespace, appId, instanceId, h.generateUniqueInstId)
	resp := h.namingServer.DeregisterInstance(ctx, &apiservice.Instance{Id: &wrappers.StringValue{Value: instanceId}})
	return resp.GetCode().GetValue()
}

func (h *EurekaServer) updateStatus(
	ctx context.Context, namespace string, appId string, instanceId string, status string, replicated bool) uint32 {
	ctx = context.WithValue(
		ctx, model.CtxEventKeyMetadata, map[string]string{
			MetadataReplicate:  strconv.FormatBool(replicated),
			MetadataInstanceId: instanceId,
		})
	instanceId = checkOrBuildNewInstanceIdByNamespace(namespace, h.namespace, appId, instanceId, h.generateUniqueInstId)

	svr := h.originDiscoverSvr.(*service.Server)
	saveIns, err := svr.Store().GetInstance(instanceId)
	if err != nil {
		eurekalog.Error("[EUREKA-SERVER] get instance from store when update status", zap.Error(err))
		return uint32(commonstore.StoreCode2APICode(err))
	}
	if saveIns == nil {
		return uint32(apimodel.Code_NotFoundInstance)
	}

	metadata := saveIns.Metadata()
	metadata[InternalMetadataStatus] = status
	isolated := status != StatusUp

	updateIns := &apiservice.Instance{
		Id:       &wrappers.StringValue{Value: instanceId},
		Isolate:  &wrappers.BoolValue{Value: isolated},
		Metadata: metadata,
	}

	resp := h.namingServer.UpdateInstance(ctx, updateIns)
	return resp.GetCode().GetValue()
}

func (h *EurekaServer) renew(ctx context.Context, namespace string, appId string,
	instanceId string, replicated bool) uint32 {
	ctx = context.WithValue(
		ctx, model.CtxEventKeyMetadata, map[string]string{
			MetadataReplicate:  strconv.FormatBool(replicated),
			MetadataInstanceId: instanceId,
		})
	instanceId = checkOrBuildNewInstanceIdByNamespace(namespace, h.namespace, appId, instanceId, h.generateUniqueInstId)
	resp := h.healthCheckServer.Report(ctx, &apiservice.Instance{Id: &wrappers.StringValue{Value: instanceId}})
	code := resp.GetCode().GetValue()

	// 如果目标实例存在，但是没有开启心跳，对于 eureka 来说，仍然属于心跳上报成功
	if code == api.HeartbeatOnDisabledIns {
		return api.ExecuteSuccess
	}
	return code
}

func (h *EurekaServer) updateMetadata(
	ctx context.Context, namespace string, appId string, instanceId string, metadata map[string]string) uint32 {
	metadata[MetadataInstanceId] = instanceId
	instanceId = checkOrBuildNewInstanceIdByNamespace(namespace, h.namespace, appId, instanceId, h.generateUniqueInstId)
	resp := h.namingServer.UpdateInstance(ctx,
		&apiservice.Instance{Id: &wrappers.StringValue{Value: instanceId}, Metadata: metadata})
	return resp.GetCode().GetValue()
}
