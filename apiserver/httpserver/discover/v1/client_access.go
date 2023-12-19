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

package v1

import (
	"context"

	"github.com/emicklei/go-restful/v3"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"

	httpcommon "github.com/polarismesh/polaris/apiserver/httpserver/utils"
	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/metrics"
	commontime "github.com/polarismesh/polaris/common/time"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/plugin"
)

// ReportClient 客户端上报信息
func (h *HTTPServerV1) ReportClient(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}
	client := &apiservice.Client{}
	ctx, err := handler.Parse(client)
	if err != nil {
		handler.WriteHeaderAndProto(api.NewResponseWithMsg(apimodel.Code_ParseException, err.Error()))
		return
	}

	handler.WriteHeaderAndProto(h.namingServer.ReportClient(ctx, client))
}

// RegisterInstance 注册服务实例
func (h *HTTPServerV1) RegisterInstance(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	instance := &apiservice.Instance{}
	ctx, err := handler.Parse(instance)
	if err != nil {
		handler.WriteHeaderAndProto(api.NewResponseWithMsg(apimodel.Code_ParseException, err.Error()))
		return
	}
	// 客户端请求中带了 token 的，优先已请求中的为准
	if instance.GetServiceToken().GetValue() != "" {
		ctx = context.WithValue(ctx, utils.ContextAuthTokenKey, instance.GetServiceToken().GetValue())
	}

	handler.WriteHeaderAndProto(h.namingServer.RegisterInstance(ctx, instance))
}

// DeregisterInstance 反注册服务实例
func (h *HTTPServerV1) DeregisterInstance(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	instance := &apiservice.Instance{}
	ctx, err := handler.Parse(instance)
	if err != nil {
		handler.WriteHeaderAndProto(api.NewResponseWithMsg(apimodel.Code_ParseException, err.Error()))
		return
	}
	// 客户端请求中带了 token 的，优先已请求中的为准
	if instance.GetServiceToken().GetValue() != "" {
		ctx = context.WithValue(ctx, utils.ContextAuthTokenKey, instance.GetServiceToken().GetValue())
	}
	handler.WriteHeaderAndProto(h.namingServer.DeregisterInstance(ctx, instance))
}

// Discover 统一发现接口
func (h *HTTPServerV1) Discover(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	discoverRequest := &apiservice.DiscoverRequest{}
	ctx, err := handler.Parse(discoverRequest)
	if err != nil {
		handler.WriteHeaderAndProto(api.NewResponseWithMsg(apimodel.Code_ParseException, err.Error()))
		return
	}

	startTime := commontime.CurrentMillisecond()
	var ret *apiservice.DiscoverResponse
	var action string
	defer func() {
		plugin.GetStatis().ReportDiscoverCall(metrics.ClientDiscoverMetric{
			Action:    action,
			ClientIP:  utils.ParseClientAddress(ctx),
			Namespace: discoverRequest.GetService().GetNamespace().GetValue(),
			Resource:  discoverRequest.GetType().String() + ":" + discoverRequest.GetService().GetName().GetValue(),
			Timestamp: startTime,
			CostTime:  commontime.CurrentMillisecond() - startTime,
			Revision:  ret.GetService().GetRevision().GetValue(),
			Success:   ret.GetCode().GetValue() > uint32(apimodel.Code_DataNoChange),
		})
	}()

	switch discoverRequest.Type {
	case apiservice.DiscoverRequest_INSTANCE:
		action = metrics.ActionDiscoverInstance
		ret = h.namingServer.ServiceInstancesCache(ctx, discoverRequest.Filter, discoverRequest.Service)
	case apiservice.DiscoverRequest_ROUTING:
		action = metrics.ActionDiscoverRouterRule
		ret = h.namingServer.GetRoutingConfigWithCache(ctx, discoverRequest.Service)
	case apiservice.DiscoverRequest_RATE_LIMIT:
		action = metrics.ActionDiscoverRateLimit
		ret = h.namingServer.GetRateLimitWithCache(ctx, discoverRequest.Service)
	case apiservice.DiscoverRequest_CIRCUIT_BREAKER:
		action = metrics.ActionDiscoverCircuitBreaker
		ret = h.namingServer.GetCircuitBreakerWithCache(ctx, discoverRequest.Service)
	case apiservice.DiscoverRequest_SERVICES:
		action = metrics.ActionDiscoverServices
		ret = h.namingServer.GetServiceWithCache(ctx, discoverRequest.Service)
	case apiservice.DiscoverRequest_FAULT_DETECTOR:
		action = metrics.ActionDiscoverFaultDetect
		ret = h.namingServer.GetFaultDetectWithCache(ctx, discoverRequest.Service)
	default:
		ret = api.NewDiscoverRoutingResponse(apimodel.Code_InvalidDiscoverResource, discoverRequest.Service)
	}

	handler.WriteHeaderAndProto(ret)
}

// Heartbeat 服务实例心跳
func (h *HTTPServerV1) Heartbeat(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	instance := &apiservice.Instance{}
	ctx, err := handler.Parse(instance)
	if err != nil {
		handler.WriteHeaderAndProto(api.NewResponseWithMsg(apimodel.Code_ParseException, err.Error()))
		return
	}

	handler.WriteHeaderAndProto(h.healthCheckServer.Report(ctx, instance))
}
