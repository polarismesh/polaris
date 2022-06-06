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
	"github.com/golang/protobuf/proto"
	"go.uber.org/zap"

	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/utils"
)

// CreateConfigFileGroup 创建配置文件组
func (h *HTTPServer) CreateConfigFileGroup(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	configFileGroup := &api.ConfigFileGroup{}
	ctx, err := handler.Parse(configFileGroup)
	requestId := ctx.Value(utils.StringContext("request-id"))

	if err != nil {
		configLog.Error("[Config][HttpServer] parse config file group from request error.",
			zap.String("requestId", requestId.(string)),
			zap.String("error", err.Error()))
		handler.WriteHeaderAndProto(api.NewConfigFileGroupResponseWithMessage(api.ParseException, err.Error()))
		return
	}

	handler.WriteHeaderAndProto(h.configServer.CreateConfigFileGroup(ctx, configFileGroup))
}

// QueryConfigFileGroups 查询配置文件组，group 模糊搜索
func (h *HTTPServer) QueryConfigFileGroups(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	namespace := handler.QueryParameter("namespace")
	group := handler.QueryParameter("group")
	fileName := handler.QueryParameter("fileName")
	offset, _ := strconv.ParseUint(handler.QueryParameter("offset"), 10, 64)
	limit, _ := strconv.ParseUint(handler.QueryParameter("limit"), 10, 64)

	response := h.configServer.QueryConfigFileGroups(handler.ParseHeaderContext(), namespace, group, fileName,
		uint32(offset), uint32(limit))

	handler.WriteHeaderAndProto(response)
}

// DeleteConfigFileGroup 删除配置文件组
func (h *HTTPServer) DeleteConfigFileGroup(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	namespace := handler.QueryParameter("namespace")
	group := handler.QueryParameter("group")

	response := h.configServer.DeleteConfigFileGroup(handler.ParseHeaderContext(), namespace, group)
	handler.WriteHeaderAndProto(response)
}

// UpdateConfigFileGroup 更新配置文件组，只能更新 comment
func (h *HTTPServer) UpdateConfigFileGroup(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	configFileGroup := &api.ConfigFileGroup{}
	ctx, err := handler.Parse(configFileGroup)
	requestId := ctx.Value(utils.StringContext("request-id"))

	if err != nil {
		configLog.Error("[Config][HttpServer] parse config file group from request error.",
			zap.String("requestId", requestId.(string)),
			zap.String("error", err.Error()))
		handler.WriteHeaderAndProto(api.NewConfigFileGroupResponseWithMessage(api.ParseException, err.Error()))
		return
	}

	handler.WriteHeaderAndProto(h.configServer.UpdateConfigFileGroup(ctx, configFileGroup))
}

// CreateConfigFile 创建配置文件
func (h *HTTPServer) CreateConfigFile(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	configFile := &api.ConfigFile{}
	ctx, err := handler.Parse(configFile)
	requestId := ctx.Value(utils.StringContext("request-id"))

	if err != nil {
		configLog.Error("[Config][HttpServer] parse config file from request error.",
			zap.String("requestId", requestId.(string)),
			zap.String("error", err.Error()))
		handler.WriteHeaderAndProto(api.NewConfigFileResponseWithMessage(api.ParseException, err.Error()))
		return
	}

	handler.WriteHeaderAndProto(h.configServer.CreateConfigFile(ctx, configFile))
}

// GetConfigFile 获取单个配置文件
func (h *HTTPServer) GetConfigFile(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	namespace := handler.QueryParameter("namespace")
	group := handler.QueryParameter("group")
	name := handler.QueryParameter("name")

	response := h.configServer.GetConfigFileRichInfo(handler.ParseHeaderContext(), namespace, group, name)
	handler.WriteHeaderAndProto(response)
}

// SearchConfigFile 按照 group 和 name 模糊搜索配置文件，按照 tag 搜索，多个tag之间或的关系
func (h *HTTPServer) SearchConfigFile(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	namespace := handler.QueryParameter("namespace")
	group := handler.QueryParameter("group")
	name := handler.QueryParameter("name")
	tags := handler.QueryParameter("tags")
	offset, _ := strconv.ParseUint(handler.QueryParameter("offset"), 10, 64)
	limit, _ := strconv.ParseUint(handler.QueryParameter("limit"), 10, 64)

	response := h.configServer.SearchConfigFile(handler.ParseHeaderContext(), namespace, group, name, tags,
		uint32(offset), uint32(limit))

	handler.WriteHeaderAndProto(response)
}

