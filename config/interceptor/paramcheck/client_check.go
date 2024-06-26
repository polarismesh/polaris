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

package paramcheck

import (
	"context"

	apiconfig "github.com/polarismesh/specification/source/go/api/v1/config_manage"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"

	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/config"
)

// UpsertAndReleaseConfigFileFromClient 创建/更新配置文件并发布
func (s *Server) UpsertAndReleaseConfigFileFromClient(ctx context.Context,
	req *apiconfig.ConfigFilePublishInfo) *apiconfig.ConfigResponse {
	if err := utils.CheckResourceName(req.GetNamespace()); err != nil {
		return api.NewConfigResponseWithInfo(apimodel.Code_BadRequest, "invalid config namespace")
	}
	if err := utils.CheckResourceName(req.GetGroup()); err != nil {
		return api.NewConfigResponseWithInfo(apimodel.Code_BadRequest, "invalid config group")
	}
	if err := CheckFileName(req.GetFileName()); err != nil {
		return api.NewConfigResponseWithInfo(apimodel.Code_BadRequest, "invalid config file_name")
	}
	return s.nextServer.UpsertAndReleaseConfigFileFromClient(ctx, req)
}

// CreateConfigFileFromClient 调用config_file的方法创建配置文件
func (s *Server) CreateConfigFileFromClient(ctx context.Context,
	req *apiconfig.ConfigFile) *apiconfig.ConfigClientResponse {
	if checkRsp := s.checkConfigFileParams(req); checkRsp != nil {
		return api.NewConfigClientResponseFromConfigResponse(checkRsp)
	}
	return s.nextServer.CreateConfigFileFromClient(ctx, req)
}

// UpdateConfigFileFromClient 调用config_file的方法更新配置文件
func (s *Server) UpdateConfigFileFromClient(ctx context.Context,
	req *apiconfig.ConfigFile) *apiconfig.ConfigClientResponse {
	if checkRsp := s.checkConfigFileParams(req); checkRsp != nil {
		return api.NewConfigClientResponseFromConfigResponse(checkRsp)
	}
	return s.nextServer.UpdateConfigFileFromClient(ctx, req)
}

// DeleteConfigFileFromClient 删除配置文件，删除配置文件同时会通知客户端 Not_Found
func (s *Server) DeleteConfigFileFromClient(ctx context.Context,
	req *apiconfig.ConfigFile) *apiconfig.ConfigResponse {

	if req.GetNamespace().GetValue() == "" {
		return api.NewConfigResponseWithInfo(
			apimodel.Code_BadRequest, "namespace is empty")
	}

	if req.GetGroup().GetValue() == "" {
		return api.NewConfigResponseWithInfo(
			apimodel.Code_BadRequest, "file group is empty")
	}

	if req.GetName().GetValue() == "" {
		return api.NewConfigResponseWithInfo(
			apimodel.Code_BadRequest, "filename is empty")
	}

	return s.nextServer.DeleteConfigFileFromClient(ctx, req)
}

// PublishConfigFileFromClient 调用config_file_release的方法发布配置文件
func (s *Server) PublishConfigFileFromClient(ctx context.Context,
	req *apiconfig.ConfigFileRelease) *apiconfig.ConfigClientResponse {

	if err := CheckFileName(req.GetFileName()); err != nil {
		ret := api.NewConfigResponse(apimodel.Code_InvalidConfigFileName)
		return api.NewConfigClientResponseFromConfigResponse(ret)
	}
	if err := utils.CheckResourceName(req.GetNamespace()); err != nil {
		ret := api.NewConfigResponse(apimodel.Code_InvalidNamespaceName)
		return api.NewConfigClientResponseFromConfigResponse(ret)
	}
	if err := utils.CheckResourceName(req.GetGroup()); err != nil {
		ret := api.NewConfigResponse(apimodel.Code_InvalidConfigFileGroupName)
		return api.NewConfigClientResponseFromConfigResponse(ret)
	}
	if !s.checkNamespaceExisted(req.GetNamespace().GetValue()) {
		ret := api.NewConfigResponse(apimodel.Code_NotFoundNamespace)
		return api.NewConfigClientResponseFromConfigResponse(ret)
	}
	if req.GetReleaseType().GetValue() == model.ReleaseTypeGray && len(req.GetBetaLabels()) == 0 {
		ret := api.NewConfigResponse(apimodel.Code_InvalidMatchRule)
		return api.NewConfigClientResponseFromConfigResponse(ret)
	}

	return s.nextServer.PublishConfigFileFromClient(ctx, req)
}

