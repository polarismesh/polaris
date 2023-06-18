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

func (s *Server) prepareCreateConfigFile(ctx context.Context,
	configFile *apiconfig.ConfigFile) *apiconfig.ConfigResponse {

	userName := utils.ParseUserName(ctx)
	configFile.CreateBy = utils.NewStringValue(userName)
	configFile.ModifyBy = utils.NewStringValue(userName)

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
	return api.NewConfigFileResponse(apimodel.Code_ExecuteSuccess, nil)
}

// CreateConfigFile 创建配置文件
func (s *Server) CreateConfigFile(ctx context.Context, req *apiconfig.ConfigFile) *apiconfig.ConfigResponse {
	if req.Format.GetValue() == "" {
		req.Format = utils.NewStringValue(utils.FileFormatText)
	}
	if checkRsp := s.checkConfigFileParams(req); checkRsp != nil {
		return checkRsp
	}

	namespace := req.Namespace.GetValue()
	group := req.Group.GetValue()
	name := req.Name.GetValue()

	req.ModifyBy = req.CreateBy
	managedFile, err := s.storage.GetConfigFile(s.getTx(ctx), namespace, group, name)
	if err != nil {
		log.Error("[Config][Service] get config file error.", utils.ZapRequestIDByCtx(ctx),
			utils.ZapNamespace(namespace), utils.ZapGroup(group), utils.ZapFileName(name), zap.Error(err))

		return api.NewConfigFileResponse(commonstore.StoreCode2APICode(err), req)
	}
	if managedFile != nil {
		return api.NewConfigFileResponse(apimodel.Code_ExistedResource, req)
	}

	return s.createConfigFile(s.getTx(ctx), ctx, req)
}

func (s *Server) createConfigFile(tx store.Tx, ctx context.Context,
	req *apiconfig.ConfigFile) *apiconfig.ConfigResponse {

	if rsp := s.prepareCreateConfigFile(ctx, req); rsp.Code.Value != api.ExecuteSuccess {
		return rsp
	}

	for i := range s.chains {
		if errResp := s.chains[i].BeforeCreateFile(ctx, req); errResp != nil {
			return errResp
		}
	}

	savaData := model.ToConfigFileStore(req)
	// 创建配置文件
	createdFile, err := s.storage.CreateConfigFile(tx, savaData)
	if err != nil {
		log.Error("[Config][Service] create config file error.", utils.ZapRequestIDByCtx(ctx),
			utils.ZapNamespace(req.GetNamespace().GetValue()), utils.ZapGroup(req.GetGroup().GetValue()),
			utils.ZapFileName(req.GetName().GetValue()), zap.Error(err))
		return api.NewConfigFileResponse(commonstore.StoreCode2APICode(err), req)
	}
	// 创建配置文件标签
	response, success := s.createOrUpdateConfigFileTags(ctx, req, savaData.ModifyBy)
	if !success {
		return response
	}

	log.Info("[Config][Service] create config file success.", utils.ZapRequestIDByCtx(ctx),
		utils.ZapNamespace(req.GetNamespace().GetValue()), utils.ZapGroup(req.GetGroup().GetValue()),
		utils.ZapFileName(req.GetName().GetValue()))
	s.RecordHistory(ctx, configFileRecordEntry(ctx, req, model.OCreate))
	return api.NewConfigFileResponse(apimodel.Code_ExecuteSuccess, model.ToConfigFileAPI(createdFile))
}

// UpdateConfigFile 更新配置文件
func (s *Server) UpdateConfigFile(ctx context.Context, configFile *apiconfig.ConfigFile) *apiconfig.ConfigResponse {
	if checkRsp := s.checkConfigFileParams(configFile); checkRsp != nil {
		return checkRsp
	}

	namespace := configFile.Namespace.GetValue()
	group := configFile.Group.GetValue()
	name := configFile.Name.GetValue()

	saveData, err := s.storage.GetConfigFile(s.getTx(ctx), namespace, group, name)
	if err != nil {
		log.Error("[Config][Service] get config file error.", utils.ZapRequestIDByCtx(ctx),
			utils.ZapNamespace(namespace), utils.ZapGroup(group), utils.ZapFileName(name), zap.Error(err))

		return api.NewConfigFileResponse(commonstore.StoreCode2APICode(err), configFile)
	}
	if saveData == nil {
		return api.NewConfigFileResponse(apimodel.Code_NotFoundResource, configFile)
	}

	configFile.ModifyBy = utils.NewStringValue(utils.ParseUserName(ctx))
	if configFile.Format.GetValue() == "" {
		configFile.Format = wrapperspb.String(saveData.Format)
	}
	return s.updateConfigFile(ctx, configFile)
}

