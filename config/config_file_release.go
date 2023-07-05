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
	"time"

	"github.com/gogo/protobuf/jsonpb"
	apiconfig "github.com/polarismesh/specification/source/go/api/v1/config_manage"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	"go.uber.org/zap"

	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/model"
	commonstore "github.com/polarismesh/polaris/common/store"
	"github.com/polarismesh/polaris/common/utils"
)

// PublishConfigFile 发布配置文件
func (s *Server) PublishConfigFile(
	ctx context.Context, configFileRelease *apiconfig.ConfigFileRelease) *apiconfig.ConfigResponse {
	namespace := configFileRelease.Namespace.GetValue()
	group := configFileRelease.Group.GetValue()
	fileName := configFileRelease.FileName.GetValue()

	if err := CheckFileName(utils.NewStringValue(fileName)); err != nil {
		return api.NewConfigFileResponse(apimodel.Code_InvalidConfigFileName, nil)
	}
	if err := CheckResourceName(utils.NewStringValue(namespace)); err != nil {
		return api.NewConfigFileResponse(apimodel.Code_InvalidNamespaceName, nil)
	}
	if err := CheckResourceName(utils.NewStringValue(group)); err != nil {
		return api.NewConfigFileResponse(apimodel.Code_InvalidConfigFileGroupName, nil)
	}
	if !s.checkNamespaceExisted(namespace) {
		return api.NewConfigFileReleaseResponse(apimodel.Code_NotFoundNamespace, configFileRelease)
	}

	userName := utils.ParseUserName(ctx)
	configFileRelease.CreateBy = utils.NewStringValue(userName)
	configFileRelease.ModifyBy = utils.NewStringValue(userName)

	tx := s.getTx(ctx)
	// 获取待发布的 configFile 信息
	toPublishFile, err := s.storage.GetConfigFile(tx, namespace, group, fileName)
	if err != nil {
		log.Error("[Config][Service] get config file error.", utils.ZapRequestIDByCtx(ctx),
			utils.ZapNamespace(namespace), utils.ZapGroup(group), utils.ZapFileName(fileName), zap.Error(err))
		s.recordReleaseFail(ctx, model.ToConfigFileReleaseStore(configFileRelease))
		return api.NewConfigFileResponse(commonstore.StoreCode2APICode(err), nil)
	}

	if toPublishFile == nil {
		return api.NewConfigFileResponse(apimodel.Code_NotFoundResource, nil)
	}

	md5 := CalMd5(toPublishFile.Content)

	// 获取 configFileRelease 信息
	managedFileRelease, err := s.storage.GetConfigFileReleaseWithAllFlag(tx, namespace, group, fileName)
	if err != nil {
		log.Error("[Config][Service] get config file release error.", utils.ZapRequestIDByCtx(ctx),
			utils.ZapNamespace(namespace), utils.ZapGroup(group), utils.ZapFileName(fileName), zap.Error(err))

		s.recordReleaseFail(ctx, model.ToConfigFileReleaseStore(configFileRelease))
		return api.NewConfigFileResponse(commonstore.StoreCode2APICode(err), nil)
	}

	releaseName := configFileRelease.Name.GetValue()
	if releaseName == "" {
		if managedFileRelease == nil {
			releaseName = GenReleaseName("", fileName)
		} else {
			releaseName = GenReleaseName(managedFileRelease.Name, fileName)
		}
	}

	// 第一次发布
	if managedFileRelease == nil {
		fileRelease := &model.ConfigFileRelease{
			Name:      releaseName,
			Namespace: namespace,
			Group:     group,
			FileName:  fileName,
			Content:   toPublishFile.Content,
			Comment:   configFileRelease.Comment.GetValue(),
			Md5:       md5,
			Version:   1,
			Flag:      0,
			CreateBy:  configFileRelease.CreateBy.GetValue(),
			ModifyBy:  configFileRelease.CreateBy.GetValue(),
		}

		createdFileRelease, err := s.storage.CreateConfigFileRelease(tx, fileRelease)
		if err != nil {
			log.Error("[Config][Service] create config file release error.", utils.ZapRequestIDByCtx(ctx),
				utils.ZapNamespace(namespace), utils.ZapGroup(group), utils.ZapFileName(fileName), zap.Error(err))
			s.recordReleaseFail(ctx, model.ToConfigFileReleaseStore(configFileRelease))
			return api.NewConfigFileResponse(commonstore.StoreCode2APICode(err), nil)
		}

		s.RecordHistory(ctx, configFileReleaseRecordEntry(ctx, configFileRelease, createdFileRelease, model.OCreate))
		s.recordReleaseHistory(ctx, createdFileRelease, utils.ReleaseTypeNormal, utils.ReleaseStatusSuccess)
		return api.NewConfigFileReleaseResponse(apimodel.Code_ExecuteSuccess,
			model.ToConfiogFileReleaseApi(createdFileRelease))
	}

	// 更新发布
	fileRelease := &model.ConfigFileRelease{
		Name:      releaseName,
		Namespace: namespace,
		Group:     group,
		FileName:  fileName,
		Content:   toPublishFile.Content,
		Comment:   configFileRelease.Comment.GetValue(),
		Md5:       md5,
		Version:   managedFileRelease.Version + 1,
		ModifyBy:  configFileRelease.CreateBy.GetValue(),
	}

	updatedFileRelease, err := s.storage.UpdateConfigFileRelease(tx, fileRelease)
	if err != nil {
		log.Error("[Config][Service] update config file release error.", utils.ZapRequestIDByCtx(ctx),
			utils.ZapNamespace(namespace), utils.ZapGroup(group), utils.ZapFileName(fileName), zap.Error(err))
		s.recordReleaseFail(ctx, model.ToConfigFileReleaseStore(configFileRelease))
		return api.NewConfigFileResponse(commonstore.StoreCode2APICode(err), nil)
	}

	s.recordReleaseHistory(ctx, updatedFileRelease, utils.ReleaseTypeNormal, utils.ReleaseStatusSuccess)
	s.RecordHistory(ctx, configFileReleaseRecordEntry(ctx, configFileRelease, updatedFileRelease, model.OCreate))
	return api.NewConfigFileReleaseResponse(apimodel.Code_ExecuteSuccess,
		model.ToConfiogFileReleaseApi(updatedFileRelease))
}

