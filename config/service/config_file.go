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

package service

import (
	"context"
	"errors"
	"strings"

	"go.uber.org/zap"

	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/common/time"
	"github.com/polarismesh/polaris-server/common/utils"
	utils2 "github.com/polarismesh/polaris-server/config/utils"
)

// CreateConfigFile 创建配置文件
func (cs *Impl) CreateConfigFile(ctx context.Context, configFile *api.ConfigFile) *api.ConfigResponse {
	if configFile.Format.GetValue() == "" {
		configFile.Format = utils.NewStringValue(utils.FileFormatText)
	}

	if checkRsp := checkConfigFileParams(configFile); checkRsp != nil {
		return checkRsp
	}

	namespace := configFile.Namespace.GetValue()
	group := configFile.Group.GetValue()
	name := configFile.Name.GetValue()

	requestID, _ := ctx.Value(utils.StringContext("request-id")).(string)

	//如果 namespace 不存在则自动创建
	if err := cs.createNamespaceIfAbsent(namespace, configFile.CreateBy.GetValue(), requestID); err != nil {
		log.ConfigScope().Error("[Config][Service] create config file error because of create namespace failed.",
			zap.String("request-id", requestID),
			zap.String("namespace", namespace),
			zap.String("group", group),
			zap.String("name", name),
			zap.Error(err))
		return api.NewConfigFileResponse(api.StoreLayerException, configFile)
	}

	//如果配置文件组不存在则自动创建
	createGroupRsp := cs.CreateConfigFileGroupIfAbsent(ctx, &api.ConfigFileGroup{
		Namespace: configFile.Namespace,
		Name:      configFile.Group,
		CreateBy:  configFile.CreateBy,
		Comment:   utils.NewStringValue("auto created"),
	})

	if createGroupRsp.Code.GetValue() != api.ExecuteSuccess {
		return api.NewConfigFileResponse(createGroupRsp.Code.GetValue(), configFile)
	}

	managedFile, err := cs.storage.GetConfigFile(cs.getTx(ctx), namespace, group, name)

	if err != nil {
		log.ConfigScope().Error("[Config][Service] get config file error.",
			zap.String("request-id", requestID),
			zap.String("namespace", namespace),
			zap.String("group", group),
			zap.String("name", name),
			zap.Error(err))

		return api.NewConfigFileResponse(api.StoreLayerException, configFile)
	}

	if managedFile != nil {
		return api.NewConfigFileResponse(api.ExistedResource, configFile)
	}

	fileStoreModel := transferConfigFileAPIModel2StoreModel(configFile)
	fileStoreModel.ModifyBy = fileStoreModel.CreateBy

	//创建配置文件
	createdFile, err := cs.storage.CreateConfigFile(cs.getTx(ctx), fileStoreModel)
	if err != nil {
		log.ConfigScope().Error("[Config][Service] create config file error.",
			zap.String("request-id", requestID),
			zap.String("namespace", namespace),
			zap.String("group", group),
			zap.String("name", name),
			zap.Error(err))
		return api.NewConfigFileResponse(api.StoreLayerException, configFile)
	}

	//创建配置文件标签
	response, success := cs.createOrUpdateConfigFileTags(ctx, configFile, fileStoreModel.ModifyBy, requestID)
	if !success {
		return response
	}

	//创建成功
	log.ConfigScope().Info("[Config][Service] create config file success.",
		zap.String("request-id", requestID),
		zap.String("namespace", namespace),
		zap.String("group", group),
		zap.String("name", name),
		zap.Error(err))

	return api.NewConfigFileResponse(api.ExecuteSuccess, transferConfigFileStoreModel2APIModel(createdFile))
}

