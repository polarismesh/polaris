/*
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

package service

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
func (cs *Impl) PublishConfigFile(ctx context.Context, configFileRelease *api.ConfigFileRelease) *api.ConfigResponse {
	namespace := configFileRelease.Namespace.GetValue()
	group := configFileRelease.Group.GetValue()
	fileName := configFileRelease.FileName.GetValue()

	if err := utils2.CheckResourceName(utils.NewStringValue(fileName)); err != nil {
		return api.NewConfigFileResponse(api.InvalidConfigFileName, nil)
	}

	if err := utils2.CheckResourceName(utils.NewStringValue(namespace)); err != nil {
		return api.NewConfigFileResponse(api.InvalidNamespaceName, nil)
	}

	if err := utils2.CheckResourceName(utils.NewStringValue(group)); err != nil {
		return api.NewConfigFileResponse(api.InvalidConfigFileGroupName, nil)
	}

	if !cs.checkNamespaceExisted(namespace) {
		return api.NewConfigFileReleaseResponse(api.NotFoundNamespace, configFileRelease)
	}

	tx := cs.getTx(ctx)
	//获取待发布的 configFileRelease 信息
	toPublishFile, err := cs.storage.GetConfigFile(tx, namespace, group, fileName)

	requestID, _ := ctx.Value(utils.StringContext("request-id")).(string)
	if err != nil {
		log.ConfigScope().Error("[Config][Service] get config file error.",
			zap.String("request-id", requestID),
			zap.String("namespace", namespace),
			zap.String("group", group),
			zap.String("fileName", fileName),
			zap.Error(err))

		cs.recordReleaseFail(transferConfigFileReleaseAPIModel2StoreModel(configFileRelease))

		return api.NewConfigFileResponse(api.StoreLayerException, nil)
	}

	if toPublishFile == nil {
		return api.NewConfigFileResponse(api.NotFoundResource, nil)
	}

	md5 := utils2.CalMd5(toPublishFile.Content)

	//获取 configFileRelease 信息
	managedFileRelease, err := cs.storage.GetConfigFileReleaseWithAllFlag(tx, namespace, group, fileName)
	if err != nil {
		log.ConfigScope().Error("[Config][Service] get config file release error.",
			zap.String("request-id", requestID),
			zap.String("namespace", namespace),
			zap.String("group", group),
			zap.String("fileName", fileName),
			zap.Error(err))

		cs.recordReleaseFail(transferConfigFileReleaseAPIModel2StoreModel(configFileRelease))

		return api.NewConfigFileResponse(api.StoreLayerException, nil)
	}

	//第一次发布
	if managedFileRelease == nil {
		fileRelease := &model.ConfigFileRelease{
			Name:      configFileRelease.Name.GetValue(),
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

		createdFileRelease, err := cs.storage.CreateConfigFileRelease(tx, fileRelease)
		if err != nil {
			log.ConfigScope().Error("[Config][Service] create config file release error.",
				zap.String("request-id", requestID),
				zap.String("namespace", namespace),
				zap.String("group", group),
				zap.String("fileName", fileName),
				zap.Error(err))

			cs.recordReleaseFail(transferConfigFileReleaseAPIModel2StoreModel(configFileRelease))

			return api.NewConfigFileResponse(api.StoreLayerException, nil)
		}

		cs.RecordConfigFileReleaseHistory(ctx, createdFileRelease, utils.ReleaseTypeNormal, utils.ReleaseStatusSuccess)

		return api.NewConfigFileReleaseResponse(api.ExecuteSuccess,
			transferConfigFileReleaseStoreModel2APIModel(createdFileRelease))
	}

	//更新发布
	fileRelease := &model.ConfigFileRelease{
		Name:      configFileRelease.Name.GetValue(),
		Namespace: namespace,
		Group:     group,
		FileName:  fileName,
		Content:   toPublishFile.Content,
		Comment:   configFileRelease.Comment.GetValue(),
		Md5:       md5,
		Version:   managedFileRelease.Version + 1,
		ModifyBy:  configFileRelease.CreateBy.GetValue(),
	}

	updatedFileRelease, err := cs.storage.UpdateConfigFileRelease(tx, fileRelease)
	if err != nil {
		log.ConfigScope().Error("[Config][Service] update config file release error.",
			zap.String("request-id", requestID),
			zap.String("namespace", namespace),
			zap.String("group", group),
			zap.String("fileName", fileName),
			zap.Error(err))

		cs.recordReleaseFail(transferConfigFileReleaseAPIModel2StoreModel(configFileRelease))

		return api.NewConfigFileResponse(api.StoreLayerException, nil)
	}

	cs.RecordConfigFileReleaseHistory(ctx, updatedFileRelease, utils.ReleaseTypeNormal, utils.ReleaseStatusSuccess)

	return api.NewConfigFileReleaseResponse(api.ExecuteSuccess,
		transferConfigFileReleaseStoreModel2APIModel(updatedFileRelease))
}

// GetConfigFileRelease 获取配置文件发布内容
func (cs *Impl) GetConfigFileRelease(ctx context.Context, namespace, group, fileName string) *api.ConfigResponse {
	if err := utils2.CheckResourceName(utils.NewStringValue(fileName)); err != nil {
		return api.NewConfigFileResponse(api.InvalidConfigFileName, nil)
	}

	if err := utils2.CheckResourceName(utils.NewStringValue(namespace)); err != nil {
		return api.NewConfigFileResponse(api.InvalidNamespaceName, nil)
	}

	if err := utils2.CheckResourceName(utils.NewStringValue(group)); err != nil {
		return api.NewConfigFileResponse(api.InvalidConfigFileGroupName, nil)
	}

	fileRelease, err := cs.storage.GetConfigFileRelease(nil, namespace, group, fileName)

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
func (cs *Impl) DeleteConfigFileRelease(ctx context.Context, namespace, group, fileName, deleteBy string) *api.ConfigResponse {
	if err := utils2.CheckResourceName(utils.NewStringValue(fileName)); err != nil {
		return api.NewConfigFileResponse(api.InvalidConfigFileName, nil)
	}

	if err := utils2.CheckResourceName(utils.NewStringValue(namespace)); err != nil {
		return api.NewConfigFileResponse(api.InvalidNamespaceName, nil)
	}

	if err := utils2.CheckResourceName(utils.NewStringValue(group)); err != nil {
		return api.NewConfigFileResponse(api.InvalidConfigFileGroupName, nil)
	}

	err := cs.storage.DeleteConfigFileRelease(cs.getTx(ctx), namespace, group, fileName, deleteBy)

	requestID, _ := ctx.Value(utils.StringContext("request-id")).(string)
	if err != nil {
		log.ConfigScope().Error("[Config][Service] delete config file release error.",
			zap.String("request-id", requestID),
			zap.String("namespace", namespace),
			zap.String("group", group),
			zap.String("fileName", fileName),
			zap.Error(err))

		cs.RecordConfigFileReleaseHistory(nil, &model.ConfigFileRelease{
			Namespace: namespace,
			Group:     group,
			FileName:  fileName,
			ModifyBy:  deleteBy,
		}, utils.ReleaseTypeDelete, utils.ReleaseStatusFail)

		return api.NewConfigFileResponse(api.StoreLayerException, nil)
	}

	cs.RecordConfigFileReleaseHistory(ctx, &model.ConfigFileRelease{
		Namespace: namespace,
		Group:     group,
		FileName:  fileName,
		ModifyBy:  deleteBy,
	}, utils.ReleaseTypeDelete, utils.ReleaseStatusSuccess)

	return api.NewConfigFileReleaseResponse(api.ExecuteSuccess, nil)
}

func (cs *Impl) recordReleaseFail(configFileRelease *model.ConfigFileRelease) {
	cs.RecordConfigFileReleaseHistory(nil, configFileRelease, utils.ReleaseTypeNormal, utils.ReleaseStatusFail)
}

func transferConfigFileReleaseAPIModel2StoreModel(release *api.ConfigFileRelease) *model.ConfigFileRelease {
	if release == nil {
		return nil
	}

	return &model.ConfigFileRelease{
		Id:        release.Id.GetValue(),
		Namespace: release.Namespace.GetValue(),
		Group:     release.Group.GetValue(),
		FileName:  release.FileName.GetValue(),
		Content:   release.Content.GetValue(),
		Comment:   release.Comment.GetValue(),
		Md5:       release.Md5.GetValue(),
		Version:   release.Version.GetValue(),
		CreateBy:  release.CreateBy.GetValue(),
		ModifyBy:  release.ModifyBy.GetValue(),
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
