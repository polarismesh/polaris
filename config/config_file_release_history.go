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
	releaseType, status string) {

	namespace, group, fileName := fileRelease.Namespace, fileRelease.Group, fileRelease.FileName

	// 获取 format 信息
	var format string
	configFileResponse := s.GetConfigFileBaseInfo(ctx, namespace, group, fileName)
	if configFileResponse.ConfigFile != nil {
		format = configFileResponse.ConfigFile.Format.GetValue()
	}

	// 获取配置文件标签信息
	tags, _ := s.queryTagsByConfigFileWithAPIModels(ctx, namespace, group, fileName)
	releaseHistory := &model.ConfigFileReleaseHistory{
		Name:      fileRelease.Name,
		Namespace: namespace,
		Group:     group,
		FileName:  fileName,
		Content:   fileRelease.Content,
		Format:    format,
		Tags:      ToTagJsonStr(tags),
		Comment:   fileRelease.Comment,
		Md5:       fileRelease.Md5,
		Type:      releaseType,
		Status:    status,
		CreateBy:  fileRelease.ModifyBy,
		ModifyBy:  fileRelease.ModifyBy,
	}

	if err := s.storage.CreateConfigFileReleaseHistory(s.getTx(ctx), releaseHistory); err != nil {
		log.Error("[Config][Service] create config file release history error.", utils.ZapRequestIDByCtx(ctx),
			utils.ZapNamespace(fileRelease.Namespace), utils.ZapGroup(fileRelease.Group),
			utils.ZapFileName(fileRelease.FileName), zap.Error(err))
	}
}

// GetConfigFileReleaseHistory 获取配置文件发布历史记录
func (s *Server) GetConfigFileReleaseHistory(ctx context.Context, namespace, group, fileName string, offset,
	limit uint32, endId uint64) *apiconfig.ConfigBatchQueryResponse {

	if limit > MaxPageSize {
		return api.NewConfigFileReleaseHistoryBatchQueryResponse(apimodel.Code_InvalidParameter, 0, nil)
	}

	count, releaseHistories, err := s.storage.QueryConfigFileReleaseHistories(namespace,
		group, fileName, offset, limit, endId)
	if err != nil {
		log.Error("[Config][Service] get config file release history error.", utils.ZapRequestIDByCtx(ctx),
			utils.ZapNamespace(namespace), utils.ZapGroup(group), utils.ZapFileName(fileName), zap.Error(err))
		return api.NewConfigFileReleaseHistoryBatchQueryResponse(commonstore.StoreCode2APICode(err), 0, nil)
	}

	if len(releaseHistories) == 0 {
		return api.NewConfigFileReleaseHistoryBatchQueryResponse(apimodel.Code_ExecuteSuccess, count, nil)
	}

	var apiReleaseHistory []*apiconfig.ConfigFileReleaseHistory
	for _, history := range releaseHistories {
		historyAPIModel := model.ToReleaseHistoryAPI(history)
		apiReleaseHistory = append(apiReleaseHistory, historyAPIModel)
	}

	if err := s.decryptMultiConfigFileReleaseHistory(apiReleaseHistory); err != nil {
		log.Error("[Config][Service] decrypt config file release history error.", utils.ZapRequestIDByCtx(ctx),
			utils.ZapNamespace(namespace), utils.ZapGroup(group), utils.ZapFileName(fileName), zap.Error(err))
		return api.NewConfigFileReleaseHistoryBatchQueryResponse(apimodel.Code_EncryptConfigFileException, 0, nil)
	}

	return api.NewConfigFileReleaseHistoryBatchQueryResponse(apimodel.Code_ExecuteSuccess, count, apiReleaseHistory)
}

// GetConfigFileLatestReleaseHistory 获取配置文件最后一次发布记录
func (s *Server) GetConfigFileLatestReleaseHistory(ctx context.Context, namespace, group,
	fileName string) *apiconfig.ConfigResponse {

	if err := CheckResourceName(utils.NewStringValue(namespace)); err != nil {
		return api.NewConfigFileReleaseHistoryResponse(apimodel.Code_InvalidNamespaceName, nil)
	}

	if err := CheckResourceName(utils.NewStringValue(group)); err != nil {
		return api.NewConfigFileReleaseHistoryResponse(apimodel.Code_InvalidNamespaceName, nil)
	}

	if err := CheckFileName(utils.NewStringValue(fileName)); err != nil {
		return api.NewConfigFileReleaseHistoryResponse(apimodel.Code_InvalidNamespaceName, nil)
	}

	history, err := s.storage.GetLatestConfigFileReleaseHistory(namespace, group, fileName)
	if err != nil {
		log.Error("[Config][Service] get latest config file release error", utils.ZapRequestIDByCtx(ctx),
			utils.ZapNamespace(namespace), utils.ZapGroup(group), utils.ZapFileName(fileName), zap.Error(err),
		)
		return api.NewConfigFileReleaseHistoryResponse(commonstore.StoreCode2APICode(err), nil)
	}
	apiHistory := model.ToReleaseHistoryAPI(history)
	s.decryptConfigFileReleaseHistory(apiHistory)
	return api.NewConfigFileReleaseHistoryResponse(apimodel.Code_ExecuteSuccess, apiHistory)
}

// decryptMultiConfigFileReleaseHistory 解密多个配置文件发布历史
func (s *Server) decryptMultiConfigFileReleaseHistory(releaseHistories []*apiconfig.ConfigFileReleaseHistory) error {
	for _, releaseHistory := range releaseHistories {
		_ = s.decryptConfigFileReleaseHistory(releaseHistory)
	}
	return nil
}

// decryptConfigFileReleaseHistory 解密配置文件发布历史
func (s *Server) decryptConfigFileReleaseHistory(releaseHistory *apiconfig.ConfigFileReleaseHistory) error {
	if releaseHistory == nil {
		return nil
	}
	var (
		dataKey   = ""
		algorithm = ""
	)
	for _, tag := range releaseHistory.GetTags() {
		if tag.Key.GetValue() == utils.ConfigFileTagKeyDataKey {
			dataKey = tag.Value.GetValue()
		}
		if tag.Key.GetValue() == utils.ConfigFileTagKeyEncryptAlgo {
			algorithm = tag.Value.GetValue()
		}
	}
	plainContent, err := s.decryptConfigFileContent(dataKey, algorithm, releaseHistory.GetContent().GetValue())
	if err != nil {
		return err
	}
	if plainContent == "" {
		return nil
	}
	releaseHistory.Content = utils.NewStringValue(plainContent)
	return nil
}
