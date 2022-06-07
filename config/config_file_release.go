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

// PublishConfigFile 发布配置文件
func (s *Server) PublishConfigFile(ctx context.Context, configFileRelease *api.ConfigFileRelease) *api.ConfigResponse {
	namespace := configFileRelease.Namespace.GetValue()
	group := configFileRelease.Group.GetValue()
	fileName := configFileRelease.FileName.GetValue()

	if err := utils2.CheckFileName(utils.NewStringValue(fileName)); err != nil {
		return api.NewConfigFileResponse(api.InvalidConfigFileName, nil)
	}

	if err := utils2.CheckResourceName(utils.NewStringValue(namespace)); err != nil {
		return api.NewConfigFileResponse(api.InvalidNamespaceName, nil)
	}

	if err := utils2.CheckResourceName(utils.NewStringValue(group)); err != nil {
		return api.NewConfigFileResponse(api.InvalidConfigFileGroupName, nil)
	}

	if !s.checkNamespaceExisted(namespace) {
		return api.NewConfigFileReleaseResponse(api.NotFoundNamespace, configFileRelease)
	}

	userName := utils.ParseUserName(ctx)
	configFileRelease.CreateBy = utils.NewStringValue(userName)
	configFileRelease.ModifyBy = utils.NewStringValue(userName)

	tx := s.getTx(ctx)
	// 获取待发布的 configFile 信息
	toPublishFile, err := s.storage.GetConfigFile(tx, namespace, group, fileName)

	requestID, _ := ctx.Value(utils.StringContext("request-id")).(string)
	if err != nil {
		log.ConfigScope().Error("[Config][Service] get config file error.",
			zap.String("request-id", requestID),
			zap.String("namespace", namespace),
			zap.String("group", group),
			zap.String("fileName", fileName),
			zap.Error(err))

		s.recordReleaseFail(ctx, transferConfigFileReleaseAPIModel2StoreModel(configFileRelease))

		return api.NewConfigFileResponse(api.StoreLayerException, nil)
	}

	if toPublishFile == nil {
		return api.NewConfigFileResponse(api.NotFoundResource, nil)
	}

	md5 := utils2.CalMd5(toPublishFile.Content)

	// 获取 configFileRelease 信息
	managedFileRelease, err := s.storage.GetConfigFileReleaseWithAllFlag(tx, namespace, group, fileName)
	if err != nil {
		log.ConfigScope().Error("[Config][Service] get config file release error.",
			zap.String("request-id", requestID),
			zap.String("namespace", namespace),
			zap.String("group", group),
			zap.String("fileName", fileName),
			zap.Error(err))

		s.recordReleaseFail(ctx, transferConfigFileReleaseAPIModel2StoreModel(configFileRelease))

		return api.NewConfigFileResponse(api.StoreLayerException, nil)
	}

	releaseName := configFileRelease.Name.GetValue()
	if releaseName == "" {
		if managedFileRelease == nil {
			releaseName = utils2.GenReleaseName("", fileName)
		} else {
			releaseName = utils2.GenReleaseName(managedFileRelease.Name, fileName)
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
			log.ConfigScope().Error("[Config][Service] create config file release error.",
				zap.String("request-id", requestID),
				zap.String("namespace", namespace),
				zap.String("group", group),
				zap.String("fileName", fileName),
				zap.Error(err))

			s.recordReleaseFail(ctx, transferConfigFileReleaseAPIModel2StoreModel(configFileRelease))

			return api.NewConfigFileResponse(api.StoreLayerException, nil)
		}

		s.RecordConfigFileReleaseHistory(ctx, createdFileRelease, utils.ReleaseTypeNormal, utils.ReleaseStatusSuccess)

		return api.NewConfigFileReleaseResponse(api.ExecuteSuccess,
			transferConfigFileReleaseStoreModel2APIModel(createdFileRelease))
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
		log.ConfigScope().Error("[Config][Service] update config file release error.",
			zap.String("request-id", requestID),
			zap.String("namespace", namespace),
			zap.String("group", group),
			zap.String("fileName", fileName),
			zap.Error(err))

		s.recordReleaseFail(ctx, transferConfigFileReleaseAPIModel2StoreModel(configFileRelease))

		return api.NewConfigFileResponse(api.StoreLayerException, nil)
	}

	s.RecordConfigFileReleaseHistory(ctx, updatedFileRelease, utils.ReleaseTypeNormal, utils.ReleaseStatusSuccess)

	return api.NewConfigFileReleaseResponse(api.ExecuteSuccess,
		transferConfigFileReleaseStoreModel2APIModel(updatedFileRelease))
}

