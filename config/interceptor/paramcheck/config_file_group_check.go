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

// CreateConfigFileGroup 创建配置文件组
func (s *Server) CreateConfigFileGroup(ctx context.Context,
	req *apiconfig.ConfigFileGroup) *apiconfig.ConfigResponse {
	if checkError := checkConfigFileGroupParams(req); checkError != nil {
		return checkError
	}
	return s.nextServer.CreateConfigFileGroup(ctx, req)
}

// QueryConfigFileGroups 查询配置文件组
func (s *Server) QueryConfigFileGroups(ctx context.Context,
	filter map[string]string) *apiconfig.ConfigBatchQueryResponse {

	offset, limit, err := utils.ParseOffsetAndLimit(filter)
	if err != nil {
		resp := api.NewConfigBatchQueryResponse(apimodel.Code_BadRequest)
		resp.Info = utils.NewStringValue(err.Error())
		return resp
	}

	searchFilters := map[string]string{
		"offset": strconv.FormatInt(int64(offset), 10),
		"limit":  strconv.FormatInt(int64(limit), 10),
	}
	for k, v := range filter {
		if newK, ok := availableSearch["config_file_group"][k]; ok {
			searchFilters[newK] = v
		}
	}

	return s.nextServer.QueryConfigFileGroups(ctx, searchFilters)
}

// DeleteConfigFileGroup 删除配置文件组
func (s *Server) DeleteConfigFileGroup(
	ctx context.Context, namespace, name string) *apiconfig.ConfigResponse {
	if err := utils.CheckResourceName(utils.NewStringValue(namespace)); err != nil {
		return api.NewConfigResponse(apimodel.Code_InvalidNamespaceName)
	}
	if err := utils.CheckResourceName(utils.NewStringValue(name)); err != nil {
		return api.NewConfigResponse(apimodel.Code_InvalidConfigFileGroupName)
	}
	return s.nextServer.DeleteConfigFileGroup(ctx, namespace, name)
}

// UpdateConfigFileGroup 更新配置文件组
func (s *Server) UpdateConfigFileGroup(ctx context.Context,
	req *apiconfig.ConfigFileGroup) *apiconfig.ConfigResponse {
	if checkError := checkConfigFileGroupParams(req); checkError != nil {
		return checkError
	}
	return s.nextServer.UpdateConfigFileGroup(ctx, req)
}

func checkConfigFileGroupParams(configFileGroup *apiconfig.ConfigFileGroup) *apiconfig.ConfigResponse {
	if configFileGroup == nil {
		return api.NewConfigResponse(apimodel.Code_InvalidParameter)
	}
	if err := utils.CheckResourceName(configFileGroup.Name); err != nil {
		return api.NewConfigResponse(apimodel.Code_InvalidConfigFileGroupName)
	}
	if err := utils.CheckResourceName(configFileGroup.Namespace); err != nil {
		return api.NewConfigResponse(apimodel.Code_InvalidNamespaceName)
	}
	if len(configFileGroup.GetMetadata()) > utils.MaxMetadataLength {
		return api.NewConfigResponse(apimodel.Code_InvalidMetadata)
	}
	return nil
}
