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

package service

import (
	"context"
	"errors"

	"go.uber.org/zap"

	"github.com/polarismesh/polaris-server/auth"
	"github.com/polarismesh/polaris-server/cache"
	api "github.com/polarismesh/polaris-server/common/api/v1"
	commonlog "github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/common/utils"
)

// serverAuthAbility 带有鉴权能力的 discoverServer
//  该层会对请求参数做一些调整，根据具体的请求发起人，设置为数据对应的 owner，不可为为别人进行创建资源
type serverAuthAbility struct {
	targetServer *Server
	authSvr      auth.AuthServer
	authMgn      auth.AuthChecker
}

func newServerAuthAbility(targetServer *Server, authSvr auth.AuthServer) DiscoverServer {
	proxy := &serverAuthAbility{
		targetServer: targetServer,
		authSvr:      authSvr,
		authMgn:      authSvr.GetAuthChecker(),
	}

	targetServer.SetResourceHooks(proxy)

	return proxy
}

// Cache Get cache management
func (svr *serverAuthAbility) Cache() *cache.CacheManager {
	return svr.targetServer.Cache()
}

// GetServiceInstanceRevision 获取服务实例的版本号
func (svr *serverAuthAbility) GetServiceInstanceRevision(serviceID string,
	instances []*model.Instance) (string, error) {
	return svr.targetServer.GetServiceInstanceRevision(serviceID, instances)
}

// collectServiceAuthContext 对于服务的处理，收集所有的与鉴权的相关信息
//  @receiver svr serverAuthAbility
//  @param ctx 请求上下文 ctx
//  @param req 实际请求对象
//  @param resourceOp 该接口的数据操作类型
//  @return *model.AcquireContext 返回鉴权上下文
func (svr *serverAuthAbility) collectServiceAuthContext(ctx context.Context, req []*api.Service,
	resourceOp model.ResourceOperation, methodName string) *model.AcquireContext {

	return model.NewAcquireContext(
		model.WithRequestContext(ctx),
		model.WithOperation(resourceOp),
		model.WithToken(utils.ParseAuthToken(ctx)),
		model.WithModule(model.DiscoverModule),
		model.WithMethod(methodName),
		model.WithAccessResources(svr.queryServiceResource(req)),
	)
}

// collectServiceAliasAuthContext 对于服务别名的处理，收集所有的与鉴权的相关信息
//  @receiver svr serverAuthAbility
//  @param ctx 请求上下文 ctx
//  @param req 实际请求对象
//  @param resourceOp 该接口的数据操作类型
//  @return *model.AcquireContext 返回鉴权上下文
func (svr *serverAuthAbility) collectServiceAliasAuthContext(ctx context.Context, req []*api.ServiceAlias,
	resourceOp model.ResourceOperation, methodName string) *model.AcquireContext {

	return model.NewAcquireContext(
		model.WithRequestContext(ctx),
		model.WithOperation(resourceOp),
		model.WithToken(utils.ParseAuthToken(ctx)),
		model.WithModule(model.DiscoverModule),
		model.WithMethod(methodName),
		model.WithAccessResources(svr.queryServiceAliasResource(req)),
	)
}

// collectInstanceAuthContext 对于服务实例的处理，收集所有的与鉴权的相关信息
//  @receiver svr serverAuthAbility
//  @param ctx 请求上下文 ctx
//  @param req 实际请求对象
//  @param resourceOp 该接口的数据操作类型
//  @return *model.AcquireContext 返回鉴权上下文
func (svr *serverAuthAbility) collectInstanceAuthContext(ctx context.Context, req []*api.Instance,
	resourceOp model.ResourceOperation, methodName string) *model.AcquireContext {

	return model.NewAcquireContext(
		model.WithRequestContext(ctx),
		model.WithOperation(resourceOp),
		model.WithToken(utils.ParseAuthToken(ctx)),
		model.WithModule(model.DiscoverModule),
		model.WithMethod(methodName),
		model.WithAccessResources(svr.queryInstanceResource(req)),
	)
}