func (s *Server) updateConfigFile(ctx context.Context, configFile *apiconfig.ConfigFile) *apiconfig.ConfigResponse {
	namespace := configFile.Namespace.GetValue()
	group := configFile.Group.GetValue()
	name := configFile.Name.GetValue()

	for i := range s.chains {
		if errResp := s.chains[i].BeforeUpdateFile(ctx, configFile); errResp != nil {
			return errResp
		}
	}

	updateData := model.ToConfigFileStore(configFile)
	updateData.ModifyBy = configFile.GetModifyBy().GetValue()

	updatedFile, err := s.storage.UpdateConfigFile(s.getTx(ctx), updateData)
	if err != nil {
		log.Error("[Config][Service] update config file error.", utils.ZapRequestIDByCtx(ctx),
			utils.ZapNamespace(namespace), utils.ZapGroup(group), utils.ZapFileName(name), zap.Error(err))
		return api.NewConfigFileResponse(commonstore.StoreCode2APICode(err), configFile)
	}

	response, success := s.createOrUpdateConfigFileTags(ctx, configFile, updateData.ModifyBy)
	if !success {
		return response
	}
	s.RecordHistory(ctx, configFileRecordEntry(ctx, configFile, model.OUpdate))
	return api.NewConfigFileResponse(apimodel.Code_ExecuteSuccess, model.ToConfigFileAPI(updatedFile))
}

// DeleteConfigFile 删除配置文件，删除配置文件同时会通知客户端 Not_Found
func (s *Server) DeleteConfigFile(
	ctx context.Context, namespace, group, name, deleteBy string) *apiconfig.ConfigResponse {
	if err := CheckResourceName(utils.NewStringValue(namespace)); err != nil {
		return api.NewConfigFileResponse(apimodel.Code_InvalidNamespaceName, nil)
	}
	if err := CheckResourceName(utils.NewStringValue(group)); err != nil {
		return api.NewConfigFileResponse(apimodel.Code_InvalidConfigFileGroupName, nil)
	}
	if err := CheckFileName(utils.NewStringValue(name)); err != nil {
		return api.NewConfigFileResponse(apimodel.Code_InvalidConfigFileName, nil)
	}

	file, err := s.storage.GetConfigFile(nil, namespace, group, name)
	if err != nil {
		log.Error("[Config][Service] get config file error.", utils.ZapRequestIDByCtx(ctx),
			utils.ZapNamespace(namespace), utils.ZapGroup(group), utils.ZapFileName(name), zap.Error(err))
		return api.NewConfigFileResponse(commonstore.StoreCode2APICode(err), nil)
	}
	if file == nil {
		return api.NewConfigFileResponse(apimodel.Code_ExecuteSuccess, nil)
	}

	tx, ctx, _ := s.StartTxAndSetToContext(ctx)
	defer func() { _ = tx.Rollback() }()

	if deleteBy == "" {
		deleteBy = utils.ParseUserName(ctx)
	}

	// 1. 删除配置文件发布内容
	deleteFileReleaseRsp := s.DeleteConfigFileRelease(ctx, namespace, group, name, deleteBy)
	if deleteFileReleaseRsp.Code.GetValue() != uint32(apimodel.Code_ExecuteSuccess) {
		return api.NewConfigFileResponse(apimodel.Code(deleteFileReleaseRsp.Code.GetValue()), nil)
	}

	// 2. 删除配置文件
	if err = s.storage.DeleteConfigFile(tx, namespace, group, name); err != nil {
		log.Error("[Config][Service] delete config file error.", utils.ZapRequestIDByCtx(ctx),
			utils.ZapNamespace(namespace), utils.ZapGroup(group), utils.ZapFileName(name), zap.Error(err))
		return api.NewConfigFileResponse(commonstore.StoreCode2APICode(err), nil)
	}

	// 3. 删除配置文件关联的 tag
	if err = s.storage.DeleteTagByConfigFile(tx, namespace, group, name); err != nil {
		log.Error("[Config][Service] delete config file tags error.", utils.ZapRequestIDByCtx(ctx),
			utils.ZapNamespace(namespace), utils.ZapGroup(group), utils.ZapFileName(name), zap.Error(err))
		return api.NewConfigFileResponse(commonstore.StoreCode2APICode(err), nil)
	}

	if err := tx.Commit(); err != nil {
		log.Error("[Config][Service] commit delete config file tx error.", utils.ZapRequestIDByCtx(ctx),
			utils.ZapNamespace(namespace), utils.ZapGroup(group), utils.ZapFileName(name), zap.Error(err))
		return api.NewConfigFileResponse(commonstore.StoreCode2APICode(err), nil)
	}

	log.Info("[Config][Service] delete config file success.", utils.ZapRequestIDByCtx(ctx),
		utils.ZapNamespace(namespace), utils.ZapGroup(group), utils.ZapFileName(name))
	s.RecordHistory(ctx, configFileRecordEntry(ctx, &apiconfig.ConfigFile{
		Namespace: utils.NewStringValue(namespace),
		Group:     utils.NewStringValue(group),
		Name:      utils.NewStringValue(name),
	}, model.ODelete))
	return api.NewConfigFileResponse(apimodel.Code_ExecuteSuccess, nil)
}

