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
	"strings"
	"time"

	"github.com/gogo/protobuf/jsonpb"
	apiconfig "github.com/polarismesh/specification/source/go/api/v1/config_manage"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/wrapperspb"

	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/model"
	commonstore "github.com/polarismesh/polaris/common/store"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/store"
)

// CreateConfigFile 创建配置文件
func (s *Server) CreateConfigFile(ctx context.Context, req *apiconfig.ConfigFile) *apiconfig.ConfigResponse {
	if rsp := s.prepareCreateConfigFile(ctx, req); rsp.Code.Value != api.ExecuteSuccess {
		return rsp
	}

	tx, err := s.storage.StartTx()
	if err != nil {
		log.Error("[Config][File] create config file begin tx.", utils.RequestID(ctx), zap.Error(err))
		return api.NewConfigResponse(commonstore.StoreCode2APICode(err))
	}
	defer func() {
		_ = tx.Rollback()
	}()

	resp := s.handleCreateConfigFile(ctx, tx, req)
	if resp.GetCode().GetValue() != uint32(apimodel.Code_ExecuteSuccess) {
		return resp
	}
	if err := tx.Commit(); err != nil {
		log.Error("[Config][File] create config file commit tx.", utils.RequestID(ctx), zap.Error(err))
		return api.NewConfigResponse(commonstore.StoreCode2APICode(err))
	}
	s.RecordHistory(ctx, configFileRecordEntry(ctx, req, model.OCreate))
	resp.ConfigFile = req
	return resp
}

func (s *Server) handleCreateConfigFile(ctx context.Context, tx store.Tx,
	req *apiconfig.ConfigFile) *apiconfig.ConfigResponse {

	data, err := s.storage.GetConfigFileTx(tx, req.GetNamespace().GetValue(), req.GetGroup().GetValue(),
		req.GetName().GetValue())
	if err != nil {
		log.Error("[Config][File] create config file when get save data.", utils.RequestID(ctx),
			utils.ZapNamespace(req.GetNamespace().GetValue()), utils.ZapGroup(req.GetGroup().GetValue()),
			utils.ZapFileName(req.GetName().GetValue()), zap.Error(err))
		return api.NewConfigResponse(commonstore.StoreCode2APICode(err))
	}
	if data != nil {
		return api.NewConfigResponse(apimodel.Code_ExistedResource)
	}

	savaData := model.ToConfigFileStore(req)
	if errResp := s.chains.BeforeCreateFile(ctx, savaData); errResp != nil {
		return errResp
	}
	// 创建配置文件
	if err := s.storage.CreateConfigFileTx(tx, savaData); err != nil {
		log.Error("[Config][File] create config file error.", utils.RequestID(ctx),
			utils.ZapNamespace(req.GetNamespace().GetValue()), utils.ZapGroup(req.GetGroup().GetValue()),
			utils.ZapFileName(req.GetName().GetValue()), zap.Error(err))
		return api.NewConfigResponse(commonstore.StoreCode2APICode(err))
	}
	return api.NewConfigResponse(apimodel.Code_ExecuteSuccess)
}

// UpdateConfigFile 更新配置文件
func (s *Server) UpdateConfigFile(ctx context.Context, req *apiconfig.ConfigFile) *apiconfig.ConfigResponse {
	tx, err := s.storage.StartTx()
	if err != nil {
		log.Error("[Config][File] update config file begin tx.", utils.RequestID(ctx), zap.Error(err))
		return api.NewConfigResponse(commonstore.StoreCode2APICode(err))
	}
	defer func() {
		_ = tx.Rollback()
	}()

	resp := s.handleUpdateConfigFile(ctx, tx, req)
	if resp.GetCode().GetValue() != uint32(apimodel.Code_ExecuteSuccess) {
		return resp
	}
	if err := tx.Commit(); err != nil {
		log.Error("[Config][File] update config file commit tx.", utils.RequestID(ctx), zap.Error(err))
		return api.NewConfigResponseWithInfo(commonstore.StoreCode2APICode(err), err.Error())
	}
	s.RecordHistory(ctx, configFileRecordEntry(ctx, req, model.OUpdate))
	return resp
}

