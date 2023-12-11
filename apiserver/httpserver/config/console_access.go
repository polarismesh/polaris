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
	"net/http"
	"strings"

	"github.com/emicklei/go-restful/v3"
	"github.com/golang/protobuf/proto"
	apiconfig "github.com/polarismesh/specification/source/go/api/v1/config_manage"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	"go.uber.org/zap"

	httpcommon "github.com/polarismesh/polaris/apiserver/httpserver/utils"
	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/utils"
)

// CreateConfigFileGroup 创建配置文件组
func (h *HTTPServer) CreateConfigFileGroup(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	configFileGroup := &apiconfig.ConfigFileGroup{}
	ctx, err := handler.Parse(configFileGroup)
	requestId := ctx.Value(utils.StringContext("request-id"))

	if err != nil {
		configLog.Error("[Config][HttpServer] parse config file group from request error.",
			zap.String("requestId", requestId.(string)),
			zap.String("error", err.Error()))
		handler.WriteHeaderAndProto(api.NewConfigFileGroupResponseWithMessage(apimodel.Code_ParseException, err.Error()))
		return
	}

	handler.WriteHeaderAndProto(h.configServer.CreateConfigFileGroup(ctx, configFileGroup))
}

// QueryConfigFileGroups 查询配置文件组，group 模糊搜索
func (h *HTTPServer) QueryConfigFileGroups(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	filter := httpcommon.ParseQueryParams(req)
	response := h.configServer.QueryConfigFileGroups(handler.ParseHeaderContext(), filter)

	handler.WriteHeaderAndProto(response)
}

// DeleteConfigFileGroup 删除配置文件组
func (h *HTTPServer) DeleteConfigFileGroup(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	namespace := handler.Request.QueryParameter("namespace")
	group := handler.Request.QueryParameter("group")

	response := h.configServer.DeleteConfigFileGroup(handler.ParseHeaderContext(), namespace, group)
	handler.WriteHeaderAndProto(response)
}

// UpdateConfigFileGroup 更新配置文件组，只能更新 comment
func (h *HTTPServer) UpdateConfigFileGroup(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	configFileGroup := &apiconfig.ConfigFileGroup{}
	ctx, err := handler.Parse(configFileGroup)
	if err != nil {
		configLog.Error("[Config][HttpServer] parse config file group from request error.",
			utils.RequestID(ctx), zap.Error(err))
		handler.WriteHeaderAndProto(api.NewConfigResponseWithInfo(apimodel.Code_ParseException, err.Error()))
		return
	}

	handler.WriteHeaderAndProto(h.configServer.UpdateConfigFileGroup(ctx, configFileGroup))
}

// CreateConfigFile 创建配置文件
func (h *HTTPServer) CreateConfigFile(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	configFile := &apiconfig.ConfigFile{}
	ctx, err := handler.Parse(configFile)
	if err != nil {
		configLog.Error("[Config][HttpServer] parse config file from request error.",
			utils.RequestID(ctx), zap.Error(err))
		handler.WriteHeaderAndProto(api.NewConfigResponseWithInfo(apimodel.Code_ParseException, err.Error()))
		return
	}

	handler.WriteHeaderAndProto(h.configServer.CreateConfigFile(ctx, configFile))
}

// GetConfigFile 获取单个配置文件
func (h *HTTPServer) GetConfigFile(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	namespace := handler.Request.QueryParameter("namespace")
	group := handler.Request.QueryParameter("group")
	name := handler.Request.QueryParameter("name")

	fileReq := &apiconfig.ConfigFile{
		Namespace: utils.NewStringValue(namespace),
		Group:     utils.NewStringValue(group),
		Name:      utils.NewStringValue(name),
	}

	response := h.configServer.GetConfigFileRichInfo(handler.ParseHeaderContext(), fileReq)
	handler.WriteHeaderAndProto(response)
}

// SearchConfigFile 按照 group 和 name 模糊搜索配置文件，按照 tag 搜索，多个tag之间或的关系
func (h *HTTPServer) SearchConfigFile(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	filters := httpcommon.ParseQueryParams(req)
	response := h.configServer.SearchConfigFile(handler.ParseHeaderContext(), filters)

	handler.WriteHeaderAndProto(response)
}

// UpdateConfigFile 更新配置文件
func (h *HTTPServer) UpdateConfigFile(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	configFile := &apiconfig.ConfigFile{}
	ctx, err := handler.Parse(configFile)
	if err != nil {
		configLog.Error("[Config][HttpServer] parse config file from request error.",
			utils.RequestID(ctx), zap.Error(err))
		handler.WriteHeaderAndProto(api.NewConfigResponseWithInfo(apimodel.Code_ParseException, err.Error()))
		return
	}

	handler.WriteHeaderAndProto(h.configServer.UpdateConfigFile(ctx, configFile))
}