// BatchDeleteConfigFile 批量删除配置文件
func (s *Server) BatchDeleteConfigFile(ctx context.Context, configFiles []*apiconfig.ConfigFile,
	operator string) *apiconfig.ConfigResponse {
	if len(configFiles) == 0 {
		api.NewConfigFileResponse(apimodel.Code_ExecuteSuccess, nil)
	}
	for _, configFile := range configFiles {
		rsp := s.DeleteConfigFile(ctx, configFile.Namespace.GetValue(),
			configFile.Group.GetValue(), configFile.Name.GetValue(), operator)
		if rsp.Code.GetValue() != uint32(apimodel.Code_ExecuteSuccess) {
			return rsp
		}
	}
	return api.NewConfigFileResponse(apimodel.Code_ExecuteSuccess, nil)
}

// GetConfigFileBaseInfo 获取配置文件，只返回基础元信息
func (s *Server) GetConfigFileBaseInfo(ctx context.Context, namespace, group, name string) *apiconfig.ConfigResponse {
	if err := CheckResourceName(utils.NewStringValue(namespace)); err != nil {
		return api.NewConfigFileResponse(apimodel.Code_InvalidNamespaceName, nil)
	}
	if err := CheckResourceName(utils.NewStringValue(group)); err != nil {
		return api.NewConfigFileResponse(apimodel.Code_InvalidConfigFileGroupName, nil)
	}
	if err := CheckFileName(utils.NewStringValue(name)); err != nil {
		return api.NewConfigFileResponse(apimodel.Code_InvalidConfigFileName, nil)
	}

	file, err := s.storage.GetConfigFile(s.getTx(ctx), namespace, group, name)
	if err != nil {
		log.Error("[Config][Service] get config file error.", utils.ZapRequestIDByCtx(ctx),
			utils.ZapNamespace(namespace), utils.ZapGroup(group), utils.ZapFileName(name), zap.Error(err))
		return api.NewConfigFileResponse(commonstore.StoreCode2APICode(err), nil)
	}
	if file == nil {
		return api.NewConfigFileResponse(apimodel.Code_NotFoundResource, nil)
	}

	retConfigFile, err := s.enrichConfigFile(ctx, model.ToConfigFileAPI(file))
	if err != nil {
		return api.NewConfigFileResponseWithMessage(apimodel.Code_ExecuteException, err.Error())
	}
	return api.NewConfigFileResponse(apimodel.Code_ExecuteSuccess, retConfigFile)
}

