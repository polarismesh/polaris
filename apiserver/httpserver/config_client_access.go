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
	"strconv"

	"github.com/emicklei/go-restful"
	"google.golang.org/protobuf/types/known/wrapperspb"

	api "github.com/polarismesh/polaris-server/common/api/v1"
)

func (h *HTTPServer) getConfigFile(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	version, err := strconv.ParseUint(handler.QueryParameter("version"), 10, 64)
	if err != nil {
		handler.WriteHeaderAndProto(api.NewConfigClientResponseWithMessage(api.BadRequest, "version must be number"))
	}

	configFile := &api.ClientConfigFileInfo{
		Namespace: &wrapperspb.StringValue{Value: handler.QueryParameter("namespace")},
		Group: &wrapperspb.StringValue{Value: handler.QueryParameter("group")},
		FileName: &wrapperspb.StringValue{Value: handler.QueryParameter("fileName")},
		Version: &wrapperspb.UInt64Value{Value: version},
	}

	response := h.configServer.GetConfigFileForClient(handler.ParseHeaderContext(), configFile)

	handler.WriteHeaderAndProto(response)
}

func (h *HTTPServer) watchConfigFile(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	// 1. 解析出客户端监听的配置文件列表
	watchConfigFileRequest := &api.ClientWatchConfigFileRequest{}
	_, err := handler.Parse(watchConfigFileRequest)
	if err != nil {
		handler.WriteHeaderAndProto(api.NewResponseWithMsg(api.ParseException, err.Error()))
		return
	}

	// 阻塞等待响应
	callback, err := h.configServer.WatchConfigFiles(handler.ParseHeaderContext(), watchConfigFileRequest)
	if err != nil {

	}

	handler.WriteHeaderAndProto(callback())
}
