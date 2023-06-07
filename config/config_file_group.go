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
	"sort"
	"strings"
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

// CreateConfigFileGroup 创建配置文件组
func (s *Server) CreateConfigFileGroup(ctx context.Context,
	configFileGroup *apiconfig.ConfigFileGroup) *apiconfig.ConfigResponse {
	if checkError := checkConfigFileGroupParams(configFileGroup); checkError != nil {
		return checkError
	}

	userName := utils.ParseUserName(ctx)
	configFileGroup.CreateBy = utils.NewStringValue(userName)
	configFileGroup.ModifyBy = utils.NewStringValue(userName)

	namespace := configFileGroup.Namespace.GetValue()
	groupName := configFileGroup.Name.GetValue()

	// 如果 namespace 不存在则自动创建
	if _, errResp := s.namespaceOperator.CreateNamespaceIfAbsent(ctx, &apimodel.Namespace{
		Name: utils.NewStringValue(namespace),
	}); errResp != nil {
		log.Error("[Config][Service] create namespace failed.", utils.ZapRequestIDByCtx(ctx),
			utils.ZapNamespace(namespace), utils.ZapGroup(groupName), zap.String("err", errResp.String()))
		return api.NewConfigFileGroupResponse(apimodel.Code(errResp.Code.GetValue()), configFileGroup)
	}

	fileGroup, err := s.storage.GetConfigFileGroup(namespace, groupName)
	if err != nil {
		log.Error("[Config][Service] get config file group error.",
			utils.ZapRequestIDByCtx(ctx),
			zap.Error(err))
		return api.NewConfigFileGroupResponse(commonstore.StoreCode2APICode(err), configFileGroup)
	}

	if fileGroup != nil {
		return api.NewConfigFileGroupResponse(apimodel.Code_ExistedResource, configFileGroup)
	}

	toCreateGroup := model.ToConfigGroupStore(configFileGroup)
	toCreateGroup.ModifyBy = toCreateGroup.CreateBy

	createdGroup, err := s.storage.CreateConfigFileGroup(toCreateGroup)
	if err != nil {
		log.Error("[Config][Service] create config file group error.", utils.ZapRequestIDByCtx(ctx),
			utils.ZapNamespace(namespace), utils.ZapGroup(groupName), zap.Error(err))
		return api.NewConfigFileGroupResponse(commonstore.StoreCode2APICode(err), configFileGroup)
	}

	log.Info("[Config][Service] create config file group successful.", utils.ZapRequestIDByCtx(ctx),
		utils.ZapNamespace(namespace), utils.ZapGroup(groupName))

	// 这里设置在 config-group 的 id 信息
	configFileGroup.Id = utils.NewUInt64Value(createdGroup.Id)
	if err := s.afterConfigGroupResource(ctx, configFileGroup); err != nil {
		log.Error("[Config][Service] create config_file_group after resource",
			utils.ZapRequestIDByCtx(ctx), zap.Error(err))
		return api.NewConfigFileGroupResponse(apimodel.Code_ExecuteException, nil)
	}

	s.RecordHistory(ctx, configGroupRecordEntry(ctx, configFileGroup, createdGroup, model.OCreate))
	return api.NewConfigFileGroupResponse(apimodel.Code_ExecuteSuccess, model.ToConfigGroupAPI(createdGroup))
}

// createConfigFileGroupIfAbsent 如果不存在配置文件组，则自动创建
func (s *Server) createConfigFileGroupIfAbsent(ctx context.Context,
	configFileGroup *apiconfig.ConfigFileGroup) *apiconfig.ConfigResponse {
	var (
		namespace = configFileGroup.Namespace.GetValue()
		name      = configFileGroup.Name.GetValue()
	)

	group, err := s.storage.GetConfigFileGroup(namespace, name)
	if err != nil {
		log.Error("[Config][Service] query config file group error.", utils.ZapRequestIDByCtx(ctx),
			utils.ZapNamespace(namespace), utils.ZapGroup(name), zap.Error(err))
		return api.NewConfigFileGroupResponse(commonstore.StoreCode2APICode(err), nil)
	}
	if group != nil {
		return api.NewConfigFileGroupResponse(apimodel.Code_ExecuteSuccess, model.ToConfigGroupAPI(group))
	}
	return s.CreateConfigFileGroup(ctx, configFileGroup)
}