// GetConfigFileRichInfo 获取单个配置文件基础信息，包含发布状态等信息
func (s *Server) GetConfigFileRichInfo(ctx context.Context, namespace, group, name string) *apiconfig.ConfigResponse {
	configFileBaseInfoRsp := s.GetConfigFileBaseInfo(ctx, namespace, group, name)
	if configFileBaseInfoRsp.Code.GetValue() != uint32(apimodel.Code_ExecuteSuccess) {
		log.Error("[Config][Service] get config file release error.", utils.ZapRequestIDByCtx(ctx),
			utils.ZapNamespace(namespace), utils.ZapGroup(group), utils.ZapFileName(name))
		return api.NewConfigFileResponse(apimodel.Code(configFileBaseInfoRsp.Code.GetValue()), nil)
	}

	configFileBaseInfo := configFileBaseInfoRsp.ConfigFile
	// 填充发布信息、标签信息等
	configFileBaseInfo, err := s.enrichConfigFile(ctx, configFileBaseInfo)
	if err != nil {
		return api.NewConfigFileResponse(commonstore.StoreCode2APICode(err), nil)
	}
	return api.NewConfigFileResponse(apimodel.Code_ExecuteSuccess, configFileBaseInfo)
}

// QueryConfigFilesByGroup querying configuration files
func (s *Server) QueryConfigFilesByGroup(ctx context.Context, namespace, group string,
	offset, limit uint32) *apiconfig.ConfigBatchQueryResponse {
	if err := CheckResourceName(utils.NewStringValue(namespace)); err != nil {
		return api.NewConfigFileBatchQueryResponse(apimodel.Code_InvalidNamespaceName, 0, nil)
	}
	if err := CheckResourceName(utils.NewStringValue(group)); err != nil {
		return api.NewConfigFileBatchQueryResponse(apimodel.Code_InvalidConfigFileGroupName, 0, nil)
	}
	if limit > MaxPageSize {
		limit = MaxPageSize
	}
	count, files, err := s.storage.QueryConfigFilesByGroup(namespace, group, offset, limit)
	if err != nil {
		log.Error("[Config][Service]get config files by group error.", utils.ZapRequestIDByCtx(ctx),
			utils.ZapNamespace(namespace), utils.ZapGroup(group), zap.Error(err))
		return api.NewConfigFileBatchQueryResponse(commonstore.StoreCode2APICode(err), 0, nil)
	}

	if len(files) == 0 {
		return api.NewConfigFileBatchQueryResponse(apimodel.Code_ExecuteSuccess, count, nil)
	}

	var fileAPIModels []*apiconfig.ConfigFile
	for _, file := range files {
		baseFile := model.ToConfigFileAPI(file)
		baseFile, err = s.enrichConfigFile(ctx, baseFile)
		if err != nil {
			return api.NewConfigFileBatchQueryResponse(commonstore.StoreCode2APICode(err), 0, nil)
		}
		fileAPIModels = append(fileAPIModels, baseFile)
	}
	return api.NewConfigFileBatchQueryResponse(apimodel.Code_ExecuteSuccess, count, fileAPIModels)
}

