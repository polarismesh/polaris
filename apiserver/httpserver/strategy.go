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
	"github.com/emicklei/go-restful"
	api "github.com/polarismesh/polaris-server/common/api/v1"
)

// GetMaintainAccessServer 运维接口
func (h *HTTPServer) GetAuthStrategyAccessServer() *restful.WebService {
	ws := new(restful.WebService)
	ws.Path("/core/v1").Consumes(restful.MIME_JSON).Produces(restful.MIME_JSON)

	//
	ws.Route(ws.POST("/auth/strategy").To(h.CreateAuthStrategy))
	ws.Route(ws.PUT("/auth/strategy").To(h.UpdateAuthStrategy))
	ws.Route(ws.POST("/auth/strategies/delete").To(h.DeleteStrategy))
	ws.Route(ws.GET("/auth/strategies").To(h.ListStrategy))
	ws.Route(ws.POST("/auth/strategy/resource").To(h.AddStrategyResources))
	ws.Route(ws.POST("/auth/strategy/resource/delete").To(h.DeleteStrategyResources))
	return ws
}

func (h *HTTPServer) CreateAuthStrategy(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	strategy := &api.AuthStrategy{}

	ctx, err := handler.Parse(strategy)
	if err != nil {
		handler.WriteHeaderAndProto(api.NewResponseWithMsg(api.ParseException, err.Error()))
		return
	}

	handler.WriteHeaderAndProto(h.strategyServer.CreateStrategy(ctx, strategy))
}

func (h *HTTPServer) UpdateAuthStrategy(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	strategy := &api.AuthStrategy{}

	ctx, err := handler.Parse(strategy)
	if err != nil {
		handler.WriteHeaderAndProto(api.NewResponseWithMsg(api.ParseException, err.Error()))
		return
	}

	handler.WriteHeaderAndProto(h.strategyServer.UpdateStrategy(ctx, strategy))
}

func (h *HTTPServer) DeleteStrategy(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	strategy := &api.AuthStrategy{}

	ctx, err := handler.Parse(strategy)
	if err != nil {
		handler.WriteHeaderAndProto(api.NewResponseWithMsg(api.ParseException, err.Error()))
		return
	}

	handler.WriteHeaderAndProto(h.strategyServer.DeleteStrategy(ctx, strategy))
}

func (h *HTTPServer) ListStrategy(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	queryParams := parseQueryParams(req)
	ctx := handler.ParseHeaderContext()

	handler.WriteHeaderAndProto(h.strategyServer.ListStrategy(ctx, queryParams))
}

func (h *HTTPServer) AddStrategyResources(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	resources := &api.StrategyResource{}

	ctx, err := handler.Parse(resources)
	if err != nil {
		handler.WriteHeaderAndProto(api.NewResponseWithMsg(api.ParseException, err.Error()))
		return
	}

	handler.WriteHeaderAndProto(h.strategyServer.AddStrategyResources(ctx, resources))
}

func (h *HTTPServer) DeleteStrategyResources(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	resources := &api.StrategyResource{}

	ctx, err := handler.Parse(resources)
	if err != nil {
		handler.WriteHeaderAndProto(api.NewResponseWithMsg(api.ParseException, err.Error()))
		return
	}

	handler.WriteHeaderAndProto(h.strategyServer.DeleteStrategyResources(ctx, resources))
}