// GetConfigFileRelease 获取配置文件发布内容
func (s *Server) GetConfigFileRelease(
	ctx context.Context, namespace, group, fileName string) *apiconfig.ConfigResponse {
	if err := CheckFileName(utils.NewStringValue(fileName)); err != nil {
		return api.NewConfigFileResponse(apimodel.Code_InvalidConfigFileName, nil)
	}
	if err := CheckResourceName(utils.NewStringValue(namespace)); err != nil {
		return api.NewConfigFileResponse(apimodel.Code_InvalidNamespaceName, nil)
	}
	if err := CheckResourceName(utils.NewStringValue(group)); err != nil {
		return api.NewConfigFileResponse(apimodel.Code_InvalidConfigFileGroupName, nil)
	}

	fileRelease, err := s.storage.GetConfigFileRelease(s.getTx(ctx), namespace, group, fileName)
	if err != nil {
		log.Error("[Config][Service]get config file release error.", utils.ZapRequestIDByCtx(ctx),
			utils.ZapNamespace(namespace), utils.ZapGroup(group), utils.ZapFileName(fileName), zap.Error(err))
		return api.NewConfigFileResponse(commonstore.StoreCode2APICode(err), nil)
	}
	if fileRelease == nil {
		return api.NewConfigFileReleaseResponse(apimodel.Code_ExecuteSuccess, nil)
	}
	release := model.ToConfiogFileReleaseApi(fileRelease)
	release, err = s.chains.AfterGetFileRelease(ctx, release)
	if err != nil {
		return api.NewConfigFileReleaseResponseWithMessage(apimodel.Code_ExecuteException, err.Error())
	}
	return api.NewConfigFileReleaseResponse(apimodel.Code_ExecuteSuccess, release)
}