func (s *Server) handleUpdateConfigFile(ctx context.Context, tx store.Tx,
	req *apiconfig.ConfigFile) *apiconfig.ConfigResponse {

	namespace := req.Namespace.GetValue()
	group := req.Group.GetValue()
	name := req.Name.GetValue()

	saveData, err := s.storage.GetConfigFileTx(tx, req.GetNamespace().GetValue(), req.GetGroup().GetValue(),
		req.GetName().GetValue())
	if err != nil {
		log.Error("[Config][File] update config file when get save data.", utils.RequestID(ctx), zap.Error(err))
		return api.NewConfigResponse(commonstore.StoreCode2APICode(err))
	}
	if saveData == nil {
		return api.NewConfigResponse(apimodel.Code_NotFoundResource)
	}
	updateData, needUpdate := s.updateConfigFileAttribute(saveData, model.ToConfigFileStore(req))
	if !needUpdate {
		return api.NewConfigResponse(apimodel.Code_NoNeedUpdate)
	}

	if errResp := s.chains.BeforeUpdateFile(ctx, updateData); errResp != nil {
		return errResp
	}

	if err := s.storage.UpdateConfigFileTx(tx, updateData); err != nil {
		log.Error("[Config][File] update config file error.", utils.RequestID(ctx),
			utils.ZapNamespace(namespace), utils.ZapGroup(group), utils.ZapFileName(name), zap.Error(err))
		return api.NewConfigFileResponse(commonstore.StoreCode2APICode(err), req)
	}
	return api.NewConfigFileResponse(apimodel.Code_ExecuteSuccess, model.ToConfigFileAPI(updateData))
}

func (s *Server) updateConfigFileAttribute(saveData, updateData *model.ConfigFile) (*model.ConfigFile, bool) {
	needUpdate := false
	oldMetadata := saveData.Metadata
	oldEncrtptAlgo := saveData.EncryptAlgo
	if saveData.Comment != updateData.Comment {
		needUpdate = true
		saveData.Comment = updateData.Comment
	}
	if saveData.Comment != updateData.Content {
		needUpdate = true
		saveData.Content = updateData.Content
	}
	if saveData.Format != updateData.Format {
		needUpdate = true
		saveData.Format = updateData.Format
	}
	if utils.IsNotEqualMap(updateData.Metadata, saveData.Metadata) {
		needUpdate = true
		saveData.Metadata = updateData.Metadata
	}
	if saveData.Encrypt != updateData.Encrypt {
		needUpdate = true
		saveData.Encrypt = updateData.Encrypt
	}
	if saveData.EncryptAlgo != updateData.EncryptAlgo {
		needUpdate = true
		saveData.EncryptAlgo = updateData.EncryptAlgo
	}
	// 填充加密所需要的 Metadata Key 数据
	if saveData.Encrypt && saveData.EncryptAlgo == oldEncrtptAlgo {
		if len(saveData.Metadata) == 0 {
			saveData.Metadata = map[string]string{}
		}
		saveData.Metadata[model.MetaKeyConfigFileDataKey] = oldMetadata[model.MetaKeyConfigFileDataKey]
		saveData.Metadata[model.MetaKeyConfigFileEncryptAlgo] = oldMetadata[model.MetaKeyConfigFileEncryptAlgo]
	}

	return saveData, needUpdate
}

