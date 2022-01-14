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

	"github.com/polarismesh/polaris-server/auth"
	"github.com/polarismesh/polaris-server/cache"
	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/common/utils"

	api "github.com/polarismesh/polaris-server/common/api/v1"
)

// serverAuthAbility 带有鉴权能力的 discoverServer
//  该层会对请求参数做一些调整，根据具体的请求发起人，设置为数据对应的 owner，不可为为别人进行创建资源
type serverAuthAbility struct {
	targetServer *Server
	authMgn      auth.AuthManager
}

func newServerAuthAbility(targetServer *Server, authMgn auth.AuthManager) DiscoverServer {
	proxy := &serverAuthAbility{
		targetServer: targetServer,
		authMgn:      authMgn,
	}

	targetServer.SetResourceHook([]ResourceHook{proxy})

	return proxy
}

// Get cache management
func (svr *serverAuthAbility) Cache() *cache.NamingCache {
	return svr.targetServer.Cache()
}

func (svr *serverAuthAbility) GetServiceInstanceRevision(serviceID string, instances []*model.Instance) (string, error) {
	return svr.targetServer.GetServiceInstanceRevision(serviceID, instances)
}

// collectNamespaceAuthContext 对于命名空间的处理，收集所有的与鉴权的相关信息
//  @receiver svr serverAuthAbility
//  @param ctx 请求上下文 ctx
//  @param req 实际请求对象
//  @param resourceOp 该接口的数据操作类型
//  @return *model.AcquireContext 返回鉴权上下文
func (svr *serverAuthAbility) collectNamespaceAuthContext(ctx context.Context, req []*api.Namespace, resourceOp model.ResourceOperation) *model.AcquireContext {
	authToken := utils.ParseAuthToken(ctx)

	authCtx := model.NewAcquireContext(
		model.WithRequestContext(ctx),
		model.WithOperation(resourceOp),
		model.WithToken(authToken),
		model.WithModule(model.CoreModule),
		model.WithAccessResources(svr.queryNamespaceResource(req)),
	)

	return authCtx
}

// collectServiceAuthContext 对于服务的处理，收集所有的与鉴权的相关信息
//  @receiver svr serverAuthAbility
//  @param ctx 请求上下文 ctx
//  @param req 实际请求对象
//  @param resourceOp 该接口的数据操作类型
//  @return *model.AcquireContext 返回鉴权上下文
func (svr *serverAuthAbility) collectServiceAuthContext(ctx context.Context, req []*api.Service, resourceOp model.ResourceOperation) *model.AcquireContext {
	authToken := utils.ParseAuthToken(ctx)

	namespaceNames := make([]string, 0)
	serviceNames := make([]string, 0, len(req))
	for i := range req {
		service := req[i]
		namespaceNames = append(namespaceNames, service.GetNamespace().GetValue())
		serviceNames = append(serviceNames, service.GetName().GetValue())
	}

	authCtx := model.NewAcquireContext(
		model.WithRequestContext(ctx),
		model.WithOperation(resourceOp),
		model.WithToken(authToken),
		model.WithModule(model.CoreModule),
		model.WithAccessResources(svr.queryServiceResource(req)),
	)

	return authCtx
}

// collectServiceAliasAuthContext 对于服务别名的处理，收集所有的与鉴权的相关信息
//  @receiver svr serverAuthAbility
//  @param ctx 请求上下文 ctx
//  @param req 实际请求对象
//  @param resourceOp 该接口的数据操作类型
//  @return *model.AcquireContext 返回鉴权上下文
func (svr *serverAuthAbility) collectServiceAliasAuthContext(ctx context.Context, req []*api.ServiceAlias, resourceOp model.ResourceOperation) *model.AcquireContext {
	authToken := utils.ParseAuthToken(ctx)

	namespaceNames := make([]string, 0)
	serviceNames := make([]string, 0, len(req))
	for i := range req {
		service := req[i]
		namespaceNames = append(namespaceNames, service.GetNamespace().GetValue())
		serviceNames = append(serviceNames, service.GetAlias().GetValue())
		serviceNames = append(serviceNames, service.GetService().GetValue())
	}

	authCtx := model.NewAcquireContext(
		model.WithRequestContext(ctx),
		model.WithOperation(resourceOp),
		model.WithToken(authToken),
		model.WithModule(model.CoreModule),
		model.WithAccessResources(svr.queryServiceAliasResource(req)),
	)

	return authCtx
}

