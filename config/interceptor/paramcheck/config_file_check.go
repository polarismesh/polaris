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
	"strconv"

	apiconfig "github.com/polarismesh/specification/source/go/api/v1/config_manage"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"

	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/utils"
)

// CreateConfigFile 创建配置文件
func (s *Server) CreateConfigFile(ctx context.Context,
	configFile *apiconfig.ConfigFile) *apiconfig.ConfigResponse {

	return s.nextServer.CreateConfigFile(ctx, configFile)
}

// GetConfigFileRichInfo 获取单个配置文件基础信息，包含发布状态等信息
func (s *Server) GetConfigFileRichInfo(ctx context.Context,
	req *apiconfig.ConfigFile) *apiconfig.ConfigResponse {
	if errResp := checkReadFileParameter(req); errResp != nil {
		return errResp
	}
	return s.nextServer.GetConfigFileRichInfo(ctx, req)
}

// SearchConfigFile 查询配置文件
func (s *Server) SearchConfigFile(ctx context.Context,
	filter map[string]string) *apiconfig.ConfigBatchQueryResponse {

	offset, limit, err := utils.ParseOffsetAndLimit(filter)
	if err != nil {
		out := api.NewConfigBatchQueryResponse(apimodel.Code_BadRequest)
		out.Info = utils.NewStringValue(err.Error())
		return out
	}
	searchFilters := map[string]string{
		"offset": strconv.FormatInt(int64(offset), 10),
		"limit":  strconv.FormatInt(int64(limit), 10),
	}
	for k, v := range filter {
		// 无效查询参数自动忽略
		if v == "" {
			continue
		}
		if _, ok := availableSearch["config_file"][k]; ok {
			searchFilters[k] = v
		}
	}
	return s.nextServer.SearchConfigFile(ctx, searchFilters)
}

// UpdateConfigFile 更新配置文件
func (s *Server) UpdateConfigFile(
	ctx context.Context, configFile *apiconfig.ConfigFile) *apiconfig.ConfigResponse {

	return s.nextServer.UpdateConfigFile(ctx, configFile)
}

// DeleteConfigFile 删除配置文件，删除配置文件同时会通知客户端 Not_Found
func (s *Server) DeleteConfigFile(ctx context.Context,
	req *apiconfig.ConfigFile) *apiconfig.ConfigResponse {

	return s.nextServer.DeleteConfigFile(ctx, req)
}

// BatchDeleteConfigFile 批量删除配置文件
func (s *Server) BatchDeleteConfigFile(ctx context.Context,
	req []*apiconfig.ConfigFile) *apiconfig.ConfigResponse {

	return s.nextServer.BatchDeleteConfigFile(ctx, req)
}

func (s *Server) ExportConfigFile(ctx context.Context,
	configFileExport *apiconfig.ConfigFileExportRequest) *apiconfig.ConfigExportResponse {

	return s.nextServer.ExportConfigFile(ctx, configFileExport)
}

func (s *Server) ImportConfigFile(ctx context.Context,
	configFiles []*apiconfig.ConfigFile, conflictHandling string) *apiconfig.ConfigImportResponse {
	for _, configFile := range configFiles {
		if checkRsp := s.checkConfigFileParams(configFile); checkRsp != nil {
			return api.NewConfigFileImportResponse(apimodel.Code(checkRsp.Code.GetValue()), nil, nil, nil)
		}
	}
	return s.nextServer.ImportConfigFile(ctx, configFiles, conflictHandling)
}

func (s *Server) GetAllConfigEncryptAlgorithms(
	ctx context.Context) *apiconfig.ConfigEncryptAlgorithmResponse {
	return s.nextServer.GetAllConfigEncryptAlgorithms(ctx)
}