func (s *Server) prepareCreateConfigFile(ctx context.Context,
	configFile *apiconfig.ConfigFile) *apiconfig.ConfigResponse {

	configFile.CreateBy = utils.NewStringValue(utils.ParseUserName(ctx))
	configFile.ModifyBy = utils.NewStringValue(utils.ParseUserName(ctx))

	// 如果配置文件组不存在则自动创建
	createGroupRsp := s.createConfigFileGroupIfAbsent(ctx, &apiconfig.ConfigFileGroup{
		Namespace: configFile.Namespace,
		Name:      configFile.Group,
		CreateBy:  configFile.CreateBy,
		Comment:   utils.NewStringValue("auto created"),
	})

	if createGroupRsp.Code.GetValue() != uint32(apimodel.Code_ExecuteSuccess) {
		return api.NewConfigFileResponse(apimodel.Code(createGroupRsp.Code.GetValue()), configFile)
	}
	return api.NewConfigResponse(apimodel.Code_ExecuteSuccess)
}

// BatchDeleteConfigFile 批量删除配置文件
func (s *Server) BatchDeleteConfigFile(ctx context.Context, req []*apiconfig.ConfigFile) *apiconfig.ConfigResponse {
	if len(req) == 0 {
		api.NewConfigFileResponse(apimodel.Code_ExecuteSuccess, nil)
	}
	for _, configFile := range req {
		rsp := s.DeleteConfigFile(ctx, configFile)
		if rsp.Code.GetValue() != uint32(apimodel.Code_ExecuteSuccess) {
			return rsp
		}
	}
	return api.NewConfigFileResponse(apimodel.Code_ExecuteSuccess, nil)
}

// DeleteConfigFile 删除配置文件，删除配置文件同时会通知客户端 Not_Found
func (s *Server) DeleteConfigFile(ctx context.Context, req *apiconfig.ConfigFile) *apiconfig.ConfigResponse {
	namespace := req.GetNamespace().GetValue()
	group := req.GetGroup().GetValue()
	fileName := req.GetName().GetValue()

	tx, err := s.storage.StartTx()
	if err != nil {
		log.Error("[Config][File] delete config file begin tx.", utils.RequestID(ctx), zap.Error(err))
		return api.NewConfigResponse(commonstore.StoreCode2APICode(err))
	}
	defer func() { _ = tx.Rollback() }()

	file, err := s.storage.LockConfigFile(tx, &model.ConfigFileKey{
		Namespace: namespace,
		Group:     group,
		Name:      fileName,
	})
	if err != nil {
		log.Error("[Config][File] delete config file when lock.", utils.RequestID(ctx),
			utils.ZapNamespace(namespace), utils.ZapGroup(group), utils.ZapFileName(fileName), zap.Error(err))
		return api.NewConfigResponse(commonstore.StoreCode2APICode(err))
	}
	if file == nil {
		log.Info("[Config][File] delete config file not found, so skip.", utils.RequestID(ctx),
			utils.ZapNamespace(namespace), utils.ZapGroup(group), utils.ZapFileName(fileName), zap.Error(err))
		return api.NewConfigResponse(apimodel.Code_ExecuteSuccess)
	}
	// 1. 删除配置文件发布内容
	if errResp := s.cleanConfigFileReleases(ctx, tx, file); errResp != nil {
		return errResp
	}
	// 2. 删除配置文件
	if err = s.storage.DeleteConfigFileTx(tx, namespace, group, fileName); err != nil {
		log.Error("[Config][File] delete config file error.", utils.RequestID(ctx),
			utils.ZapNamespace(namespace), utils.ZapGroup(group), utils.ZapFileName(fileName), zap.Error(err))
		return api.NewConfigResponse(commonstore.StoreCode2APICode(err))
	}
	if err := tx.Commit(); err != nil {
		log.Error("[Config][File] delete config file when commit tx.", utils.RequestID(ctx),
			utils.ZapNamespace(namespace), utils.ZapGroup(group), utils.ZapFileName(fileName), zap.Error(err))
		return api.NewConfigResponse(commonstore.StoreCode2APICode(err))
	}
	s.RecordHistory(ctx, configFileRecordEntry(ctx, &apiconfig.ConfigFile{
		Namespace: utils.NewStringValue(namespace),
		Group:     utils.NewStringValue(group),
		Name:      utils.NewStringValue(fileName),
	}, model.ODelete))
	return api.NewConfigResponse(apimodel.Code_ExecuteSuccess)
}

