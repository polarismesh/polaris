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

package config_auth

import (
	"context"

	apiconfig "github.com/polarismesh/specification/source/go/api/v1/config_manage"

	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/model/auth"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/config"
)

// UpsertAndReleaseConfigFileFromClient 创建/更新配置文件并发布
func (s *ServerAuthability) UpsertAndReleaseConfigFileFromClient(ctx context.Context,
	req *apiconfig.ConfigFilePublishInfo) *apiconfig.ConfigResponse {
	authCtx := s.collectConfigFilePublishAuthContext(ctx, []*apiconfig.ConfigFilePublishInfo{req},
		auth.Modify, auth.PublishConfigFile)
	if _, err := s.policyMgr.GetAuthChecker().CheckClientPermission(authCtx); err != nil {
		return api.NewConfigFileResponse(auth.ConvertToErrCode(err), nil)
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return s.nextServer.UpsertAndReleaseConfigFileFromClient(ctx, req)
}

// CreateConfigFileFromClient 调用config_file的方法创建配置文件
func (s *ServerAuthability) CreateConfigFileFromClient(ctx context.Context,
	fileInfo *apiconfig.ConfigFile) *apiconfig.ConfigClientResponse {
	authCtx := s.collectClientConfigFileAuthContext(ctx,
		[]*apiconfig.ConfigFile{{
			Namespace: fileInfo.Namespace,
			Name:      fileInfo.Name,
			Group:     fileInfo.Group},
		}, auth.Create, auth.CreateConfigFile)
	if _, err := s.policyMgr.GetAuthChecker().CheckClientPermission(authCtx); err != nil {
		return api.NewConfigClientResponseWithInfo(auth.ConvertToErrCode(err), err.Error())
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return s.nextServer.CreateConfigFileFromClient(ctx, fileInfo)
}

// UpdateConfigFileFromClient 调用config_file的方法更新配置文件
func (s *ServerAuthability) UpdateConfigFileFromClient(ctx context.Context,
	fileInfo *apiconfig.ConfigFile) *apiconfig.ConfigClientResponse {
	authCtx := s.collectClientConfigFileAuthContext(ctx,
		[]*apiconfig.ConfigFile{fileInfo}, auth.Modify, auth.UpdateConfigFile)
	if _, err := s.policyMgr.GetAuthChecker().CheckClientPermission(authCtx); err != nil {
		return api.NewConfigClientResponseWithInfo(auth.ConvertToErrCode(err), err.Error())
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return s.nextServer.UpdateConfigFileFromClient(ctx, fileInfo)
}

// DeleteConfigFileFromClient 删除配置文件，删除配置文件同时会通知客户端 Not_Found
func (s *ServerAuthability) DeleteConfigFileFromClient(ctx context.Context,
	req *apiconfig.ConfigFile) *apiconfig.ConfigResponse {

	authCtx := s.collectConfigFileAuthContext(ctx,
		[]*apiconfig.ConfigFile{req}, auth.Delete, auth.DeleteConfigFile)
	if _, err := s.policyMgr.GetAuthChecker().CheckClientPermission(authCtx); err != nil {
		return api.NewConfigResponseWithInfo(auth.ConvertToErrCode(err), err.Error())
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return s.nextServer.DeleteConfigFileFromClient(ctx, req)
}

// PublishConfigFileFromClient 调用config_file_release的方法发布配置文件
func (s *ServerAuthability) PublishConfigFileFromClient(ctx context.Context,
	fileInfo *apiconfig.ConfigFileRelease) *apiconfig.ConfigClientResponse {
	authCtx := s.collectClientConfigFileReleaseAuthContext(ctx,
		[]*apiconfig.ConfigFileRelease{{
			Namespace: fileInfo.Namespace,
			Name:      fileInfo.FileName,
			Group:     fileInfo.Group},
		}, auth.Create, auth.PublishConfigFile)
	if _, err := s.policyMgr.GetAuthChecker().CheckClientPermission(authCtx); err != nil {
		return api.NewConfigClientResponseWithInfo(auth.ConvertToErrCode(err), err.Error())
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return s.nextServer.PublishConfigFileFromClient(ctx, fileInfo)
}

// GetConfigFileWithCache 从缓存中获取配置文件，如果客户端的版本号大于服务端，则服务端重新加载缓存
func (s *ServerAuthability) GetConfigFileWithCache(ctx context.Context,
	fileInfo *apiconfig.ClientConfigFileInfo) *apiconfig.ConfigClientResponse {
	authCtx := s.collectClientConfigFileAuthContext(ctx,
		[]*apiconfig.ConfigFile{{
			Namespace: fileInfo.Namespace,
			Name:      fileInfo.FileName,
			Group:     fileInfo.Group},
		}, auth.Read, auth.DiscoverConfigFile)
	if _, err := s.policyMgr.GetAuthChecker().CheckClientPermission(authCtx); err != nil {
		return api.NewConfigClientResponseWithInfo(auth.ConvertToErrCode(err), err.Error())
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)
	return s.nextServer.GetConfigFileWithCache(ctx, fileInfo)
}

// WatchConfigFiles 监听配置文件变化
func (s *ServerAuthability) LongPullWatchFile(ctx context.Context,
	request *apiconfig.ClientWatchConfigFileRequest) (config.WatchCallback, error) {
	authCtx := s.collectClientWatchConfigFiles(ctx, request, auth.Read, auth.WatchConfigFile)
	if _, err := s.policyMgr.GetAuthChecker().CheckClientPermission(authCtx); err != nil {
		return func() *apiconfig.ConfigClientResponse {
			return api.NewConfigClientResponseWithInfo(auth.ConvertToErrCode(err), err.Error())
		}, nil
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return s.nextServer.LongPullWatchFile(ctx, request)
}

// GetConfigFileNamesWithCache 获取某个配置分组下的配置文件
func (s *ServerAuthability) GetConfigFileNamesWithCache(ctx context.Context,
	req *apiconfig.ConfigFileGroupRequest) *apiconfig.ConfigClientListResponse {

	authCtx := s.collectClientConfigFileReleaseAuthContext(ctx, []*apiconfig.ConfigFileRelease{
		{
			Namespace: req.GetConfigFileGroup().GetNamespace(),
			Group:     req.GetConfigFileGroup().GetName(),
		},
	}, auth.Read, auth.DiscoverConfigFileNames)
	if _, err := s.policyMgr.GetAuthChecker().CheckClientPermission(authCtx); err != nil {
		out := api.NewConfigClientListResponse(auth.ConvertToErrCode(err))
		return out
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)
	return s.nextServer.GetConfigFileNamesWithCache(ctx, req)
}

// GetConfigGroupsWithCache 获取某个命名空间下的配置分组列表
func (s *ServerAuthability) GetConfigGroupsWithCache(ctx context.Context,
	req *apiconfig.ClientConfigFileInfo) *apiconfig.ConfigDiscoverResponse {

	authCtx := s.collectClientConfigFileReleaseAuthContext(ctx, []*apiconfig.ConfigFileRelease{
		{
			Namespace: req.GetNamespace(),
		},
	}, auth.Read, auth.DiscoverConfigGroups)
	if _, err := s.policyMgr.GetAuthChecker().CheckClientPermission(authCtx); err != nil {
		out := api.NewConfigDiscoverResponse(auth.ConvertToErrCode(err))
		return out
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)
	return s.nextServer.GetConfigGroupsWithCache(ctx, req)
}

// CasUpsertAndReleaseConfigFileFromClient 创建/更新配置文件并发布
func (s *ServerAuthability) CasUpsertAndReleaseConfigFileFromClient(ctx context.Context,
	req *apiconfig.ConfigFilePublishInfo) *apiconfig.ConfigResponse {

	authCtx := s.collectConfigFilePublishAuthContext(ctx, []*apiconfig.ConfigFilePublishInfo{req},
		auth.Modify, auth.UpsertAndReleaseConfigFile)
	if _, err := s.policyMgr.GetAuthChecker().CheckClientPermission(authCtx); err != nil {
		return api.NewConfigFileResponse(auth.ConvertToErrCode(err), nil)
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return s.nextServer.CasUpsertAndReleaseConfigFileFromClient(ctx, req)
}
