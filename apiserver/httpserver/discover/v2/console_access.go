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
	"io"
	"strings"

	"github.com/emicklei/go-restful/v3"
	"github.com/golang/protobuf/proto"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	apitraffic "github.com/polarismesh/specification/source/go/api/v1/traffic_manage"

	v1 "github.com/polarismesh/polaris/apiserver/httpserver/discover/v1"
	httpcommon "github.com/polarismesh/polaris/apiserver/httpserver/utils"
	apiv1 "github.com/polarismesh/polaris/common/api/v1"
)

const (
	deprecatedRoutingV2TypeUrl = "type.googleapis.com/v2."
	newRoutingV2TypeUrl        = "type.googleapis.com/v1."
)

func (h *HTTPServerV2) replaceV2TypeUrl(req *restful.Request) (string, error) {
	requestBytes, err := io.ReadAll(req.Request.Body)
	if err != nil {
		return "", err
	}
	requestText := strings.ReplaceAll(string(requestBytes), deprecatedRoutingV2TypeUrl, newRoutingV2TypeUrl)
	return requestText, nil
}

// CreateRoutings 创建规则路由
func (h *HTTPServerV2) CreateRoutings(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	requestText, err := h.replaceV2TypeUrl(req)
	if err != nil {
		handler.WriteHeaderAndProto(apiv1.NewBatchWriteResponseWithMsg(apimodel.Code_ParseException, err.Error()))
		return
	}
	var routings v1.RouterArr
	ctx, err := handler.ParseArrayByText(func() proto.Message {
		msg := &apitraffic.RouteRule{}
		routings = append(routings, msg)
		return msg
	}, requestText)
	if err != nil {
		handler.WriteHeaderAndProto(apiv1.NewBatchWriteResponseWithMsg(apimodel.Code_ParseException, err.Error()))
		return
	}

	ret := h.namingServer.CreateRoutingConfigsV2(ctx, routings)
	handler.WriteHeaderAndProto(ret)
}

// DeleteRoutings 删除规则路由
func (h *HTTPServerV2) DeleteRoutings(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}
	requestText, err := h.replaceV2TypeUrl(req)
	if err != nil {
		handler.WriteHeaderAndProto(apiv1.NewBatchWriteResponseWithMsg(apimodel.Code_ParseException, err.Error()))
		return
	}
	var routings v1.RouterArr
	ctx, err := handler.ParseArrayByText(func() proto.Message {
		msg := &apitraffic.RouteRule{}
		routings = append(routings, msg)
		return msg
	}, requestText)
	if err != nil {
		handler.WriteHeaderAndProto(apiv1.NewBatchWriteResponseWithMsg(apimodel.Code_ParseException, err.Error()))
		return
	}

	ret := h.namingServer.DeleteRoutingConfigsV2(ctx, routings)
	handler.WriteHeaderAndProto(ret)
}

// UpdateRoutings 修改规则路由
func (h *HTTPServerV2) UpdateRoutings(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}
	requestText, err := h.replaceV2TypeUrl(req)
	if err != nil {
		handler.WriteHeaderAndProto(apiv1.NewBatchWriteResponseWithMsg(apimodel.Code_ParseException, err.Error()))
		return
	}
	var routings v1.RouterArr
	ctx, err := handler.ParseArrayByText(func() proto.Message {
		msg := &apitraffic.RouteRule{}
		routings = append(routings, msg)
		return msg
	}, requestText)
	if err != nil {
		handler.WriteHeaderAndProto(apiv1.NewBatchWriteResponseWithMsg(apimodel.Code_ParseException, err.Error()))
		return
	}

	ret := h.namingServer.UpdateRoutingConfigsV2(ctx, routings)
	handler.WriteHeaderAndProto(ret)
}

// GetRoutings 查询规则路由
func (h *HTTPServerV2) GetRoutings(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	queryParams := httpcommon.ParseQueryParams(req)
	ret := h.namingServer.QueryRoutingConfigsV2(handler.ParseHeaderContext(), queryParams)
	handler.WriteHeaderAndProto(ret)
}

// EnableRoutings 启用规则路由
func (h *HTTPServerV2) EnableRoutings(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}
	requestText, err := h.replaceV2TypeUrl(req)
	if err != nil {
		handler.WriteHeaderAndProto(apiv1.NewBatchWriteResponseWithMsg(apimodel.Code_ParseException, err.Error()))
		return
	}
	var routings v1.RouterArr
	ctx, err := handler.ParseArrayByText(func() proto.Message {
		msg := &apitraffic.RouteRule{}
		routings = append(routings, msg)
		return msg
	}, requestText)
	if err != nil {
		handler.WriteHeaderAndProto(apiv1.NewBatchWriteResponseWithMsg(apimodel.Code_ParseException, err.Error()))
		return
	}

	ret := h.namingServer.EnableRoutings(ctx, routings)
	handler.WriteHeaderAndProto(ret)
}