// DeleteConfigFile 删除单个配置文件，删除配置文件也会删除配置文件发布内容，客户端将获取不到配置文件
func (h *HTTPServer) DeleteConfigFile(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	namespace := handler.Request.QueryParameter("namespace")
	group := handler.Request.QueryParameter("group")
	name := handler.Request.QueryParameter("name")

	fileReq := &apiconfig.ConfigFile{
		Namespace: utils.NewStringValue(namespace),
		Group:     utils.NewStringValue(group),
		Name:      utils.NewStringValue(name),
	}

	response := h.configServer.DeleteConfigFile(handler.ParseHeaderContext(), fileReq)
	handler.WriteHeaderAndProto(response)
}

// BatchDeleteConfigFile 批量删除配置文件
func (h *HTTPServer) BatchDeleteConfigFile(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	var configFiles ConfigFileArr
	ctx, err := handler.ParseArray(func() proto.Message {
		msg := &apiconfig.ConfigFile{}
		configFiles = append(configFiles, msg)
		return msg
	})
	if err != nil {
		handler.WriteHeaderAndProto(api.NewBatchWriteResponseWithMsg(apimodel.Code_ParseException, err.Error()))
		return
	}

	response := h.configServer.BatchDeleteConfigFile(ctx, configFiles)
	handler.WriteHeaderAndProto(response)
}

// ExportConfigFile 导出配置文件
func (h *HTTPServer) ExportConfigFile(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	configFileExport := &apiconfig.ConfigFileExportRequest{}
	ctx, err := handler.Parse(configFileExport)
	if err != nil {
		handler.WriteHeaderAndProto(api.NewBatchWriteResponseWithMsg(apimodel.Code_ParseException, err.Error()))
		return
	}
	response := h.configServer.ExportConfigFile(ctx, configFileExport)
	if response.Code.Value != api.ExecuteSuccess {
		handler.WriteHeaderAndProto(response)
	} else {
		handler.WriteHeader(api.ExecuteSuccess, http.StatusOK)
		handler.Response.AddHeader("Content-Type", "application/zip")
		handler.Response.AddHeader("Content-Disposition", "attachment; filename=config.zip")
		if _, err := handler.Response.ResponseWriter.Write(response.Data.Value); err != nil {
			configLog.Error("[Config][HttpServer] response write error.",
				utils.RequestID(ctx),
				zap.String("error", err.Error()))
		}
	}
}

// ImportConfigFile 导入配置文件
func (h *HTTPServer) ImportConfigFile(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	ctx := handler.ParseHeaderContext()
	configFiles, err := handler.ParseFile()
	if err != nil {
		handler.WriteHeaderAndProto(api.NewResponseWithMsg(apimodel.Code_ParseException, err.Error()))
		return
	}
	namespace := handler.Request.QueryParameter("namespace")
	group := handler.Request.QueryParameter("group")
	conflictHandling := handler.Request.QueryParameter("conflict_handling")

	for _, file := range configFiles {
		file.Namespace = utils.NewStringValue(namespace)
		if group != "" {
			file.Group = utils.NewStringValue(group)
		}
	}

	var filenames []string
	for _, file := range configFiles {
		filenames = append(filenames, file.String())
	}
	configLog.Info("[Config][HttpServer]import config file",
		zap.String("namespace", namespace),
		zap.String("group", group),
		zap.String("conflict_handling", conflictHandling),
		zap.String("files", strings.Join(filenames, ",")),
	)

	response := h.configServer.ImportConfigFile(ctx, configFiles, conflictHandling)
	handler.WriteHeaderAndProto(response)
}

// PublishConfigFile 发布配置文件
func (h *HTTPServer) PublishConfigFile(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	configFile := &apiconfig.ConfigFileRelease{}
	ctx, err := handler.Parse(configFile)
	if err != nil {
		configLog.Error("[Config][HttpServer] parse config file release from request error.",
			zap.String("error", err.Error()))
		handler.WriteHeaderAndProto(api.NewConfigFileReleaseResponseWithMessage(apimodel.Code_ParseException, err.Error()))
		return
	}
	handler.WriteHeaderAndProto(h.configServer.PublishConfigFile(ctx, configFile))
}

// RollbackConfigFileReleases 获取配置文件最后一次发布内容
func (h *HTTPServer) RollbackConfigFileReleases(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	var releases []*apiconfig.ConfigFileRelease
	ctx, err := handler.ParseArray(func() proto.Message {
		msg := &apiconfig.ConfigFileRelease{}
		releases = append(releases, msg)
		return msg
	})
	if err != nil {
		handler.WriteHeaderAndProto(api.NewBatchWriteResponseWithMsg(apimodel.Code_ParseException, err.Error()))
		return
	}
	response := h.configServer.RollbackConfigFileReleases(ctx, releases)

	handler.WriteHeaderAndProto(response)
}

// DeleteConfigFileReleases
func (h *HTTPServer) DeleteConfigFileReleases(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	var releases []*apiconfig.ConfigFileRelease
	ctx, err := handler.ParseArray(func() proto.Message {
		msg := &apiconfig.ConfigFileRelease{}
		releases = append(releases, msg)
		return msg
	})
	if err != nil {
		handler.WriteHeaderAndProto(api.NewBatchWriteResponseWithMsg(apimodel.Code_ParseException, err.Error()))
		return
	}

	response := h.configServer.DeleteConfigFileReleases(ctx, releases)
	handler.WriteHeaderAndProto(response)
}

