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

	"github.com/polarismesh/specification/source/go/api/v1/security"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"
	"google.golang.org/protobuf/types/known/wrapperspb"

	cachetypes "github.com/polarismesh/polaris/cache/api"
	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/model"
	authcommon "github.com/polarismesh/polaris/common/model/auth"
	"github.com/polarismesh/polaris/common/utils"
)

// RegisterInstance create one instance
func (svr *Server) RegisterInstance(ctx context.Context, req *apiservice.Instance) *apiservice.Response {
	authCtx := svr.collectClientInstanceAuthContext(
		ctx, []*apiservice.Instance{req}, authcommon.Create, authcommon.RegisterInstance)

	_, err := svr.policySvr.GetAuthChecker().CheckClientPermission(authCtx)
	if err != nil {
		resp := api.NewResponseWithMsg(authcommon.ConvertToErrCode(err), err.Error())
		return resp
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return svr.nextSvr.RegisterInstance(ctx, req)
}

// DeregisterInstance delete onr instance
func (svr *Server) DeregisterInstance(ctx context.Context, req *apiservice.Instance) *apiservice.Response {
	authCtx := svr.collectClientInstanceAuthContext(
		ctx, []*apiservice.Instance{req}, authcommon.Create, authcommon.DeregisterInstance)

	if _, err := svr.policySvr.GetAuthChecker().CheckClientPermission(authCtx); err != nil {
		return api.NewResponse(authcommon.ConvertToErrCode(err))
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return svr.nextSvr.DeregisterInstance(ctx, req)
}

// ReportClient is the interface for reporting client authability
func (svr *Server) ReportClient(ctx context.Context, req *apiservice.Client) *apiservice.Response {
	return svr.nextSvr.ReportClient(ctx, req)
}

// ReportServiceContract .
func (svr *Server) ReportServiceContract(ctx context.Context, req *apiservice.ServiceContract) *apiservice.Response {
	authCtx := svr.collectServiceAuthContext(
		ctx, []*apiservice.Service{{
			Name:      wrapperspb.String(req.GetService()),
			Namespace: wrapperspb.String(req.GetNamespace()),
		}}, authcommon.Create, authcommon.ReportServiceContract)

	if _, err := svr.policySvr.GetAuthChecker().CheckClientPermission(authCtx); err != nil {
		return api.NewResponse(authcommon.ConvertToErrCode(err))
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)
	return svr.nextSvr.ReportServiceContract(ctx, req)
}

// GetPrometheusTargets Used for client acquisition service information
func (svr *Server) GetPrometheusTargets(ctx context.Context,
	query map[string]string) *model.PrometheusDiscoveryResponse {

	return svr.nextSvr.GetPrometheusTargets(ctx, query)
}

// GetServiceWithCache is the interface for getting service with cache
func (svr *Server) GetServiceWithCache(
	ctx context.Context, req *apiservice.Service) *apiservice.DiscoverResponse {

	authCtx := svr.collectServiceAuthContext(
		ctx, []*apiservice.Service{req}, authcommon.Read, authcommon.DiscoverServices)
	if _, err := svr.policySvr.GetAuthChecker().CheckClientPermission(authCtx); err != nil {
		return api.NewDiscoverResponse(authcommon.ConvertToErrCode(err))
	}
	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	ctx = cachetypes.AppendServicePredicate(ctx, func(ctx context.Context, cbr *model.Service) bool {
		return svr.policySvr.GetAuthChecker().ResourcePredicate(authCtx, &authcommon.ResourceEntry{
			Type:     security.ResourceType_Services,
			ID:       cbr.ID,
			Metadata: cbr.Meta,
		})
	})
	authCtx.SetRequestContext(ctx)

	return svr.nextSvr.GetServiceWithCache(ctx, req)
}

// ServiceInstancesCache is the interface for getting service instances cache
func (svr *Server) ServiceInstancesCache(
	ctx context.Context, filter *apiservice.DiscoverFilter, req *apiservice.Service) *apiservice.DiscoverResponse {

	authCtx := svr.collectServiceAuthContext(
		ctx, []*apiservice.Service{req}, authcommon.Read, authcommon.DiscoverInstances)
	if _, err := svr.policySvr.GetAuthChecker().CheckClientPermission(authCtx); err != nil {
		return api.NewDiscoverResponse(authcommon.ConvertToErrCode(err))
	}
	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return svr.nextSvr.ServiceInstancesCache(ctx, filter, req)
}

// GetRoutingConfigWithCache is the interface for getting routing config with cache
func (svr *Server) GetRoutingConfigWithCache(
	ctx context.Context, req *apiservice.Service) *apiservice.DiscoverResponse {

	authCtx := svr.collectServiceAuthContext(
		ctx, []*apiservice.Service{req}, authcommon.Read, authcommon.DiscoverRouterRule)
	if _, err := svr.policySvr.GetAuthChecker().CheckClientPermission(authCtx); err != nil {
		return api.NewDiscoverResponse(authcommon.ConvertToErrCode(err))
	}
	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return svr.nextSvr.GetRoutingConfigWithCache(ctx, req)
}

// GetRateLimitWithCache is the interface for getting rate limit with cache
func (svr *Server) GetRateLimitWithCache(
	ctx context.Context, req *apiservice.Service) *apiservice.DiscoverResponse {

	authCtx := svr.collectServiceAuthContext(
		ctx, []*apiservice.Service{req}, authcommon.Read, authcommon.DiscoverRateLimitRule)
	if _, err := svr.policySvr.GetAuthChecker().CheckClientPermission(authCtx); err != nil {
		return api.NewDiscoverResponse(authcommon.ConvertToErrCode(err))
	}
	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return svr.nextSvr.GetRateLimitWithCache(ctx, req)
}

// GetCircuitBreakerWithCache is the interface for getting a circuit breaker with cache
func (svr *Server) GetCircuitBreakerWithCache(
	ctx context.Context, req *apiservice.Service) *apiservice.DiscoverResponse {

	authCtx := svr.collectServiceAuthContext(
		ctx, []*apiservice.Service{req}, authcommon.Read, authcommon.DiscoverCircuitBreakerRule)
	if _, err := svr.policySvr.GetAuthChecker().CheckClientPermission(authCtx); err != nil {
		return api.NewDiscoverResponse(authcommon.ConvertToErrCode(err))
	}
	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return svr.nextSvr.GetCircuitBreakerWithCache(ctx, req)
}

// GetFaultDetectWithCache 获取主动探测规则列表
func (svr *Server) GetFaultDetectWithCache(
	ctx context.Context, req *apiservice.Service) *apiservice.DiscoverResponse {

	authCtx := svr.collectServiceAuthContext(
		ctx, []*apiservice.Service{req}, authcommon.Read, authcommon.DiscoverFaultDetectRule)
	if _, err := svr.policySvr.GetAuthChecker().CheckClientPermission(authCtx); err != nil {
		return api.NewDiscoverResponse(authcommon.ConvertToErrCode(err))
	}
	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return svr.nextSvr.GetFaultDetectWithCache(ctx, req)
}

// UpdateInstance update single instance
func (svr *Server) UpdateInstance(ctx context.Context, req *apiservice.Instance) *apiservice.Response {
	authCtx := svr.collectClientInstanceAuthContext(
		ctx, []*apiservice.Instance{req}, authcommon.Modify, authcommon.UpdateInstance)

	if _, err := svr.policySvr.GetAuthChecker().CheckClientPermission(authCtx); err != nil {
		return api.NewResponse(authcommon.ConvertToErrCode(err))
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return svr.nextSvr.UpdateInstance(ctx, req)
}

// GetServiceContractWithCache User Client Get ServiceContract Rule Information
func (svr *Server) GetServiceContractWithCache(ctx context.Context,
	req *apiservice.ServiceContract) *apiservice.Response {
	authCtx := svr.collectServiceAuthContext(ctx, []*apiservice.Service{{
		Namespace: wrapperspb.String(req.Namespace),
		Name:      wrapperspb.String(req.Service),
	}}, authcommon.Read, authcommon.DiscoverServiceContract)

	if _, err := svr.policySvr.GetAuthChecker().CheckClientPermission(authCtx); err != nil {
		return api.NewResponse(authcommon.ConvertToErrCode(err))
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return svr.nextSvr.GetServiceContractWithCache(ctx, req)
}

// GetLaneRuleWithCache fetch lane rules by client
func (svr *Server) GetLaneRuleWithCache(ctx context.Context, req *apiservice.Service) *apiservice.DiscoverResponse {
	authCtx := svr.collectServiceAuthContext(
		ctx, []*apiservice.Service{req}, authcommon.Read, authcommon.DiscoverLaneRule)
	if _, err := svr.policySvr.GetAuthChecker().CheckClientPermission(authCtx); err != nil {
		return api.NewDiscoverResponse(authcommon.ConvertToErrCode(err))
	}
	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return svr.nextSvr.GetLaneRuleWithCache(ctx, req)
}

// GetRouterRuleWithCache .
func (svr *Server) GetRouterRuleWithCache(ctx context.Context, req *apiservice.Service) *apiservice.DiscoverResponse {
	authCtx := svr.collectServiceAuthContext(
		ctx, []*apiservice.Service{req}, authcommon.Read, authcommon.DiscoverRouterRule)
	if _, err := svr.policySvr.GetAuthChecker().CheckClientPermission(authCtx); err != nil {
		return api.NewDiscoverResponse(authcommon.ConvertToErrCode(err))
	}
	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return svr.nextSvr.GetRouterRuleWithCache(ctx, req)
}