// collectInstanceAuthContext 对于服务实例的处理，收集所有的与鉴权的相关信息
//  @receiver svr serverAuthAbility
//  @param ctx 请求上下文 ctx
//  @param req 实际请求对象
//  @param resourceOp 该接口的数据操作类型
//  @return *model.AcquireContext 返回鉴权上下文
func (svr *serverAuthAbility) collectInstanceAuthContext(ctx context.Context, req []*api.Instance, resourceOp model.ResourceOperation) *model.AcquireContext {
	authToken := utils.ParseAuthToken(ctx)

	namespaceNames := make([]string, 0)
	serviceNames := make([]string, 0, len(req))
	for i := range req {
		ns := req[i]
		namespaceNames = append(namespaceNames, ns.GetNamespace().GetValue())
		serviceNames = append(serviceNames, ns.GetService().GetValue())
	}

	authCtx := model.NewAcquireContext(
		model.WithRequestContext(ctx),
		model.WithOperation(resourceOp),
		model.WithToken(authToken),
		model.WithModule(model.DiscoverModule),
		model.WithAccessResources(svr.queryInstanceResource(req)),
	)

	return authCtx
}

// collectCircuitBreakerAuthContext 对于服务熔断的处理，收集所有的与鉴权的相关信息
//  @receiver svr serverAuthAbility
//  @param ctx 请求上下文 ctx
//  @param req 实际请求对象
//  @param resourceOp 该接口的数据操作类型
//  @return *model.AcquireContext 返回鉴权上下文
func (svr *serverAuthAbility) collectCircuitBreakerAuthContext(ctx context.Context, req []*api.CircuitBreaker, resourceOp model.ResourceOperation) *model.AcquireContext {
	authToken := utils.ParseAuthToken(ctx)

	namespaceNames := make([]string, 0)
	serviceNames := make([]string, 0, len(req))
	for i := range req {
		ns := req[i]
		namespaceNames = append(namespaceNames, ns.GetNamespace().GetValue())
		serviceNames = append(serviceNames, ns.GetService().GetValue())
	}

	authCtx := model.NewAcquireContext(
		model.WithRequestContext(ctx),
		model.WithOperation(resourceOp),
		model.WithToken(authToken),
		model.WithModule(model.DiscoverModule),
		model.WithAccessResources(svr.queryCircuitBreakerResource(req)),
	)

	return authCtx
}

// collectCircuitBreakerReleaseAuthContext
//  @receiver svr
//  @param ctx
//  @param req
//  @param resourceOp
//  @return *model.AcquireContext
func (svr *serverAuthAbility) collectCircuitBreakerReleaseAuthContext(ctx context.Context, req []*api.ConfigRelease, resourceOp model.ResourceOperation) *model.AcquireContext {
	authToken := utils.ParseAuthToken(ctx)

	namespaceNames := make([]string, 0)
	serviceNames := make([]string, 0, len(req))
	for i := range req {
		cfg := req[i]
		namespaceNames = append(namespaceNames, cfg.GetCircuitBreaker().GetNamespace().GetValue())
		serviceNames = append(serviceNames, cfg.GetCircuitBreaker().GetService().GetValue())
	}

	authCtx := model.NewAcquireContext(
		model.WithRequestContext(ctx),
		model.WithOperation(resourceOp),
		model.WithToken(authToken),
		model.WithModule(model.DiscoverModule),
		model.WithAccessResources(svr.queryCircuitBreakerReleaseResource(req)),
	)

	return authCtx
}

// collectRouteRuleAuthContext 对于服务路由规则的处理，收集所有的与鉴权的相关信息
//  @receiver svr serverAuthAbility
//  @param ctx 请求上下文 ctx
//  @param req 实际请求对象
//  @param resourceOp 该接口的数据操作类型
//  @return *model.AcquireContext 返回鉴权上下文
func (svr *serverAuthAbility) collectRouteRuleAuthContext(ctx context.Context, req []*api.Routing, resourceOp model.ResourceOperation) *model.AcquireContext {
	authToken := utils.ParseAuthToken(ctx)

	namespaceNames := make([]string, 0)
	serviceNames := make([]string, 0, len(req))
	for i := range req {
		ns := req[i]
		namespaceNames = append(namespaceNames, ns.GetNamespace().GetValue())
		serviceNames = append(serviceNames, ns.GetService().GetValue())
	}

	authCtx := model.NewAcquireContext(
		model.WithRequestContext(ctx),
		model.WithOperation(resourceOp),
		model.WithToken(authToken),
		model.WithModule(model.DiscoverModule),
		model.WithAccessResources(svr.queryRouteRuleResource(req)),
	)

	return authCtx
}