// GetConfigFileRichInfo 获取单个配置文件基础信息，包含发布状态等信息
func (s *Server) GetConfigFileRichInfo(ctx context.Context, req *apiconfig.ConfigFile) *apiconfig.ConfigResponse {
	namespace := req.GetNamespace().GetValue()
	group := req.GetGroup().GetValue()
	fileName := req.GetName().GetValue()

	file, err := s.storage.GetConfigFile(namespace, group, fileName)
	if err != nil {
		log.Error("[Config][File] get config file error.", utils.RequestID(ctx),
			utils.ZapNamespace(namespace), utils.ZapGroup(group), utils.ZapFileName(fileName), zap.Error(err))
		return api.NewConfigResponse(commonstore.StoreCode2APICode(err))
	}
	if file == nil {
		return api.NewConfigResponse(apimodel.Code_NotFoundResource)
	}

	// 填充发布信息、标签信息等
	richFile, err := s.chains.AfterGetFile(ctx, file)
	if err != nil {
		return api.NewConfigResponseWithInfo(apimodel.Code_ExecuteException, err.Error())
	}
	ret := model.ToConfigFileAPI(richFile)
	return api.NewConfigFileResponse(apimodel.Code_ExecuteSuccess, ret)
}

// SearchConfigFile 查询配置文件
func (s *Server) SearchConfigFile(ctx context.Context, searchFilters map[string]string) *apiconfig.ConfigBatchQueryResponse {
	offset, limit, _ := utils.ParseOffsetAndLimit(searchFilters)
	count, files, err := s.storage.QueryConfigFiles(searchFilters, offset, limit)
	if err != nil {
		log.Error("[Config][File] search config files.", utils.RequestID(ctx), zap.Error(err))
		out := api.NewConfigBatchQueryResponse(commonstore.StoreCode2APICode(err))
		return out
	}

	if len(files) == 0 {
		out := api.NewConfigBatchQueryResponse(apimodel.Code_ExecuteSuccess)
		out.Total = utils.NewUInt32Value(count)
		return out
	}

	_ = s.caches.ConfigFile().Update()
	_ = s.caches.Gray().Update()
	ret := make([]*apiconfig.ConfigFile, 0, len(files))
	for _, file := range files {
		file, err := s.chains.AfterGetFile(ctx, file)
		if err != nil {
			log.Error("[Config][File] search files run chain after-get file.", utils.RequestID(ctx),
				zap.Error(err))
			return api.NewConfigBatchQueryResponse(apimodel.Code_ExecuteException)
		}
		ret = append(ret, model.ToConfigFileAPI(file))
	}
	out := api.NewConfigBatchQueryResponse(apimodel.Code_ExecuteSuccess)
	out.Total = utils.NewUInt32Value(count)
	out.ConfigFiles = ret
	return out
}

