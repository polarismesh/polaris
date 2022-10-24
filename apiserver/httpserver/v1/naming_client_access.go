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
	"fmt"

	"github.com/emicklei/go-restful/v3"
	restfulspec "github.com/polarismesh/go-restful-openapi/v2"
	"go.uber.org/zap"

	"github.com/polarismesh/polaris/apiserver"
	httpcommon "github.com/polarismesh/polaris/apiserver/httpserver/http"
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
	tags := []string{"DiscoverAccess"}
	ws.Route(ws.POST("/ReportClient").To(h.ReportClient).
		Doc("上报客户端").
		Writes(&api.Client{}).
		Metadata(restfulspec.KeyOpenAPITags, tags))
	ws.Route(ws.POST("/Discover").To(h.Discover).
		Doc("服务发现").
		Metadata(restfulspec.KeyOpenAPITags, tags))
}

// addRegisterAccess 增加注册/反注册接口
func (h *HTTPServerV1) addRegisterAccess(ws *restful.WebService) {
	tags := []string{"Instances", "RegisterAccess"}
	ws.Route(ws.POST("/RegisterInstance").
		Doc("注册实例").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		To(h.RegisterInstance))

	ws.Route(ws.POST("/DeregisterInstance").
		Doc("移除注册实例").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		To(h.DeregisterInstance))
}

// addHealthCheckAccess 增加健康检查接口
func (h *HTTPServerV1) addHealthCheckAccess(ws *restful.WebService) {
	ws.Route(ws.POST("/Heartbeat").To(h.Heartbeat))
}

// ReportClient 客户端上报信息
func (h *HTTPServerV1) ReportClient(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}
	client := &api.Client{}
	ctx, err := handler.Parse(client)
	if err != nil {
		handler.WriteHeaderAndProto(api.NewResponseWithMsg(api.ParseException, err.Error()))
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

	instance := &api.Instance{}
	ctx, err := handler.Parse(instance)
	if err != nil {
		handler.WriteHeaderAndProto(api.NewResponseWithMsg(api.ParseException, err.Error()))
		return
	}

	handler.WriteHeaderAndProto(h.namingServer.RegisterInstance(ctx, instance))
}

// DeregisterInstance 反注册服务实例
func (h *HTTPServerV1) DeregisterInstance(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	instance := &api.Instance{}
	ctx, err := handler.Parse(instance)
	if err != nil {
		handler.WriteHeaderAndProto(api.NewResponseWithMsg(api.ParseException, err.Error()))
		return
	}

	handler.WriteHeaderAndProto(h.namingServer.DeregisterInstance(ctx, instance))
}

// Discover 统一发现接口
func (h *HTTPServerV1) Discover(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	discoverRequest := &api.DiscoverRequest{}
	ctx, err := handler.Parse(discoverRequest)
	if err != nil {
		handler.WriteHeaderAndProto(api.NewResponseWithMsg(api.ParseException, err.Error()))
		return
	}

	msg := fmt.Sprintf("receive http discover request: %s", discoverRequest.Service.String())
	namingLog.Info(msg,
		zap.String("type", api.DiscoverRequest_DiscoverRequestType_name[int32(discoverRequest.Type)]),
		zap.String("client-address", req.Request.RemoteAddr),
		zap.String("user-agent", req.HeaderParameter("User-Agent")),
		utils.ZapRequestID(req.HeaderParameter("Request-Id")),
	)

	var ret *api.DiscoverResponse
	switch discoverRequest.Type {
	case api.DiscoverRequest_INSTANCE:
		ret = h.namingServer.ServiceInstancesCache(ctx, discoverRequest.Service)
	case api.DiscoverRequest_ROUTING:
		ret = h.namingServer.GetRoutingConfigWithCache(ctx, discoverRequest.Service)
	case api.DiscoverRequest_RATE_LIMIT:
		ret = h.namingServer.GetRateLimitWithCache(ctx, discoverRequest.Service)
	case api.DiscoverRequest_CIRCUIT_BREAKER:
		ret = h.namingServer.GetCircuitBreakerWithCache(ctx, discoverRequest.Service)
	case api.DiscoverRequest_SERVICES:
		ret = h.namingServer.GetServiceWithCache(ctx, discoverRequest.Service)
	default:
		ret = api.NewDiscoverRoutingResponse(api.InvalidDiscoverResource, discoverRequest.Service)
	}

	handler.WriteHeaderAndProto(ret)
}

// Heartbeat 服务实例心跳
func (h *HTTPServerV1) Heartbeat(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	instance := &api.Instance{}
	ctx, err := handler.Parse(instance)
	if err != nil {
		handler.WriteHeaderAndProto(api.NewResponseWithMsg(api.ParseException, err.Error()))
		return
	}

	handler.WriteHeaderAndProto(h.healthCheckServer.Report(ctx, instance))
}
