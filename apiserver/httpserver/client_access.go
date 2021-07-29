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

package httpserver

import (
	"fmt"

	"github.com/polarismesh/polaris-server/apiserver"
	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/log"
	"github.com/emicklei/go-restful"
	"go.uber.org/zap"
)

/**
 * @brief 注册客户端接口
 */
func (h *Httpserver) GetClientAccessServer(include []string) (*restful.WebService, error) {
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

/**
 * @brief 增加服务发现接口
 */
func (h *Httpserver) addDiscoverAccess(ws *restful.WebService) {
	ws.Route(ws.POST("/ReportClient").To(h.ReportClient))
	ws.Route(ws.POST("/Discover").To(h.Discover))
}

/**
 * @brief 增加注册/反注册接口
 */
func (h *Httpserver) addRegisterAccess(ws *restful.WebService) {
	ws.Route(ws.POST("/RegisterInstance").To(h.RegisterInstance))
	ws.Route(ws.POST("/DeregisterInstance").To(h.DeregisterInstance))
}

/**
 * @brief 增加健康检查接口
 */
func (h *Httpserver) addHealthCheckAccess(ws *restful.WebService) {
	ws.Route(ws.POST("/Heartbeat").To(h.Heartbeat))
}

/**
 * @brief 客户端上报信息
 */
func (h *Httpserver) ReportClient(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	client := &api.Client{}
	context, err := handler.Parse(client)
	if context == nil {
		ret := api.NewResponseWithMsg(api.ParseException, err.Error())
		handler.WriteHeaderAndProto(ret)
		return
	}

	ret := h.namingServer.ReportClient(context, client)
	handler.WriteHeaderAndProto(ret)
}

/**
 * @brief 注册服务实例
 */
func (h *Httpserver) RegisterInstance(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	instance := &api.Instance{}
	context, err := handler.Parse(instance)
	if context == nil {
		ret := api.NewResponseWithMsg(api.ParseException, err.Error())
		handler.WriteHeaderAndProto(ret)
		return
	}

	ret := h.namingServer.CreateInstance(context, instance)
	handler.WriteHeaderAndProto(ret)
}

/**
 * @brief 反注册服务实例
 */
func (h *Httpserver) DeregisterInstance(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	instance := &api.Instance{}
	context, err := handler.Parse(instance)
	if context == nil {
		ret := api.NewResponseWithMsg(api.ParseException, err.Error())
		handler.WriteHeaderAndProto(ret)
		return
	}

	ret := h.namingServer.DeleteInstance(context, instance)
	handler.WriteHeaderAndProto(ret)
}

/**
 * @brief 统一发现接口
 */
func (h *Httpserver) Discover(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	discoverRequest := &api.DiscoverRequest{}
	context, err := handler.Parse(discoverRequest)
	if context == nil {
		ret := api.NewResponseWithMsg(api.ParseException, err.Error())
		handler.WriteHeaderAndProto(ret)
		return
	}

	var msg string
	msg = fmt.Sprintf("receive http discover request: %s", discoverRequest.Service.String())
	log.Info(msg,
		zap.String("type", api.DiscoverRequest_DiscoverRequestType_name[int32(discoverRequest.Type)]),
		zap.String("client-address", req.Request.RemoteAddr),
		zap.String("user-agent", req.HeaderParameter("User-Agent")),
		zap.String("request-id", req.HeaderParameter("Request-Id")),
	)

	var ret *api.DiscoverResponse
	switch discoverRequest.Type {
	case api.DiscoverRequest_INSTANCE:
		ret = h.namingServer.ServiceInstancesCache(context, discoverRequest.Service)
	case api.DiscoverRequest_ROUTING:
		ret = h.namingServer.GetRoutingConfigWithCache(context, discoverRequest.Service)
	case api.DiscoverRequest_RATE_LIMIT:
		ret = h.namingServer.GetRateLimitWithCache(context, discoverRequest.Service)
	case api.DiscoverRequest_CIRCUIT_BREAKER:
		ret = h.namingServer.GetCircuitBreakerWithCache(context, discoverRequest.Service)
	case api.DiscoverRequest_SERVICES:
		ret = h.namingServer.GetServiceWithCache(context, discoverRequest.Service)
	default:
		ret = api.NewDiscoverRoutingResponse(api.InvalidDiscoverResource, discoverRequest.Service)
	}

	handler.WriteHeaderAndProto(ret)
}

/**
 * @brief 服务实例心跳
 */
func (h *Httpserver) Heartbeat(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	instance := &api.Instance{}
	context, err := handler.Parse(instance)
	if context == nil {
		ret := api.NewResponseWithMsg(api.ParseException, err.Error())
		handler.WriteHeaderAndProto(ret)
		return
	}

	ret := h.namingServer.Heartbeat(context, instance)
	handler.WriteHeaderAndProto(ret)
}