// GetConfigFileBaseInfo 获取配置文件，只返回基础元信息
func (cs *Impl) GetConfigFileBaseInfo(ctx context.Context, namespace, group, name string) *api.ConfigResponse {
	if err := utils2.CheckResourceName(utils.NewStringValue(namespace)); err != nil {
		return api.NewConfigFileResponse(api.InvalidNamespaceName, nil)
	}

	if err := utils2.CheckResourceName(utils.NewStringValue(group)); err != nil {
		return api.NewConfigFileResponse(api.InvalidConfigFileGroupName, nil)
	}

	if err := utils2.CheckResourceName(utils.NewStringValue(name)); err != nil {
		return api.NewConfigFileResponse(api.InvalidConfigFileName, nil)
	}

	requestID, _ := ctx.Value(utils.StringContext("request-id")).(string)

	file, err := cs.storage.GetConfigFile(nil, namespace, group, name)
	if err != nil {
		log.ConfigScope().Error("[Config][Service] get config file error.",
			zap.String("request-id", requestID),
			zap.String("namespace", namespace),
			zap.String("group", group),
			zap.String("name", name),
			zap.Error(err))

		return api.NewConfigFileResponse(api.StoreLayerException, nil)
	}

	if file == nil {
		return api.NewConfigFileResponse(api.NotFoundResource, nil)
	}

	return api.NewConfigFileResponse(api.ExecuteSuccess, transferConfigFileStoreModel2APIModel(file))
}

// GetConfigFileRichInfo 获取单个配置文件基础信息，包含发布状态等信息
func (cs *Impl) GetConfigFileRichInfo(ctx context.Context, namespace, group, name string) *api.ConfigResponse {
	requestID, _ := ctx.Value(utils.StringContext("request-id")).(string)

	configFileBaseInfoRsp := cs.GetConfigFileBaseInfo(ctx, namespace, group, name)
	if configFileBaseInfoRsp.Code.GetValue() != api.ExecuteSuccess {
		log.ConfigScope().Error("[Config][Service] get config file release error.",
			zap.String("request-id", requestID),
			zap.String("namespace", namespace),
			zap.String("group", group),
			zap.String("name", name))
		return api.NewConfigFileResponse(configFileBaseInfoRsp.Code.GetValue(), nil)
	}

	configFileBaseInfo := configFileBaseInfoRsp.ConfigFile

	//填充发布信息、标签信息等
	err := cs.enrich(ctx, configFileBaseInfo, requestID)

	if err != nil {
		return api.NewConfigFileResponse(api.StoreLayerException, nil)
	}

	return api.NewConfigFileResponse(api.ExecuteSuccess, configFileBaseInfo)
}

// SearchConfigFile 查询配置文件
func (cs *Impl) SearchConfigFile(ctx context.Context, namespace, group, name, tags string, offset, limit int) *api.ConfigBatchQueryResponse {
	if err := utils2.CheckResourceName(utils.NewStringValue(namespace)); err != nil {
		return api.NewConfigFileBatchQueryResponse(api.InvalidNamespaceName, 0, nil)
	}

	if offset < 0 || limit <= 0 || limit > MaxPageSize {
		return api.NewConfigFileBatchQueryResponse(api.InvalidParameter, 0, nil)
	}

	if len(tags) == 0 {
		return cs.queryConfigFileWithoutTags(ctx, namespace, group, name, offset, limit)
	}

	//按tag搜索，内存分页
	var tagKVs []string
	tagKVs = strings.Split(tags, ",")
	if len(tagKVs)%2 != 0 {
		return api.NewConfigFileBatchQueryResponse(api.InvalidConfigFileTags, 0, nil)
	}

	requestID, _ := ctx.Value(utils.StringContext("request-id")).(string)

	count, files, err := cs.QueryConfigFileByTags(ctx, namespace, group, name, offset, limit, tagKVs...)
	if err != nil {
		log.ConfigScope().Error("[Config][Service] query config file tags error.",
			zap.String("request-id", requestID),
			zap.String("namespace", namespace),
			zap.String("group", group),
			zap.String("fileName", name),
			zap.Error(err))
		return api.NewConfigFileBatchQueryResponse(api.StoreLayerException, 0, nil)
	}

	//渲染配置文件，因为从 tag 表获取的只有主键信息
	var enrichedFiles []*api.ConfigFile
	for _, file := range files {
		rsp := cs.GetConfigFileRichInfo(ctx, file.Namespace, file.Group, file.FileName)
		if rsp.Code.GetValue() != api.ExecuteSuccess {
			return api.NewConfigFileBatchQueryResponse(rsp.Code.GetValue(), 0, nil)
		}
		enrichedFiles = append(enrichedFiles, rsp.ConfigFile)
	}

	return api.NewConfigFileBatchQueryResponse(api.ExecuteSuccess, uint32(count), enrichedFiles)
}