// GetConfigFileWithCache 从缓存中获取配置文件，如果客户端的版本号大于服务端，则服务端重新加载缓存
func (s *Server) GetConfigFileWithCache(ctx context.Context,
	req *apiconfig.ClientConfigFileInfo) *apiconfig.ConfigClientResponse {

	if req.GetNamespace().GetValue() == "" {
		return api.NewConfigClientResponseWithInfo(
			apimodel.Code_BadRequest, "namespace is empty")
	}

	if req.GetGroup().GetValue() == "" {
		return api.NewConfigClientResponseWithInfo(
			apimodel.Code_BadRequest, "file group is empty")
	}

	if req.GetFileName().GetValue() == "" {
		return api.NewConfigClientResponseWithInfo(
			apimodel.Code_BadRequest, "filename is empty")
	}

	return s.nextServer.GetConfigFileWithCache(ctx, req)
}

// WatchConfigFiles 监听配置文件变化
func (s *Server) LongPullWatchFile(ctx context.Context,
	request *apiconfig.ClientWatchConfigFileRequest) (config.WatchCallback, error) {

	watchFiles := request.WatchFiles
	if len(watchFiles) == 0 {
		return func() *apiconfig.ConfigClientResponse {
			return api.NewConfigClientResponse0(apimodel.Code_InvalidWatchConfigFileFormat)
		}, nil
	}

	for _, configFile := range watchFiles {
		namespace := configFile.GetNamespace().GetValue()
		group := configFile.GetGroup().GetValue()
		fileName := configFile.GetFileName().GetValue()
		if namespace == "" {
			return func() *apiconfig.ConfigClientResponse {
				return api.NewConfigClientResponseWithInfo(
					apimodel.Code_BadRequest, "namespace is empty")
			}, nil
		}
		if group == "" {
			return func() *apiconfig.ConfigClientResponse {
				return api.NewConfigClientResponseWithInfo(
					apimodel.Code_BadRequest, "file group is empty")
			}, nil
		}
		if fileName == "" {
			return func() *apiconfig.ConfigClientResponse {
				return api.NewConfigClientResponseWithInfo(
					apimodel.Code_BadRequest, "filename is empty")
			}, nil
		}
	}

	return s.nextServer.LongPullWatchFile(ctx, request)
}

// GetConfigFileNamesWithCache 获取某个配置分组下的配置文件
func (s *Server) GetConfigFileNamesWithCache(ctx context.Context,
	req *apiconfig.ConfigFileGroupRequest) *apiconfig.ConfigClientListResponse {

	if req.GetConfigFileGroup().GetNamespace().GetValue() == "" {
		return api.NewConfigClientListResponseWithInfo(
			apimodel.Code_BadRequest, "namespace is empty")
	}

	if req.GetConfigFileGroup().GetName().GetValue() == "" {
		return api.NewConfigClientListResponseWithInfo(
			apimodel.Code_BadRequest, "file group is empty")
	}

	return s.nextServer.GetConfigFileNamesWithCache(ctx, req)
}

func (s *Server) GetConfigGroupsWithCache(ctx context.Context,
	req *apiconfig.ClientConfigFileInfo) *apiconfig.ConfigDiscoverResponse {

	namespace := req.GetNamespace().GetValue()
	out := api.NewConfigDiscoverResponse(apimodel.Code_ExecuteSuccess)
	if namespace == "" {
		out.Code = uint32(apimodel.Code_BadRequest)
		out.Info = "invalid namespace"
		return out
	}

	return s.nextServer.GetConfigGroupsWithCache(ctx, req)
}

// CasUpsertAndReleaseConfigFileFromClient 创建/更新配置文件并发布
func (s *Server) CasUpsertAndReleaseConfigFileFromClient(ctx context.Context,
	req *apiconfig.ConfigFilePublishInfo) *apiconfig.ConfigResponse {
	if err := CheckFileName(req.GetFileName()); err != nil {
		return api.NewConfigResponse(apimodel.Code_InvalidConfigFileName)
	}
	if err := utils.CheckResourceName(req.GetNamespace()); err != nil {
		return api.NewConfigResponse(apimodel.Code_InvalidNamespaceName)
	}
	if err := utils.CheckResourceName(req.GetGroup()); err != nil {
		return api.NewConfigResponse(apimodel.Code_InvalidConfigFileGroupName)
	}
	return s.nextServer.CasUpsertAndReleaseConfigFileFromClient(ctx, req)
}
