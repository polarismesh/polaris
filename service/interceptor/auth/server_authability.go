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

package service_auth

import (
	"context"
	"errors"

	apifault "github.com/polarismesh/specification/source/go/api/v1/fault_tolerance"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	apisecurity "github.com/polarismesh/specification/source/go/api/v1/security"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"
	apitraffic "github.com/polarismesh/specification/source/go/api/v1/traffic_manage"
	"go.uber.org/zap"

	"github.com/polarismesh/polaris/auth"
	"github.com/polarismesh/polaris/cache"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/service"
)

// ServerAuthAbility 带有鉴权能力的 discoverServer
//
//	该层会对请求参数做一些调整，根据具体的请求发起人，设置为数据对应的 owner，不可为为别人进行创建资源
type ServerAuthAbility struct {
	targetServer *service.Server
	userMgn      auth.UserServer
	strategyMgn  auth.StrategyServer
}

func NewServerAuthAbility(targetServer *service.Server,
	userMgn auth.UserServer, strategyMgn auth.StrategyServer) service.DiscoverServer {
	proxy := &ServerAuthAbility{
		targetServer: targetServer,
		userMgn:      userMgn,
		strategyMgn:  strategyMgn,
	}

	targetServer.SetResourceHooks(proxy)

	return proxy
}

// Cache Get cache management
func (svr *ServerAuthAbility) Cache() *cache.CacheManager {
	return svr.targetServer.Cache()
}

// GetServiceInstanceRevision 获取服务实例的版本号
func (svr *ServerAuthAbility) GetServiceInstanceRevision(serviceID string,
	instances []*model.Instance) (string, error) {
	return svr.targetServer.GetServiceInstanceRevision(serviceID, instances)
}

// collectServiceAuthContext 对于服务的处理，收集所有的与鉴权的相关信息
//
//	@receiver svr ServerAuthAbility
//	@param ctx 请求上下文 ctx
//	@param req 实际请求对象
//	@param resourceOp 该接口的数据操作类型
//	@return *model.AcquireContext 返回鉴权上下文
func (svr *ServerAuthAbility) collectServiceAuthContext(ctx context.Context, req []*apiservice.Service,
	resourceOp model.ResourceOperation, methodName string) *model.AcquireContext {
	return model.NewAcquireContext(
		model.WithRequestContext(ctx),
		model.WithOperation(resourceOp),
		model.WithModule(model.DiscoverModule),
		model.WithMethod(methodName),
		model.WithAccessResources(svr.queryServiceResource(req)),
	)
}

// collectServiceAliasAuthContext 对于服务别名的处理，收集所有的与鉴权的相关信息
//
//	@receiver svr ServerAuthAbility
//	@param ctx 请求上下文 ctx
//	@param req 实际请求对象
//	@param resourceOp 该接口的数据操作类型
//	@return *model.AcquireContext 返回鉴权上下文
func (svr *ServerAuthAbility) collectServiceAliasAuthContext(ctx context.Context, req []*apiservice.ServiceAlias,
	resourceOp model.ResourceOperation, methodName string) *model.AcquireContext {
	return model.NewAcquireContext(
		model.WithRequestContext(ctx),
		model.WithOperation(resourceOp),
		model.WithModule(model.DiscoverModule),
		model.WithMethod(methodName),
		model.WithAccessResources(svr.queryServiceAliasResource(req)),
	)
}

// collectInstanceAuthContext 对于服务实例的处理，收集所有的与鉴权的相关信息
//
//	@receiver svr ServerAuthAbility
//	@param ctx 请求上下文 ctx
//	@param req 实际请求对象
//	@param resourceOp 该接口的数据操作类型
//	@return *model.AcquireContext 返回鉴权上下文
func (svr *ServerAuthAbility) collectInstanceAuthContext(ctx context.Context, req []*apiservice.Instance,
	resourceOp model.ResourceOperation, methodName string) *model.AcquireContext {
	return model.NewAcquireContext(
		model.WithRequestContext(ctx),
		model.WithOperation(resourceOp),
		model.WithModule(model.DiscoverModule),
		model.WithMethod(methodName),
		model.WithAccessResources(svr.queryInstanceResource(req)),
	)
}