func (cs *Impl) queryConfigFileWithoutTags(ctx context.Context, namespace string, group string, name string, offset, limit int) *api.ConfigBatchQueryResponse {
	requestID, _ := ctx.Value(utils.StringContext("request-id")).(string)
	count, files, err := cs.storage.QueryConfigFiles(namespace, group, name, offset, limit)
	if err != nil {
		log.ConfigScope().Error("[Config][Service]search config file error.",
			zap.String("request-id", requestID),
			zap.String("namespace", namespace),
			zap.String("group", group),
			zap.String("name", name),
			zap.Error(err))

		return api.NewConfigFileBatchQueryResponse(api.StoreLayerException, 0, nil)
	}

	if len(files) == 0 {
		return api.NewConfigFileBatchQueryResponse(api.ExecuteSuccess, count, nil)
	}

	var fileAPIModels []*api.ConfigFile
	for _, file := range files {
		baseFile := transferConfigFileStoreModel2APIModel(file)
		err = cs.enrich(ctx, baseFile, requestID)
		if err != nil {
			return api.NewConfigFileBatchQueryResponse(api.StoreLayerException, 0, nil)
		}
		fileAPIModels = append(fileAPIModels, baseFile)
	}

	return api.NewConfigFileBatchQueryResponse(api.ExecuteSuccess, count, fileAPIModels)
}

// UpdateConfigFile 更新配置文件
func (cs *Impl) UpdateConfigFile(ctx context.Context, configFile *api.ConfigFile) *api.ConfigResponse {
	if configFile.Format.GetValue() == "" {
		configFile.Format = utils.NewStringValue(utils.FileFormatText)
	}

	if checkRsp := checkConfigFileParams(configFile); checkRsp != nil {
		return checkRsp
	}

	namespace := configFile.Namespace.GetValue()
	group := configFile.Group.GetValue()
	name := configFile.Name.GetValue()

	managedFile, err := cs.storage.GetConfigFile(cs.getTx(ctx), namespace, group, name)

	requestID, _ := ctx.Value(utils.StringContext("request-id")).(string)
	if err != nil {
		log.ConfigScope().Error("[Config][Service] get config file error.",
			zap.String("request-id", requestID),
			zap.String("namespace", namespace),
			zap.String("group", group),
			zap.String("name", name),
			zap.Error(err))

		return api.NewConfigFileResponse(api.StoreLayerException, configFile)
	}

	if managedFile == nil {
		return api.NewConfigFileResponse(api.NotFoundResource, configFile)
	}

	toUpdateFile := transferConfigFileAPIModel2StoreModel(configFile)
	toUpdateFile.ModifyBy = configFile.ModifyBy.GetValue()

	updatedFile, err := cs.storage.UpdateConfigFile(cs.getTx(ctx), toUpdateFile)
	if err != nil {
		log.ConfigScope().Error("[Config][Service] update config file error.",
			zap.String("request-id", requestID),
			zap.String("namespace", namespace),
			zap.String("group", group),
			zap.String("name", name),
			zap.Error(err))

		return api.NewConfigFileResponse(api.StoreLayerException, configFile)
	}

	response, success := cs.createOrUpdateConfigFileTags(ctx, configFile, toUpdateFile.ModifyBy, requestID)
	if !success {
		return response
	}

	baseFile := transferConfigFileStoreModel2APIModel(updatedFile)
	_ = cs.enrich(ctx, baseFile, requestID)

	return api.NewConfigFileResponse(api.ExecuteSuccess, baseFile)
}

