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
 * CONDITIONS OF ANY KIND, either express or Serveried. See the License for the
 * specific language governing permissions and limitations under the License.
 */

package config

import (
	"context"

	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/common/time"
	"github.com/polarismesh/polaris-server/common/utils"
	utils2 "github.com/polarismesh/polaris-server/config/utils"
	"go.uber.org/zap"
)

// RecordConfigFileReleaseHistory 新增配置文件发布历史记录
func (s *Server) RecordConfigFileReleaseHistory(ctx context.Context, fileRelease *model.ConfigFileRelease, releaseType, status string) {
	requestID, _ := ctx.Value(utils.StringContext("request-id")).(string)

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
		Tags:      utils2.ToTagJsonStr(tags),
		Comment:   fileRelease.Comment,
		Md5:       fileRelease.Md5,
		Type:      releaseType,
		Status:    status,
		CreateBy:  fileRelease.ModifyBy,
		ModifyBy:  fileRelease.ModifyBy,
	}

	err := s.storage.CreateConfigFileReleaseHistory(s.getTx(ctx), releaseHistory)

	if err != nil {
		log.ConfigScope().Error("[Config][Service] create config file release history error.",
			zap.String("request-id", requestID),
			zap.String("namespace", fileRelease.Namespace),
			zap.String("group", fileRelease.Group),
			zap.String("fileName", fileRelease.FileName),
			zap.Error(err))
	}
}

// GetConfigFileReleaseHistory 获取配置文件发布历史记录
func (s *Server) GetConfigFileReleaseHistory(ctx context.Context, namespace, group, fileName string, offset,
	limit uint32, endId uint64) *api.ConfigBatchQueryResponse {

	if offset < 0 || limit <= 0 || limit > MaxPageSize {
		return api.NewConfigFileReleaseHistoryBatchQueryResponse(api.InvalidParameter, 0, nil)
	}

	count, releaseHistories, err := s.storage.QueryConfigFileReleaseHistories(namespace, group, fileName, offset, limit, endId)

	requestID, _ := ctx.Value(utils.StringContext("request-id")).(string)
	if err != nil {
		log.ConfigScope().Error("[Config][Service] get config file release history error.",
			zap.String("request-id", requestID),
			zap.String("namespace", namespace),
			zap.String("group", group),
			zap.String("fileName", fileName),
			zap.Error(err))
		return api.NewConfigFileReleaseHistoryBatchQueryResponse(api.StoreLayerException, 0, nil)
	}

	if len(releaseHistories) == 0 {
		return api.NewConfigFileReleaseHistoryBatchQueryResponse(api.ExecuteSuccess, count, nil)
	}

	var apiReleaseHistory []*api.ConfigFileReleaseHistory
	for _, history := range releaseHistories {
		historyAPIModel := transferReleaseHistoryStoreModel2APIModel(history)
		apiReleaseHistory = append(apiReleaseHistory, historyAPIModel)
	}

	return api.NewConfigFileReleaseHistoryBatchQueryResponse(api.ExecuteSuccess, count, apiReleaseHistory)
}

// GetConfigFileLatestReleaseHistory 获取配置文件最后一次发布记录
func (s *Server) GetConfigFileLatestReleaseHistory(ctx context.Context, namespace, group,
	fileName string) *api.ConfigResponse {

	if err := utils2.CheckResourceName(utils.NewStringValue(namespace)); err != nil {
		return api.NewConfigFileReleaseHistoryResponse(api.InvalidNamespaceName, nil)
	}

	if err := utils2.CheckResourceName(utils.NewStringValue(group)); err != nil {
		return api.NewConfigFileReleaseHistoryResponse(api.InvalidNamespaceName, nil)
	}

	if err := utils2.CheckFileName(utils.NewStringValue(fileName)); err != nil {
		return api.NewConfigFileReleaseHistoryResponse(api.InvalidNamespaceName, nil)
	}

	history, err := s.storage.GetLatestConfigFileReleaseHistory(namespace, group, fileName)

	requestID, _ := ctx.Value(utils.StringContext("request-id")).(string)

	if err != nil {
		log.ConfigScope().Error("[Config][Service] get latest config file release error",
			zap.String("request-id", requestID),
			zap.String("namespace", namespace),
			zap.String("group", group),
			zap.String("fileName", fileName),
			zap.Error(err),
		)
		return api.NewConfigFileReleaseHistoryResponse(api.StoreLayerException, nil)
	}

	return api.NewConfigFileReleaseHistoryResponse(api.ExecuteSuccess, transferReleaseHistoryStoreModel2APIModel(history))
}

func transferReleaseHistoryStoreModel2APIModel(releaseHistory *model.ConfigFileReleaseHistory) *api.ConfigFileReleaseHistory {
	if releaseHistory == nil {
		return nil
	}
	return &api.ConfigFileReleaseHistory{
		Id:         utils.NewUInt64Value(releaseHistory.Id),
		Name:       utils.NewStringValue(releaseHistory.Name),
		Namespace:  utils.NewStringValue(releaseHistory.Namespace),
		Group:      utils.NewStringValue(releaseHistory.Group),
		FileName:   utils.NewStringValue(releaseHistory.FileName),
		Content:    utils.NewStringValue(releaseHistory.Content),
		Comment:    utils.NewStringValue(releaseHistory.Comment),
		Format:     utils.NewStringValue(releaseHistory.Format),
		Tags:       utils2.FromTagJson(releaseHistory.Tags),
		Md5:        utils.NewStringValue(releaseHistory.Md5),
		Type:       utils.NewStringValue(releaseHistory.Type),
		Status:     utils.NewStringValue(releaseHistory.Status),
		CreateBy:   utils.NewStringValue(releaseHistory.CreateBy),
		CreateTime: utils.NewStringValue(time.Time2String(releaseHistory.CreateTime)),
		ModifyBy:   utils.NewStringValue(releaseHistory.ModifyBy),
		ModifyTime: utils.NewStringValue(time.Time2String(releaseHistory.ModifyTime)),
	}
}