// DeleteConfigFileRelease 删除配置文件发布，删除配置文件的时候，同步删除配置文件发布数据
func (s *Server) DeleteConfigFileRelease(ctx context.Context, namespace,
	group, fileName, deleteBy string) *apiconfig.ConfigResponse {

	if err := CheckFileName(utils.NewStringValue(fileName)); err != nil {
		return api.NewConfigFileResponse(apimodel.Code_InvalidConfigFileName, nil)
	}
	if err := CheckResourceName(utils.NewStringValue(namespace)); err != nil {
		return api.NewConfigFileResponse(apimodel.Code_InvalidNamespaceName, nil)
	}
	if err := CheckResourceName(utils.NewStringValue(group)); err != nil {
		return api.NewConfigFileResponse(apimodel.Code_InvalidConfigFileGroupName, nil)
	}

	latestReleaseRsp := s.GetConfigFileRelease(ctx, namespace, group, fileName)
	if latestReleaseRsp.Code.GetValue() != uint32(apimodel.Code_ExecuteSuccess) {
		return api.NewConfigFileResponse(apimodel.Code(latestReleaseRsp.Code.GetValue()), nil)
	}

	var releaseName string
	latestRelease := latestReleaseRsp.ConfigFileRelease
	if latestRelease == nil {
		// 从来没有发布过，无需删除
		return api.NewConfigFileResponse(apimodel.Code_ExecuteSuccess, nil)
	}

	releaseName = GenReleaseName(latestRelease.Name.GetValue(), fileName)
	if releaseName != latestRelease.Name.GetValue() {
		// 更新releaseName
		releaseModel := model.ToConfigFileReleaseStore(latestRelease)
		releaseModel.Name = releaseName
		_, err := s.storage.UpdateConfigFileRelease(s.getTx(ctx), releaseModel)
		if err != nil {
			log.Error("[Config][Service] update release name error when delete release.",
				utils.ZapRequestIDByCtx(ctx), utils.ZapNamespace(namespace), utils.ZapGroup(group),
				utils.ZapFileName(fileName), zap.Error(err))
			return api.NewConfigFileResponse(commonstore.StoreCode2APICode(err), nil)
		}
	}

	if err := s.storage.DeleteConfigFileRelease(s.getTx(ctx), namespace, group, fileName, deleteBy); err != nil {
		log.Error("[Config][Service] delete config file release error.", utils.ZapRequestIDByCtx(ctx),
			utils.ZapNamespace(namespace), utils.ZapGroup(group), utils.ZapFileName(fileName), zap.Error(err))
		s.recordReleaseHistory(ctx, &model.ConfigFileRelease{
			Name:      releaseName,
			Namespace: namespace,
			Group:     group,
			FileName:  fileName,
			ModifyBy:  deleteBy,
		}, utils.ReleaseTypeDelete, utils.ReleaseStatusFail)
		return api.NewConfigFileResponse(commonstore.StoreCode2APICode(err), nil)
	}

	data := &model.ConfigFileRelease{
		Name:      releaseName,
		Namespace: namespace,
		Group:     group,
		FileName:  fileName,
		ModifyBy:  deleteBy,
	}
	s.recordReleaseHistory(ctx, data, utils.ReleaseTypeDelete, utils.ReleaseStatusSuccess)
	s.RecordHistory(ctx, configFileReleaseRecordEntry(ctx, &apiconfig.ConfigFileRelease{
		Namespace: utils.NewStringValue(namespace),
		Name:      utils.NewStringValue(releaseName),
		Group:     utils.NewStringValue(group),
		FileName:  utils.NewStringValue(fileName),
	}, data, model.ODelete))
	return api.NewConfigFileReleaseResponse(apimodel.Code_ExecuteSuccess, nil)
}

func (s *Server) recordReleaseFail(ctx context.Context, configFileRelease *model.ConfigFileRelease) {
	s.recordReleaseHistory(ctx, configFileRelease, utils.ReleaseTypeNormal, utils.ReleaseStatusFail)
}

// configFileReleaseRecordEntry 生成服务的记录entry
func configFileReleaseRecordEntry(ctx context.Context, req *apiconfig.ConfigFileRelease, md *model.ConfigFileRelease,
	operationType model.OperationType) *model.RecordEntry {

	marshaler := jsonpb.Marshaler{}
	detail, _ := marshaler.MarshalToString(req)

	entry := &model.RecordEntry{
		ResourceType:  model.RConfigFileRelease,
		ResourceName:  req.GetName().GetValue(),
		Namespace:     req.GetNamespace().GetValue(),
		OperationType: operationType,
		Operator:      utils.ParseOperator(ctx),
		Detail:        detail,
		HappenTime:    time.Now(),
	}

	return entry
}