// collectClientInstanceAuthContext 对于服务实例的处理，收集所有的与鉴权的相关信息
func (svr *ServerAuthAbility) collectClientInstanceAuthContext(ctx context.Context, req []*apiservice.Instance,
	resourceOp model.ResourceOperation, methodName string) *model.AcquireContext {
	return model.NewAcquireContext(
		model.WithRequestContext(ctx),
		model.WithOperation(resourceOp),
		model.WithModule(model.DiscoverModule),
		model.WithMethod(methodName),
		model.WithFromClient(),
		model.WithAccessResources(svr.queryInstanceResource(req)),
	)
}

// collectCircuitBreakerAuthContext 对于服务熔断的处理，收集所有的与鉴权的相关信息
//
//	@receiver svr ServerAuthAbility
//	@param ctx 请求上下文 ctx
//	@param req 实际请求对象
//	@param resourceOp 该接口的数据操作类型
//	@return *model.AcquireContext 返回鉴权上下文
func (svr *ServerAuthAbility) collectCircuitBreakerAuthContext(ctx context.Context, req []*apifault.CircuitBreaker,
	resourceOp model.ResourceOperation, methodName string) *model.AcquireContext {
	return model.NewAcquireContext(
		model.WithRequestContext(ctx),
		model.WithOperation(resourceOp),
		model.WithModule(model.DiscoverModule),
		model.WithMethod(methodName),
		model.WithAccessResources(svr.queryCircuitBreakerResource(req)),
	)
}

// collectCircuitBreakerReleaseAuthContext
//
//	@receiver svr
//	@param ctx
//	@param req
//	@param resourceOp
//	@return *model.AcquireContext
func (svr *ServerAuthAbility) collectCircuitBreakerReleaseAuthContext(ctx context.Context,
	req []*apiservice.ConfigRelease, resourceOp model.ResourceOperation, methodName string) *model.AcquireContext {
	return model.NewAcquireContext(
		model.WithRequestContext(ctx),
		model.WithOperation(resourceOp),
		model.WithModule(model.DiscoverModule),
		model.WithMethod(methodName),
		model.WithAccessResources(svr.queryCircuitBreakerReleaseResource(req)),
	)
}

// collectRouteRuleAuthContext 对于服务路由规则的处理，收集所有的与鉴权的相关信息
//
//	@receiver svr ServerAuthAbility
//	@param ctx 请求上下文 ctx
//	@param req 实际请求对象
//	@param resourceOp 该接口的数据操作类型
//	@return *model.AcquireContext 返回鉴权上下文
func (svr *ServerAuthAbility) collectRouteRuleAuthContext(ctx context.Context, req []*apitraffic.Routing,
	resourceOp model.ResourceOperation, methodName string) *model.AcquireContext {
	return model.NewAcquireContext(
		model.WithRequestContext(ctx),
		model.WithOperation(resourceOp),
		model.WithModule(model.DiscoverModule),
		model.WithMethod(methodName),
		model.WithAccessResources(svr.queryRouteRuleResource(req)),
	)
}

// collectRateLimitAuthContext 对于服务限流规则的处理，收集所有的与鉴权的相关信息
//
//	@receiver svr ServerAuthAbility
//	@param ctx 请求上下文 ctx
//	@param req 实际请求对象
//	@param resourceOp 该接口的数据操作类型
//	@return *model.AcquireContext 返回鉴权上下文
func (svr *ServerAuthAbility) collectRateLimitAuthContext(ctx context.Context, req []*apitraffic.Rule,
	resourceOp model.ResourceOperation, methodName string) *model.AcquireContext {
	return model.NewAcquireContext(
		model.WithRequestContext(ctx),
		model.WithOperation(resourceOp),
		model.WithModule(model.DiscoverModule),
		model.WithMethod(methodName),
		model.WithAccessResources(svr.queryRateLimitConfigResource(req)),
	)
}

