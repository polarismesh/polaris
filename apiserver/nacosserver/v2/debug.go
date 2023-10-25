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

package v2

import (
	"net/http"

	"github.com/emicklei/go-restful/v3"

	v1 "github.com/polarismesh/polaris/apiserver/nacosserver/v1"
	nacoshttp "github.com/polarismesh/polaris/apiserver/nacosserver/v1/http"
)

func (n *NacosV2Server) RegistryDebugRoute() {
	_ = v1.RegistryDebugRoute("DescribeConnections", func(ws *restful.WebService) *restful.RouteBuilder {
		return ws.GET("/2.x/connections").To(n.DescribeConnections)
	})
	_ = v1.RegistryDebugRoute("DescribeConnectionDetail", func(ws *restful.WebService) *restful.RouteBuilder {
		return ws.GET("/2.x/connections/detail").To(n.DescribeConnectionDetail)
	})
}

// DescribeConnections 查询 V2 客户端的连接
func (n *NacosV2Server) DescribeConnections(req *restful.Request, rsp *restful.Response) {
	connections := n.connectionManager.ListConnections()

	ret := map[string]interface{}{}
	ret["code"] = 200
	ret["count"] = len(connections)
	ret["connections"] = connections

	nacoshttp.WrirteNacosResponse(ret, rsp)
}

// DescribeConnectionDetail 查询某一个连接ID的详细信息
func (n *NacosV2Server) DescribeConnectionDetail(req *restful.Request, rsp *restful.Response) {
	connID := req.QueryParameter("conn_id")
	client, exist := n.connectionManager.GetClient(connID)

	ret := map[string]interface{}{}
	if exist {
		ret["code"] = 200
		ret["connection"] = client
	} else {
		ret["code"] = 404
	}

	nacoshttp.WrirteNacosResponseWithCode(http.StatusNotFound, ret, rsp)
}
