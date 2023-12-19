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
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
)

// CreateConfigFile 创建配置文件
func (s *ServerAuthability) CreateConfigFile(ctx context.Context,
	configFile *apiconfig.ConfigFile) *apiconfig.ConfigResponse {
	authCtx := s.collectConfigFileAuthContext(
		ctx, []*apiconfig.ConfigFile{configFile}, model.Create, "CreateConfigFile")
	if _, err := s.strategyMgn.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewConfigResponseWithInfo(model.ConvertToErrCode(err), err.Error())
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return s.targetServer.CreateConfigFile(ctx, configFile)
}

// GetConfigFileRichInfo 获取单个配置文件基础信息，包含发布状态等信息
func (s *ServerAuthability) GetConfigFileRichInfo(ctx context.Context,
	req *apiconfig.ConfigFile) *apiconfig.ConfigResponse {

	authCtx := s.collectConfigFileAuthContext(
		ctx, []*apiconfig.ConfigFile{req}, model.Read, "GetConfigFileRichInfo")
	if _, err := s.strategyMgn.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewConfigResponseWithInfo(model.ConvertToErrCode(err), err.Error())
	}
	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)
	return s.targetServer.GetConfigFileRichInfo(ctx, req)
}

// SearchConfigFile 查询配置文件
func (s *ServerAuthability) SearchConfigFile(ctx context.Context,
	filter map[string]string) *apiconfig.ConfigBatchQueryResponse {

	authCtx := s.collectConfigFileAuthContext(ctx, nil, model.Read, "SearchConfigFile")
	if _, err := s.strategyMgn.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewConfigFileBatchQueryResponseWithMessage(model.ConvertToErrCode(err), err.Error())
	}
	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return s.targetServer.SearchConfigFile(ctx, filter)
}

// UpdateConfigFile 更新配置文件
func (s *ServerAuthability) UpdateConfigFile(
	ctx context.Context, configFile *apiconfig.ConfigFile) *apiconfig.ConfigResponse {
	authCtx := s.collectConfigFileAuthContext(
		ctx, []*apiconfig.ConfigFile{configFile}, model.Modify, "UpdateConfigFile")
	if _, err := s.strategyMgn.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewConfigResponseWithInfo(model.ConvertToErrCode(err), err.Error())
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return s.targetServer.UpdateConfigFile(ctx, configFile)
}

// DeleteConfigFile 删除配置文件，删除配置文件同时会通知客户端 Not_Found
func (s *ServerAuthability) DeleteConfigFile(ctx context.Context,
	req *apiconfig.ConfigFile) *apiconfig.ConfigResponse {

	authCtx := s.collectConfigFileAuthContext(ctx,
		[]*apiconfig.ConfigFile{req}, model.Delete, "DeleteConfigFile")
	if _, err := s.strategyMgn.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewConfigResponseWithInfo(model.ConvertToErrCode(err), err.Error())
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return s.targetServer.DeleteConfigFile(ctx, req)
}

// BatchDeleteConfigFile 批量删除配置文件
func (s *ServerAuthability) BatchDeleteConfigFile(ctx context.Context,
	req []*apiconfig.ConfigFile) *apiconfig.ConfigResponse {

	authCtx := s.collectConfigFileAuthContext(ctx, req, model.Delete, "BatchDeleteConfigFile")
	if _, err := s.strategyMgn.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewConfigResponseWithInfo(model.ConvertToErrCode(err), err.Error())
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return s.targetServer.BatchDeleteConfigFile(ctx, req)
}

func (s *ServerAuthability) ExportConfigFile(ctx context.Context,
	configFileExport *apiconfig.ConfigFileExportRequest) *apiconfig.ConfigExportResponse {
	var configFiles []*apiconfig.ConfigFile
	for _, group := range configFileExport.Groups {
		configFile := &apiconfig.ConfigFile{
			Namespace: configFileExport.Namespace,
			Group:     group,
		}
		configFiles = append(configFiles, configFile)
	}
	authCtx := s.collectConfigFileAuthContext(ctx, configFiles, model.Read, "ExportConfigFile")
	if _, err := s.strategyMgn.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewConfigFileExportResponseWithMessage(model.ConvertToErrCode(err), err.Error())
	}
	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return s.targetServer.ExportConfigFile(ctx, configFileExport)
}

func (s *ServerAuthability) ImportConfigFile(ctx context.Context,
	configFiles []*apiconfig.ConfigFile, conflictHandling string) *apiconfig.ConfigImportResponse {
	authCtx := s.collectConfigFileAuthContext(ctx, configFiles, model.Create, "ImportConfigFile")
	if _, err := s.strategyMgn.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewConfigFileImportResponseWithMessage(model.ConvertToErrCode(err), err.Error())
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)
	return s.targetServer.ImportConfigFile(ctx, configFiles, conflictHandling)
}

func (s *ServerAuthability) GetAllConfigEncryptAlgorithms(
	ctx context.Context) *apiconfig.ConfigEncryptAlgorithmResponse {
	return s.targetServer.GetAllConfigEncryptAlgorithms(ctx)
}
