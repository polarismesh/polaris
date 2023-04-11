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
	"fmt"

	"github.com/emicklei/go-restful/v3"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"
	"go.uber.org/zap"

	"github.com/polarismesh/polaris/apiserver"
	"github.com/polarismesh/polaris/apiserver/httpserver/docs"
	httpcommon "github.com/polarismesh/polaris/apiserver/httpserver/utils"
	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/utils"
)

// GetClientAccessServer get client access server
func (h *HTTPServerV1) GetClientAccessServer(include []string) (*restful.WebService, error) {
	clientAccess := []string{apiserver.DiscoverAccess, apiserver.RegisterAccess, apiserver.HealthcheckAccess}

	ws := new(restful.WebService)

	ws.Path("/v1").Consumes(restful.MIME_JSON).Produces(restful.MIME_JSON)

	// 如果为空，则开启全部接口
	if len(include) == 0 {
		include = clientAccess
	}

	// 客户端接口：增删改请求操作存储层，查请求访问缓存
	for _, item := range include {
		switch item {
		case apiserver.DiscoverAccess:
			h.addDiscoverAccess(ws)
		case apiserver.RegisterAccess:
			h.addRegisterAccess(ws)
		case apiserver.HealthcheckAccess:
			h.addHealthCheckAccess(ws)
		default:
			log.Errorf("method %s does not exist in httpserver client access", item)
			return nil, fmt.Errorf("method %s does not exist in httpserver client access", item)
		}
	}

	return ws, nil
}

// addDiscoverAccess 增加服务发现接口
func (h *HTTPServerV1) addDiscoverAccess(ws *restful.WebService) {
	ws.Route(docs.EnrichReportClientApiDocs(ws.POST("/ReportClient").To(h.ReportClient)))
	ws.Route(docs.EnrichDiscoverApiDocs(ws.POST("/Discover").To(h.Discover)))
}

// addRegisterAccess 增加注册/反注册接口
func (h *HTTPServerV1) addRegisterAccess(ws *restful.WebService) {
	ws.Route(docs.EnrichRegisterInstanceApiDocs(ws.POST("/RegisterInstance").To(h.RegisterInstance)))
	ws.Route(docs.EnrichDeregisterInstanceApiDocs(ws.POST("/DeregisterInstance").To(h.DeregisterInstance)))
}

// addHealthCheckAccess 增加健康检查接口
func (h *HTTPServerV1) addHealthCheckAccess(ws *restful.WebService) {
	ws.Route(docs.EnrichHeartbeatApiDocs(ws.POST("/Heartbeat").To(h.Heartbeat)))
}

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

	msg := fmt.Sprintf("receive http discover request: %s", discoverRequest.Service.String())
	namingLog.Info(msg,
		zap.String("type", apiservice.DiscoverRequest_DiscoverRequestType_name[int32(discoverRequest.Type)]),
		zap.String("client-address", req.Request.RemoteAddr),
		zap.String("user-agent", req.HeaderParameter("User-Agent")),
		utils.ZapRequestID(req.HeaderParameter("Request-Id")),
	)

	var ret *apiservice.DiscoverResponse
	switch discoverRequest.Type {
	case apiservice.DiscoverRequest_INSTANCE:
		ret = h.namingServer.ServiceInstancesCache(ctx, discoverRequest.Service)
	case apiservice.DiscoverRequest_ROUTING:
		ret = h.namingServer.GetRoutingConfigWithCache(ctx, discoverRequest.Service)
	case apiservice.DiscoverRequest_RATE_LIMIT:
		ret = h.namingServer.GetRateLimitWithCache(ctx, discoverRequest.Service)
	case apiservice.DiscoverRequest_CIRCUIT_BREAKER:
		ret = h.namingServer.GetCircuitBreakerWithCache(ctx, discoverRequest.Service)
	case apiservice.DiscoverRequest_SERVICES:
		ret = h.namingServer.GetServiceWithCache(ctx, discoverRequest.Service)
	case apiservice.DiscoverRequest_FAULT_DETECTOR:
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
