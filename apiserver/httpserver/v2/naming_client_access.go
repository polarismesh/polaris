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
	restfulspec "github.com/polarismesh/go-restful-openapi/v2"
	"go.uber.org/zap"

	"github.com/polarismesh/polaris/apiserver"
	httpcommon "github.com/polarismesh/polaris/apiserver/httpserver/http"
	api "github.com/polarismesh/polaris/common/api/v1"
	apiv2 "github.com/polarismesh/polaris/common/api/v2"
	"github.com/polarismesh/polaris/common/utils"
)

// GetClientAccessServer get client access server
func (h *HTTPServerV2) GetClientAccessServer(include []string) (*restful.WebService, error) {
	clientAccess := []string{apiserver.DiscoverAccess, apiserver.RegisterAccess, apiserver.HealthcheckAccess}

	ws := new(restful.WebService)

	ws.Path("/v2").Consumes(restful.MIME_JSON).Produces(restful.MIME_JSON)

	// 如果为空，则开启全部接口
	if len(include) == 0 {
		include = clientAccess
	}

	// 客户端接口：增删改请求操作存储层，查请求访问缓存
	for _, item := range include {
		switch item {
		case apiserver.DiscoverAccess:
			h.addDiscoverAccess(ws)
		}
	}

	return ws, nil
}

// addDiscoverAccess 增加服务发现接口
func (h *HTTPServerV2) addDiscoverAccess(ws *restful.WebService) {
	tags := []string{"DiscoverAccess"}
	ws.Route(ws.POST("/Discover").To(h.Discover).
		Doc("服务发现").
		Metadata(restfulspec.KeyOpenAPITags, tags))
}

// Discover 统一发现接口
func (h *HTTPServerV2) Discover(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	discoverRequest := &apiv2.DiscoverRequest{}
	ctx, err := handler.Parse(discoverRequest)
	if err != nil {
		handler.WriteHeaderAndProto(api.NewResponseWithMsg(api.ParseException, err.Error()))
		return
	}

	msg := fmt.Sprintf("receive http discover request: %s", discoverRequest.GetSerivce().String())
	namingLog.Info(msg,
		zap.String("type", api.DiscoverRequest_DiscoverRequestType_name[int32(discoverRequest.Type)]),
		zap.String("client-address", req.Request.RemoteAddr),
		zap.String("user-agent", req.HeaderParameter("User-Agent")),
		utils.ZapRequestID(req.HeaderParameter("Request-Id")),
	)

	var ret *apiv2.DiscoverResponse
	switch discoverRequest.Type {
	case apiv2.DiscoverRequest_ROUTING:
		ret = h.namingServer.GetRoutingConfigV2WithCache(ctx, discoverRequest.GetSerivce())
	default:
		ret = apiv2.NewDiscoverRoutingResponse(api.InvalidDiscoverResource, discoverRequest.GetSerivce())
	}

	handler.WriteHeaderAndProtoV2(ret)
}
