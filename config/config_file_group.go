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
	"google.golang.org/protobuf/types/known/wrapperspb"

	cachetypes "github.com/polarismesh/polaris/cache/api"
	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/model"
	commonstore "github.com/polarismesh/polaris/common/store"
	"github.com/polarismesh/polaris/common/utils"
)

// CreateConfigFileGroup 创建配置文件组
func (s *Server) CreateConfigFileGroup(ctx context.Context, req *apiconfig.ConfigFileGroup) *apiconfig.ConfigResponse {
	namespace := req.Namespace.GetValue()
	groupName := req.Name.GetValue()

	// 如果 namespace 不存在则自动创建
	if _, errResp := s.namespaceOperator.CreateNamespaceIfAbsent(ctx, &apimodel.Namespace{
		Name: req.GetNamespace(),
	}); errResp != nil {
		log.Error("[Config][Group] create namespace failed.", utils.RequestID(ctx),
			utils.ZapNamespace(namespace), utils.ZapGroup(groupName), zap.String("err", errResp.String()))
		return api.NewConfigResponse(apimodel.Code(errResp.Code.GetValue()))
	}

	fileGroup, err := s.storage.GetConfigFileGroup(namespace, groupName)
	if err != nil {
		log.Error("[Config][Group] get config file group error.", utils.RequestID(ctx), zap.Error(err))
		return api.NewConfigResponse(commonstore.StoreCode2APICode(err))
	}
	if fileGroup != nil {
		return api.NewConfigResponse(apimodel.Code_ExistedResource)
	}

	saveData := model.ToConfigGroupStore(req)
	saveData.CreateBy = utils.ParseUserName(ctx)
	saveData.ModifyBy = utils.ParseUserName(ctx)

	ret, err := s.storage.CreateConfigFileGroup(saveData)
	if err != nil {
		log.Error("[Config][Group] create config file group error.", utils.RequestID(ctx),
			utils.ZapNamespace(namespace), utils.ZapGroup(groupName), zap.Error(err))
		return api.NewConfigResponse(commonstore.StoreCode2APICode(err))
	}

	log.Info("[Config][Group] create config file group successful.", utils.RequestID(ctx),
		utils.ZapNamespace(namespace), utils.ZapGroup(groupName))

	// 这里设置在 config-group 的 id 信息
	req.Id = utils.NewUInt64Value(ret.Id)
	if err := s.afterConfigGroupResource(ctx, req); err != nil {
		log.Error("[Config][Group] create config_file_group after resource",
			utils.RequestID(ctx), zap.Error(err))
		return api.NewConfigResponse(apimodel.Code_ExecuteException)
	}

	s.RecordHistory(ctx, configGroupRecordEntry(ctx, req, saveData, model.OCreate))
	return api.NewConfigResponse(apimodel.Code_ExecuteSuccess)
}

// UpdateConfigFileGroup 更新配置文件组
func (s *Server) UpdateConfigFileGroup(ctx context.Context, req *apiconfig.ConfigFileGroup) *apiconfig.ConfigResponse {
	namespace := req.Namespace.GetValue()
	groupName := req.Name.GetValue()

	saveData, err := s.storage.GetConfigFileGroup(namespace, groupName)
	if err != nil {
		log.Error("[Config][Group] get config file group failed. ", utils.RequestID(ctx),
			utils.ZapNamespace(namespace), utils.ZapGroup(groupName), zap.Error(err))
		return api.NewConfigResponse(commonstore.StoreCode2APICode(err))
	}
	if saveData == nil {
		return api.NewConfigResponse(apimodel.Code_NotFoundResource)
	}

	updateData := model.ToConfigGroupStore(req)
	updateData.ModifyBy = utils.ParseOperator(ctx)
	updateData, needUpdate := s.UpdateGroupAttribute(saveData, updateData)
	if !needUpdate {
		return api.NewConfigResponse(apimodel.Code_NoNeedUpdate)
	}

	if err := s.storage.UpdateConfigFileGroup(updateData); err != nil {
		log.Error("[Config][Group] update config file group failed. ", utils.RequestID(ctx),
			utils.ZapNamespace(namespace), utils.ZapGroup(groupName), zap.Error(err))
		return api.NewConfigResponse(commonstore.StoreCode2APICode(err))
	}

	req.Id = utils.NewUInt64Value(saveData.Id)
	if err := s.afterConfigGroupResource(ctx, req); err != nil {
		log.Error("[Config][Group] update config_file_group after resource",
			utils.RequestID(ctx), zap.Error(err))
		return api.NewConfigResponse(apimodel.Code_ExecuteException)
	}

	s.RecordHistory(ctx, configGroupRecordEntry(ctx, req, updateData, model.OUpdate))
	return api.NewConfigResponse(apimodel.Code_ExecuteSuccess)
}

func (s *Server) UpdateGroupAttribute(saveData, updateData *model.ConfigFileGroup) (*model.ConfigFileGroup, bool) {
	needUpdate := false
	if saveData.Comment != updateData.Comment {
		needUpdate = true
		saveData.Comment = updateData.Comment
	}
	if saveData.Business != updateData.Business {
		needUpdate = true
		saveData.Business = updateData.Business
	}
	if saveData.Department != updateData.Department {
		needUpdate = true
		saveData.Department = updateData.Department
	}
	if utils.IsNotEqualMap(updateData.Metadata, saveData.Metadata) {
		needUpdate = true
		saveData.Metadata = updateData.Metadata
	}
	return saveData, needUpdate
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
		log.Error("[Config][Group] query config file group error.", utils.RequestID(ctx),
			utils.ZapNamespace(namespace), utils.ZapGroup(name), zap.Error(err))
		return api.NewConfigResponse(commonstore.StoreCode2APICode(err))
	}
	if group != nil {
		return api.NewConfigResponse(apimodel.Code_ExecuteSuccess)
	}
	return s.CreateConfigFileGroup(ctx, configFileGroup)
}