// collectRateLimitAuthContext 对于服务限流规则的处理，收集所有的与鉴权的相关信息
//  @receiver svr serverAuthAbility
//  @param ctx 请求上下文 ctx
//  @param req 实际请求对象
//  @param resourceOp 该接口的数据操作类型
//  @return *model.AcquireContext 返回鉴权上下文
func (svr *serverAuthAbility) collectRateLimitAuthContext(ctx context.Context, req []*api.Rule, resourceOp model.ResourceOperation) *model.AcquireContext {
	authToken := utils.ParseAuthToken(ctx)

	namespaceNames := make([]string, 0)
	serviceNames := make([]string, 0, len(req))
	for i := range req {
		ns := req[i]
		namespaceNames = append(namespaceNames, ns.GetNamespace().GetValue())
		serviceNames = append(serviceNames, ns.GetService().GetValue())
	}

	authCtx := model.NewAcquireContext(
		model.WithRequestContext(ctx),
		model.WithOperation(resourceOp),
		model.WithToken(authToken),
		model.WithModule(model.DiscoverModule),
		model.WithAccessResources(svr.queryRateLimitConfigResource(req)),
	)

	return authCtx
}

// queryNamespaceResource 根据所给的 namespace 信息，收集对应的 ResourceEntry 列表
//  @receiver svr
//  @param req
//  @return map
func (svr *serverAuthAbility) queryNamespaceResource(req []*api.Namespace) map[api.ResourceType][]model.ResourceEntry {

	names := utils.NewStringSet()
	for index := range req {
		names.Add(req[index].Name.GetValue())
	}
	param := names.ToSlice()
	nsArr := svr.Cache().Namespace().GetNamespacesByName(param)

	ret := make([]model.ResourceEntry, 0, len(nsArr))

	for index := range nsArr {
		ns := nsArr[index]
		ret = append(ret, model.ResourceEntry{
			ID:    ns.Name,
			Owner: ns.Owner,
		})
	}

	return map[api.ResourceType][]model.ResourceEntry{
		api.ResourceType_Namespaces: ret,
	}
}

// queryServiceResource  根据所给的 service 信息，收集对应的 ResourceEntry 列表
//  @receiver svr
//  @param req
//  @return map
func (svr *serverAuthAbility) queryServiceResource(req []*api.Service) map[api.ResourceType][]model.ResourceEntry {

	names := utils.NewStringSet()
	svcSet := utils.NewServiceSet()

	for index := range req {
		names.Add(req[index].Namespace.GetValue())
		svc := svr.Cache().Service().GetServiceByName(req[index].Name.GetValue(), req[index].Namespace.GetValue())
		if svc != nil {
			svcSet.Add(svc)
		}
	}

	return svr.convertToDiscoverResourceEntryMaps(names, svcSet)
}

// queryServiceAliasResource  根据所给的 servicealias 信息，收集对应的 ResourceEntry 列表
//  @receiver svr
//  @param req
//  @return map
func (svr *serverAuthAbility) queryServiceAliasResource(req []*api.ServiceAlias) map[api.ResourceType][]model.ResourceEntry {
	names := utils.NewStringSet()
	svcSet := utils.NewServiceSet()

	for index := range req {
		names.Add(req[index].Namespace.GetValue())
		svc := svr.Cache().Service().GetServiceByName(req[index].Service.GetValue(), req[index].Namespace.GetValue())
		if svc != nil {
			svcSet.Add(svc)
		}
	}

	return svr.convertToDiscoverResourceEntryMaps(names, svcSet)
}

// queryInstanceResource 根据所给的 instances 信息，收集对应的 ResourceEntry 列表
//  @receiver svr
//  @param req
//  @return map
func (svr *serverAuthAbility) queryInstanceResource(req []*api.Instance) map[api.ResourceType][]model.ResourceEntry {

	names := utils.NewStringSet()
	svcSet := utils.NewServiceSet()

	for index := range req {
		names.Add(req[index].Namespace.GetValue())
		svc := svr.Cache().Service().GetServiceByName(req[index].Service.GetValue(), req[index].Namespace.GetValue())
		if svc != nil {
			svcSet.Add(svc)
		}
	}

	return svr.convertToDiscoverResourceEntryMaps(names, svcSet)

}

