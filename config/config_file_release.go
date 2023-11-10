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
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/golang/protobuf/jsonpb"
	apiconfig "github.com/polarismesh/specification/source/go/api/v1/config_manage"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	"go.uber.org/zap"

	cachetypes "github.com/polarismesh/polaris/cache/api"
	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/model"
	commonstore "github.com/polarismesh/polaris/common/store"
	commontime "github.com/polarismesh/polaris/common/time"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/store"
)

// PublishConfigFile 发布配置文件
func (s *Server) PublishConfigFile(ctx context.Context, req *apiconfig.ConfigFileRelease) *apiconfig.ConfigResponse {

	if err := CheckFileName(req.GetFileName()); err != nil {
		return api.NewConfigResponse(apimodel.Code_InvalidConfigFileName)
	}
	if err := utils.CheckResourceName(req.GetNamespace()); err != nil {
		return api.NewConfigResponse(apimodel.Code_InvalidNamespaceName)
	}
	if err := utils.CheckResourceName(req.GetGroup()); err != nil {
		return api.NewConfigResponse(apimodel.Code_InvalidConfigFileGroupName)
	}
	if !s.checkNamespaceExisted(req.GetNamespace().GetValue()) {
		return api.NewConfigResponse(apimodel.Code_NotFoundNamespace)
	}

	if req.GetType().GetValue() != uint32(model.ReleaseTypeGray) && req.GetType().GetValue() != uint32(model.ReleaseTypeFull) {
		return api.NewConfigResponse(apimodel.Code_InvalidParameter)
	}
	if req.GetType().GetValue() == uint32(model.ReleaseTypeGray) && req.GetGrayRule() == nil {
		return api.NewConfigResponse(apimodel.Code_InvalidMatchRule)
	}

	tx, err := s.storage.StartTx()
	if err != nil {
		log.Error("[Config][Release] publish config file begin tx.", utils.RequestID(ctx), zap.Error(err))
		return api.NewConfigResponse(commonstore.StoreCode2APICode(err))
	}
	defer func() {
		_ = tx.Rollback()
	}()

	data, resp := s.handlePublishConfigFile(ctx, tx, req)
	if resp.GetCode().GetValue() != uint32(apimodel.Code_ExecuteSuccess) {
		_ = tx.Rollback()
		if data != nil {
			s.recordReleaseFail(ctx, utils.ReleaseTypeNormal, data, errors.New(resp.GetInfo().GetValue()))
		}
		return resp
	}

	if err := tx.Commit(); err != nil {
		s.recordReleaseFail(ctx, utils.ReleaseTypeNormal, data, err)
		log.Error("[Config][Release] publish config file commit tx.", utils.RequestID(ctx), zap.Error(err))
		return api.NewConfigResponse(commonstore.StoreCode2APICode(err))
	}
	if req.GetType().GetValue() == uint32(model.ReleaseTypeFull) {
		s.recordReleaseSuccess(ctx, utils.ReleaseTypeNormal, data)
	} else {
		s.recordReleaseSuccess(ctx, utils.ReleaseTypeGray, data)
	}

	resp.ConfigFileRelease = req
	return resp
}

func (s *Server) nextSequence() int64 {
	return atomic.AddInt64(&s.sequence, 1)
}