// SearchConfigFile 查询配置文件
func (s *Server) SearchConfigFile(ctx context.Context, namespace, group, name, tags string,
	offset, limit uint32) *apiconfig.ConfigBatchQueryResponse {
	if err := CheckResourceName(utils.NewStringValue(namespace)); err != nil {
		return api.NewConfigFileBatchQueryResponse(apimodel.Code_InvalidNamespaceName, 0, nil)
	}
	if limit > MaxPageSize {
		limit = MaxPageSize
	}
	if len(tags) == 0 {
		return s.queryConfigFileWithoutTags(ctx, namespace, group, name, offset, limit)
	}

	// 按tag搜索，内存分页
	tagKVs := strings.Split(tags, ",")
	if len(tagKVs)%2 != 0 {
		return api.NewConfigFileBatchQueryResponse(apimodel.Code_InvalidConfigFileTags, 0, nil)
	}

	count, files, err := s.queryConfigFileByTags(ctx, namespace, group, name, offset, limit, tagKVs...)
	if err != nil {
		log.Error("[Config][Service] query config file tags error.", utils.ZapRequestIDByCtx(ctx),
			utils.ZapNamespace(namespace), utils.ZapGroup(group), utils.ZapFileName(name), zap.Error(err))
		return api.NewConfigFileBatchQueryResponse(commonstore.StoreCode2APICode(err), 0, nil)
	}

	// Rendering configuration files, because only the main key information is obtained from the TAG table
	enrichedFiles := make([]*apiconfig.ConfigFile, 0, len(files))
	for _, file := range files {
		rsp := s.GetConfigFileRichInfo(ctx, file.Namespace, file.Group, file.FileName)
		if rsp.Code.GetValue() != uint32(apimodel.Code_ExecuteSuccess) {
			return api.NewConfigFileBatchQueryResponse(apimodel.Code(rsp.Code.GetValue()), 0, nil)
		}
		enrichedFiles = append(enrichedFiles, rsp.ConfigFile)
	}
	return api.NewConfigFileBatchQueryResponse(apimodel.Code_ExecuteSuccess, uint32(count), enrichedFiles)
}