// collectClientInstanceAuthContext 对于服务实例的处理，收集所有的与鉴权的相关信息
func (svr *serverAuthAbility) collectClientInstanceAuthContext(ctx context.Context, req []*api.Instance,
	resourceOp model.ResourceOperation, methodName string) *model.AcquireContext {

	return model.NewAcquireContext(
		model.WithRequestContext(ctx),
		model.WithOperation(resourceOp),
		model.WithToken(utils.ParseAuthToken(ctx)),
		model.WithModule(model.DiscoverModule),
		model.WithMethod(methodName),
		model.WithFromClient(),
		model.WithAccessResources(svr.queryInstanceResource(req)),
	)
}

// collectCircuitBreakerAuthContext 对于服务熔断的处理，收集所有的与鉴权的相关信息
//  @receiver svr serverAuthAbility
//  @param ctx 请求上下文 ctx
//  @param req 实际请求对象
//  @param resourceOp 该接口的数据操作类型
//  @return *model.AcquireContext 返回鉴权上下文
func (svr *serverAuthAbility) collectCircuitBreakerAuthContext(ctx context.Context, req []*api.CircuitBreaker,
	resourceOp model.ResourceOperation, methodName string) *model.AcquireContext {

	return model.NewAcquireContext(
		model.WithRequestContext(ctx),
		model.WithOperation(resourceOp),
		model.WithToken(utils.ParseAuthToken(ctx)),
		model.WithModule(model.DiscoverModule),
		model.WithMethod(methodName),
		model.WithAccessResources(svr.queryCircuitBreakerResource(req)),
	)
}

// collectCircuitBreakerReleaseAuthContext
//  @receiver svr
//  @param ctx
//  @param req
//  @param resourceOp
//  @return *model.AcquireContext
func (svr *serverAuthAbility) collectCircuitBreakerReleaseAuthContext(ctx context.Context, req []*api.ConfigRelease,
	resourceOp model.ResourceOperation, methodName string) *model.AcquireContext {

	return model.NewAcquireContext(
		model.WithRequestContext(ctx),
		model.WithOperation(resourceOp),
		model.WithToken(utils.ParseAuthToken(ctx)),
		model.WithModule(model.DiscoverModule),
		model.WithMethod(methodName),
		model.WithAccessResources(svr.queryCircuitBreakerReleaseResource(req)),
	)
}

// collectRouteRuleAuthContext 对于服务路由规则的处理，收集所有的与鉴权的相关信息
//  @receiver svr serverAuthAbility
//  @param ctx 请求上下文 ctx
//  @param req 实际请求对象
//  @param resourceOp 该接口的数据操作类型
//  @return *model.AcquireContext 返回鉴权上下文
func (svr *serverAuthAbility) collectRouteRuleAuthContext(ctx context.Context, req []*api.Routing,
	resourceOp model.ResourceOperation, methodName string) *model.AcquireContext {

	return model.NewAcquireContext(
		model.WithRequestContext(ctx),
		model.WithOperation(resourceOp),
		model.WithToken(utils.ParseAuthToken(ctx)),
		model.WithModule(model.DiscoverModule),
		model.WithMethod(methodName),
		model.WithAccessResources(svr.queryRouteRuleResource(req)),
	)
}

// collectRateLimitAuthContext 对于服务限流规则的处理，收集所有的与鉴权的相关信息
//  @receiver svr serverAuthAbility
//  @param ctx 请求上下文 ctx
//  @param req 实际请求对象
//  @param resourceOp 该接口的数据操作类型
//  @return *model.AcquireContext 返回鉴权上下文
func (svr *serverAuthAbility) collectRateLimitAuthContext(ctx context.Context, req []*api.Rule,
	resourceOp model.ResourceOperation, methodName string) *model.AcquireContext {

	return model.NewAcquireContext(
		model.WithRequestContext(ctx),
		model.WithOperation(resourceOp),
		model.WithToken(utils.ParseAuthToken(ctx)),
		model.WithModule(model.DiscoverModule),
		model.WithMethod(methodName),
		model.WithAccessResources(svr.queryRateLimitConfigResource(req)),
	)
}

