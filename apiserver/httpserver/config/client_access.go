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
	"strconv"
	"strings"

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
		Tags: func() []*apiconfig.ConfigFileTag {
			tags := handler.Request.QueryParameters("tags")
			ret := make([]*apiconfig.ConfigFileTag, 0, len(tags))
			for i := range tags {
				kv := strings.Split(tags[i], "=")
				ret = append(ret, &apiconfig.ConfigFileTag{
					Key:   &wrapperspb.StringValue{Value: strings.TrimSpace(kv[0])},
					Value: &wrapperspb.StringValue{Value: strings.TrimSpace(kv[1])},
				})
			}
			return ret
		}(),
	}

	ctx := handler.ParseHeaderContext()
	startTime := commontime.CurrentMillisecond()
	var ret *apiconfig.ConfigClientResponse
	defer func() {
		plugin.GetStatis().ReportDiscoverCall(metrics.ClientDiscoverMetric{
			Action:    metrics.ActionGetConfigFile,
			ClientIP:  utils.ParseClientAddress(ctx),
			Namespace: configFile.GetNamespace().GetValue(),
			Resource:  metrics.ResourceOfConfigFile(configFile.GetGroup().GetValue(), configFile.GetFileName().GetValue()),
			Timestamp: startTime,
			CostTime:  commontime.CurrentMillisecond() - startTime,
			Revision:  strconv.FormatUint(ret.GetConfigFile().GetVersion().GetValue(), 10),
			Success:   ret.GetCode().GetValue() > uint32(apimodel.Code_DataNoChange),
		})
	}()

	ret = h.configServer.GetConfigFileWithCache(ctx, configFile)
	handler.WriteHeaderAndProto(ret)
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

	var out *apiconfig.ConfigClientListResponse
	startTime := commontime.CurrentMillisecond()
	defer func() {
		plugin.GetStatis().ReportDiscoverCall(metrics.ClientDiscoverMetric{
			Action:    metrics.ActionListConfigFiles,
			ClientIP:  utils.ParseClientAddress(ctx),
			Namespace: in.GetConfigFileGroup().GetNamespace().GetValue(),
			Resource:  metrics.ResourceOfConfigFileList(in.GetConfigFileGroup().GetName().GetValue()),
			Timestamp: startTime,
			CostTime:  commontime.CurrentMillisecond() - startTime,
			Revision:  out.GetRevision().GetValue(),
			Success:   out.GetCode().GetValue() > uint32(apimodel.Code_DataNoChange),
		})
	}()

	out = h.configServer.GetConfigFileNamesWithCache(ctx, in)
	handler.WriteHeaderAndProto(out)
}

// Discover 统一发现接口
func (h *HTTPServer) Discover(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	in := &apiconfig.ConfigDiscoverRequest{}
	ctx, err := handler.Parse(in)
	if err != nil {
		handler.WriteHeaderAndProto(api.NewResponseWithMsg(apimodel.Code_ParseException, err.Error()))
		return
	}

	var out *apiconfig.ConfigDiscoverResponse
	var action string
	startTime := commontime.CurrentMillisecond()
	defer func() {
		plugin.GetStatis().ReportDiscoverCall(metrics.ClientDiscoverMetric{
			Action:    action,
			ClientIP:  utils.ParseClientAddress(ctx),
			Namespace: in.GetConfigFile().GetNamespace().GetValue(),
			Resource:  metrics.ResourceOfConfigFile(in.GetConfigFile().GetGroup().GetValue(), in.GetConfigFile().GetFileName().GetValue()),
			Timestamp: startTime,
			CostTime:  commontime.CurrentMillisecond() - startTime,
			Revision:  out.GetRevision(),
			Success:   out.GetCode() > uint32(apimodel.Code_DataNoChange),
		})
	}()

	switch in.Type {
	case apiconfig.ConfigDiscoverRequest_CONFIG_FILE:
		action = metrics.ActionGetConfigFile
		ret := h.configServer.GetConfigFileWithCache(ctx, &apiconfig.ClientConfigFileInfo{})
		out = api.NewConfigDiscoverResponse(apimodel.Code(ret.GetCode().GetValue()))
		out.ConfigFile = ret.GetConfigFile()
		out.Type = apiconfig.ConfigDiscoverResponse_CONFIG_FILE
		out.Revision = strconv.Itoa(int(out.GetConfigFile().GetVersion().GetValue()))
	case apiconfig.ConfigDiscoverRequest_CONFIG_FILE_Names:
		action = metrics.ActionListConfigFiles
		ret := h.configServer.GetConfigFileNamesWithCache(ctx, &apiconfig.ConfigFileGroupRequest{
			Revision: wrapperspb.String(in.GetRevision()),
			ConfigFileGroup: &apiconfig.ConfigFileGroup{
				Namespace: in.GetConfigFile().GetNamespace(),
				Name:      in.GetConfigFile().GetGroup(),
			},
		})
		out = api.NewConfigDiscoverResponse(apimodel.Code(ret.GetCode().GetValue()))
		out.ConfigFileNames = ret.GetConfigFileInfos()
		out.Type = apiconfig.ConfigDiscoverResponse_CONFIG_FILE_Names
		out.Revision = ret.GetRevision().GetValue()
	case apiconfig.ConfigDiscoverRequest_CONFIG_FILE_GROUPS:
		action = metrics.ActionListConfigGroups
		req := in.GetConfigFile()
		req.Md5 = wrapperspb.String(in.GetRevision())
		out = h.configServer.GetConfigGroupsWithCache(ctx, req)
	default:
		out = api.NewConfigDiscoverResponse(apimodel.Code_InvalidDiscoverResource)
	}

	handler.WriteHeaderAndProtoV2(out)
}
