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

// CreateConfigFileGroup 创建配置文件组
func (cs *Impl) CreateConfigFileGroup(ctx context.Context, configFileGroup *api.ConfigFileGroup) *api.ConfigResponse {
	requestID, _ := ctx.Value(utils.StringContext("request-id")).(string)

	if checkError := checkConfigFileGroupParams(configFileGroup); checkError != nil {
		return checkError
	}

	namespace := configFileGroup.Namespace.GetValue()
	groupName := configFileGroup.Name.GetValue()

	//如果 namespace 不存在则自动创建
	if err := cs.createNamespaceIfAbsent(namespace, configFileGroup.CreateBy.GetValue(), requestID); err != nil {
		log.ConfigScope().Error("[Config][Service] create config file group error because of create namespace failed.",
			zap.String("request-id", requestID),
			zap.String("namespace", namespace),
			zap.String("group", groupName),
			zap.Error(err))
		return api.NewConfigFileGroupResponse(api.StoreLayerException, configFileGroup)
	}

	fileGroup, err := cs.storage.GetConfigFileGroup(namespace, groupName)
	if err != nil {
		log.ConfigScope().Error("[Config][Service] get config file group error.",
			zap.String("request-id", requestID),
			zap.Error(err))
		return api.NewConfigFileGroupResponse(api.StoreLayerException, configFileGroup)
	}

	if fileGroup != nil {
		return api.NewConfigFileGroupResponse(api.ExistedResource, configFileGroup)
	}

	toCreateGroup := transferConfigFileGroupAPIModel2StoreModel(configFileGroup)
	toCreateGroup.ModifyBy = toCreateGroup.CreateBy

	createdGroup, err := cs.storage.CreateConfigFileGroup(toCreateGroup)
	if err != nil {
		log.ConfigScope().Error("[Config][Service] create config file group error.",
			zap.String("request-id", requestID),
			zap.String("namespace", namespace),
			zap.String("groupName", groupName),
			zap.Error(err))
		return api.NewConfigFileGroupResponse(api.StoreLayerException, configFileGroup)
	}

	log.ConfigScope().Info("[Config][Service] create config file group successful.",
		zap.String("request-id", requestID),
		zap.String("namespace", namespace),
		zap.String("groupName", groupName))

	return api.NewConfigFileGroupResponse(api.ExecuteSuccess, transferConfigFileGroupStoreModel2APIModel(createdGroup))
}

// CreateConfigFileGroupIfAbsent 如果不存在配置文件组，则自动创建
func (cs *Impl) CreateConfigFileGroupIfAbsent(ctx context.Context, configFileGroup *api.ConfigFileGroup) *api.ConfigResponse {
	namespace := configFileGroup.Namespace.GetValue()
	name := configFileGroup.Name.GetValue()

	group, err := cs.storage.GetConfigFileGroup(namespace, name)
	if err != nil {
		requestID, _ := ctx.Value(utils.StringContext("request-id")).(string)
		log.ConfigScope().Error("[Config][Service] query config file group error.",
			zap.String("request-id", requestID),
			zap.String("namespace", namespace),
			zap.String("groupName", name),
			zap.Error(err))

		return api.NewConfigFileGroupResponse(api.StoreLayerException, nil)
	}

	if group != nil {
		return api.NewConfigFileGroupResponse(api.ExecuteSuccess, transferConfigFileGroupStoreModel2APIModel(group))
	}

	return cs.CreateConfigFileGroup(ctx, configFileGroup)
}

// QueryConfigFileGroups 查询配置文件组
func (cs *Impl) QueryConfigFileGroups(ctx context.Context, namespace, name string, offset, limit uint32) *api.ConfigBatchQueryResponse {
	if err := utils2.CheckResourceName(utils.NewStringValue(namespace)); err != nil {
		return api.NewConfigFileGroupBatchQueryResponse(api.InvalidNamespaceName, 0, nil)
	}

	if offset < 0 || limit <= 0 || limit > MaxPageSize {
		return api.NewConfigFileGroupBatchQueryResponse(api.InvalidParameter, 0, nil)
	}

	count, groups, err := cs.storage.QueryConfigFileGroups(namespace, name, offset, limit)
	if err != nil {
		requestID, _ := ctx.Value(utils.StringContext("request-id")).(string)
		log.ConfigScope().Error("[Config][Service] query config file group error.",
			zap.String("request-id", requestID),
			zap.String("namespace", namespace),
			zap.String("groupName", name),
			zap.Error(err))

		return api.NewConfigFileGroupBatchQueryResponse(api.StoreLayerException, 0, nil)
	}

	var groupAPIModels []*api.ConfigFileGroup

	if len(groups) == 0 {
		groupAPIModels = nil
		return api.NewConfigFileGroupBatchQueryResponse(api.ExecuteSuccess, count, groupAPIModels)
	}

	for _, groupStoreModel := range groups {
		groupAPIModels = append(groupAPIModels, transferConfigFileGroupStoreModel2APIModel(groupStoreModel))
	}

	return api.NewConfigFileGroupBatchQueryResponse(api.ExecuteSuccess, count, groupAPIModels)
}

