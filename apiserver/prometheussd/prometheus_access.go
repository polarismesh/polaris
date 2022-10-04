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

package prometheussd

import (
	"context"

	"github.com/emicklei/go-restful/v3"
)

// GetPrometheusDiscoveryServer 注册用于prometheus服务发现的接口
func (h *PrometheusServer) GetPrometheusDiscoveryServer(include []string) (*restful.WebService, error) {
	ws := new(restful.WebService)

	ws.Path("/prometheus/v1").Consumes(restful.MIME_JSON).Produces(restful.MIME_JSON)

	h.addPrometheusDefaultAccess(ws)

	return ws, nil
}

func (h *PrometheusServer) addPrometheusDefaultAccess(ws *restful.WebService) {
	ws.Route(ws.GET("/clients").To(h.GetPrometheusClients))
}

// GetPrometheusClients 对接 prometheus 基于 http 的 service discovery
// [
//
//	{
//	  "targets": [ "<host>", ... ],
//	  "labels": {
//	    "<labelname>": "<labelvalue>", ...
//	  }
//	},
//	...
//
// ]
func (h *PrometheusServer) GetPrometheusClients(req *restful.Request, rsp *restful.Response) {

	queryParams := ParseQueryParams(req)
	ret := h.namingServer.GetReportClientWithCache(context.Background(), queryParams)

	_ = rsp.WriteAsJson(ret.Response)
}

// parseQueryParams 解析并获取HTTP的query params
func ParseQueryParams(req *restful.Request) map[string]string {
	queryParams := make(map[string]string)
	for key, value := range req.Request.URL.Query() {
		if len(value) > 0 {
			queryParams[key] = value[0] // 暂时默认只支持一个查询
		}
	}

	return queryParams
}
