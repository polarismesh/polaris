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
	"context"

	"github.com/emicklei/go-restful/v3"

	httpcommon "github.com/polarismesh/polaris/apiserver/httpserver/utils"
)

func (h *HTTPServer) GetClientServer(ws *restful.WebService) error {
	ws.Route(ws.GET("/clients").To(h.GetReportClients))
	return nil
}

func (h *HTTPServer) GetReportClients(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	queryParams := httpcommon.ParseQueryParams(req)
	ctx := handler.ParseHeaderContext()
	ret := h.namingServer.GetPrometheusTargets(ctx, queryParams)

	_ = rsp.WriteAsJson(ret)
}

// GetPrometheusDiscoveryServer 注册用于prometheus服务发现的接口
func (h *HTTPServer) GetPrometheusDiscoveryServer(include []string) (*restful.WebService, error) {
	ws := new(restful.WebService)
	ws.Path("/prometheus/v1").Consumes(restful.MIME_JSON).Produces(restful.MIME_JSON)
	h.addPrometheusDefaultAccess(ws)
	return ws, nil
}

func (h *HTTPServer) addPrometheusDefaultAccess(ws *restful.WebService) {
	ws.Route(ws.GET("/clients").To(h.GetPrometheusClients))
}

// GetPrometheusClients 对接 prometheus 基于 http 的 service discovery
func (h *HTTPServer) GetPrometheusClients(req *restful.Request, rsp *restful.Response) {
	queryParams := httpcommon.ParseQueryParams(req)
	ret := h.namingServer.GetPrometheusTargets(context.Background(), queryParams)
	_ = rsp.WriteAsJson(ret.Response)
}