// ExportConfigFile 导出配置文件
func (s *Server) ExportConfigFile(ctx context.Context,
	configFileExport *apiconfig.ConfigFileExportRequest) *apiconfig.ConfigExportResponse {
	namespace := configFileExport.Namespace.GetValue()
	var groups []string
	for _, group := range configFileExport.Groups {
		groups = append(groups, group.GetValue())
	}
	var names []string
	for _, name := range configFileExport.Names {
		names = append(names, name.GetValue())
	}
	// 检查参数
	if err := utils.CheckResourceName(configFileExport.Namespace); err != nil {
		return api.NewConfigFileExportResponse(apimodel.Code_InvalidNamespaceName, nil)
	}
	var (
		isExportGroup bool
		configFiles   []*model.ConfigFile
	)
	if len(groups) >= 1 && len(names) == 0 {
		// 导出配置组
		isExportGroup = true
		for _, group := range groups {
			files, err := s.getGroupAllConfigFiles(namespace, group)
			if err != nil {
				log.Error("[Config][File] get config file by group error.", utils.RequestID(ctx),
					utils.ZapNamespace(namespace), utils.ZapGroup(group), zap.Error(err))
				return api.NewConfigFileExportResponse(commonstore.StoreCode2APICode(err), nil)
			}
			configFiles = append(configFiles, files...)
		}
	} else if len(groups) == 1 && len(names) > 0 {
		// 导出配置文件
		for _, name := range names {
			file, err := s.storage.GetConfigFile(namespace, groups[0], name)
			if err != nil {
				log.Error("[Config][File] get config file error.", utils.RequestID(ctx),
					utils.ZapNamespace(namespace), utils.ZapGroup(groups[0]), utils.ZapFileName(name),
					zap.Error(err))
				return api.NewConfigFileExportResponse(commonstore.StoreCode2APICode(err), nil)
			}
			configFiles = append(configFiles, file)
		}
	} else {
		log.Error("[Config][File] export config file error.", utils.RequestID(ctx),
			utils.ZapNamespace(namespace), zap.String("groups", strings.Join(groups, ",")),
			zap.String("names", strings.Join(names, ",")))
		return api.NewConfigFileExportResponse(apimodel.Code_InvalidParameter, nil)
	}
	if len(configFiles) == 0 {
		return api.NewConfigFileExportResponse(apimodel.Code_NotFoundResourceConfigFile, nil)
	}
	// 查询配置文件的标签
	fileID2Tags := make(map[uint64][]*model.ConfigFileTag)
	for _, file := range configFiles {
		filterTags := make([]*model.ConfigFileTag, 0, len(file.Metadata))
		for tagKey, tagVal := range file.Metadata {
			filterTags = append(filterTags, &model.ConfigFileTag{
				Key:   tagKey,
				Value: tagVal,
			})
		}
		fileID2Tags[file.Id] = filterTags
	}
	// 生成ZIP文件
	buf, err := CompressConfigFiles(configFiles, fileID2Tags, isExportGroup)
	if err != nil {
		log.Error("[Config][Servie]export config files compress to zip error.", zap.Error(err))
	}
	return api.NewConfigFileExportResponse(apimodel.Code_ExecuteSuccess, buf.Bytes())
}

