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
)

// PublishConfigFile 发布配置文件
func (s *ServerAuthability) PublishConfigFile(ctx context.Context,
	configFileRelease *apiconfig.ConfigFileRelease) *apiconfig.ConfigResponse {

	authCtx := s.collectConfigFileReleaseAuthContext(ctx,
		[]*apiconfig.ConfigFileRelease{configFileRelease}, auth.Modify, "PublishConfigFile")

	if _, err := s.policyMgr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewConfigResponseWithInfo(auth.ConvertToErrCode(err), err.Error())
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return s.nextServer.PublishConfigFile(ctx, configFileRelease)
}

// GetConfigFileRelease 获取配置文件发布内容
func (s *ServerAuthability) GetConfigFileRelease(ctx context.Context,
	req *apiconfig.ConfigFileRelease) *apiconfig.ConfigResponse {

	authCtx := s.collectConfigFileReleaseAuthContext(ctx,
		[]*apiconfig.ConfigFileRelease{req}, auth.Read, "GetConfigFileRelease")

	if _, err := s.policyMgr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewConfigResponseWithInfo(auth.ConvertToErrCode(err), err.Error())
	}
	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)
	return s.nextServer.GetConfigFileRelease(ctx, req)
}

// DeleteConfigFileReleases implements ConfigCenterServer.
func (s *ServerAuthability) DeleteConfigFileReleases(ctx context.Context,
	reqs []*apiconfig.ConfigFileRelease) *apiconfig.ConfigBatchWriteResponse {

	authCtx := s.collectConfigFileReleaseAuthContext(ctx, reqs, auth.Delete, "DeleteConfigFileReleases")

	if _, err := s.policyMgr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewConfigBatchWriteResponseWithInfo(auth.ConvertToErrCode(err), err.Error())
	}
	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)
	return s.nextServer.DeleteConfigFileReleases(ctx, reqs)
}

// DeleteConfigFileRelease implements ConfigCenterServer.
func (s *ServerAuthability) DeleteConfigFileRelease(ctx context.Context, req *apiconfig.ConfigFileRelease) *apiconfig.ConfigResponse {
	authCtx := s.collectConfigFileReleaseAuthContext(ctx, []*apiconfig.ConfigFileRelease{
		req,
	}, auth.Delete, "DeleteConfigFileRelease")

	if _, err := s.policyMgr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewConfigResponseWithInfo(auth.ConvertToErrCode(err), err.Error())
	}
	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)
	return s.nextServer.DeleteConfigFileRelease(ctx, req)
}

// GetConfigFileReleaseVersions implements ConfigCenterServer.
func (s *ServerAuthability) GetConfigFileReleaseVersions(ctx context.Context,
	filters map[string]string) *apiconfig.ConfigBatchQueryResponse {

	authCtx := s.collectConfigFileReleaseAuthContext(ctx, nil, auth.Read, "GetConfigFileReleaseVersions")

	if _, err := s.policyMgr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewConfigBatchQueryResponseWithInfo(auth.ConvertToErrCode(err), err.Error())
	}
	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)
	return s.nextServer.GetConfigFileReleaseVersions(ctx, filters)
}

// GetConfigFileReleases implements ConfigCenterServer.
func (s *ServerAuthability) GetConfigFileReleases(ctx context.Context,
	filters map[string]string) *apiconfig.ConfigBatchQueryResponse {

	authCtx := s.collectConfigFileReleaseAuthContext(ctx, nil, auth.Read, "GetConfigFileReleases")

	if _, err := s.policyMgr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewConfigBatchQueryResponseWithInfo(auth.ConvertToErrCode(err), err.Error())
	}
	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)
	return s.nextServer.GetConfigFileReleases(ctx, filters)
}

// RollbackConfigFileReleases implements ConfigCenterServer.
func (s *ServerAuthability) RollbackConfigFileReleases(ctx context.Context,
	reqs []*apiconfig.ConfigFileRelease) *apiconfig.ConfigBatchWriteResponse {

	authCtx := s.collectConfigFileReleaseAuthContext(ctx, reqs, auth.Modify, "RollbackConfigFileReleases")

	if _, err := s.policyMgr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewConfigBatchWriteResponseWithInfo(auth.ConvertToErrCode(err), err.Error())
	}
	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)
	return s.nextServer.RollbackConfigFileReleases(ctx, reqs)
}

func (s *ServerAuthability) RollbackConfigFileRelease(ctx context.Context,
	req *apiconfig.ConfigFileRelease) *apiconfig.ConfigResponse {

	authCtx := s.collectConfigFileReleaseAuthContext(ctx, []*apiconfig.ConfigFileRelease{
		req,
	}, auth.Modify, "RollbackConfigFileRelease")

	if _, err := s.policyMgr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewConfigResponseWithInfo(auth.ConvertToErrCode(err), err.Error())
	}
	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)
	return s.nextServer.RollbackConfigFileRelease(ctx, req)
}

// UpsertAndReleaseConfigFile .
func (s *ServerAuthability) UpsertAndReleaseConfigFile(ctx context.Context,
	req *apiconfig.ConfigFilePublishInfo) *apiconfig.ConfigResponse {
	authCtx := s.collectConfigFilePublishAuthContext(ctx, []*apiconfig.ConfigFilePublishInfo{req},
		auth.Modify, "UpsertAndReleaseConfigFile")
	if _, err := s.policyMgr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewConfigFileResponse(auth.ConvertToErrCode(err), nil)
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return s.nextServer.UpsertAndReleaseConfigFile(ctx, req)
}

func (s *ServerAuthability) StopGrayConfigFileReleases(ctx context.Context,
	reqs []*apiconfig.ConfigFileRelease) *apiconfig.ConfigBatchWriteResponse {

	authCtx := s.collectConfigFileReleaseAuthContext(ctx, reqs,
		auth.Modify, "StopGrayConfigFileReleases")
	if _, err := s.policyMgr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewConfigBatchWriteResponse(auth.ConvertToErrCode(err))
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return s.nextServer.StopGrayConfigFileReleases(ctx, reqs)
}
