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
	"context"

	apiconfig "github.com/polarismesh/specification/source/go/api/v1/config_manage"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	"go.uber.org/zap"

	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/model"
	commonstore "github.com/polarismesh/polaris/common/store"
	"github.com/polarismesh/polaris/common/utils"
)

// recordReleaseHistory 新增配置文件发布历史记录
func (s *Server) recordReleaseHistory(ctx context.Context, fileRelease *model.ConfigFileRelease,
	releaseType, status, reason string) {

	releaseHistory := &model.ConfigFileReleaseHistory{
		Name:               fileRelease.Name,
		Namespace:          fileRelease.Namespace,
		Group:              fileRelease.Group,
		FileName:           fileRelease.FileName,
		Content:            fileRelease.Content,
		Format:             fileRelease.Format,
		Metadata:           fileRelease.Metadata,
		Comment:            fileRelease.Comment,
		Md5:                fileRelease.Md5,
		Version:            fileRelease.Version,
		Type:               releaseType,
		Status:             status,
		Reason:             reason,
		CreateBy:           utils.ParseUserName(ctx),
		ModifyBy:           utils.ParseUserName(ctx),
		ReleaseDescription: fileRelease.ReleaseDescription,
	}

	if err := s.storage.CreateConfigFileReleaseHistory(releaseHistory); err != nil {
		log.Error("[Config][History] create config file release history error.", utils.RequestID(ctx),
			utils.ZapNamespace(fileRelease.Namespace), utils.ZapGroup(fileRelease.Group),
			utils.ZapFileName(fileRelease.FileName), zap.Error(err))
	}
}

// GetConfigFileReleaseHistories 获取配置文件发布历史记录
func (s *Server) GetConfigFileReleaseHistories(ctx context.Context,
	searchFilter map[string]string) *apiconfig.ConfigBatchQueryResponse {

	offset, limit, _ := utils.ParseOffsetAndLimit(searchFilter)

	count, saveDatas, err := s.storage.QueryConfigFileReleaseHistories(searchFilter, offset, limit)
	if err != nil {
		log.Error("[Config][History] get config file release history error.", utils.RequestID(ctx),
			zap.Any("filter", searchFilter), zap.Error(err))
		return api.NewConfigBatchQueryResponseWithInfo(commonstore.StoreCode2APICode(err), err.Error())
	}

	if len(saveDatas) == 0 {
		out := api.NewConfigBatchQueryResponse(apimodel.Code_ExecuteSuccess)
		out.Total = utils.NewUInt32Value(0)
		return out
	}

	var histories []*apiconfig.ConfigFileReleaseHistory
	for _, data := range saveDatas {
		data, err := s.chains.AfterGetFileHistory(ctx, data)
		if err != nil {
			return api.NewConfigBatchQueryResponseWithInfo(apimodel.Code_ExecuteException, err.Error())
		}
		history := model.ToReleaseHistoryAPI(data)
		histories = append(histories, history)
	}
	out := api.NewConfigBatchQueryResponse(apimodel.Code_ExecuteSuccess)
	out.Total = utils.NewUInt32Value(count)
	out.ConfigFileReleaseHistories = histories
	return out
}
