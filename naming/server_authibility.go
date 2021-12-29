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

package naming

import (
	"context"

	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/common/utils"
	"github.com/polarismesh/polaris-server/core/auth"
	"github.com/polarismesh/polaris-server/naming/cache"

	api "github.com/polarismesh/polaris-server/common/api/v1"
)

// serverAuthAbility 带有鉴权能力的 discoverServer
type serverAuthAbility struct {
	targetServer *Server
	authMgn      auth.AuthManager
}

// Get cache management
func (svr *serverAuthAbility) Cache() *cache.NamingCache {
	return svr.targetServer.Cache()
}

// collectNamespaceAuthContext 对于命名空间的处理，收集所有的与鉴权的相关信息
//  @receiver svr serverAuthAbility
//  @param ctx 请求上下文 ctx
//  @param req 实际请求对象
//  @param resourceOp 该接口的数据操作类型
//  @return *model.AcquireContext 返回鉴权上下文
func (svr *serverAuthAbility) collectNamespaceAuthContext(ctx context.Context, req []*api.Namespace, resourceOp model.ResourceOperation) *model.AcquireContext {
	authToken := utils.ParseAuthToken(ctx)

	nsSlice := make([]string, 0, len(req))
	for i := range req {
		ns := req[i]
		nsSlice = append(nsSlice, ns.GetName().GetValue())
	}

	authCtx := &model.AcquireContext{
		RequestContext: ctx,
		Token:          authToken,
		Module:         model.CoreModule,
		Operation:      resourceOp,
		Resources: map[api.ResourceType][]string{
			api.ResourceType_Namespaces: nsSlice,
		},
		Attachment: make(map[string]interface{}),
	}

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

	authCtx := &model.AcquireContext{
		RequestContext: ctx,
		Token:          authToken,
		Module:         model.CoreModule,
		Operation:      resourceOp,
		Resources: map[api.ResourceType][]string{
			api.ResourceType_Namespaces: utils.StringSliceDeDuplication(namespaceNames),
			api.ResourceType_Services:   serviceNames,
		},
		Attachment: make(map[string]interface{}),
	}

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

	authCtx := &model.AcquireContext{
		RequestContext: ctx,
		Token:          authToken,
		Module:         model.CoreModule,
		Operation:      resourceOp,
		Resources: map[api.ResourceType][]string{
			api.ResourceType_Namespaces: utils.StringSliceDeDuplication(namespaceNames),
			api.ResourceType_Services:   serviceNames,
		},
		Attachment: make(map[string]interface{}),
	}

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

	authCtx := &model.AcquireContext{
		RequestContext: ctx,
		Token:          authToken,
		Module:         model.CoreModule,
		Operation:      resourceOp,
		Resources: map[api.ResourceType][]string{
			api.ResourceType_Namespaces: utils.StringSliceDeDuplication(namespaceNames),
			api.ResourceType_Services:   utils.StringSliceDeDuplication(serviceNames),
		},
		Attachment: make(map[string]interface{}),
	}

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

	authCtx := &model.AcquireContext{
		RequestContext: ctx,
		Token:          authToken,
		Module:         model.CoreModule,
		Operation:      resourceOp,
		Resources: map[api.ResourceType][]string{
			api.ResourceType_Namespaces: utils.StringSliceDeDuplication(namespaceNames),
			api.ResourceType_Services:   utils.StringSliceDeDuplication(serviceNames),
		},
		Attachment: make(map[string]interface{}),
	}

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

	authCtx := &model.AcquireContext{
		RequestContext: ctx,
		Token:          authToken,
		Module:         model.CoreModule,
		Operation:      resourceOp,
		Resources: map[api.ResourceType][]string{
			api.ResourceType_Namespaces: utils.StringSliceDeDuplication(namespaceNames),
			api.ResourceType_Services:   utils.StringSliceDeDuplication(serviceNames),
		},
		Attachment: make(map[string]interface{}),
	}

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

	authCtx := &model.AcquireContext{
		RequestContext: ctx,
		Token:          authToken,
		Module:         model.CoreModule,
		Operation:      resourceOp,
		Resources: map[api.ResourceType][]string{
			api.ResourceType_Namespaces: utils.StringSliceDeDuplication(namespaceNames),
			api.ResourceType_Services:   utils.StringSliceDeDuplication(serviceNames),
		},
		Attachment: make(map[string]interface{}),
	}

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

	authCtx := &model.AcquireContext{
		RequestContext: ctx,
		Token:          authToken,
		Module:         model.CoreModule,
		Operation:      resourceOp,
		Resources: map[api.ResourceType][]string{
			api.ResourceType_Namespaces: utils.StringSliceDeDuplication(namespaceNames),
			api.ResourceType_Services:   utils.StringSliceDeDuplication(serviceNames),
		},
		Attachment: make(map[string]interface{}),
	}

	return authCtx
}