// queryCircuitBreakerResource 根据所给的 CircuitBreaker 信息，收集对应的 ResourceEntry 列表
//  @receiver svr
//  @param req
//  @return map
func (svr *serverAuthAbility) queryCircuitBreakerResource(req []*api.CircuitBreaker) map[api.ResourceType][]model.ResourceEntry {
	names := utils.NewStringSet()
	svcSet := utils.NewServiceSet()

	for index := range req {
		names.Add(req[index].Namespace.GetValue())
		svc := svr.Cache().Service().GetServiceByName(req[index].Service.GetValue(), req[index].Namespace.GetValue())
		if svc != nil {
			svcSet.Add(svc)
		}
	}
	return svr.convertToDiscoverResourceEntryMaps(names, svcSet)
}

// queryCircuitBreakerReleaseResource 根据所给的 CircuitBreakerRelease 信息，收集对应的 ResourceEntry 列表
//  @receiver svr
//  @param req
//  @return map
func (svr *serverAuthAbility) queryCircuitBreakerReleaseResource(req []*api.ConfigRelease) map[api.ResourceType][]model.ResourceEntry {
	names := utils.NewStringSet()
	svcSet := utils.NewServiceSet()

	for index := range req {
		names.Add(req[index].Service.Namespace.GetValue())
		svc := svr.Cache().Service().GetServiceByName(req[index].Service.Name.GetValue(), req[index].Service.Namespace.GetValue())
		if svc != nil {
			svcSet.Add(svc)
		}
	}

	return svr.convertToDiscoverResourceEntryMaps(names, svcSet)
}

// queryRouteRuleResource 根据所给的 RouteRule 信息，收集对应的 ResourceEntry 列表
//  @receiver svr
//  @param req
//  @return map
func (svr *serverAuthAbility) queryRouteRuleResource(req []*api.Routing) map[api.ResourceType][]model.ResourceEntry {
	names := utils.NewStringSet()
	svcSet := utils.NewServiceSet()

	for index := range req {
		names.Add(req[index].Namespace.GetValue())
		svc := svr.Cache().Service().GetServiceByName(req[index].Service.GetValue(), req[index].Namespace.GetValue())
		if svc != nil {
			svcSet.Add(svc)
		}
	}

	return svr.convertToDiscoverResourceEntryMaps(names, svcSet)
}

// queryRateLimitConfigResource 根据所给的 RateLimit 信息，收集对应的 ResourceEntry 列表
//  @receiver svr
//  @param req
//  @return map
func (svr *serverAuthAbility) queryRateLimitConfigResource(req []*api.Rule) map[api.ResourceType][]model.ResourceEntry {
	names := utils.NewStringSet()
	svcSet := utils.NewServiceSet()

	for index := range req {
		names.Add(req[index].Namespace.GetValue())
		svc := svr.Cache().Service().GetServiceByName(req[index].Service.GetValue(), req[index].Namespace.GetValue())
		if svc != nil {
			svcSet.Add(svc)
		}
	}

	return svr.convertToDiscoverResourceEntryMaps(names, svcSet)
}

// convertToDiscoverResourceEntryMaps 通用方法，进行转换为期望的、服务相关的 ResourceEntry
//  @receiver svr
//  @param nsSet
//  @param svcSet
//  @return map
func (svr *serverAuthAbility) convertToDiscoverResourceEntryMaps(nsSet utils.StringSet, svcSet *utils.ServiceSet) map[api.ResourceType][]model.ResourceEntry {
	param := nsSet.ToSlice()
	nsArr := svr.Cache().Namespace().GetNamespacesByName(param)

	ret := make([]model.ResourceEntry, 0, len(nsArr))
	for index := range nsArr {
		ns := nsArr[index]
		ret = append(ret, model.ResourceEntry{
			ID:    ns.Name,
			Owner: ns.Owner,
		})
	}

	svcParam := svcSet.ToSlice()
	svcRet := make([]model.ResourceEntry, 0, len(svcParam))
	for index := range svcRet {
		svc := nsArr[index]
		ret = append(ret, model.ResourceEntry{
			ID:    svc.Name,
			Owner: svc.Owner,
		})
	}

	return map[api.ResourceType][]model.ResourceEntry{
		api.ResourceType_Namespaces: ret,
		api.ResourceType_Services:   svcRet,
	}
}