// GetConfigFileRelease 获取配置文件发布内容
func (s *Server) GetConfigFileRelease(ctx context.Context, namespace, group, fileName string) *api.ConfigResponse {
	if err := utils2.CheckFileName(utils.NewStringValue(fileName)); err != nil {
		return api.NewConfigFileResponse(api.InvalidConfigFileName, nil)
	}

	if err := utils2.CheckResourceName(utils.NewStringValue(namespace)); err != nil {
		return api.NewConfigFileResponse(api.InvalidNamespaceName, nil)
	}

	if err := utils2.CheckResourceName(utils.NewStringValue(group)); err != nil {
		return api.NewConfigFileResponse(api.InvalidConfigFileGroupName, nil)
	}

	fileRelease, err := s.storage.GetConfigFileRelease(s.getTx(ctx), namespace, group, fileName)

	if err != nil {
		requestID, _ := ctx.Value(utils.StringContext("request-id")).(string)
		log.ConfigScope().Error("[Config][Service]get config file release error.",
			zap.String("request-id", requestID),
			zap.String("namespace", namespace),
			zap.String("group", group),
			zap.String("fileName", fileName),
			zap.Error(err))

		return api.NewConfigFileResponse(api.StoreLayerException, nil)
	}

	return api.NewConfigFileReleaseResponse(api.ExecuteSuccess,
		transferConfigFileReleaseStoreModel2APIModel(fileRelease))
}

// DeleteConfigFileRelease 删除配置文件发布，删除配置文件的时候，同步删除配置文件发布数据
func (s *Server) DeleteConfigFileRelease(ctx context.Context, namespace, group, fileName, deleteBy string) *api.ConfigResponse {
	if err := utils2.CheckFileName(utils.NewStringValue(fileName)); err != nil {
		return api.NewConfigFileResponse(api.InvalidConfigFileName, nil)
	}

	if err := utils2.CheckResourceName(utils.NewStringValue(namespace)); err != nil {
		return api.NewConfigFileResponse(api.InvalidNamespaceName, nil)
	}

	if err := utils2.CheckResourceName(utils.NewStringValue(group)); err != nil {
		return api.NewConfigFileResponse(api.InvalidConfigFileGroupName, nil)
	}

	latestReleaseRsp := s.GetConfigFileRelease(ctx, namespace, group, fileName)
	if latestReleaseRsp.Code.GetValue() != api.ExecuteSuccess {
		return api.NewConfigFileResponse(latestReleaseRsp.Code.GetValue(), nil)
	}

	requestID, _ := ctx.Value(utils.StringContext("request-id")).(string)

	var releaseName string
	latestRelease := latestReleaseRsp.ConfigFileRelease
	if latestRelease == nil {
		// 从来没有发布过，无需删除
		return api.NewConfigFileResponse(api.ExecuteSuccess, nil)
	}

	releaseName = utils2.GenReleaseName(latestRelease.Name.GetValue(), fileName)
	if releaseName != latestRelease.Name.GetValue() {
		// 更新releaseName
		releaseModel := transferConfigFileReleaseAPIModel2StoreModel(latestRelease)
		releaseModel.Name = releaseName
		_, err := s.storage.UpdateConfigFileRelease(s.getTx(ctx), releaseModel)
		if err != nil {
			log.ConfigScope().Error("[Config][Service] update release name error when delete release.",
				zap.String("request-id", requestID),
				zap.String("namespace", namespace),
				zap.String("group", group),
				zap.String("fileName", fileName),
				zap.Error(err))
		}
		return api.NewConfigFileResponse(api.StoreLayerException, nil)
	}

	err := s.storage.DeleteConfigFileRelease(s.getTx(ctx), namespace, group, fileName, deleteBy)

	if err != nil {
		log.ConfigScope().Error("[Config][Service] delete config file release error.",
			zap.String("request-id", requestID),
			zap.String("namespace", namespace),
			zap.String("group", group),
			zap.String("fileName", fileName),
			zap.Error(err))

		s.RecordConfigFileReleaseHistory(ctx, &model.ConfigFileRelease{
			Name:      releaseName,
			Namespace: namespace,
			Group:     group,
			FileName:  fileName,
			ModifyBy:  deleteBy,
		}, utils.ReleaseTypeDelete, utils.ReleaseStatusFail)

		return api.NewConfigFileResponse(api.StoreLayerException, nil)
	}

	s.RecordConfigFileReleaseHistory(ctx, &model.ConfigFileRelease{
		Name:      releaseName,
		Namespace: namespace,
		Group:     group,
		FileName:  fileName,
		ModifyBy:  deleteBy,
	}, utils.ReleaseTypeDelete, utils.ReleaseStatusSuccess)

	return api.NewConfigFileReleaseResponse(api.ExecuteSuccess, nil)
}