// ImportConfigFile 导入配置文件
func (s *Server) ImportConfigFile(ctx context.Context,
	configFiles []*apiconfig.ConfigFile, conflictHandling string) *apiconfig.ConfigImportResponse {
	// 预创建命名空间和分组
	for _, configFile := range configFiles {
		if rsp := s.prepareCreateConfigFile(ctx, configFile); rsp.Code.Value != api.ExecuteSuccess {
			return api.NewConfigFileImportResponse(apimodel.Code(rsp.Code.GetValue()), nil, nil, nil)
		}
	}

	// 开启事务
	tx, err := s.storage.StartTx()
	if err != nil {
		return api.NewConfigFileImportResponse(commonstore.StoreCode2APICode(err), nil, nil, nil)
	}
	defer func() { _ = tx.Rollback() }()

	// 记录创建，跳过，覆盖的配置文件
	var (
		createConfigFiles    []*apiconfig.ConfigFile
		skipConfigFiles      []*apiconfig.ConfigFile
		overwriteConfigFiles []*apiconfig.ConfigFile
	)
	for _, configFile := range configFiles {
		namespace := configFile.Namespace.GetValue()
		group := configFile.Group.GetValue()
		name := configFile.Name.GetValue()

		managedFile, err := s.storage.GetConfigFileTx(tx, namespace, group, name)
		if err != nil {
			log.Error("[Config][File] get config file error.", utils.RequestID(ctx),
				utils.ZapNamespace(namespace), utils.ZapGroup(group), utils.ZapFileName(name), zap.Error(err))
			return api.NewConfigFileImportResponse(commonstore.StoreCode2APICode(err), nil, nil, nil)
		}
		// 如果配置文件存在
		if managedFile != nil {
			if conflictHandling == utils.ConfigFileImportConflictSkip {
				skipConfigFiles = append(skipConfigFiles, configFile)
				continue
			} else if conflictHandling == utils.ConfigFileImportConflictOverwrite {
				resp := s.handleUpdateConfigFile(ctx, tx, configFile)
				if resp.GetCode().GetValue() != uint32(apimodel.Code_ExecuteSuccess) {
					log.Error("[Config][File] update config file error.", utils.RequestID(ctx),
						utils.ZapNamespace(namespace), utils.ZapGroup(group), utils.ZapFileName(name), zap.Error(err))
					return api.NewConfigFileImportResponse(commonstore.StoreCode2APICode(err), nil, nil, nil)
				}
				overwriteConfigFiles = append(overwriteConfigFiles, configFile)
				s.RecordHistory(ctx, configFileRecordEntry(ctx, configFile, model.OUpdate))
			}
		} else {
			// 配置文件不存在则创建
			resp := s.handleCreateConfigFile(ctx, tx, configFile)
			if resp.GetCode().GetValue() != uint32(apimodel.Code_ExecuteSuccess) {
				log.Error("[Config][File] create config file error.", utils.RequestID(ctx),
					utils.ZapNamespace(namespace), utils.ZapGroup(group), utils.ZapFileName(name), zap.Error(err))
				return api.NewConfigFileImportResponse(commonstore.StoreCode2APICode(err), nil, nil, nil)
			}
			createConfigFiles = append(createConfigFiles, configFile)
			s.RecordHistory(ctx, configFileRecordEntry(ctx, configFile, model.OCreate))
		}
	}

	if err := tx.Commit(); err != nil {
		log.Error("[Config][File] commit import config file tx error.", utils.RequestID(ctx), zap.Error(err))
		return api.NewConfigFileImportResponse(commonstore.StoreCode2APICode(err), nil, nil, nil)
	}

	return api.NewConfigFileImportResponse(apimodel.Code_ExecuteSuccess,
		createConfigFiles, skipConfigFiles, overwriteConfigFiles)
}

func (s *Server) getGroupAllConfigFiles(namespace, group string) ([]*model.ConfigFile, error) {
	var configFiles []*model.ConfigFile
	offset := uint32(0)
	limit := uint32(100)
	for {
		_, files, err := s.storage.QueryConfigFiles(map[string]string{
			"namespace": namespace,
			"group":     group,
		}, offset, limit)
		if err != nil {
			return nil, err
		}
		if len(files) == 0 {
			break
		}
		configFiles = append(configFiles, files...)
		offset += uint32(len(files))
	}
	return configFiles, nil
}

// GetAllConfigEncryptAlgorithms 获取配置加密算法
func (s *Server) GetAllConfigEncryptAlgorithms(ctx context.Context) *apiconfig.ConfigEncryptAlgorithmResponse {
	if s.cryptoManager == nil {
		return api.NewConfigEncryptAlgorithmResponse(apimodel.Code_ExecuteSuccess, nil)
	}
	var algorithms []*wrapperspb.StringValue
	for _, name := range s.cryptoManager.GetCryptoAlgoNames() {
		algorithms = append(algorithms, utils.NewStringValue(name))
	}
	return api.NewConfigEncryptAlgorithmResponse(apimodel.Code_ExecuteSuccess, algorithms)
}

// configFileRecordEntry 生成服务的记录entry
func configFileRecordEntry(ctx context.Context, req *apiconfig.ConfigFile,
	operationType model.OperationType) *model.RecordEntry {

	marshaler := jsonpb.Marshaler{}
	detail, _ := marshaler.MarshalToString(req)

	entry := &model.RecordEntry{
		ResourceType:  model.RConfigFile,
		ResourceName:  req.GetName().GetValue(),
		Namespace:     req.GetNamespace().GetValue(),
		OperationType: operationType,
		Operator:      utils.ParseOperator(ctx),
		Detail:        detail,
		HappenTime:    time.Now(),
	}

	return entry
}
