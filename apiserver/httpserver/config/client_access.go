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

package config

import (
	"fmt"
	"strconv"

	"github.com/emicklei/go-restful/v3"
	apiconfig "github.com/polarismesh/specification/source/go/api/v1/config_manage"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	"google.golang.org/protobuf/types/known/wrapperspb"

	httpcommon "github.com/polarismesh/polaris/apiserver/httpserver/utils"
	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/metrics"
	commontime "github.com/polarismesh/polaris/common/time"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/plugin"
)

func (h *HTTPServer) ClientGetConfigFile(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	version, _ := strconv.ParseUint(handler.Request.QueryParameter("version"), 10, 64)
	configFile := &apiconfig.ClientConfigFileInfo{
		Namespace: &wrapperspb.StringValue{Value: handler.Request.QueryParameter("namespace")},
		Group:     &wrapperspb.StringValue{Value: handler.Request.QueryParameter("group")},
		FileName:  &wrapperspb.StringValue{Value: handler.Request.QueryParameter("fileName")},
		Version:   &wrapperspb.UInt64Value{Value: version},
	}

	ctx := handler.ParseHeaderContext()
	startTime := commontime.CurrentMillisecond()
	defer func() {
		plugin.GetStatis().ReportDiscoverCall(metrics.ClientDiscoverMetric{
			ClientIP:  utils.ParseClientAddress(ctx),
			Namespace: configFile.GetNamespace().GetValue(),
			Resource: fmt.Sprintf("CONFIG_FILE:%s|%s|%d", configFile.GetGroup().GetValue(),
				configFile.GetFileName().GetValue(), version),
			Timestamp: startTime,
			CostTime:  commontime.CurrentMillisecond() - startTime,
		})
	}()

	response := h.configServer.GetConfigFileForClient(ctx, configFile)
	handler.WriteHeaderAndProto(response)
}

func (h *HTTPServer) ClientWatchConfigFile(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	// 1. 解析出客户端监听的配置文件列表
	watchConfigFileRequest := &apiconfig.ClientWatchConfigFileRequest{}
	if _, err := handler.Parse(watchConfigFileRequest); err != nil {
		handler.WriteHeaderAndProto(api.NewResponseWithMsg(apimodel.Code_ParseException, err.Error()))
		return
	}

	// 阻塞等待响应
	callback, err := h.configServer.LongPullWatchFile(handler.ParseHeaderContext(), watchConfigFileRequest)
	if err != nil {
		handler.WriteHeaderAndProto(api.NewResponseWithMsg(apimodel.Code_ExecuteException, err.Error()))
		return
	}
	handler.WriteHeaderAndProto(callback())
}

// GetConfigFileMetadataList 统一发现接口
func (h *HTTPServer) GetConfigFileMetadataList(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	in := &apiconfig.ConfigFileGroupRequest{}
	ctx, err := handler.Parse(in)
	if err != nil {
		handler.WriteHeaderAndProto(api.NewResponseWithMsg(apimodel.Code_ParseException, err.Error()))
		return
	}

	startTime := commontime.CurrentMillisecond()
	defer func() {
		plugin.GetStatis().ReportDiscoverCall(metrics.ClientDiscoverMetric{
			ClientIP:  utils.ParseClientAddress(ctx),
			Namespace: in.GetConfigFileGroup().GetNamespace().GetValue(),
			Resource: fmt.Sprintf("CONFIG_FILE_LIST:%s|%s", in.GetConfigFileGroup().GetName().GetValue(),
				in.GetRevision().GetValue()),
			Timestamp: startTime,
			CostTime:  commontime.CurrentMillisecond() - startTime,
		})
	}()

	out := h.configServer.GetConfigFileNamesWithCache(ctx, in)
	handler.WriteHeaderAndProto(out)
}