// queryServiceResource  根据所给的 service 信息，收集对应的 ResourceEntry 列表
func (svr *serverAuthAbility) queryServiceResource(
	req []*api.Service) map[api.ResourceType][]model.ResourceEntry {
	if len(req) == 0 {
		return make(map[api.ResourceType][]model.ResourceEntry)
	}

	names := utils.NewStringSet()
	svcSet := utils.NewServiceSet()

	for index := range req {
		names.Add(req[index].Namespace.GetValue())
		svc := svr.Cache().Service().GetServiceByName(req[index].Name.GetValue(), req[index].Namespace.GetValue())
		if svc != nil {
			svcSet.Add(svc)
		}
	}

	ret := svr.convertToDiscoverResourceEntryMaps(names, svcSet)
	commonlog.AuthScope().Debug("[Auth][Server] collect service access res", zap.Any("res", ret))
	return ret
}

// queryServiceAliasResource  根据所给的 servicealias 信息，收集对应的 ResourceEntry 列表
func (svr *serverAuthAbility) queryServiceAliasResource(
	req []*api.ServiceAlias) map[api.ResourceType][]model.ResourceEntry {
	if len(req) == 0 {
		return make(map[api.ResourceType][]model.ResourceEntry)
	}

	names := utils.NewStringSet()
	svcSet := utils.NewServiceSet()

	for index := range req {
		names.Add(req[index].Namespace.GetValue())
		alias := svr.Cache().Service().GetServiceByName(req[index].Alias.GetValue(),
			req[index].AliasNamespace.GetValue())
		if alias != nil {
			svc := svr.Cache().Service().GetServiceByID(alias.Reference)
			if svc != nil {
				svcSet.Add(svc)
			}
		}
	}

	ret := svr.convertToDiscoverResourceEntryMaps(names, svcSet)
	commonlog.AuthScope().Debug("[Auth][Server] collect service alias access res", zap.Any("res", ret))
	return ret
}

// queryInstanceResource 根据所给的 instances 信息，收集对应的 ResourceEntry 列表
// 由于实例是注册到服务下的，因此只需要判断，是否有对应服务的权限即可
func (svr *serverAuthAbility) queryInstanceResource(
	req []*api.Instance) map[api.ResourceType][]model.ResourceEntry {
	if len(req) == 0 {
		return make(map[api.ResourceType][]model.ResourceEntry)
	}

	names := utils.NewStringSet()
	svcSet := utils.NewServiceSet()

	for index := range req {
		item := req[index]
		if item.Namespace.GetValue() != "" && item.Service.GetValue() != "" {
			svc := svr.Cache().Service().GetServiceByName(req[index].Service.GetValue(),
				req[index].Namespace.GetValue())
			if svc != nil {
				svcSet.Add(svc)
			} else {
				names.Add(req[index].Namespace.GetValue())
			}
		} else {
			ins := svr.Cache().Instance().GetInstance(item.GetId().GetValue())
			if ins != nil {
				svc := svr.Cache().Service().GetServiceByID(ins.ServiceID)
				if svc != nil {
					svcSet.Add(svc)
				} else {
					names.Add(req[index].Namespace.GetValue())
				}
			}
		}
	}

	ret := svr.convertToDiscoverResourceEntryMaps(names, svcSet)
	commonlog.AuthScope().Debug("[Auth][Server] collect instance access res", zap.Any("res", ret))
	return ret
}

// queryCircuitBreakerResource 根据所给的 CircuitBreaker 信息，收集对应的 ResourceEntry 列表
func (svr *serverAuthAbility) queryCircuitBreakerResource(
	req []*api.CircuitBreaker) map[api.ResourceType][]model.ResourceEntry {
	if len(req) == 0 {
		return make(map[api.ResourceType][]model.ResourceEntry)
	}

	names := utils.NewStringSet()
	svcSet := utils.NewServiceSet()

	for index := range req {
		svc := svr.Cache().Service().GetServiceByName(req[index].Service.GetValue(),
			req[index].Namespace.GetValue())
		if svc != nil {
			svcSet.Add(svc)
		}
	}
	ret := svr.convertToDiscoverResourceEntryMaps(names, svcSet)
	commonlog.AuthScope().Debug("[Auth][Server] collect circuit-breaker access res", zap.Any("res", ret))
	return ret
}