// collectRouteRuleV2AuthContext 收集路由v2规则
func (svr *ServerAuthAbility) collectRouteRuleV2AuthContext(ctx context.Context, req []*apitraffic.RouteRule,
	resourceOp model.ResourceOperation, methodName string) *model.AcquireContext {
	return model.NewAcquireContext(
		model.WithRequestContext(ctx),
		model.WithOperation(resourceOp),
		model.WithModule(model.DiscoverModule),
		model.WithMethod(methodName),
		model.WithAccessResources(map[apisecurity.ResourceType][]model.ResourceEntry{}),
	)
}

// collectRouteRuleV2AuthContext 收集熔断v2规则
func (svr *ServerAuthAbility) collectCircuitBreakerRuleV2AuthContext(ctx context.Context,
	req []*apifault.CircuitBreakerRule,
	resourceOp model.ResourceOperation, methodName string) *model.AcquireContext {
	return model.NewAcquireContext(
		model.WithRequestContext(ctx),
		model.WithOperation(resourceOp),
		model.WithModule(model.DiscoverModule),
		model.WithMethod(methodName),
		model.WithAccessResources(map[apisecurity.ResourceType][]model.ResourceEntry{}),
	)
}

// collectRouteRuleV2AuthContext 收集主动探测规则
func (svr *ServerAuthAbility) collectFaultDetectAuthContext(ctx context.Context,
	req []*apifault.FaultDetectRule,
	resourceOp model.ResourceOperation, methodName string) *model.AcquireContext {
	return model.NewAcquireContext(
		model.WithRequestContext(ctx),
		model.WithOperation(resourceOp),
		model.WithModule(model.DiscoverModule),
		model.WithMethod(methodName),
		model.WithAccessResources(map[apisecurity.ResourceType][]model.ResourceEntry{}),
	)
}

// queryServiceResource  根据所给的 service 信息，收集对应的 ResourceEntry 列表
func (svr *ServerAuthAbility) queryServiceResource(
	req []*apiservice.Service) map[apisecurity.ResourceType][]model.ResourceEntry {
	if len(req) == 0 {
		return make(map[apisecurity.ResourceType][]model.ResourceEntry)
	}

	names := utils.NewSet[string]()
	svcSet := utils.NewMap[string, *model.Service]()

	for index := range req {
		svcName := req[index].GetName().GetValue()
		svcNamespace := req[index].GetNamespace().GetValue()
		names.Add(svcNamespace)
		svc := svr.Cache().Service().GetServiceByName(svcName, svcNamespace)
		if svc != nil {
			svcSet.Store(svc.ID, svc)
		}
	}

	ret := svr.convertToDiscoverResourceEntryMaps(names, svcSet)
	if authLog.DebugEnabled() {
		authLog.Debug("[Auth][Server] collect service access res", zap.Any("res", ret))
	}
	return ret
}

// queryServiceAliasResource  根据所给的 servicealias 信息，收集对应的 ResourceEntry 列表
func (svr *ServerAuthAbility) queryServiceAliasResource(
	req []*apiservice.ServiceAlias) map[apisecurity.ResourceType][]model.ResourceEntry {
	if len(req) == 0 {
		return make(map[apisecurity.ResourceType][]model.ResourceEntry)
	}

	names := utils.NewSet[string]()
	svcSet := utils.NewMap[string, *model.Service]()

	for index := range req {
		refSvcName := req[index].GetService().GetValue()
		refSvcNamespace := req[index].GetNamespace().GetValue()
		svcNamespace := req[index].GetNamespace().GetValue()
		names.Add(svcNamespace)
		refSvc := svr.Cache().Service().GetServiceByName(refSvcName, refSvcNamespace)
		if refSvc != nil {
			svcSet.Store(refSvc.ID, refSvc)
		}
	}

	ret := svr.convertToDiscoverResourceEntryMaps(names, svcSet)
	if authLog.DebugEnabled() {
		authLog.Debug("[Auth][Server] collect service alias access res", zap.Any("res", ret))
	}
	return ret
}