// QueryConfigFileGroups 查询配置文件组
func (s *Server) QueryConfigFileGroups(ctx context.Context, namespace, groupName,
	fileName string, offset, limit uint32) *apiconfig.ConfigBatchQueryResponse {
	if limit > MaxPageSize {
		return api.NewConfigFileGroupBatchQueryResponse(apimodel.Code_InvalidParameter, 0, nil)
	}

	// 按分组名搜索
	if fileName == "" {
		return s.queryByGroupName(ctx, namespace, groupName, offset, limit)
	}

	// 按文件搜索
	return s.queryByFileName(ctx, namespace, groupName, fileName, offset, limit)
}

func (s *Server) queryByGroupName(ctx context.Context, namespace, groupName string,
	offset, limit uint32) *apiconfig.ConfigBatchQueryResponse {
	count, groups, err := s.storage.QueryConfigFileGroups(namespace, groupName, offset, limit)
	if err != nil {
		log.Error("[Config][Service] query config file group error.", utils.ZapRequestIDByCtx(ctx),
			utils.ZapNamespace(namespace), utils.ZapGroup(groupName), zap.Error(err))
		return api.NewConfigFileGroupBatchQueryResponse(commonstore.StoreCode2APICode(err), 0, nil)
	}

	if len(groups) == 0 {
		return api.NewConfigFileGroupBatchQueryResponse(apimodel.Code_ExecuteSuccess, count, nil)
	}

	groupAPIModels, err := s.batchTransfer(ctx, groups)
	if err != nil {
		return api.NewConfigFileGroupBatchQueryResponse(commonstore.StoreCode2APICode(err), 0, nil)
	}
	return api.NewConfigFileGroupBatchQueryResponse(apimodel.Code_ExecuteSuccess, count, groupAPIModels)
}

func (s *Server) queryByFileName(ctx context.Context, namespace, groupName,
	fileName string, offset uint32, limit uint32) *apiconfig.ConfigBatchQueryResponse {
	// 内存分页，先获取到所有配置文件
	rsp := s.queryConfigFileWithoutTags(ctx, namespace, groupName, fileName, 0, 10000)
	if rsp.Code.GetValue() != uint32(apimodel.Code_ExecuteSuccess) {
		return rsp
	}

	// 获取所有的 group 信息
	groupMap := make(map[string]bool)
	for _, configFile := range rsp.ConfigFiles {
		// namespace+group 是唯一键
		groupMap[configFile.Namespace.Value+"+"+configFile.Group.Value] = true
	}

	if len(groupMap) == 0 {
		return api.NewConfigFileGroupBatchQueryResponse(apimodel.Code_ExecuteSuccess, 0, nil)
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
		return api.NewConfigFileGroupBatchQueryResponse(apimodel.Code_ExecuteSuccess, uint32(total), nil)
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
			log.Error("[Config][Service] get config file group error.", utils.ZapRequestIDByCtx(ctx),
				utils.ZapNamespace(namespaceAndGroup[0]), utils.ZapGroup(namespaceAndGroup[1]), zap.Error(err))
			return api.NewConfigFileGroupBatchQueryResponse(commonstore.StoreCode2APICode(err), 0, nil)
		}
		configFileGroups = append(configFileGroups, configFileGroup)
	}

	groupAPIModels, err := s.batchTransfer(ctx, configFileGroups)
	if err != nil {
		return api.NewConfigFileGroupBatchQueryResponse(commonstore.StoreCode2APICode(err), 0, nil)
	}

	return api.NewConfigFileGroupBatchQueryResponse(apimodel.Code_ExecuteSuccess, uint32(total), groupAPIModels)
}