// queryCircuitBreakerReleaseResource 根据所给的 CircuitBreakerRelease 信息，收集对应的 ResourceEntry 列表
func (svr *serverAuthAbility) queryCircuitBreakerReleaseResource(
	req []*api.ConfigRelease) map[api.ResourceType][]model.ResourceEntry {
	if len(req) == 0 {
		return make(map[api.ResourceType][]model.ResourceEntry)
	}

	names := utils.NewStringSet()
	svcSet := utils.NewServiceSet()

	for index := range req {
		svc := svr.Cache().Service().GetServiceByName(req[index].Service.Name.GetValue(),
			req[index].Service.Namespace.GetValue())
		if svc != nil {
			svcSet.Add(svc)
		}
	}

	ret := svr.convertToDiscoverResourceEntryMaps(names, svcSet)
	commonlog.AuthScope().Debug("[Auth][Server] collect circuit-breaker-release access res", zap.Any("res", ret))
	return ret
}

// queryRouteRuleResource 根据所给的 RouteRule 信息，收集对应的 ResourceEntry 列表
func (svr *serverAuthAbility) queryRouteRuleResource(
	req []*api.Routing) map[api.ResourceType][]model.ResourceEntry {
	if len(req) == 0 {
		return make(map[api.ResourceType][]model.ResourceEntry)
	}

	names := utils.NewStringSet()
	svcSet := utils.NewServiceSet()

	for index := range req {
		svc := svr.Cache().Service().GetServiceByName(req[index].Service.GetValue(),
			req[index].Namespace.GetValue())
		if svc != nil {
			svcSet.Add(svc)
		}
	}

	ret := svr.convertToDiscoverResourceEntryMaps(names, svcSet)
	commonlog.AuthScope().Debug("[Auth][Server] collect route-rule access res", zap.Any("res", ret))
	return ret
}

// queryRateLimitConfigResource 根据所给的 RateLimit 信息，收集对应的 ResourceEntry 列表
func (svr *serverAuthAbility) queryRateLimitConfigResource(
	req []*api.Rule) map[api.ResourceType][]model.ResourceEntry {
	if len(req) == 0 {
		return make(map[api.ResourceType][]model.ResourceEntry)
	}

	names := utils.NewStringSet()
	svcSet := utils.NewServiceSet()

	for index := range req {
		svc := svr.Cache().Service().GetServiceByName(req[index].Service.GetValue(),
			req[index].Namespace.GetValue())
		if svc != nil {
			svcSet.Add(svc)
		}
	}

	ret := svr.convertToDiscoverResourceEntryMaps(names, svcSet)
	commonlog.AuthScope().Debug("[Auth][Server] collect rate-limit access res", zap.Any("res", ret))
	return ret
}

// convertToDiscoverResourceEntryMaps 通用方法，进行转换为期望的、服务相关的 ResourceEntry
func (svr *serverAuthAbility) convertToDiscoverResourceEntryMaps(nsSet utils.StringSet,
	svcSet *utils.ServiceSet) map[api.ResourceType][]model.ResourceEntry {
	param := nsSet.ToSlice()
	nsArr := svr.Cache().Namespace().GetNamespacesByName(param)

	nsRet := make([]model.ResourceEntry, 0, len(nsArr))
	for index := range nsArr {
		ns := nsArr[index]
		nsRet = append(nsRet, model.ResourceEntry{
			ID:    ns.Name,
			Owner: ns.Owner,
		})
	}

	svcParam := svcSet.ToSlice()
	svcRet := make([]model.ResourceEntry, 0, len(svcParam))
	for index := range svcParam {
		svc := svcParam[index]
		svcRet = append(svcRet, model.ResourceEntry{
			ID:    svc.ID,
			Owner: svc.Owner,
		})
	}

	return map[api.ResourceType][]model.ResourceEntry{
		api.ResourceType_Namespaces: nsRet,
		api.ResourceType_Services:   svcRet,
	}
}

func convertToErrCode(err error) uint32 {
	if errors.Is(err, model.ErrorTokenNotExist) {
		return api.TokenNotExisted
	}
	if errors.Is(err, model.ErrorTokenDisabled) {
		return api.TokenDisabled
	}
	return api.NotAllowedAccess
}