// queryInstanceResource 根据所给的 instances 信息，收集对应的 ResourceEntry 列表
// 由于实例是注册到服务下的，因此只需要判断，是否有对应服务的权限即可
func (svr *ServerAuthAbility) queryInstanceResource(
	req []*apiservice.Instance) map[apisecurity.ResourceType][]model.ResourceEntry {
	if len(req) == 0 {
		return make(map[apisecurity.ResourceType][]model.ResourceEntry)
	}

	names := utils.NewSet[string]()
	svcSet := utils.NewMap[string, *model.Service]()

	for index := range req {
		svcName := req[index].GetService().GetValue()
		svcNamespace := req[index].GetNamespace().GetValue()
		item := req[index]
		if svcNamespace != "" && svcName != "" {
			svc := svr.Cache().Service().GetServiceByName(svcName, svcNamespace)
			if svc != nil {
				svcSet.Store(svc.ID, svc)
			} else {
				names.Add(svcNamespace)
			}
		} else {
			ins := svr.Cache().Instance().GetInstance(item.GetId().GetValue())
			if ins != nil {
				svc := svr.Cache().Service().GetServiceByID(ins.ServiceID)
				if svc != nil {
					svcSet.Store(svc.ID, svc)
				} else {
					names.Add(svcNamespace)
				}
			}
		}
	}

	ret := svr.convertToDiscoverResourceEntryMaps(names, svcSet)
	if authLog.DebugEnabled() {
		authLog.Debug("[Auth][Server] collect instance access res", zap.Any("res", ret))
	}
	return ret
}

// queryCircuitBreakerResource 根据所给的 CircuitBreaker 信息，收集对应的 ResourceEntry 列表
func (svr *ServerAuthAbility) queryCircuitBreakerResource(
	req []*apifault.CircuitBreaker) map[apisecurity.ResourceType][]model.ResourceEntry {
	if len(req) == 0 {
		return make(map[apisecurity.ResourceType][]model.ResourceEntry)
	}

	names := utils.NewSet[string]()
	svcSet := utils.NewMap[string, *model.Service]()

	for index := range req {
		svcName := req[index].GetService().GetValue()
		svcNamespace := req[index].GetNamespace().GetValue()
		svc := svr.Cache().Service().GetServiceByName(svcName, svcNamespace)
		if svc != nil {
			svcSet.Store(svc.ID, svc)
		}
	}
	ret := svr.convertToDiscoverResourceEntryMaps(names, svcSet)
	if authLog.DebugEnabled() {
		authLog.Debug("[Auth][Server] collect circuit-breaker access res", zap.Any("res", ret))
	}
	return ret
}

// queryCircuitBreakerReleaseResource 根据所给的 CircuitBreakerRelease 信息，收集对应的 ResourceEntry 列表
func (svr *ServerAuthAbility) queryCircuitBreakerReleaseResource(
	req []*apiservice.ConfigRelease) map[apisecurity.ResourceType][]model.ResourceEntry {
	if len(req) == 0 {
		return make(map[apisecurity.ResourceType][]model.ResourceEntry)
	}

	names := utils.NewSet[string]()
	svcSet := utils.NewMap[string, *model.Service]()

	for index := range req {
		svcName := req[index].GetService().GetName().GetValue()
		svcNamespace := req[index].GetService().GetNamespace().GetValue()
		svc := svr.Cache().Service().GetServiceByName(svcName, svcNamespace)
		if svc != nil {
			svcSet.Store(svc.ID, svc)
		}
	}

	ret := svr.convertToDiscoverResourceEntryMaps(names, svcSet)
	if authLog.DebugEnabled() {
		authLog.Debug("[Auth][Server] collect circuit-breaker-release access res", zap.Any("res", ret))
	}
	return ret
}