func (s *Server) batchTransfer(ctx context.Context,
	groups []*model.ConfigFileGroup) ([]*apiconfig.ConfigFileGroup, error) {
	var result []*apiconfig.ConfigFileGroup
	for _, groupStoreModel := range groups {
		configFileGroup := model.ToConfigGroupAPI(groupStoreModel)
		// enrich config file count
		fileCount, err := s.storage.CountByConfigFileGroup(groupStoreModel.Namespace, groupStoreModel.Name)
		if err != nil {
			log.Error("[Config][Service] get config file count for group error.", utils.ZapRequestIDByCtx(ctx),
				utils.ZapNamespace(groupStoreModel.Namespace), utils.ZapGroup(groupStoreModel.Name), zap.Error(err))
			return nil, err
		}
		configFileGroup.FileCount = utils.NewUInt64Value(fileCount)
		result = append(result, configFileGroup)
	}
	return result, nil
}

// DeleteConfigFileGroup 删除配置文件组
func (s *Server) DeleteConfigFileGroup(ctx context.Context, namespace, name string) *apiconfig.ConfigResponse {
	if err := CheckResourceName(utils.NewStringValue(namespace)); err != nil {
		return api.NewConfigFileGroupResponse(apimodel.Code_InvalidNamespaceName, nil)
	}
	if err := CheckResourceName(utils.NewStringValue(name)); err != nil {
		return api.NewConfigFileGroupResponse(apimodel.Code_InvalidConfigFileGroupName, nil)
	}

	log.Info("[Config][Service] delete config file group. ", utils.ZapRequestIDByCtx(ctx),
		utils.ZapNamespace(namespace), utils.ZapGroup(name))

	// 删除配置文件组，同时删除组下面所有的配置文件
	startOffset := uint32(0)
	hasMore := true
	for hasMore {
		queryRsp := s.QueryConfigFilesByGroup(ctx, namespace, name, startOffset, MaxPageSize)
		if queryRsp.Code.GetValue() != uint32(apimodel.Code_ExecuteSuccess) {
			log.Error("[Config][Service] get group's config file failed. ", utils.ZapRequestIDByCtx(ctx),
				utils.ZapNamespace(namespace), utils.ZapGroup(name))
			return api.NewConfigFileGroupResponse(apimodel.Code(queryRsp.Code.GetValue()), nil)
		}
		configFiles := queryRsp.ConfigFiles

		deleteRsp := s.BatchDeleteConfigFile(ctx, configFiles, utils.ParseUserName(ctx))
		if deleteRsp.Code.GetValue() != uint32(apimodel.Code_ExecuteSuccess) {
			log.Error("[Config][Service] batch delete group's config file failed. ", utils.ZapRequestIDByCtx(ctx),
				utils.ZapNamespace(namespace), utils.ZapGroup(name))
			return api.NewConfigFileGroupResponse(apimodel.Code(deleteRsp.Code.GetValue()), nil)
		}

		if hasMore = len(queryRsp.ConfigFiles) >= MaxPageSize; hasMore {
			startOffset += MaxPageSize
		}
	}

	configGroup, err := s.storage.GetConfigFileGroup(namespace, name)
	if err != nil {
		log.Error("[Config][Service] get config file group failed. ", utils.ZapRequestIDByCtx(ctx),
			utils.ZapNamespace(namespace), utils.ZapGroup(name), zap.Error(err))
		return api.NewConfigFileGroupResponse(commonstore.StoreCode2APICode(err), nil)
	}
	if configGroup == nil {
		return api.NewConfigFileGroupResponse(apimodel.Code_NotFoundResource, nil)
	}

	if err := s.storage.DeleteConfigFileGroup(namespace, name); err != nil {
		log.Error("[Config][Service] delete config file group failed. ", utils.ZapRequestIDByCtx(ctx),
			utils.ZapNamespace(namespace), utils.ZapGroup(name), zap.Error(err))
		return api.NewConfigFileGroupResponse(commonstore.StoreCode2APICode(err), nil)
	}

	if err := s.afterConfigGroupResource(ctx, &apiconfig.ConfigFileGroup{
		Id:        utils.NewUInt64Value(configGroup.Id),
		Namespace: utils.NewStringValue(configGroup.Namespace),
		Name:      utils.NewStringValue(configGroup.Name),
	}); err != nil {
		log.Error("[Config][Service] delete config_file_group after resource",
			utils.ZapRequestIDByCtx(ctx), zap.Error(err))
		return api.NewConfigFileGroupResponse(apimodel.Code_ExecuteException, nil)
	}
	s.RecordHistory(ctx, configGroupRecordEntry(ctx, &apiconfig.ConfigFileGroup{
		Namespace: utils.NewStringValue(namespace),
		Name:      utils.NewStringValue(name),
	}, configGroup, model.ODelete))
	return api.NewConfigFileGroupResponse(apimodel.Code_ExecuteSuccess, nil)
}