// DeleteConfigFile 删除配置文件，删除配置文件同时会通知客户端 Not_Found
func (cs *Impl) DeleteConfigFile(ctx context.Context, namespace, group, name, deleteBy string) *api.ConfigResponse {
	if err := utils2.CheckResourceName(utils.NewStringValue(namespace)); err != nil {
		return api.NewConfigFileResponse(api.InvalidNamespaceName, nil)
	}

	if err := utils2.CheckResourceName(utils.NewStringValue(group)); err != nil {
		return api.NewConfigFileResponse(api.InvalidConfigFileGroupName, nil)
	}

	if err := utils2.CheckResourceName(utils.NewStringValue(name)); err != nil {
		return api.NewConfigFileResponse(api.InvalidConfigFileName, nil)
	}

	requestID, _ := ctx.Value(utils.StringContext("request-id")).(string)

	log.ConfigScope().Info("[Config][Service] delete config file.",
		zap.String("request-id", requestID),
		zap.String("namespace", namespace),
		zap.String("group", group),
		zap.String("name", name))

	file, err := cs.storage.GetConfigFile(nil, namespace, group, name)
	if err != nil {
		log.ConfigScope().Error("[Config][Service] get config file error.",
			zap.String("request-id", requestID),
			zap.String("namespace", namespace),
			zap.String("group", group),
			zap.String("name", name),
			zap.Error(err))
		return api.NewConfigFileResponse(api.StoreLayerException, nil)
	}

	if file == nil {
		return api.NewConfigFileResponse(api.ExecuteSuccess, nil)
	}

	tx, newCtx, _ := cs.StartTxAndSetToContext(ctx)
	defer func() { _ = tx.Rollback() }()

	//1. 删除配置文件发布内容
	deleteFileReleaseRsp := cs.DeleteConfigFileRelease(newCtx, namespace, group, name, deleteBy)
	if deleteFileReleaseRsp.Code.GetValue() != api.ExecuteSuccess {
		return api.NewConfigFileResponse(deleteFileReleaseRsp.Code.GetValue(), nil)
	}

	//2. 删除配置文件
	err = cs.storage.DeleteConfigFile(tx, namespace, group, name)
	if err != nil {
		log.ConfigScope().Error("[Config][Service] delete config file error.",
			zap.String("request-id", requestID),
			zap.String("namespace", namespace),
			zap.String("group", group),
			zap.String("name", name),
			zap.Error(err))
		return api.NewConfigFileResponse(api.StoreLayerException, nil)
	}

	//3. 删除配置文件关联的 tag
	err = cs.DeleteTagByConfigFile(ctx, namespace, group, name)
	if err != nil {
		log.ConfigScope().Error("[Config][Service] delete config file tags error.",
			zap.String("request-id", requestID),
			zap.String("namespace", namespace),
			zap.String("group", group),
			zap.String("name", name),
			zap.Error(err))
		return api.NewConfigFileResponse(api.StoreLayerException, nil)
	}

	err = tx.Commit()
	if err != nil {
		log.ConfigScope().Error("[Config][Service] commit delete config file tx error.",
			zap.String("request-id", requestID),
			zap.String("namespace", namespace),
			zap.String("group", group),
			zap.String("name", name),
			zap.Error(err))
		return api.NewConfigFileResponse(api.StoreLayerException, nil)
	}

	return api.NewConfigFileResponse(api.ExecuteSuccess, nil)
}

func checkConfigFileParams(configFile *api.ConfigFile) *api.ConfigResponse {
	if configFile == nil {
		return api.NewConfigFileResponse(api.InvalidParameter, configFile)
	}

	if err := utils2.CheckResourceName(configFile.Name); err != nil {
		return api.NewConfigFileResponse(api.InvalidConfigFileName, configFile)
	}

	if err := utils2.CheckResourceName(configFile.Namespace); err != nil {
		return api.NewConfigFileResponse(api.InvalidNamespaceName, configFile)
	}

	if err := utils2.CheckContentLength(configFile.Content.GetValue()); err != nil {
		return api.NewConfigFileResponse(api.InvalidConfigFileContentLength, configFile)
	}

	if !utils.IsValidFileFormat(configFile.Format.GetValue()) {
		return api.NewConfigFileResponse(api.InvalidConfigFileFormat, configFile)
	}

	if len(configFile.Tags) > 0 {
		for _, tag := range configFile.Tags {
			if tag.Key.GetValue() == "" || tag.Value.GetValue() == "" {
				return api.NewConfigFileResponse(api.InvalidConfigFileTags, configFile)
			}
		}
	}

	return nil
}

func transferConfigFileAPIModel2StoreModel(file *api.ConfigFile) *model.ConfigFile {
	return &model.ConfigFile{
		Name:      file.Name.GetValue(),
		Namespace: file.Namespace.GetValue(),
		Group:     file.Group.GetValue(),
		Content:   file.Content.GetValue(),
		Comment:   file.Comment.GetValue(),
		Format:    file.Format.GetValue(),
		CreateBy:  file.CreateBy.GetValue(),
	}
}