// PublishConfigFile 发布配置文件
func (s *Server) handlePublishConfigFile(ctx context.Context, tx store.Tx,
	req *apiconfig.ConfigFileRelease) (*model.ConfigFileRelease, *apiconfig.ConfigResponse) {
	namespace := req.GetNamespace().GetValue()
	group := req.GetGroup().GetValue()
	fileName := req.GetFileName().GetValue()

	// 获取待发布的 configFile 信息
	toPublishFile, err := s.storage.GetConfigFileTx(tx, namespace, group, fileName)
	if err != nil {
		log.Error("[Config][Release] publish config file when get file.", utils.RequestID(ctx),
			utils.ZapNamespace(namespace), utils.ZapGroup(group), utils.ZapFileName(fileName),
			zap.Error(err))
		s.recordReleaseFail(ctx, utils.ReleaseTypeNormal, model.ToConfigFileReleaseStore(req), err)
		return nil, api.NewConfigResponse(commonstore.StoreCode2APICode(err))
	}
	if toPublishFile == nil {
		return nil, api.NewConfigResponse(apimodel.Code_NotFoundResource)
	}
	if releaseName := req.GetName().GetValue(); releaseName == "" {
		// 这里要保证每一次发布都有唯一的 release_name 名称
		req.Name = utils.NewStringValue(fmt.Sprintf("%s-%d-%d", fileName, time.Now().Unix(), s.nextSequence()))
	}

	fileRelease := &model.ConfigFileRelease{
		SimpleConfigFileRelease: &model.SimpleConfigFileRelease{
			ConfigFileReleaseKey: &model.ConfigFileReleaseKey{
				Name:      req.GetName().GetValue(),
				Namespace: namespace,
				Group:     group,
				FileName:  fileName,
				Typ:       model.ReleaseType(req.GetType().GetValue()),
			},
			Format:             toPublishFile.Format,
			Metadata:           toPublishFile.Metadata,
			Comment:            req.GetComment().GetValue(),
			Md5:                CalMd5(toPublishFile.Content),
			CreateBy:           utils.ParseUserName(ctx),
			ModifyBy:           utils.ParseUserName(ctx),
			ReleaseDescription: req.GetReleaseDescription().GetValue(),
		},
		Content: toPublishFile.Content,
	}
	saveRelease, err := s.storage.GetConfigFileReleaseTx(tx, fileRelease.ConfigFileReleaseKey)
	if err != nil {
		log.Error("[Config][Release] publish config file when get release.",
			utils.RequestID(ctx), utils.ZapNamespace(namespace), utils.ZapGroup(group),
			utils.ZapFileName(fileName), zap.Error(err))
		return fileRelease, api.NewConfigResponse(commonstore.StoreCode2APICode(err))
	}
	// 重新激活
	if saveRelease != nil {
		if err := s.storage.ActiveConfigFileReleaseTx(tx, fileRelease); err != nil {
			log.Error("[Config][Release] re-active config file release error.",
				utils.RequestID(ctx), utils.ZapNamespace(namespace), utils.ZapGroup(group),
				utils.ZapFileName(fileName), zap.Error(err))
			return fileRelease, api.NewConfigFileResponse(commonstore.StoreCode2APICode(err), nil)
		}
	} else {
		if err := s.storage.CreateConfigFileReleaseTx(tx, fileRelease); err != nil {
			log.Error("[Config][Release] publish config file when create release.",
				utils.RequestID(ctx), utils.ZapNamespace(namespace), utils.ZapGroup(group),
				utils.ZapFileName(fileName), zap.Error(err))
			return fileRelease, api.NewConfigResponse(commonstore.StoreCode2APICode(err))
		}
	}
	if req.GetType().GetValue() == uint32(model.ReleaseTypeGray) {
		grayRule := req.GetGrayRule()
		var buffer bytes.Buffer
		marshaler := jsonpb.Marshaler{}
		err := marshaler.Marshal(&buffer, grayRule)
		if err != nil {
			if err != nil {
				log.Error("[Config][Release] marshal gary rule error.",
					utils.RequestID(ctx), utils.ZapNamespace(namespace), utils.ZapGroup(group),
					utils.ZapFileName(fileName), zap.Error(err))
				return fileRelease, api.NewConfigResponse(apimodel.Code_InvalidMatchRule)
			}
		}
		grayResource := &model.GrayResource{
			Name:      model.GetGrayConfigRealseKey(fileRelease.SimpleConfigFileRelease),
			MatchRule: buffer.String(),
			CreateBy:  utils.ParseUserName(ctx),
			ModifyBy:  utils.ParseUserName(ctx),
		}
		if err := s.storage.CreateGrayResourceTx(tx, grayResource); err != nil {
			log.Error("[Config][Release] create gray resource error.",
				utils.RequestID(ctx), utils.ZapNamespace(namespace), utils.ZapGroup(group),
				utils.ZapFileName(fileName), zap.Error(err))
			return fileRelease, api.NewConfigFileResponse(commonstore.StoreCode2APICode(err), nil)
		}
	}

	s.RecordHistory(ctx, configFileReleaseRecordEntry(ctx, req, fileRelease, model.OCreate))
	return fileRelease, api.NewConfigResponse(apimodel.Code_ExecuteSuccess)
}