// UpdateConfigFileGroup 更新配置文件组
func (s *Server) UpdateConfigFileGroup(ctx context.Context,
	configFileGroup *apiconfig.ConfigFileGroup) *apiconfig.ConfigResponse {
	if resp := checkConfigFileGroupParams(configFileGroup); resp != nil {
		return resp
	}

	namespace := configFileGroup.Namespace.GetValue()
	groupName := configFileGroup.Name.GetValue()

	fileGroup, err := s.storage.GetConfigFileGroup(namespace, groupName)
	if err != nil {
		log.Error("[Config][Service] get config file group failed. ", utils.ZapRequestIDByCtx(ctx),
			utils.ZapNamespace(namespace), utils.ZapGroup(groupName), zap.Error(err))
		return api.NewConfigFileGroupResponse(commonstore.StoreCode2APICode(err), configFileGroup)
	}

	if fileGroup == nil {
		return api.NewConfigFileGroupResponse(apimodel.Code_NotFoundResource, configFileGroup)
	}

	configFileGroup.ModifyBy = utils.NewStringValue(utils.ParseUserName(ctx))

	toUpdateGroup := model.ToConfigGroupStore(configFileGroup)
	toUpdateGroup.ModifyBy = configFileGroup.ModifyBy.GetValue()

	updatedGroup, err := s.storage.UpdateConfigFileGroup(toUpdateGroup)
	if err != nil {
		log.Error("[Config][Service] update config file group failed. ", utils.ZapRequestIDByCtx(ctx),
			utils.ZapNamespace(namespace), utils.ZapGroup(groupName), zap.Error(err))

		return api.NewConfigFileGroupResponse(commonstore.StoreCode2APICode(err), configFileGroup)
	}

	configFileGroup.Id = utils.NewUInt64Value(fileGroup.Id)
	if err := s.afterConfigGroupResource(ctx, configFileGroup); err != nil {
		log.Error("[Config][Service] update config_file_group after resource",
			utils.ZapRequestIDByCtx(ctx), zap.Error(err))
		return api.NewConfigFileGroupResponse(apimodel.Code_ExecuteException, nil)
	}

	s.RecordHistory(ctx, configGroupRecordEntry(ctx, configFileGroup, fileGroup, model.OUpdate))
	return api.NewConfigFileGroupResponse(apimodel.Code_ExecuteSuccess, model.ToConfigGroupAPI(updatedGroup))
}

func checkConfigFileGroupParams(configFileGroup *apiconfig.ConfigFileGroup) *apiconfig.ConfigResponse {
	if configFileGroup == nil {
		return api.NewConfigFileGroupResponse(apimodel.Code_InvalidParameter, configFileGroup)
	}

	if err := CheckResourceName(configFileGroup.Name); err != nil {
		return api.NewConfigFileGroupResponse(apimodel.Code_InvalidConfigFileGroupName, configFileGroup)
	}

	if err := CheckResourceName(configFileGroup.Namespace); err != nil {
		return api.NewConfigFileGroupResponse(apimodel.Code_InvalidNamespaceName, configFileGroup)
	}

	return nil
}

// configGroupRecordEntry 生成服务的记录entry
func configGroupRecordEntry(ctx context.Context, req *apiconfig.ConfigFileGroup, md *model.ConfigFileGroup,
	operationType model.OperationType) *model.RecordEntry {

	marshaler := jsonpb.Marshaler{}
	detail, _ := marshaler.MarshalToString(req)

	entry := &model.RecordEntry{
		ResourceType:  model.RConfigGroup,
		ResourceName:  req.GetName().GetValue(),
		Namespace:     req.GetNamespace().GetValue(),
		OperationType: operationType,
		Operator:      utils.ParseOperator(ctx),
		Detail:        detail,
		HappenTime:    time.Now(),
	}

	return entry
}