func (s *Server) queryConfigFileWithoutTags(ctx context.Context, namespace, group, name string,
	offset, limit uint32) *apiconfig.ConfigBatchQueryResponse {
	count, files, err := s.storage.QueryConfigFiles(namespace, group, name, offset, limit)
	if err != nil {
		log.Error("[Config][Service]search config file error.", utils.ZapRequestIDByCtx(ctx),
			utils.ZapNamespace(namespace), utils.ZapGroup(group), utils.ZapFileName(name), zap.Error(err))
		return api.NewConfigFileBatchQueryResponse(commonstore.StoreCode2APICode(err), 0, nil)
	}

	if len(files) == 0 {
		return api.NewConfigFileBatchQueryResponse(apimodel.Code_ExecuteSuccess, count, nil)
	}

	fileAPIModels := make([]*apiconfig.ConfigFile, 0, len(files))

	for _, file := range files {
		baseFile, err := s.enrichConfigFile(ctx, model.ToConfigFileAPI(file))
		if err != nil {
			return api.NewConfigFileBatchQueryResponse(commonstore.StoreCode2APICode(err), 0, nil)
		}
		fileAPIModels = append(fileAPIModels, baseFile)
	}
	return api.NewConfigFileBatchQueryResponse(apimodel.Code_ExecuteSuccess, count, fileAPIModels)
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
	if err := CheckResourceName(configFileExport.Namespace); err != nil {
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
				log.Error("[Config][Service] get config file by group error.", utils.ZapRequestIDByCtx(ctx),
					utils.ZapNamespace(namespace), utils.ZapGroup(group), zap.Error(err))
				return api.NewConfigFileExportResponse(commonstore.StoreCode2APICode(err), nil)
			}
			configFiles = append(configFiles, files...)
		}
	} else if len(groups) == 1 && len(names) > 0 {
		// 导出配置文件
		for _, name := range names {
			file, err := s.storage.GetConfigFile(nil, namespace, groups[0], name)
			if err != nil {
				log.Error("[Config][Service] get config file error.", utils.ZapRequestIDByCtx(ctx),
					utils.ZapNamespace(namespace), utils.ZapGroup(groups[0]), utils.ZapFileName(name),
					zap.Error(err))
				return api.NewConfigFileExportResponse(commonstore.StoreCode2APICode(err), nil)
			}
			configFiles = append(configFiles, file)
		}
	} else {
		log.Error("[Config][Service] export config file error.", utils.ZapRequestIDByCtx(ctx),
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
		tags, err := s.storage.QueryTagByConfigFile(file.Namespace, file.Group, file.Name)
		if err != nil {
			log.Error("[Config][Servie]query config file tag error.", utils.ZapRequestIDByCtx(ctx),
				utils.ZapNamespace(file.Namespace), utils.ZapGroup(file.Group), utils.ZapFileName(file.Name),
				zap.Error(err))
			return api.NewConfigFileExportResponse(commonstore.StoreCode2APICode(err), nil)
		}
		// 加密配置创建人可以导出加密密钥
		userName := utils.ParseUserName(ctx)
		filterTags := make([]*model.ConfigFileTag, 0, len(tags))
		for _, tag := range tags {
			if tag.Key == utils.ConfigFileTagKeyDataKey {
				if userName == file.CreateBy {
					filterTags = append(filterTags, tag)
				}
			} else {
				filterTags = append(filterTags, tag)
			}
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
		if checkRsp := s.checkConfigFileParams(configFile); checkRsp != nil {
			return api.NewConfigFileImportResponse(apimodel.Code(checkRsp.Code.GetValue()), nil, nil, nil)
		}
		if rsp := s.prepareCreateConfigFile(ctx, configFile); rsp.Code.Value != api.ExecuteSuccess {
			return api.NewConfigFileImportResponse(apimodel.Code(rsp.Code.GetValue()), nil, nil, nil)
		}
	}

	// 开启事务
	tx, ctx, _ := s.StartTxAndSetToContext(ctx)
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

		managedFile, err := s.storage.GetConfigFile(s.getTx(ctx), namespace, group, name)
		if err != nil {
			log.Error("[Config][Service] get config file error.", utils.ZapRequestIDByCtx(ctx),
				utils.ZapNamespace(namespace), utils.ZapGroup(group), utils.ZapFileName(name), zap.Error(err))
			return api.NewConfigFileImportResponse(commonstore.StoreCode2APICode(err), nil, nil, nil)
		}
		// 如果配置文件存在
		if managedFile != nil {
			if conflictHandling == utils.ConfigFileImportConflictSkip {
				skipConfigFiles = append(skipConfigFiles, configFile)
				continue
			} else if conflictHandling == utils.ConfigFileImportConflictOverwrite {
				updatedFile, err := s.storage.UpdateConfigFile(s.getTx(ctx), model.ToConfigFileStore(configFile))
				if err != nil {
					log.Error("[Config][Service] update config file error.", utils.ZapRequestIDByCtx(ctx),
						utils.ZapNamespace(namespace), utils.ZapGroup(group), utils.ZapFileName(name), zap.Error(err))
					return api.NewConfigFileImportResponse(commonstore.StoreCode2APICode(err), nil, nil, nil)
				}
				if response, success := s.createOrUpdateConfigFileTags(ctx, configFile, utils.ParseUserName(ctx)); !success {
					return api.NewConfigFileImportResponse(apimodel.Code(response.Code.GetValue()), nil, nil, nil)
				}
				overwriteConfigFiles = append(overwriteConfigFiles, model.ToConfigFileAPI(updatedFile))
				s.RecordHistory(ctx, configFileRecordEntry(ctx, configFile, model.OUpdate))
			}
		} else {
			// 配置文件不存在则创建
			createdFile, err := s.storage.CreateConfigFile(s.getTx(ctx), model.ToConfigFileStore(configFile))
			if err != nil {
				log.Error("[Config][Service] create config file error.", utils.ZapRequestIDByCtx(ctx),
					utils.ZapNamespace(namespace), utils.ZapGroup(group), utils.ZapFileName(name), zap.Error(err))
				return api.NewConfigFileImportResponse(commonstore.StoreCode2APICode(err), nil, nil, nil)
			}
			if response, success := s.createOrUpdateConfigFileTags(ctx, configFile, utils.ParseUserName(ctx)); !success {
				return api.NewConfigFileImportResponse(apimodel.Code(response.Code.GetValue()), nil, nil, nil)
			}
			createConfigFiles = append(createConfigFiles, model.ToConfigFileAPI(createdFile))
			s.RecordHistory(ctx, configFileRecordEntry(ctx, configFile, model.OCreate))
		}
	}

	if err := tx.Commit(); err != nil {
		log.Error("[Config][Service] commit import config file tx error.", utils.ZapRequestIDByCtx(ctx), zap.Error(err))
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
		_, files, err := s.storage.QueryConfigFilesByGroup(namespace, group, offset, limit)
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

func (s *Server) checkConfigFileParams(configFile *apiconfig.ConfigFile) *apiconfig.ConfigResponse {
	if configFile == nil {
		return api.NewConfigFileResponse(apimodel.Code_InvalidParameter, configFile)
	}
	if err := CheckFileName(configFile.Name); err != nil {
		return api.NewConfigFileResponse(apimodel.Code_InvalidConfigFileName, configFile)
	}
	if err := CheckResourceName(configFile.Namespace); err != nil {
		return api.NewConfigFileResponse(apimodel.Code_InvalidNamespaceName, configFile)
	}
	if err := CheckContentLength(configFile.Content.GetValue(), int(s.cfg.ContentMaxLength)); err != nil {
		return api.NewConfigFileResponse(apimodel.Code_InvalidConfigFileContentLength, configFile)
	}
	if len(configFile.Tags) > 0 {
		for _, tag := range configFile.Tags {
			if tag.Key.GetValue() == "" || tag.Value.GetValue() == "" {
				return api.NewConfigFileResponse(apimodel.Code_InvalidConfigFileTags, configFile)
			}
		}
	}
	return nil
}

func (s *Server) createOrUpdateConfigFileTags(ctx context.Context, configFile *apiconfig.ConfigFile,
	operator string) (*apiconfig.ConfigResponse, bool) {
	var (
		namespace = configFile.Namespace.GetValue()
		group     = configFile.Group.GetValue()
		name      = configFile.Name.GetValue()
		tags      = make([]string, 0, len(configFile.Tags)*2)
	)

	for _, tag := range configFile.Tags {
		tags = append(tags, tag.Key.GetValue())
		tags = append(tags, tag.Value.GetValue())
	}
	if err := s.createConfigFileTags(ctx, namespace, group, name, operator, tags...); err != nil {
		log.Error("[Config][Service] create or update config file tags error.", utils.ZapRequestIDByCtx(ctx),
			utils.ZapNamespace(namespace), utils.ZapGroup(group), utils.ZapFileName(name), zap.Error(err))
		return api.NewConfigFileResponse(commonstore.StoreCode2APICode(err), configFile), false
	}
	return nil, true
}

func (s *Server) enrichConfigFile(ctx context.Context, file *apiconfig.ConfigFile) (*apiconfig.ConfigFile, error) {
	namespace := file.Namespace.GetValue()
	group := file.Group.GetValue()
	name := file.Name.GetValue()

	// 填充标签信息
	tags, err := s.queryTagsByConfigFileWithAPIModels(ctx, namespace, group, name)
	if err != nil {
		log.Error("[Config][Service] create config file error.", utils.ZapRequestIDByCtx(ctx),
			utils.ZapNamespace(namespace), utils.ZapGroup(group), utils.ZapFileName(name), zap.Error(err))
		return nil, err
	}
	file.Tags = tags

	for i := range s.chains {
		_file, err := s.chains[i].AfterGetFile(ctx, file)
		if err != nil {
			return nil, err
		}
		file = _file
	}
	return file, nil
}

// getConfigFileDataKey 获取加密配置文件数据密钥
func (s *Server) getEncryptAlgorithmAndDataKey(ctx context.Context,
	namespace, group, fileName string) (string, string, error) {
	tags, err := s.queryTagsByConfigFileWithAPIModels(ctx, namespace, group, fileName)
	if err != nil {
		return "", "", err
	}
	var (
		algorithm string
		dataKey   string
	)
	for _, tag := range tags {
		if tag.Key.GetValue() == utils.ConfigFileTagKeyDataKey {
			dataKey = tag.Value.GetValue()
		}
		if tag.Key.GetValue() == utils.ConfigFileTagKeyEncryptAlgo {
			algorithm = tag.Value.GetValue()
		}
	}
	return algorithm, dataKey, nil
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
