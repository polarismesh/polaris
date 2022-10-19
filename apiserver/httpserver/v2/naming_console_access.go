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
	"fmt"

	"github.com/emicklei/go-restful/v3"
	"github.com/golang/protobuf/proto"

	httpcommon "github.com/polarismesh/polaris/apiserver/httpserver/http"
	apiv1 "github.com/polarismesh/polaris/common/api/v1"
	apiv2 "github.com/polarismesh/polaris/common/api/v2"
)

const (
	defaultReadAccess string = "default-read"
	defaultAccess     string = "default"
)

// GetNamingConsoleAccessServer 注册管理端接口
func (h *HTTPServerV2) GetNamingConsoleAccessServer(include []string) (*restful.WebService, error) {
	consoleAccess := []string{defaultAccess}

	ws := new(restful.WebService)

	ws.Path("/naming/v2").Consumes(restful.MIME_JSON).Produces(restful.MIME_JSON)

	// 如果为空，则开启全部接口
	if len(include) == 0 {
		include = consoleAccess
	}

	var hasDefault = false
	for _, item := range include {
		if item == defaultAccess {
			hasDefault = true
			break
		}
	}
	for _, item := range include {
		switch item {
		case defaultReadAccess:
			if !hasDefault {
				h.addDefaultReadAccess(ws)
			}
		case defaultAccess:
			h.addDefaultAccess(ws)
		default:
			log.Errorf("method %s does not exist in HTTPServerV2 console access", item)
			return nil, fmt.Errorf("method %s does not exist in HTTPServerV2 console access", item)
		}
	}
	return ws, nil
}

// addDefaultReadAccess 增加默认读接口
func (h *HTTPServerV2) addDefaultReadAccess(ws *restful.WebService) {
	ws.Route(ws.POST("/routings").To(h.CreateRoutings))
	ws.Route(ws.GET("/routings").To(h.GetRoutings))
}

// addDefaultAccess 增加默认接口
func (h *HTTPServerV2) addDefaultAccess(ws *restful.WebService) {
	ws.Route(ws.POST("/routings").To(h.CreateRoutings))
	ws.Route(ws.POST("/routings/delete").To(h.DeleteRoutings))
	ws.Route(ws.PUT("/routings").To(h.UpdateRoutings))
	ws.Route(ws.GET("/routings").To(h.GetRoutings))
	ws.Route(ws.PUT("/routings/enable").To(h.EnableRoutings))
}

// CreateRoutings 创建规则路由
func (h *HTTPServerV2) CreateRoutings(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	var routings RoutingArr
	ctx, err := handler.ParseArray(func() proto.Message {
		msg := &apiv2.Routing{}
		routings = append(routings, msg)
		return msg
	})
	if err != nil {
		handler.WriteHeaderAndProtoV2(apiv2.NewBatchWriteResponseWithMsg(apiv1.ParseException, err.Error()))
		return
	}

	ret := h.namingServer.CreateRoutingConfigsV2(ctx, routings)
	handler.WriteHeaderAndProtoV2(ret)
}

// DeleteRoutings 删除规则路由
func (h *HTTPServerV2) DeleteRoutings(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	var routings RoutingArr
	ctx, err := handler.ParseArray(func() proto.Message {
		msg := &apiv2.Routing{}
		routings = append(routings, msg)
		return msg
	})
	if err != nil {
		handler.WriteHeaderAndProtoV2(apiv2.NewBatchWriteResponseWithMsg(apiv1.ParseException, err.Error()))
		return
	}

	ret := h.namingServer.DeleteRoutingConfigsV2(ctx, routings)
	handler.WriteHeaderAndProtoV2(ret)
}

// UpdateRoutings 修改规则路由
func (h *HTTPServerV2) UpdateRoutings(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	var routings RoutingArr
	ctx, err := handler.ParseArray(func() proto.Message {
		msg := &apiv2.Routing{}
		routings = append(routings, msg)
		return msg
	})
	if err != nil {
		handler.WriteHeaderAndProtoV2(apiv2.NewBatchWriteResponseWithMsg(apiv1.ParseException, err.Error()))
		return
	}

	ret := h.namingServer.UpdateRoutingConfigsV2(ctx, routings)
	handler.WriteHeaderAndProtoV2(ret)
}

// GetRoutings 查询规则路由
func (h *HTTPServerV2) GetRoutings(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	queryParams := httpcommon.ParseQueryParams(req)
	ret := h.namingServer.GetRoutingConfigsV2(handler.ParseHeaderContext(), queryParams)
	handler.WriteHeaderAndProtoV2(ret)
}

// EnableRoutings 查询规则路由
func (h *HTTPServerV2) EnableRoutings(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	var routings RoutingArr
	ctx, err := handler.ParseArray(func() proto.Message {
		msg := &apiv2.Routing{}
		routings = append(routings, msg)
		return msg
	})
	if err != nil {
		handler.WriteHeaderAndProtoV2(apiv2.NewBatchWriteResponseWithMsg(apiv1.ParseException, err.Error()))
		return
	}

	ret := h.namingServer.EnableRoutings(ctx, routings)
	handler.WriteHeaderAndProtoV2(ret)
}
