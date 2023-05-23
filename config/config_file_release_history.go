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
	"encoding/base64"

	apiconfig "github.com/polarismesh/specification/source/go/api/v1/config_manage"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	"go.uber.org/zap"

	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/model"
	commonstore "github.com/polarismesh/polaris/common/store"
	"github.com/polarismesh/polaris/common/time"
	"github.com/polarismesh/polaris/common/utils"
	utils2 "github.com/polarismesh/polaris/config/utils"
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

	// 过滤数据密钥 tag，不保存到发布历史中
	filterTags := make([]*apiconfig.ConfigFileTag, 0, len(tags))
	for _, tag := range tags {
		if tag.Key.GetValue() != utils.ConfigFileTagKeyDataKey {
			filterTags = append(filterTags, tag)
		}
	}
	releaseHistory := &model.ConfigFileReleaseHistory{
		Name:      fileRelease.Name,
		Namespace: namespace,
		Group:     group,
		FileName:  fileName,
		Content:   fileRelease.Content,
		Format:    format,
		Tags:      utils2.ToTagJsonStr(filterTags),
		Comment:   fileRelease.Comment,
		Md5:       fileRelease.Md5,
		Type:      releaseType,
		Status:    status,
		CreateBy:  fileRelease.ModifyBy,
		ModifyBy:  fileRelease.ModifyBy,
	}

	err := s.storage.CreateConfigFileReleaseHistory(s.getTx(ctx), releaseHistory)

	if err != nil {
		log.Error("[Config][Service] create config file release history error.",
			utils.ZapRequestIDByCtx(ctx),
			zap.String("namespace", fileRelease.Namespace),
			zap.String("group", fileRelease.Group),
			zap.String("fileName", fileRelease.FileName),
			zap.Error(err))
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
		log.Error("[Config][Service] get config file release history error.",
			utils.ZapRequestIDByCtx(ctx),
			zap.String("namespace", namespace),
			zap.String("group", group),
			zap.String("fileName", fileName),
			zap.Error(err))
		return api.NewConfigFileReleaseHistoryBatchQueryResponse(commonstore.StoreCode2APICode(err), 0, nil)
	}

	if len(releaseHistories) == 0 {
		return api.NewConfigFileReleaseHistoryBatchQueryResponse(apimodel.Code_ExecuteSuccess, count, nil)
	}

	var apiReleaseHistory []*apiconfig.ConfigFileReleaseHistory
	for _, history := range releaseHistories {
		historyAPIModel := transferReleaseHistoryStoreModel2APIModel(history)
		apiReleaseHistory = append(apiReleaseHistory, historyAPIModel)
	}

	if err := s.decryptMultiConfigFileReleaseHistory(ctx, apiReleaseHistory); err != nil {
		log.Error("[Config][Service] decrypt config file release history error.",
			utils.ZapRequestIDByCtx(ctx),
			zap.String("namespace", namespace),
			zap.String("group", group),
			zap.String("name", fileName),
			zap.Error(err))
		return api.NewConfigFileReleaseHistoryBatchQueryResponse(apimodel.Code_EncryptConfigFileException, 0, nil)
	}

	return api.NewConfigFileReleaseHistoryBatchQueryResponse(apimodel.Code_ExecuteSuccess, count, apiReleaseHistory)
}

// GetConfigFileLatestReleaseHistory 获取配置文件最后一次发布记录
func (s *Server) GetConfigFileLatestReleaseHistory(ctx context.Context, namespace, group,
	fileName string) *apiconfig.ConfigResponse {

	if err := utils2.CheckResourceName(utils.NewStringValue(namespace)); err != nil {
		return api.NewConfigFileReleaseHistoryResponse(apimodel.Code_InvalidNamespaceName, nil)
	}

	if err := utils2.CheckResourceName(utils.NewStringValue(group)); err != nil {
		return api.NewConfigFileReleaseHistoryResponse(apimodel.Code_InvalidNamespaceName, nil)
	}

	if err := utils2.CheckFileName(utils.NewStringValue(fileName)); err != nil {
		return api.NewConfigFileReleaseHistoryResponse(apimodel.Code_InvalidNamespaceName, nil)
	}

	history, err := s.storage.GetLatestConfigFileReleaseHistory(namespace, group, fileName)

	if err != nil {
		log.Error("[Config][Service] get latest config file release error",
			utils.ZapRequestIDByCtx(ctx),
			zap.String("namespace", namespace),
			zap.String("group", group),
			zap.String("fileName", fileName),
			zap.Error(err),
		)
		return api.NewConfigFileReleaseHistoryResponse(commonstore.StoreCode2APICode(err), nil)
	}
	apiHistory := transferReleaseHistoryStoreModel2APIModel(history)
	return api.NewConfigFileReleaseHistoryResponse(apimodel.Code_ExecuteSuccess, apiHistory)
}

func transferReleaseHistoryStoreModel2APIModel(
	releaseHistory *model.ConfigFileReleaseHistory) *apiconfig.ConfigFileReleaseHistory {

	if releaseHistory == nil {
		return nil
	}
	return &apiconfig.ConfigFileReleaseHistory{
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

// decryptMultiConfigFileReleaseHistory 解密多个配置文件发布历史
func (s *Server) decryptMultiConfigFileReleaseHistory(ctx context.Context,
	releaseHistories []*apiconfig.ConfigFileReleaseHistory) error {
	for _, releaseHistory := range releaseHistories {
		if err := s.decryptConfigFileReleaseHistory(ctx, releaseHistory); err != nil {
			return err
		}
	}
	return nil
}

// decryptConfigFileReleaseHistory 解密配置文件发布历史
func (s *Server) decryptConfigFileReleaseHistory(ctx context.Context,
	releaseHistory *apiconfig.ConfigFileReleaseHistory) error {
	if s.cryptoManager == nil || releaseHistory == nil {
		return nil
	}
	// 非创建人请求不解密
	if utils.ParseUserName(ctx) != releaseHistory.CreateBy.GetValue() {
		return nil
	}
	algorithm, dataKey, err := s.getEncryptAlgorithmAndDataKey(ctx, releaseHistory.GetNamespace().GetValue(),
		releaseHistory.GetGroup().GetValue(), releaseHistory.GetName().GetValue())
	if err != nil {
		return err
	}
	// 非加密文件不解密
	if dataKey == "" {
		return nil
	}
	dateKeyBytes, err := base64.StdEncoding.DecodeString(dataKey)
	if err != nil {
		return err
	}
	crypto, err := s.cryptoManager.GetCrypto(algorithm)
	if err != nil {
		return err
	}
	// 解密
	plainContent, err := crypto.Decrypt(releaseHistory.Content.GetValue(), dateKeyBytes)
	if err != nil {
		return err
	}
	releaseHistory.Content = utils.NewStringValue(plainContent)
	return nil
}