// GetConfigFileRelease 获取配置文件发布内容
func (s *Server) GetConfigFileRelease(ctx context.Context, req *apiconfig.ConfigFileRelease) *apiconfig.ConfigResponse {
	namespace := req.GetNamespace().GetValue()
	group := req.GetGroup().GetValue()
	fileName := req.GetFileName().GetValue()
	releaseName := req.GetName().GetValue()

	if errCode, errMsg := checkBaseReleaseParam(req, false); errCode != apimodel.Code_ExecuteSuccess {
		return api.NewConfigResponseWithInfo(errCode, errMsg)
	}
	var (
		ret *model.ConfigFileRelease
		err error
	)

	// 如果没有指定专门的 releaseName，则直接查询 active 状态的配置发布, 兼容老的控制台查询逻辑
	if releaseName != "" {
		ret, err = s.storage.GetConfigFileRelease(&model.ConfigFileReleaseKey{
			Namespace: namespace,
			Group:     group,
			FileName:  fileName,
			Name:      releaseName,
		})
	} else {
		ret, err = s.storage.GetConfigFileActiveRelease(&model.ConfigFileKey{
			Namespace: namespace,
			Group:     group,
			Name:      fileName,
		})
	}

	if err != nil {
		log.Error("[Config][Release] get config file release.", utils.RequestID(ctx),
			utils.ZapNamespace(namespace), utils.ZapGroup(group), utils.ZapFileName(fileName), zap.Error(err))
		return api.NewConfigResponse(commonstore.StoreCode2APICode(err))
	}
	if ret == nil {
		return api.NewConfigResponse(apimodel.Code_ExecuteSuccess)
	}

	ret, err = s.chains.AfterGetFileRelease(ctx, ret)
	if err != nil {
		log.Error("[Config][Release] get config file release run chain.", utils.RequestID(ctx),
			utils.ZapNamespace(namespace), utils.ZapGroup(group), utils.ZapFileName(fileName), zap.Error(err))
		out := api.NewConfigResponse(apimodel.Code_ExecuteException)
		return out
	}

	release := model.ToConfiogFileReleaseApi(ret)
	if ret.Typ == model.ReleaseTypeGray {
		key := model.GetGrayConfigRealseKey(ret.SimpleConfigFileRelease)
		if grayRule := s.grayCache.GetGrayRule(key); grayRule == nil {
			return api.NewConfigResponse(apimodel.Code_InvalidMatchRule)
		} else {
			release.GrayRule = grayRule
		}
	}
	return api.NewConfigFileReleaseResponse(apimodel.Code_ExecuteSuccess, release)
}

// DeleteConfigFileRelease 删除某个配置文件的发布 release
func (s *Server) DeleteConfigFileReleases(ctx context.Context,
	reqs []*apiconfig.ConfigFileRelease) *apiconfig.ConfigBatchWriteResponse {

	responses := api.NewConfigBatchWriteResponse(apimodel.Code_ExecuteSuccess)
	chs := make([]chan *apiconfig.ConfigResponse, 0, len(reqs))
	for i, instance := range reqs {
		chs = append(chs, make(chan *apiconfig.ConfigResponse))
		go func(index int, ins *apiconfig.ConfigFileRelease) {
			chs[index] <- s.handleDeleteConfigFileRelease(ctx, ins)
		}(i, instance)
	}

	for _, ch := range chs {
		resp := <-ch
		api.ConfigCollect(responses, resp)
	}
	return responses
}