// UpdateConfigFile 更新配置文件
func (h *HTTPServer) UpdateConfigFile(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	configFile := &api.ConfigFile{}
	ctx, err := handler.Parse(configFile)
	requestId := ctx.Value(utils.StringContext("request-id"))
	if err != nil {
		configLog.Error("[Config][HttpServer] parse config file from request error.",
			zap.String("requestId", requestId.(string)),
			zap.String("error", err.Error()))
		handler.WriteHeaderAndProto(api.NewConfigFileResponseWithMessage(api.ParseException, err.Error()))
		return
	}

	handler.WriteHeaderAndProto(h.configServer.UpdateConfigFile(ctx, configFile))
}

// DeleteConfigFile 删除单个配置文件，删除配置文件也会删除配置文件发布内容，客户端将获取不到配置文件
func (h *HTTPServer) DeleteConfigFile(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	namespace := handler.QueryParameter("namespace")
	group := handler.QueryParameter("group")
	name := handler.QueryParameter("name")
	operator := handler.QueryParameter("deleteBy")

	response := h.configServer.DeleteConfigFile(handler.ParseHeaderContext(), namespace, group, name, operator)
	handler.WriteHeaderAndProto(response)
}

// BatchDeleteConfigFile 批量删除配置文件
func (h *HTTPServer) BatchDeleteConfigFile(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	operator := handler.QueryParameter("deleteBy")

	var configFiles ConfigFileArr
	ctx, err := handler.ParseArray(func() proto.Message {
		msg := &api.ConfigFile{}
		configFiles = append(configFiles, msg)
		return msg
	})
	if err != nil {
		handler.WriteHeaderAndProto(api.NewBatchWriteResponseWithMsg(api.ParseException, err.Error()))
		return
	}

	response := h.configServer.BatchDeleteConfigFile(ctx, configFiles, operator)
	handler.WriteHeaderAndProto(response)
}

// PublishConfigFile 发布配置文件
func (h *HTTPServer) PublishConfigFile(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	configFile := &api.ConfigFileRelease{}
	ctx, err := handler.Parse(configFile)
	requestId := ctx.Value(utils.StringContext("request-id"))

	if err != nil {
		configLog.Error("[Config][HttpServer] parse config file release from request error.",
			zap.String("requestId", requestId.(string)),
			zap.String("error", err.Error()))
		handler.WriteHeaderAndProto(api.NewConfigFileReleaseResponseWithMessage(api.ParseException, err.Error()))
		return
	}

	handler.WriteHeaderAndProto(h.configServer.PublishConfigFile(ctx, configFile))
}

// GetConfigFileRelease 获取配置文件最后一次发布内容
func (h *HTTPServer) GetConfigFileRelease(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	namespace := handler.QueryParameter("namespace")
	group := handler.QueryParameter("group")
	name := handler.QueryParameter("name")

	response := h.configServer.GetConfigFileRelease(handler.ParseHeaderContext(), namespace, group, name)

	handler.WriteHeaderAndProto(response)
}

// GetConfigFileReleaseHistory 获取配置文件发布历史，按照发布时间倒序排序
func (h *HTTPServer) GetConfigFileReleaseHistory(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	namespace := handler.QueryParameter("namespace")
	group := handler.QueryParameter("group")
	name := handler.QueryParameter("name")
	endIdStr := handler.QueryParameter("endId")
	offset, _ := strconv.ParseUint(handler.QueryParameter("offset"), 10, 64)
	limit, _ := strconv.ParseUint(handler.QueryParameter("limit"), 10, 64)
	var endId uint64
	if endIdStr == "" {
		endId = 0
	} else {
		endId, _ = strconv.ParseUint(endIdStr, 10, 64)
	}

	response := h.configServer.GetConfigFileReleaseHistory(handler.ParseHeaderContext(),
		namespace, group, name, uint32(offset), uint32(limit), endId)

	handler.WriteHeaderAndProto(response)
}