// DeleteConfigFileGroup 删除配置文件组
func (s *Server) DeleteConfigFileGroup(ctx context.Context, namespace, name string) *apiconfig.ConfigResponse {
	log.Info("[Config][Group] delete config file group. ", utils.RequestID(ctx),
		utils.ZapNamespace(namespace), utils.ZapGroup(name))

	configGroup, err := s.storage.GetConfigFileGroup(namespace, name)
	if err != nil {
		log.Error("[Config][Group] get config file group failed. ", utils.RequestID(ctx),
			utils.ZapNamespace(namespace), utils.ZapGroup(name), zap.Error(err))
		return api.NewConfigResponse(commonstore.StoreCode2APICode(err))
	}
	if configGroup == nil {
		return api.NewConfigResponse(apimodel.Code_NotFoundResource)
	}
	if errResp := s.hasResourceInConfigGroup(ctx, namespace, name); errResp != nil {
		return errResp
	}

	if err := s.storage.DeleteConfigFileGroup(namespace, name); err != nil {
		log.Error("[Config][Group] delete config file group failed. ", utils.RequestID(ctx),
			utils.ZapNamespace(namespace), utils.ZapGroup(name), zap.Error(err))
		return api.NewConfigResponse(commonstore.StoreCode2APICode(err))
	}

	if err := s.afterConfigGroupResource(ctx, &apiconfig.ConfigFileGroup{
		Id:        utils.NewUInt64Value(configGroup.Id),
		Namespace: utils.NewStringValue(configGroup.Namespace),
		Name:      utils.NewStringValue(configGroup.Name),
	}); err != nil {
		log.Error("[Config][Group] delete config_file_group after resource",
			utils.RequestID(ctx), zap.Error(err))
		return api.NewConfigResponse(apimodel.Code_ExecuteException)
	}
	s.RecordHistory(ctx, configGroupRecordEntry(ctx, &apiconfig.ConfigFileGroup{
		Namespace: utils.NewStringValue(namespace),
		Name:      utils.NewStringValue(name),
	}, configGroup, model.ODelete))
	return api.NewConfigResponse(apimodel.Code_ExecuteSuccess)
}

func (s *Server) hasResourceInConfigGroup(ctx context.Context, namespace, name string) *apiconfig.ConfigResponse {
	total, err := s.storage.CountConfigFiles(namespace, name)
	if err != nil {
		log.Error("[Config][Group] get config file group failed. ", utils.RequestID(ctx),
			utils.ZapNamespace(namespace), utils.ZapGroup(name), zap.Error(err))
		return api.NewConfigResponse(commonstore.StoreCode2APICode(err))
	}
	if total != 0 {
		return api.NewConfigResponse(apimodel.Code_ExistedResource)
	}
	total, err = s.storage.CountConfigReleases(namespace, name, true)
	if err != nil {
		log.Error("[Config][Group] get config file group failed. ", utils.RequestID(ctx),
			utils.ZapNamespace(namespace), utils.ZapGroup(name), zap.Error(err))
		return api.NewConfigResponse(commonstore.StoreCode2APICode(err))
	}
	if total != 0 {
		return api.NewConfigResponse(apimodel.Code_ExistedResource)
	}
	return nil
}

// QueryConfigFileGroups 查询配置文件组
func (s *Server) QueryConfigFileGroups(ctx context.Context,
	searchFilters map[string]string) *apiconfig.ConfigBatchQueryResponse {

	offset, limit, _ := utils.ParseOffsetAndLimit(searchFilters)

	args := &cachetypes.ConfigGroupArgs{
		Namespace:  searchFilters["namespace"],
		Name:       searchFilters["name"],
		Business:   searchFilters["business"],
		Department: searchFilters["department"],
		Offset:     offset,
		Limit:      limit,
		OrderField: searchFilters["order_field"],
		OrderType:  searchFilters["order_type"],
	}

	total, ret, err := s.groupCache.Query(args)
	if err != nil {
		resp := api.NewConfigBatchQueryResponse(commonstore.StoreCode2APICode(err))
		resp.Info = utils.NewStringValue(err.Error())
		return resp
	}
	values := make([]*apiconfig.ConfigFileGroup, 0, len(ret))
	for i := range ret {
		item := model.ToConfigGroupAPI(ret[i])
		fileCount, err := s.storage.CountConfigFiles(ret[i].Namespace, ret[i].Name)
		if err != nil {
			log.Error("[Config][Service] get config file count for group error.", utils.RequestID(ctx),
				utils.ZapNamespace(ret[i].Namespace), utils.ZapGroup(ret[i].Name), zap.Error(err))
		}
		item.FileCount = wrapperspb.UInt64(fileCount)
		values = append(values, item)
	}

	resp := api.NewConfigBatchQueryResponse(apimodel.Code_ExecuteSuccess)
	resp.Total = utils.NewUInt32Value(total)
	resp.ConfigFileGroups = values
	return resp
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