func (s *Server) handleDeleteConfigFileRelease(ctx context.Context,
	req *apiconfig.ConfigFileRelease) *apiconfig.ConfigResponse {

	if errCode, errMsg := checkBaseReleaseParam(req, true); errCode != apimodel.Code_ExecuteSuccess {
		return api.NewConfigResponseWithInfo(errCode, errMsg)
	}
	release := &model.ConfigFileRelease{
		SimpleConfigFileRelease: &model.SimpleConfigFileRelease{
			ConfigFileReleaseKey: &model.ConfigFileReleaseKey{
				Name:      req.GetName().GetValue(),
				Namespace: req.GetNamespace().GetValue(),
				Group:     req.GetGroup().GetValue(),
				FileName:  req.GetFileName().GetValue(),
				Typ:       model.ReleaseType(req.GetType().GetValue()),
			},
		},
	}
	var (
		errRef     error
		needRecord = true
		recordData *model.ConfigFileRelease
	)
	defer func() {
		if !needRecord {
			return
		}
		if errRef != nil {
			s.recordReleaseFail(ctx, utils.ReleaseTypeDelete, recordData, errRef)
		} else {
			s.recordReleaseSuccess(ctx, utils.ReleaseTypeDelete, recordData)
		}
	}()

	tx, err := s.storage.StartTx()
	if err != nil {
		log.Error("[Config][File] delete config file release when begin tx.",
			utils.RequestID(ctx), zap.Error(err))
		return api.NewConfigResponse(commonstore.StoreCode2APICode(err))
	}
	defer func() {
		_ = tx.Rollback()
	}()
	if _, err := s.storage.LockConfigFile(tx, release.ToFileKey()); err != nil {
		errRef = err
		log.Error("[Config][File] delete config file release when lock.",
			utils.RequestID(ctx), zap.Error(err))
		return api.NewConfigResponse(commonstore.StoreCode2APICode(err))
	}

	saveData, err := s.storage.GetConfigFileReleaseTx(tx, release.ConfigFileReleaseKey)
	if err != nil {
		errRef = err
		return api.NewConfigResponse(commonstore.StoreCode2APICode(err))
	}
	recordData = saveData
	if saveData == nil {
		needRecord = false
		return api.NewConfigResponse(apimodel.Code_ExecuteSuccess)
	}
	// 如果存在处于 active 状态的配置，重新在激活一下，触发版本的更新变动
	if saveData.Active {
		if err := s.storage.ActiveConfigFileReleaseTx(tx, saveData); err != nil {
			errRef = err
			log.Error("[Config][File] delete config file release when re-active.",
				utils.RequestID(ctx), zap.Error(err))
			return api.NewConfigResponse(commonstore.StoreCode2APICode(err))
		}
	}

	if err := s.storage.DeleteConfigFileReleaseTx(tx, saveData.ConfigFileReleaseKey); err != nil {
		log.Error("[Config][Release] delete config file release error.",
			utils.RequestID(ctx), utils.ZapNamespace(req.GetNamespace().GetValue()),
			utils.ZapGroup(req.GetGroup().GetValue()), utils.ZapFileName(req.GetFileName().GetValue()),
			zap.Error(err))
		errRef = err
		return api.NewConfigResponse(commonstore.StoreCode2APICode(err))
	}

	if err := tx.Commit(); err != nil {
		log.Error("[Config][Release] delete config file release when commit tx.",
			utils.RequestID(ctx), utils.ZapNamespace(req.GetNamespace().GetValue()),
			utils.ZapGroup(req.GetGroup().GetValue()), utils.ZapFileName(req.GetFileName().GetValue()),
			zap.Error(err))
		errRef = err
		return api.NewConfigResponse(commonstore.StoreCode2APICode(err))
	}

	s.RecordHistory(ctx, configFileReleaseRecordEntry(ctx, req, release, model.ODelete))
	return api.NewConfigResponse(apimodel.Code_ExecuteSuccess)
}

func (s *Server) GetConfigFileReleaseVersions(ctx context.Context,
	filters map[string]string) *apiconfig.ConfigBatchQueryResponse {

	searchFilters := map[string]string{}
	for k, v := range filters {
		if nk, ok := availableSearch["config_file_release"][k]; ok {
			searchFilters[nk] = v
		}
	}

	namespace := searchFilters["namespace"]
	group := searchFilters["group"]
	fileName := searchFilters["file_name"]
	if namespace == "" {
		return api.NewConfigBatchQueryResponseWithInfo(apimodel.Code_BadRequest, "invalid namespace")
	}
	if group == "" {
		return api.NewConfigBatchQueryResponseWithInfo(apimodel.Code_BadRequest, "invalid config group")
	}
	if fileName == "" {
		return api.NewConfigBatchQueryResponseWithInfo(apimodel.Code_BadRequest, "invalid config file name")
	}
	args := cachetypes.ConfigReleaseArgs{
		BaseConfigArgs: cachetypes.BaseConfigArgs{
			Namespace: filters["namespace"],
			Group:     filters["group"],
		},
		FileName:   filters["file_name"],
		OnlyActive: false,
		NoPage:     true,
	}
	return s.handleDescribeConfigFileReleases(ctx, args)
}