// DeleteConfigFileGroup 删除配置文件组
func (cs *Impl) DeleteConfigFileGroup(ctx context.Context, namespace, name string) *api.ConfigResponse {
	if err := utils2.CheckResourceName(utils.NewStringValue(namespace)); err != nil {
		return api.NewConfigFileGroupResponse(api.InvalidNamespaceName, nil)
	}
	if err := utils2.CheckResourceName(utils.NewStringValue(name)); err != nil {
		return api.NewConfigFileGroupResponse(api.InvalidConfigFileGroupName, nil)
	}

	requestID, _ := ctx.Value(utils.StringContext("request-id")).(string)
	log.ConfigScope().Info("[Config][Service] delete config file group. ",
		zap.String("request-id", requestID),
		zap.String("namespace", namespace),
		zap.String("name", name))

	err := cs.storage.DeleteConfigFileGroup(namespace, name)
	if err != nil {
		log.ConfigScope().Error("[Config][Service] delete config file group failed. ",
			zap.String("request-id", requestID),
			zap.String("namespace", namespace),
			zap.String("name", name),
			zap.Error(err))

		return api.NewConfigFileGroupResponse(api.StoreLayerException, nil)
	}

	return api.NewConfigFileGroupResponse(api.ExecuteSuccess, nil)
}

// UpdateConfigFileGroup 更新配置文件组
func (cs *Impl) UpdateConfigFileGroup(ctx context.Context, configFileGroup *api.ConfigFileGroup) *api.ConfigResponse {
	if checkError := checkConfigFileGroupParams(configFileGroup); checkError != nil {
		return checkError
	}

	namespace := configFileGroup.Namespace.GetValue()
	groupName := configFileGroup.Name.GetValue()

	fileGroup, err := cs.storage.GetConfigFileGroup(namespace, groupName)

	requestID, _ := ctx.Value(utils.StringContext("request-id")).(string)
	if err != nil {
		log.ConfigScope().Error("[Config][Service] get config file group failed. ",
			zap.String("request-id", requestID),
			zap.String("namespace", namespace),
			zap.String("name", groupName),
			zap.Error(err))

		return api.NewConfigFileGroupResponse(api.StoreLayerException, configFileGroup)
	}

	if fileGroup == nil {
		return api.NewConfigFileGroupResponse(api.NotFoundResource, configFileGroup)
	}

	toUpdateGroup := transferConfigFileGroupAPIModel2StoreModel(configFileGroup)
	toUpdateGroup.ModifyBy = configFileGroup.ModifyBy.GetValue()

	updatedGroup, err := cs.storage.UpdateConfigFileGroup(toUpdateGroup)
	if err != nil {
		log.ConfigScope().Error("[Config][Service] update config file group failed. ",
			zap.String("request-id", requestID),
			zap.String("namespace", namespace),
			zap.String("name", groupName),
			zap.Error(err))

		return api.NewConfigFileGroupResponse(api.StoreLayerException, configFileGroup)
	}

	return api.NewConfigFileGroupResponse(api.ExecuteSuccess, transferConfigFileGroupStoreModel2APIModel(updatedGroup))
}

func checkConfigFileGroupParams(configFileGroup *api.ConfigFileGroup) *api.ConfigResponse {
	if configFileGroup == nil {
		return api.NewConfigFileGroupResponse(api.InvalidParameter, configFileGroup)
	}

	if err := utils2.CheckResourceName(configFileGroup.Name); err != nil {
		return api.NewConfigFileGroupResponse(api.InvalidConfigFileGroupName, configFileGroup)
	}

	if err := utils2.CheckResourceName(configFileGroup.Namespace); err != nil {
		return api.NewConfigFileGroupResponse(api.InvalidNamespaceName, configFileGroup)
	}

	return nil
}

func transferConfigFileGroupAPIModel2StoreModel(group *api.ConfigFileGroup) *model.ConfigFileGroup {
	return &model.ConfigFileGroup{
		Name:      group.Name.GetValue(),
		Namespace: group.Namespace.GetValue(),
		Comment:   group.Comment.Value,
		CreateBy:  group.CreateBy.GetValue(),
	}
}

func transferConfigFileGroupStoreModel2APIModel(group *model.ConfigFileGroup) *api.ConfigFileGroup {
	if group == nil {
		return nil
	}
	return &api.ConfigFileGroup{
		Id:         utils.NewUInt64Value(group.Id),
		Name:       utils.NewStringValue(group.Name),
		Namespace:  utils.NewStringValue(group.Namespace),
		Comment:    utils.NewStringValue(group.Comment),
		CreateBy:   utils.NewStringValue(group.CreateBy),
		ModifyBy:   utils.NewStringValue(group.ModifyBy),
		CreateTime: utils.NewStringValue(time.Time2String(group.CreateTime)),
		ModifyTime: utils.NewStringValue(time.Time2String(group.ModifyTime)),
	}
}
