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
	"sort"
	"strings"

	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/common/time"
	"github.com/polarismesh/polaris-server/common/utils"
	utils2 "github.com/polarismesh/polaris-server/config/utils"
	"go.uber.org/zap"
)

// CreateConfigFileGroup 创建配置文件组
func (s *Server) CreateConfigFileGroup(ctx context.Context, configFileGroup *api.ConfigFileGroup) *api.ConfigResponse {
	requestID, _ := ctx.Value(utils.StringContext("request-id")).(string)

	if checkError := checkConfigFileGroupParams(configFileGroup); checkError != nil {
		return checkError
	}

	userName := utils.ParseUserName(ctx)
	configFileGroup.CreateBy = utils.NewStringValue(userName)
	configFileGroup.ModifyBy = utils.NewStringValue(userName)

	namespace := configFileGroup.Namespace.GetValue()
	groupName := configFileGroup.Name.GetValue()

	// 如果 namespace 不存在则自动创建
	if err := s.createNamespaceIfAbsent(namespace, configFileGroup.CreateBy.GetValue(), requestID); err != nil {
		log.ConfigScope().Error("[Config][Service] create config file group error because of create namespace failed.",
			zap.String("request-id", requestID),
			zap.String("namespace", namespace),
			zap.String("group", groupName),
			zap.Error(err))
		return api.NewConfigFileGroupResponse(api.StoreLayerException, configFileGroup)
	}

	fileGroup, err := s.storage.GetConfigFileGroup(namespace, groupName)
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

	createdGroup, err := s.storage.CreateConfigFileGroup(toCreateGroup)
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

// createConfigFileGroupIfAbsent 如果不存在配置文件组，则自动创建
func (s *Server) createConfigFileGroupIfAbsent(ctx context.Context, configFileGroup *api.ConfigFileGroup) *api.ConfigResponse {
	namespace := configFileGroup.Namespace.GetValue()
	name := configFileGroup.Name.GetValue()

	group, err := s.storage.GetConfigFileGroup(namespace, name)
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

	return s.CreateConfigFileGroup(ctx, configFileGroup)
}

// QueryConfigFileGroups 查询配置文件组
func (s *Server) QueryConfigFileGroups(ctx context.Context, namespace, groupName, fileName string, offset, limit uint32) *api.ConfigBatchQueryResponse {
	if offset < 0 || limit <= 0 || limit > MaxPageSize {
		return api.NewConfigFileGroupBatchQueryResponse(api.InvalidParameter, 0, nil)
	}

	// 按分组名搜索
	if fileName == "" {
		return s.queryByGroupName(ctx, namespace, groupName, offset, limit)
	}

	// 按文件搜索
	return s.queryByFileName(ctx, namespace, groupName, fileName, offset, limit)
}

func (s *Server) queryByGroupName(ctx context.Context, namespace string, groupName string, offset uint32, limit uint32) *api.ConfigBatchQueryResponse {
	count, groups, err := s.storage.QueryConfigFileGroups(namespace, groupName, offset, limit)
	if err != nil {
		requestID, _ := ctx.Value(utils.StringContext("request-id")).(string)
		log.ConfigScope().Error("[Config][Service] query config file group error.",
			zap.String("request-id", requestID),
			zap.String("namespace", namespace),
			zap.String("groupName", groupName),
			zap.Error(err))

		return api.NewConfigFileGroupBatchQueryResponse(api.StoreLayerException, 0, nil)
	}

	if len(groups) == 0 {
		return api.NewConfigFileGroupBatchQueryResponse(api.ExecuteSuccess, count, nil)
	}

	groupAPIModels, err := s.batchTransfer(ctx, groups)
	if err != nil {
		return api.NewConfigFileGroupBatchQueryResponse(api.StoreLayerException, 0, nil)
	}

	return api.NewConfigFileGroupBatchQueryResponse(api.ExecuteSuccess, count, groupAPIModels)
}

func (s *Server) queryByFileName(ctx context.Context, namespace, groupName, fileName string, offset uint32, limit uint32) *api.ConfigBatchQueryResponse {
	// 内存分页，先获取到所有配置文件
	rsp := s.queryConfigFileWithoutTags(ctx, namespace, groupName, fileName, 0, 10000)
	if rsp.Code.GetValue() != api.ExecuteSuccess {
		return rsp
	}

	// 获取所有的 group 信息
	groupMap := make(map[string]bool)
	for _, configFile := range rsp.ConfigFiles {
		// namespace+group 是唯一键
		groupMap[configFile.Namespace.Value+"+"+configFile.Group.Value] = true
	}

	if len(groupMap) == 0 {
		return api.NewConfigFileGroupBatchQueryResponse(api.ExecuteSuccess, 0, nil)
	}

	var distinctGroupNames []string
	for key := range groupMap {
		distinctGroupNames = append(distinctGroupNames, key)
	}

	// 按 groupName 字典排序
	sort.Strings(distinctGroupNames)

	// 分页
	total := len(distinctGroupNames)
	if int(offset) >= total {
		return api.NewConfigFileGroupBatchQueryResponse(api.ExecuteSuccess, uint32(total), nil)
	}

	var pageGroupNames []string
	if int(offset+limit) >= total {
		pageGroupNames = distinctGroupNames[offset:total]
	} else {
		pageGroupNames = distinctGroupNames[offset : offset+limit]
	}

	// 渲染
	var configFileGroups []*model.ConfigFileGroup
	for _, pageGroupName := range pageGroupNames {
		namespaceAndGroup := strings.Split(pageGroupName, "+")
		configFileGroup, err := s.storage.GetConfigFileGroup(namespaceAndGroup[0], namespaceAndGroup[1])
		if err != nil {
			requestID, _ := ctx.Value(utils.StringContext("request-id")).(string)
			log.ConfigScope().Error("[Config][Service] get config file group error.",
				zap.String("request-id", requestID),
				zap.String("namespace", namespaceAndGroup[0]),
				zap.String("name", namespaceAndGroup[1]),
				zap.Error(err))
			return api.NewConfigFileGroupBatchQueryResponse(api.StoreLayerException, 0, nil)
		}
		configFileGroups = append(configFileGroups, configFileGroup)
	}

	groupAPIModels, err := s.batchTransfer(ctx, configFileGroups)
	if err != nil {
		return api.NewConfigFileGroupBatchQueryResponse(api.StoreLayerException, 0, nil)
	}

	return api.NewConfigFileGroupBatchQueryResponse(api.ExecuteSuccess, uint32(total), groupAPIModels)
}

func (s *Server) batchTransfer(ctx context.Context, groups []*model.ConfigFileGroup) ([]*api.ConfigFileGroup, error) {
	var result []*api.ConfigFileGroup

	for _, groupStoreModel := range groups {
		configFileGroup := transferConfigFileGroupStoreModel2APIModel(groupStoreModel)
		// enrich config file count
		fileCount, err := s.storage.CountByConfigFileGroup(groupStoreModel.Namespace, groupStoreModel.Name)
		if err != nil {
			requestID, _ := ctx.Value(utils.StringContext("request-id")).(string)
			log.ConfigScope().Error("[Config][Service] get config file count for group error.",
				zap.String("request-id", requestID),
				zap.String("namespace", groupStoreModel.Namespace),
				zap.String("groupName", groupStoreModel.Name),
				zap.Error(err))
			return nil, err
		}
		configFileGroup.FileCount = utils.NewUInt64Value(fileCount)

		result = append(result, configFileGroup)
	}
	return result, nil
}

// DeleteConfigFileGroup 删除配置文件组
func (s *Server) DeleteConfigFileGroup(ctx context.Context, namespace, name string) *api.ConfigResponse {
	if err := utils2.CheckResourceName(utils.NewStringValue(namespace)); err != nil {
		return api.NewConfigFileGroupResponse(api.InvalidNamespaceName, nil)
	}
	if err := utils2.CheckResourceName(utils.NewStringValue(name)); err != nil {
		return api.NewConfigFileGroupResponse(api.InvalidConfigFileGroupName, nil)
	}

	operator := utils.ParseUserName(ctx)

	requestID, _ := ctx.Value(utils.StringContext("request-id")).(string)
	log.ConfigScope().Info("[Config][Service] delete config file group. ",
		zap.String("request-id", requestID),
		zap.String("namespace", namespace),
		zap.String("name", name))

	// 删除配置文件组，同时删除组下面所有的配置文件
	startOffset := uint32(0)
	hasMore := true
	for hasMore {
		searchRsp := s.SearchConfigFile(ctx, namespace, name, "", "", startOffset, MaxPageSize)
		if searchRsp.Code.GetValue() != api.ExecuteSuccess {
			log.ConfigScope().Error("[Config][Service] get group's config file failed. ",
				zap.String("request-id", requestID),
				zap.String("namespace", namespace),
				zap.String("name", name))
			return api.NewConfigFileGroupResponse(searchRsp.Code.GetValue(), nil)
		}
		configFiles := searchRsp.ConfigFiles

		deleteRsp := s.BatchDeleteConfigFile(ctx, configFiles, operator)
		if deleteRsp.Code.GetValue() != api.ExecuteSuccess {
			log.ConfigScope().Error("[Config][Service] batch delete group's config file failed. ",
				zap.String("request-id", requestID),
				zap.String("namespace", namespace),
				zap.String("name", name))
			return api.NewConfigFileGroupResponse(deleteRsp.Code.GetValue(), nil)
		}

		hasMore = len(searchRsp.ConfigFiles) >= MaxPageSize
		if hasMore {
			startOffset += MaxPageSize
		}
	}

	err := s.storage.DeleteConfigFileGroup(namespace, name)
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
func (s *Server) UpdateConfigFileGroup(ctx context.Context, configFileGroup *api.ConfigFileGroup) *api.ConfigResponse {
	if checkError := checkConfigFileGroupParams(configFileGroup); checkError != nil {
		return checkError
	}

	namespace := configFileGroup.Namespace.GetValue()
	groupName := configFileGroup.Name.GetValue()

	fileGroup, err := s.storage.GetConfigFileGroup(namespace, groupName)

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

	userName := utils.ParseUserName(ctx)
	configFileGroup.ModifyBy = utils.NewStringValue(userName)

	toUpdateGroup := transferConfigFileGroupAPIModel2StoreModel(configFileGroup)
	toUpdateGroup.ModifyBy = configFileGroup.ModifyBy.GetValue()

	updatedGroup, err := s.storage.UpdateConfigFileGroup(toUpdateGroup)
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
	var comment string
	if group.Comment != nil {
		comment = group.Comment.Value
	}
	var createBy string
	if group.CreateBy != nil {
		createBy = group.CreateBy.Value
	}
	return &model.ConfigFileGroup{
		Name:      group.Name.GetValue(),
		Namespace: group.Namespace.GetValue(),
		Comment:   comment,
		CreateBy:  createBy,
		Valid:     true,
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