func (s *Server) GetConfigFileReleases(ctx context.Context,
	filter map[string]string) *apiconfig.ConfigBatchQueryResponse {

	searchFilters := map[string]string{}
	for k, v := range filter {
		if nK, ok := availableSearch["config_file_release"][k]; ok {
			searchFilters[nK] = v
		}
	}

	offset, limit, err := utils.ParseOffsetAndLimit(filter)
	if err != nil {
		return api.NewConfigBatchQueryResponseWithInfo(apimodel.Code_BadRequest, err.Error())
	}

	args := cachetypes.ConfigReleaseArgs{
		BaseConfigArgs: cachetypes.BaseConfigArgs{
			Namespace:  searchFilters["namespace"],
			Group:      searchFilters["group"],
			Offset:     offset,
			Limit:      limit,
			OrderField: searchFilters["order_field"],
			OrderType:  searchFilters["order_type"],
		},
		FileName:    searchFilters["file_name"],
		ReleaseName: searchFilters["release_name"],
		OnlyActive:  strings.Compare(searchFilters["only_active"], "true") == 0,
		IncludeGray: false,
	}
	return s.handleDescribeConfigFileReleases(ctx, args)
}

func (s *Server) handleDescribeConfigFileReleases(ctx context.Context,
	args cachetypes.ConfigReleaseArgs) *apiconfig.ConfigBatchQueryResponse {

	total, simpleReleases, err := s.fileCache.QueryReleases(&args)
	if err != nil {
		return api.NewConfigBatchQueryResponseWithInfo(apimodel.Code_ExecuteException, err.Error())
	}
	ret := make([]*apiconfig.ConfigFileRelease, 0, len(simpleReleases))
	for i := range simpleReleases {
		item := simpleReleases[i]
		ret = append(ret, &apiconfig.ConfigFileRelease{
			Id:                 utils.NewUInt64Value(item.Id),
			Name:               utils.NewStringValue(item.Name),
			Namespace:          utils.NewStringValue(item.Namespace),
			Group:              utils.NewStringValue(item.Group),
			FileName:           utils.NewStringValue(item.FileName),
			Format:             utils.NewStringValue(item.Format),
			Version:            utils.NewUInt64Value(item.Version),
			Active:             utils.NewBoolValue(item.Active),
			CreateTime:         utils.NewStringValue(commontime.Time2String(item.CreateTime)),
			ModifyTime:         utils.NewStringValue(commontime.Time2String(item.ModifyTime)),
			CreateBy:           utils.NewStringValue(item.CreateBy),
			ModifyBy:           utils.NewStringValue(item.ModifyBy),
			ReleaseDescription: utils.NewStringValue(item.ReleaseDescription),
			Tags:               model.FromTagMap(item.Metadata),
			Type:               utils.NewUInt32Value(uint32(item.Typ)),
		})
	}

	resp := api.NewConfigBatchQueryResponse(apimodel.Code_ExecuteSuccess)
	resp.Total = utils.NewUInt32Value(total)
	resp.ConfigFileReleases = ret
	return resp
}

func (s *Server) RollbackConfigFileReleases(ctx context.Context,
	reqs []*apiconfig.ConfigFileRelease) *apiconfig.ConfigBatchWriteResponse {

	responses := api.NewConfigBatchWriteResponse(apimodel.Code_ExecuteSuccess)
	chs := make([]chan *apiconfig.ConfigResponse, 0, len(reqs))
	for i, instance := range reqs {
		chs = append(chs, make(chan *apiconfig.ConfigResponse))
		go func(index int, ins *apiconfig.ConfigFileRelease) {
			chs[index] <- s.RollbackConfigFileRelease(ctx, ins)
		}(i, instance)
	}

	for _, ch := range chs {
		resp := <-ch
		api.ConfigCollect(responses, resp)
	}
	return responses
}