// queryRouteRuleResource 根据所给的 RouteRule 信息，收集对应的 ResourceEntry 列表
func (svr *ServerAuthAbility) queryRouteRuleResource(
	req []*apitraffic.Routing) map[apisecurity.ResourceType][]model.ResourceEntry {
	if len(req) == 0 {
		return make(map[apisecurity.ResourceType][]model.ResourceEntry)
	}

	names := utils.NewSet[string]()
	svcSet := utils.NewMap[string, *model.Service]()

	for index := range req {
		svcName := req[index].GetService().GetValue()
		svcNamespace := req[index].GetNamespace().GetValue()
		svc := svr.Cache().Service().GetServiceByName(svcName, svcNamespace)
		if svc != nil {
			svcSet.Store(svc.ID, svc)
		}
	}

	ret := svr.convertToDiscoverResourceEntryMaps(names, svcSet)
	if authLog.DebugEnabled() {
		authLog.Debug("[Auth][Server] collect route-rule access res", zap.Any("res", ret))
	}
	return ret
}

// queryRateLimitConfigResource 根据所给的 RateLimit 信息，收集对应的 ResourceEntry 列表
func (svr *ServerAuthAbility) queryRateLimitConfigResource(
	req []*apitraffic.Rule) map[apisecurity.ResourceType][]model.ResourceEntry {
	if len(req) == 0 {
		return make(map[apisecurity.ResourceType][]model.ResourceEntry)
	}

	names := utils.NewSet[string]()
	svcSet := utils.NewMap[string, *model.Service]()

	for index := range req {
		svcName := req[index].GetService().GetValue()
		svcNamespace := req[index].GetNamespace().GetValue()
		svc := svr.Cache().Service().GetServiceByName(svcName, svcNamespace)
		if svc != nil {
			svcSet.Store(svc.ID, svc)
		}
	}

	ret := svr.convertToDiscoverResourceEntryMaps(names, svcSet)
	if authLog.DebugEnabled() {
		authLog.Debug("[Auth][Server] collect rate-limit access res", zap.Any("res", ret))
	}
	return ret
}

// convertToDiscoverResourceEntryMaps 通用方法，进行转换为期望的、服务相关的 ResourceEntry
func (svr *ServerAuthAbility) convertToDiscoverResourceEntryMaps(nsSet *utils.Set[string],
	svcSet *utils.Map[string, *model.Service]) map[apisecurity.ResourceType][]model.ResourceEntry {
	var (
		param = nsSet.ToSlice()
		nsArr = svr.Cache().Namespace().GetNamespacesByName(param)
		nsRet = make([]model.ResourceEntry, 0, len(nsArr))
	)
	for index := range nsArr {
		ns := nsArr[index]
		nsRet = append(nsRet, model.ResourceEntry{
			ID:    ns.Name,
			Owner: ns.Owner,
		})
	}

	svcRet := make([]model.ResourceEntry, 0, svcSet.Len())
	svcSet.Range(func(key string, svc *model.Service) {
		svcRet = append(svcRet, model.ResourceEntry{
			ID:    svc.ID,
			Owner: svc.Owner,
		})
	})

	return map[apisecurity.ResourceType][]model.ResourceEntry{
		apisecurity.ResourceType_Namespaces: nsRet,
		apisecurity.ResourceType_Services:   svcRet,
	}
}

func convertToErrCode(err error) apimodel.Code {
	if errors.Is(err, model.ErrorTokenNotExist) {
		return apimodel.Code_TokenNotExisted
	}
	if errors.Is(err, model.ErrorTokenDisabled) {
		return apimodel.Code_TokenDisabled
	}
	return apimodel.Code_NotAllowedAccess
}