func (s *Server) recordReleaseFail(ctx context.Context, configFileRelease *model.ConfigFileRelease) {
	s.RecordConfigFileReleaseHistory(ctx, configFileRelease, utils.ReleaseTypeNormal, utils.ReleaseStatusFail)
}

func transferConfigFileReleaseAPIModel2StoreModel(release *api.ConfigFileRelease) *model.ConfigFileRelease {
	if release == nil {
		return nil
	}
	var comment string
	if release.Comment != nil {
		comment = release.Comment.Value
	}
	var content string
	if release.Content != nil {
		content = release.Content.Value
	}
	var md5 string
	if release.Md5 != nil {
		md5 = release.Md5.Value
	}
	var version uint64
	if release.Version != nil {
		version = release.Version.Value
	}
	var createBy string
	if release.CreateBy != nil {
		createBy = release.CreateBy.Value
	}
	var modifyBy string
	if release.ModifyBy != nil {
		createBy = release.ModifyBy.Value
	}
	var id uint64
	if release.Id != nil {
		id = release.Id.Value
	}

	return &model.ConfigFileRelease{
		Id:        id,
		Namespace: release.Namespace.GetValue(),
		Group:     release.Group.GetValue(),
		FileName:  release.FileName.GetValue(),
		Content:   content,
		Comment:   comment,
		Md5:       md5,
		Version:   version,
		CreateBy:  createBy,
		ModifyBy:  modifyBy,
	}
}

func transferConfigFileReleaseStoreModel2APIModel(release *model.ConfigFileRelease) *api.ConfigFileRelease {
	if release == nil {
		return nil
	}

	return &api.ConfigFileRelease{
		Id:         utils.NewUInt64Value(release.Id),
		Name:       utils.NewStringValue(release.Name),
		Namespace:  utils.NewStringValue(release.Namespace),
		Group:      utils.NewStringValue(release.Group),
		FileName:   utils.NewStringValue(release.FileName),
		Content:    utils.NewStringValue(release.Content),
		Comment:    utils.NewStringValue(release.Comment),
		Md5:        utils.NewStringValue(release.Md5),
		Version:    utils.NewUInt64Value(release.Version),
		CreateBy:   utils.NewStringValue(release.CreateBy),
		CreateTime: utils.NewStringValue(time.Time2String(release.CreateTime)),
		ModifyBy:   utils.NewStringValue(release.ModifyBy),
		ModifyTime: utils.NewStringValue(time.Time2String(release.ModifyTime)),
	}
}