// RollbackConfigFileRelease 回滚配置
func (s *Server) RollbackConfigFileRelease(ctx context.Context,
	req *apiconfig.ConfigFileRelease) *apiconfig.ConfigResponse {
	if errCode, errMsg := checkBaseReleaseParam(req, true); errCode != apimodel.Code_ExecuteSuccess {
		return api.NewConfigResponseWithInfo(errCode, errMsg)
	}
	data := &model.ConfigFileRelease{
		SimpleConfigFileRelease: &model.SimpleConfigFileRelease{
			ConfigFileReleaseKey: &model.ConfigFileReleaseKey{
				Name:      req.GetName().GetValue(),
				Namespace: req.GetNamespace().GetValue(),
				Group:     req.GetGroup().GetValue(),
				FileName:  req.GetFileName().GetValue(),
				Typ:       model.ReleaseTypeFull,
			},
		},
	}

	tx, err := s.storage.StartTx()
	if err != nil {
		log.Error("[Config][File] rollback config file releasw when begin tx.",
			utils.RequestID(ctx), zap.Error(err))
		return api.NewConfigResponse(commonstore.StoreCode2APICode(err))
	}
	defer func() {
		_ = tx.Rollback()
	}()

	targetRelease, ret := s.handleRollbackConfigFileRelease(ctx, tx, data)
	if targetRelease != nil {
		data = targetRelease
	}
	if ret != nil {
		_ = tx.Rollback()
		s.recordReleaseFail(ctx, utils.ReleaseTypeRollback, data, errors.New(ret.GetInfo().GetValue()))
		return ret
	}

	if err := tx.Commit(); err != nil {
		log.Error("[Config][File] rollback config file releasw when commit tx.",
			utils.RequestID(ctx), zap.Error(err))
		s.recordReleaseFail(ctx, utils.ReleaseTypeRollback, data, err)
		return api.NewConfigResponse(commonstore.StoreCode2APICode(err))
	}

	s.recordReleaseSuccess(ctx, utils.ReleaseTypeRollback, data)
	s.RecordHistory(ctx, configFileReleaseRecordEntry(ctx, req, data, model.ORollback))
	return api.NewConfigResponse(apimodel.Code_ExecuteSuccess)
}

// handleRollbackConfigFileRelease 回滚配置
func (s *Server) handleRollbackConfigFileRelease(ctx context.Context, tx store.Tx,
	data *model.ConfigFileRelease) (*model.ConfigFileRelease, *apiconfig.ConfigResponse) {

	targetRelease, err := s.storage.GetConfigFileReleaseTx(tx, data.ConfigFileReleaseKey)
	if err != nil {
		log.Error("[Config][Release] rollback config file get target release", zap.Error(err))
		return nil, api.NewConfigResponse(commonstore.StoreCode2APICode(err))
	}
	if targetRelease == nil {
		log.Error("[Config][Release] rollback config file to target release not found")
		return nil, api.NewConfigResponse(apimodel.Code_NotFoundResource)
	}

	if err := s.storage.ActiveConfigFileReleaseTx(tx, data); err != nil {
		log.Error("[Config][Release] rollback config file release error.",
			utils.RequestID(ctx), zap.String("namespace", data.Namespace),
			zap.String("group", data.Group), zap.String("fileName", data.FileName), zap.Error(err))
		return targetRelease, api.NewConfigResponse(commonstore.StoreCode2APICode(err))
	}
	return targetRelease, nil
}