func transferConfigFileStoreModel2APIModel(file *model.ConfigFile) *api.ConfigFile {
	if file == nil {
		return nil
	}
	return &api.ConfigFile{
		Id:         utils.NewUInt64Value(file.Id),
		Name:       utils.NewStringValue(file.Name),
		Namespace:  utils.NewStringValue(file.Namespace),
		Group:      utils.NewStringValue(file.Group),
		Content:    utils.NewStringValue(file.Content),
		Comment:    utils.NewStringValue(file.Comment),
		Format:     utils.NewStringValue(file.Format),
		CreateBy:   utils.NewStringValue(file.CreateBy),
		CreateTime: utils.NewStringValue(time.Time2String(file.CreateTime)),
		ModifyBy:   utils.NewStringValue(file.ModifyBy),
		ModifyTime: utils.NewStringValue(time.Time2String(file.ModifyTime)),
	}
}

func (cs *Impl) createOrUpdateConfigFileTags(ctx context.Context, configFile *api.ConfigFile, operator, requestID string) (*api.ConfigResponse, bool) {
	namespace := configFile.Namespace.GetValue()
	group := configFile.Group.GetValue()
	name := configFile.Name.GetValue()

	var tags []string
	for _, tag := range configFile.Tags {
		tags = append(tags, tag.Key.GetValue())
		tags = append(tags, tag.Value.GetValue())
	}
	err := cs.CreateConfigFileTags(ctx, namespace, group, name, operator, tags...)
	if err != nil {
		log.ConfigScope().Error("[Config][Service] create or update config file tags error.",
			zap.String("request-id", requestID),
			zap.String("namespace", namespace),
			zap.String("group", group),
			zap.String("fileName", name),
			zap.Error(err))
		return api.NewConfigFileResponse(api.StoreLayerException, configFile), false
	}
	return nil, true
}

func (cs *Impl) enrich(ctx context.Context, baseConfigFile *api.ConfigFile, requestID string) error {
	namespace := baseConfigFile.Namespace.GetValue()
	group := baseConfigFile.Group.GetValue()
	name := baseConfigFile.Name.GetValue()

	//填充发布信息
	latestReleaseRsp := cs.GetConfigFileLatestReleaseHistory(ctx, namespace, group, name)
	if latestReleaseRsp.Code.GetValue() != api.ExecuteSuccess {
		log.ConfigScope().Error("[Config][Service] get config file latest release error.",
			zap.String("request-id", requestID),
			zap.String("namespace", namespace),
			zap.String("group", group),
			zap.String("name", name))
		return errors.New("enrich config file release info error")
	}

	latestRelease := latestReleaseRsp.ConfigFileReleaseHistory
	if latestRelease != nil && latestRelease.Type.GetValue() == utils.ReleaseTypeNormal {
		baseConfigFile.ReleaseBy = latestRelease.CreateBy
		baseConfigFile.ReleaseTime = latestRelease.CreateTime

		//如果最后一次发布的内容和当前文件内容一致，则展示最后一次发布状态。否则说明文件有修改，待发布
		if latestRelease.Content.GetValue() == baseConfigFile.Content.GetValue() {
			baseConfigFile.Status = latestRelease.Status
		} else {
			baseConfigFile.Status = utils.NewStringValue(utils.ReleaseStatusToRelease)
		}
	} else {
		//如果从来没有发布过，也是待发布状态
		baseConfigFile.Status = utils.NewStringValue(utils.ReleaseStatusToRelease)
		baseConfigFile.ReleaseBy = utils.NewStringValue("")
		baseConfigFile.ReleaseTime = utils.NewStringValue("")
	}

	//填充标签信息
	tags, err := cs.QueryTagsByConfigFileWithAPIModels(ctx, namespace, group, name)
	if err != nil {
		log.ConfigScope().Error("[Config][Service] create config file error.",
			zap.String("request-id", requestID),
			zap.String("namespace", namespace),
			zap.String("group", group),
			zap.String("name", name),
			zap.Error(err))
		return err
	}

	baseConfigFile.Tags = tags
	return nil
}