// GetConfigFileReleaseVersions 获取配置文件最后一次发布内容
func (h *HTTPServer) GetConfigFileReleaseVersions(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	queryParams := httpcommon.ParseQueryParams(req)
	response := h.configServer.GetConfigFileReleaseVersions(handler.ParseHeaderContext(), queryParams)

	handler.WriteHeaderAndProto(response)
}

// GetConfigFileReleases 获取配置文件最后一次发布内容
func (h *HTTPServer) GetConfigFileReleases(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	queryParams := httpcommon.ParseQueryParams(req)
	response := h.configServer.GetConfigFileReleases(handler.ParseHeaderContext(), queryParams)

	handler.WriteHeaderAndProto(response)
}

// GetConfigFileRelease 获取配置文件最后一次发布内容
func (h *HTTPServer) GetConfigFileRelease(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	namespace := handler.Request.QueryParameter("namespace")
	group := handler.Request.QueryParameter("group")
	fileName := handler.Request.QueryParameter("file_name")
	name := handler.Request.QueryParameter("release_name")
	// 兼容旧的查询参数
	if fileName == "" {
		fileName = handler.Request.QueryParameter("name")
	}

	fileReq := &apiconfig.ConfigFileRelease{
		Namespace: utils.NewStringValue(namespace),
		Group:     utils.NewStringValue(group),
		FileName:  utils.NewStringValue(fileName),
		Name:      utils.NewStringValue(name),
	}

	response := h.configServer.GetConfigFileRelease(handler.ParseHeaderContext(), fileReq)
	handler.WriteHeaderAndProto(response)
}

// GetConfigFileReleaseHistory 获取配置文件发布历史，按照发布时间倒序排序
func (h *HTTPServer) GetConfigFileReleaseHistory(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	filters := httpcommon.ParseQueryParams(req)
	response := h.configServer.GetConfigFileReleaseHistories(handler.ParseHeaderContext(), filters)

	handler.WriteHeaderAndProto(response)
}

// GetAllConfigFileTemplates get all config file template
func (h *HTTPServer) GetAllConfigFileTemplates(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	response := h.configServer.GetAllConfigFileTemplates(handler.ParseHeaderContext())

	handler.WriteHeaderAndProto(response)
}

// CreateConfigFileTemplate create config file template
func (h *HTTPServer) CreateConfigFileTemplate(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	configFileTemplate := &apiconfig.ConfigFileTemplate{}
	ctx, err := handler.Parse(configFileTemplate)
	requestId := ctx.Value(utils.StringContext("request-id"))

	if err != nil {
		configLog.Error("[Config][HttpServer] parse config file template from request error.",
			zap.String("requestId", requestId.(string)),
			zap.String("error", err.Error()))
		handler.WriteHeaderAndProto(api.NewConfigFileTemplateResponseWithMessage(apimodel.Code_ParseException, err.Error()))
		return
	}

	handler.WriteHeaderAndProto(h.configServer.CreateConfigFileTemplate(ctx, configFileTemplate))
}

// GetAllConfigEncryptAlgorithm get all config encrypt algorithm
func (h *HTTPServer) GetAllConfigEncryptAlgorithms(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}
	response := h.configServer.GetAllConfigEncryptAlgorithms(handler.ParseHeaderContext())
	configLog.Info("response",
		zap.Uint32("code", response.GetCode().GetValue()),
	)
	handler.WriteHeaderAndProto(response)
}

// UpsertAndReleaseConfigFile
func (h *HTTPServer) UpsertAndReleaseConfigFile(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	configFile := &apiconfig.ConfigFilePublishInfo{}
	ctx, err := handler.Parse(configFile)
	if err != nil {
		configLog.Error("[Config][HttpServer] parse config file from request error.",
			utils.RequestID(ctx), zap.String("error", err.Error()))
		handler.WriteHeaderAndProto(api.NewConfigResponseWithInfo(apimodel.Code_ParseException, err.Error()))
		return
	}

	handler.WriteHeaderAndProto(h.configServer.UpsertAndReleaseConfigFile(ctx, configFile))
}

// StopGrayConfigFileReleases .
func (h *HTTPServer) StopGrayConfigFileReleases(req *restful.Request, rsp *restful.Response) {
	handler := &httpcommon.Handler{
		Request:  req,
		Response: rsp,
	}

	var releases []*apiconfig.ConfigFileRelease
	ctx, err := handler.ParseArray(func() proto.Message {
		msg := &apiconfig.ConfigFileRelease{}
		releases = append(releases, msg)
		return msg
	})
	if err != nil {
		handler.WriteHeaderAndProto(api.NewBatchWriteResponseWithMsg(apimodel.Code_ParseException, err.Error()))
		return
	}

	response := h.configServer.StopGrayConfigFileReleases(ctx, releases)
	handler.WriteHeaderAndProto(response)
}