func (s *Server) UpsertAndReleaseConfigFile(ctx context.Context,
	req *apiconfig.ConfigFilePublishInfo) *apiconfig.ConfigResponse {

	if err := utils.CheckResourceName(req.GetNamespace()); err != nil {
		return api.NewConfigResponseWithInfo(apimodel.Code_BadRequest, "invalid config namespace")
	}
	if err := utils.CheckResourceName(req.GetGroup()); err != nil {
		return api.NewConfigResponseWithInfo(apimodel.Code_BadRequest, "invalid config group")
	}
	if err := CheckFileName(req.GetFileName()); err != nil {
		return api.NewConfigResponseWithInfo(apimodel.Code_BadRequest, "invalid config file_name")
	}

	upsertFileReq := &apiconfig.ConfigFile{
		Name:        req.GetFileName(),
		Namespace:   req.GetNamespace(),
		Group:       req.GetGroup(),
		Content:     req.GetContent(),
		Format:      req.GetFormat(),
		Comment:     req.GetComment(),
		Tags:        req.GetTags(),
		CreateBy:    utils.NewStringValue(utils.ParseUserName(ctx)),
		ModifyBy:    utils.NewStringValue(utils.ParseUserName(ctx)),
		ReleaseTime: utils.NewStringValue(req.GetReleaseDescription().GetValue()),
	}
	if rsp := s.prepareCreateConfigFile(ctx, upsertFileReq); rsp.Code.Value != api.ExecuteSuccess {
		return rsp
	}

	tx, err := s.storage.StartTx()
	if err != nil {
		log.Error("[Config][File] upsert config file when begin tx.", utils.RequestID(ctx), zap.Error(err))
		return api.NewConfigResponse(commonstore.StoreCode2APICode(err))
	}

	defer func() {
		_ = tx.Rollback()
	}()
	upsertResp := s.handleCreateConfigFile(ctx, tx, upsertFileReq)
	if upsertResp.GetCode().GetValue() == uint32(apimodel.Code_ExistedResource) {
		upsertResp = s.handleUpdateConfigFile(ctx, tx, upsertFileReq)
	}
	if upsertResp.GetCode().GetValue() != uint32(apimodel.Code_ExecuteSuccess) {
		return upsertResp
	}

	data, releaseResp := s.handlePublishConfigFile(ctx, tx, &apiconfig.ConfigFileRelease{
		Name:               req.GetReleaseName(),
		Namespace:          req.GetNamespace(),
		Group:              req.GetGroup(),
		FileName:           req.GetFileName(),
		CreateBy:           utils.NewStringValue(utils.ParseUserName(ctx)),
		ModifyBy:           utils.NewStringValue(utils.ParseUserName(ctx)),
		ReleaseDescription: req.GetReleaseDescription(),
	})
	if releaseResp.GetCode().GetValue() != uint32(apimodel.Code_ExecuteSuccess) {
		_ = tx.Rollback()
		if data != nil {
			s.recordReleaseFail(ctx, utils.ReleaseTypeNormal, data, errors.New(releaseResp.GetInfo().GetValue()))
		}
		return releaseResp
	}

	if err := tx.Commit(); err != nil {
		log.Error("[Config][File] upsert config file when commit tx.", utils.RequestID(ctx), zap.Error(err))
		s.recordReleaseFail(ctx, utils.ReleaseTypeNormal, data, err)
		return api.NewConfigResponse(commonstore.StoreCode2APICode(err))
	}
	s.recordReleaseHistory(ctx, data, utils.ReleaseTypeNormal, utils.ReleaseStatusSuccess, "")
	return releaseResp
}

func (s *Server) cleanConfigFileReleases(ctx context.Context, tx store.Tx,
	file *model.ConfigFile) *apiconfig.ConfigResponse {

	// 先重新 active 下当前正在发布的
	saveData, err := s.storage.GetConfigFileActiveReleaseTx(tx, file.Key())
	if err != nil {
		return api.NewConfigResponse(commonstore.StoreCode2APICode(err))
	}
	if saveData != nil {
		if err := s.storage.ActiveConfigFileReleaseTx(tx, saveData); err != nil {
			return api.NewConfigResponse(commonstore.StoreCode2APICode(err))
		}
	}
	if err := s.storage.CleanConfigFileReleasesTx(tx, file.Namespace, file.Group, file.Name); err != nil {
		return api.NewConfigResponse(commonstore.StoreCode2APICode(err))
	}
	return nil
}

func (s *Server) recordReleaseSuccess(ctx context.Context, rType string, release *model.ConfigFileRelease) {
	s.recordReleaseHistory(ctx, release, rType, utils.ReleaseStatusSuccess, "")
}

func (s *Server) recordReleaseFail(ctx context.Context, rType string, release *model.ConfigFileRelease, err error) {
	s.recordReleaseHistory(ctx, release, rType, utils.ReleaseStatusFail, err.Error())
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

func checkBaseReleaseParam(req *apiconfig.ConfigFileRelease, checkRelease bool) (apimodel.Code, string) {
	namespace := req.GetNamespace().GetValue()
	group := req.GetGroup().GetValue()
	fileName := req.GetFileName().GetValue()
	releaseName := req.GetName().GetValue()
	if namespace == "" {
		return apimodel.Code_BadRequest, "invalid namespace"
	}
	if group == "" {
		return apimodel.Code_BadRequest, "invalid config group"
	}
	if fileName == "" {
		return apimodel.Code_BadRequest, "invalid config file name"
	}
	if checkRelease {
		if releaseName == "" {
			return apimodel.Code_BadRequest, "invalid config release name"
		}
	}
	return apimodel.Code_ExecuteSuccess, ""
}
