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
	"encoding/json"
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
		return resp
	}

	if err := tx.Commit(); err != nil {
		log.Error("[Config][Release] publish config file commit tx.", utils.RequestID(ctx), zap.Error(err))
		return api.NewConfigResponse(commonstore.StoreCode2APICode(err))
	}
	if req.GetReleaseType().GetValue() == model.ReleaseTypeGray {
		s.recordReleaseSuccess(ctx, utils.ReleaseTypeGray, data)
	} else {
		s.recordReleaseSuccess(ctx, utils.ReleaseTypeNormal, data)
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

	fileRelease := &model.ConfigFileRelease{
		SimpleConfigFileRelease: &model.SimpleConfigFileRelease{
			ConfigFileReleaseKey: &model.ConfigFileReleaseKey{
				Name:        req.GetName().GetValue(),
				Namespace:   namespace,
				Group:       group,
				FileName:    fileName,
				ReleaseType: model.ReleaseType(req.GetReleaseType().GetValue()),
			},
		},
	}

	// 确认是否存在正在灰度发布中的配置文件
	betaRelease, err := s.storage.GetConfigFileBetaReleaseTx(tx, fileRelease.ToFileKey())
	if err != nil {
		log.Error("[Config][File] get beta config file release in get target.", utils.RequestID(ctx), zap.Error(err))
		return nil, api.NewConfigResponse(commonstore.StoreCode2APICode(err))
	}
	if betaRelease != nil {
		log.Error("[Config][File] still exist beta config file release.", utils.RequestID(ctx), zap.Error(err))
		return nil, api.NewConfigResponse(apimodel.Code_DataConflict)
	}

	// 获取待发布的 configFile 信息
	toPublishFile, err := s.storage.GetConfigFileTx(tx, namespace, group, fileName)
	if err != nil {
		log.Error("[Config][Release] publish config file when get file.", utils.RequestID(ctx),
			utils.ZapNamespace(namespace), utils.ZapGroup(group), utils.ZapFileName(fileName),
			zap.Error(err))
		return nil, api.NewConfigResponse(commonstore.StoreCode2APICode(err))
	}
	if toPublishFile == nil {
		return nil, api.NewConfigResponse(apimodel.Code_NotFoundResource)
	}
	if releaseName := req.GetName().GetValue(); releaseName == "" {
		// 这里要保证每一次发布都有唯一的 release_name 名称
		req.Name = utils.NewStringValue(fmt.Sprintf("%s-%d-%d", fileName, time.Now().Unix(), s.nextSequence()))
	}

	fileRelease.Name = req.GetName().GetValue()
	fileRelease.Format = toPublishFile.Format
	fileRelease.Metadata = toPublishFile.Metadata
	fileRelease.Comment = req.GetComment().GetValue()
	fileRelease.Md5 = CalMd5(toPublishFile.Content)
	fileRelease.CreateBy = utils.ParseUserName(ctx)
	fileRelease.ModifyBy = utils.ParseUserName(ctx)
	fileRelease.ReleaseDescription = req.GetReleaseDescription().GetValue()
	fileRelease.Content = toPublishFile.Content

	saveRelease, err := s.storage.GetConfigFileReleaseTx(tx, fileRelease.ConfigFileReleaseKey)
	if err != nil {
		log.Error("[Config][Release] publish config file when get release.",
			utils.RequestID(ctx), utils.ZapNamespace(namespace), utils.ZapGroup(group),
			utils.ZapFileName(fileName), zap.Error(err))
		return fileRelease, api.NewConfigResponse(commonstore.StoreCode2APICode(err))
	}
	// 重新激活
	if saveRelease != nil {
		log.Debug("[Config][Release] re-active config file release.",
			utils.RequestID(ctx), utils.ZapNamespace(namespace), utils.ZapGroup(group),
			utils.ZapFileName(fileName), utils.ZapReleaseName(fileRelease.Name))
		if err := s.storage.ActiveConfigFileReleaseTx(tx, fileRelease); err != nil {
			log.Error("[Config][Release] re-active config file release error.",
				utils.RequestID(ctx), utils.ZapNamespace(namespace), utils.ZapGroup(group),
				utils.ZapFileName(fileName), zap.Error(err))
			return fileRelease, api.NewConfigFileResponse(commonstore.StoreCode2APICode(err), nil)
		}
	} else {
		if err = s.storage.CreateConfigFileReleaseTx(tx, fileRelease); err != nil {
			log.Error("[Config][Release] publish config file when create release.",
				utils.RequestID(ctx), utils.ZapNamespace(namespace), utils.ZapGroup(group),
				utils.ZapFileName(fileName), zap.Error(err))
			return fileRelease, api.NewConfigResponse(commonstore.StoreCode2APICode(err))
		}
	}
	if req.GetReleaseType().GetValue() == model.ReleaseTypeGray {
		clientLabels := req.GetBetaLabels()
		raw := make([]json.RawMessage, 0, len(clientLabels))
		marshaler := jsonpb.Marshaler{}
		for i := range clientLabels {
			data, err := marshaler.MarshalToString(clientLabels[i])
			if err != nil {
				log.Error("[Config][Release] marshal gary rule error.",
					utils.RequestID(ctx), utils.ZapNamespace(namespace), utils.ZapGroup(group),
					utils.ZapFileName(fileName), zap.Error(err))
				return fileRelease, api.NewConfigResponseWithInfo(apimodel.Code_InvalidMatchRule, err.Error())
			}
			raw = append(raw, json.RawMessage(data))
		}
		grayResource := &model.GrayResource{
			Name:      model.GetGrayConfigRealseKey(fileRelease.SimpleConfigFileRelease),
			MatchRule: string(utils.MustJson(raw)),
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

	_ = s.caches.Gray().Update()
	ret, err = s.chains.AfterGetFileRelease(ctx, ret)
	if err != nil {
		log.Error("[Config][Release] get config file release run chain.", utils.RequestID(ctx),
			utils.ZapNamespace(namespace), utils.ZapGroup(group), utils.ZapFileName(fileName), zap.Error(err))
		out := api.NewConfigResponse(apimodel.Code_ExecuteException)
		return out
	}

	release := model.ToConfiogFileReleaseApi(ret)
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
			chs[index] <- s.DeleteConfigFileRelease(ctx, ins)
		}(i, instance)
	}

	for _, ch := range chs {
		resp := <-ch
		api.ConfigCollect(responses, resp)
	}
	return responses
}

func (s *Server) DeleteConfigFileRelease(ctx context.Context,
	req *apiconfig.ConfigFileRelease) *apiconfig.ConfigResponse {
	release := &model.ConfigFileRelease{
		SimpleConfigFileRelease: &model.SimpleConfigFileRelease{
			ConfigFileReleaseKey: &model.ConfigFileReleaseKey{
				Name:        req.GetName().GetValue(),
				Namespace:   req.GetNamespace().GetValue(),
				Group:       req.GetGroup().GetValue(),
				FileName:    req.GetFileName().GetValue(),
				ReleaseType: model.ReleaseType(req.GetReleaseType().GetValue()),
			},
		},
	}
	var (
		recordData *model.ConfigFileRelease
	)

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
		log.Error("[Config][File] delete config file release when lock.",
			utils.RequestID(ctx), zap.Error(err))
		return api.NewConfigResponse(commonstore.StoreCode2APICode(err))
	}

	saveData, err := s.storage.GetConfigFileReleaseTx(tx, release.ConfigFileReleaseKey)
	if err != nil {
		return api.NewConfigResponse(commonstore.StoreCode2APICode(err))
	}
	recordData = saveData
	if saveData == nil {
		return api.NewConfigResponse(apimodel.Code_ExecuteSuccess)
	}
	// 如果存在处于 active 状态的配置，重新在激活一下，触发版本的更新变动
	if saveData.Active {
		if err := s.storage.ActiveConfigFileReleaseTx(tx, saveData); err != nil {
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
		return api.NewConfigResponse(commonstore.StoreCode2APICode(err))
	}

	if err := tx.Commit(); err != nil {
		log.Error("[Config][Release] delete config file release when commit tx.",
			utils.RequestID(ctx), utils.ZapNamespace(req.GetNamespace().GetValue()),
			utils.ZapGroup(req.GetGroup().GetValue()), utils.ZapFileName(req.GetFileName().GetValue()),
			zap.Error(err))
		return api.NewConfigResponse(commonstore.StoreCode2APICode(err))
	}
	s.recordReleaseSuccess(ctx, utils.ReleaseTypeDelete, recordData)
	s.RecordHistory(ctx, configFileReleaseRecordEntry(ctx, req, release, model.ODelete))
	return api.NewConfigResponse(apimodel.Code_ExecuteSuccess)
}

func (s *Server) GetConfigFileReleaseVersions(ctx context.Context,
	searchFilters map[string]string) *apiconfig.ConfigBatchQueryResponse {

	args := cachetypes.ConfigReleaseArgs{
		BaseConfigArgs: cachetypes.BaseConfigArgs{
			Namespace: searchFilters["namespace"],
			Group:     searchFilters["group"],
		},
		FileName:   searchFilters["file_name"],
		OnlyActive: false,
		NoPage:     true,
	}
	return s.handleDescribeConfigFileReleases(ctx, args)
}

func (s *Server) GetConfigFileReleases(ctx context.Context,
	searchFilters map[string]string) *apiconfig.ConfigBatchQueryResponse {

	offset, limit, _ := utils.ParseOffsetAndLimit(searchFilters)

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
		IncludeGray: true,
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
		viewData := &apiconfig.ConfigFileRelease{
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
			ReleaseType:        utils.NewStringValue(string(item.ReleaseType)),
		}
		// 查询配置灰度规则标签
		if item.ReleaseType == model.ReleaseTypeGray {
			viewData.BetaLabels = s.caches.Gray().GetGrayRule(model.GetGrayConfigRealseKey(item))
		}
		ret = append(ret, viewData)
	}

	resp := api.NewConfigBatchQueryResponse(apimodel.Code_ExecuteSuccess)
	resp.Total = utils.NewUInt32Value(total)
	resp.ConfigFileReleases = ret
	return resp
}

// RollbackConfigFileReleases 批量回滚配置
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
	data := &model.ConfigFileRelease{
		SimpleConfigFileRelease: &model.SimpleConfigFileRelease{
			ConfigFileReleaseKey: &model.ConfigFileReleaseKey{
				Name:        req.GetName().GetValue(),
				Namespace:   req.GetNamespace().GetValue(),
				Group:       req.GetGroup().GetValue(),
				FileName:    req.GetFileName().GetValue(),
				ReleaseType: model.ReleaseTypeFull,
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
		return ret
	}

	if err := tx.Commit(); err != nil {
		log.Error("[Config][File] rollback config file releasw when commit tx.",
			utils.RequestID(ctx), zap.Error(err))
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

// CasUpsertAndReleaseConfigFile 根据版本比对决定是否允许进行配置修改发布
func (s *Server) CasUpsertAndReleaseConfigFile(ctx context.Context,
	req *apiconfig.ConfigFilePublishInfo) *apiconfig.ConfigResponse {
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
		log.Error("[Config][File] upsert config file when begin tx.", utils.RequestID(ctx),
			zap.String("namespace", req.GetNamespace().GetValue()), zap.String("group", req.GetGroup().GetValue()),
			zap.String("fileName", req.GetFileName().GetValue()), zap.Error(err))
		return api.NewConfigResponse(commonstore.StoreCode2APICode(err))
	}

	defer func() {
		_ = tx.Rollback()
	}()
	saveFile, err := s.storage.LockConfigFile(tx, &model.ConfigFileKey{
		Namespace: req.GetNamespace().GetValue(),
		Group:     req.GetGroup().GetValue(),
		Name:      req.GetFileName().GetValue(),
	})
	if err != nil {
		log.Error("[Config][File] lock config file when begin tx.", utils.RequestID(ctx),
			zap.String("namespace", req.GetNamespace().GetValue()), zap.String("group", req.GetGroup().GetValue()),
			zap.String("fileName", req.GetFileName().GetValue()), zap.Error(err))
		return api.NewConfigResponse(commonstore.StoreCode2APICode(err))
	}

	historyRecords := []func(){}

	var upsertResp *apiconfig.ConfigResponse
	if saveFile == nil {
		upsertResp = s.handleCreateConfigFile(ctx, tx, upsertFileReq)
		historyRecords = append(historyRecords, func() {
			s.RecordHistory(ctx, configFileRecordEntry(ctx, upsertFileReq, model.OCreate))
		})
	} else {
		actualMd5 := CalMd5(saveFile.Content)
		if req.GetMd5().GetValue() != actualMd5 {
			log.Error("[Config][File] cas compare config file.", utils.RequestID(ctx),
				zap.String("namespace", req.GetNamespace().GetValue()), zap.String("group", req.GetGroup().GetValue()),
				zap.String("fileName", req.GetFileName().GetValue()),
				zap.String("expect", req.GetMd5().GetValue()), zap.String("actual", actualMd5))
			return api.NewConfigResponse(apimodel.Code_DataConflict)
		}
		upsertResp = s.handleUpdateConfigFile(ctx, tx, upsertFileReq)
		historyRecords = append(historyRecords, func() {
			s.RecordHistory(ctx, configFileRecordEntry(ctx, upsertFileReq, model.OUpdate))
		})
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
		return releaseResp
	}

	if err := tx.Commit(); err != nil {
		log.Error("[Config][File] upsert config file when commit tx.", utils.RequestID(ctx), zap.Error(err))
		return api.NewConfigResponse(commonstore.StoreCode2APICode(err))
	}
	for i := range historyRecords {
		historyRecords[i]()
	}
	s.recordReleaseHistory(ctx, data, utils.ReleaseTypeNormal, utils.ReleaseStatusSuccess, "")
	return releaseResp
}

func (s *Server) UpsertAndReleaseConfigFile(ctx context.Context,
	req *apiconfig.ConfigFilePublishInfo) *apiconfig.ConfigResponse {
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

	historyRecords := []func(){}
	upsertResp := s.handleCreateConfigFile(ctx, tx, upsertFileReq)
	if upsertResp.GetCode().GetValue() == uint32(apimodel.Code_ExistedResource) {
		upsertResp = s.handleUpdateConfigFile(ctx, tx, upsertFileReq)
		historyRecords = append(historyRecords, func() {
			s.RecordHistory(ctx, configFileRecordEntry(ctx, upsertFileReq, model.OUpdate))
		})
	} else {
		historyRecords = append(historyRecords, func() {
			s.RecordHistory(ctx, configFileRecordEntry(ctx, upsertFileReq, model.OCreate))
		})
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
		return releaseResp
	}

	if err := tx.Commit(); err != nil {
		log.Error("[Config][File] upsert config file when commit tx.", utils.RequestID(ctx), zap.Error(err))
		return api.NewConfigResponse(commonstore.StoreCode2APICode(err))
	}
	for i := range historyRecords {
		historyRecords[i]()
	}
	s.recordReleaseHistory(ctx, data, utils.ReleaseTypeNormal, utils.ReleaseStatusSuccess, "")
	return releaseResp
}

func (s *Server) StopGrayConfigFileReleases(ctx context.Context, reqs []*apiconfig.ConfigFileRelease) *apiconfig.ConfigBatchWriteResponse {
	responses := api.NewConfigBatchWriteResponse(apimodel.Code_ExecuteSuccess)
	chs := make([]chan *apiconfig.ConfigResponse, 0, len(reqs))
	for i, instance := range reqs {
		chs = append(chs, make(chan *apiconfig.ConfigResponse))
		go func(index int, ins *apiconfig.ConfigFileRelease) {
			chs[index] <- s.StopGrayConfigFileRelease(ctx, ins)
		}(i, instance)
	}

	for _, ch := range chs {
		resp := <-ch
		api.ConfigCollect(responses, resp)
	}
	return responses
}

func (s *Server) StopGrayConfigFileRelease(ctx context.Context, req *apiconfig.ConfigFileRelease) *apiconfig.ConfigResponse {
	if err := utils.CheckResourceName(req.GetNamespace()); err != nil {
		return api.NewConfigResponseWithInfo(apimodel.Code_BadRequest, "invalid config namespace")
	}
	if err := utils.CheckResourceName(req.GetGroup()); err != nil {
		return api.NewConfigResponseWithInfo(apimodel.Code_BadRequest, "invalid config group")
	}
	if err := CheckFileName(req.GetFileName()); err != nil {
		return api.NewConfigResponseWithInfo(apimodel.Code_BadRequest, "invalid config file_name")
	}
	tx, err := s.storage.StartTx()
	if err != nil {
		log.Error("[Config][File] stop beta config file when begin tx.", utils.RequestID(ctx), zap.Error(err))
		return api.NewConfigResponse(commonstore.StoreCode2APICode(err))
	}

	defer func() {
		_ = tx.Rollback()
	}()

	fileKey := &model.ConfigFileKey{
		Namespace: req.GetNamespace().GetValue(),
		Group:     req.GetGroup().GetValue(),
		Name:      req.GetFileName().GetValue(),
	}

	if _, err := s.storage.LockConfigFile(tx, fileKey); err != nil {
		log.Error("[Config][File] stop beta config file release in lock file.", utils.RequestID(ctx), zap.Error(err))
		return api.NewConfigResponse(commonstore.StoreCode2APICode(err))
	}
	betaRelease, err := s.storage.GetConfigFileBetaReleaseTx(tx, fileKey)
	if err != nil {
		log.Error("[Config][File] stop beta config file release in get target.", utils.RequestID(ctx), zap.Error(err))
		return api.NewConfigResponse(commonstore.StoreCode2APICode(err))
	}
	if betaRelease == nil {
		return api.NewConfigResponse(apimodel.Code_ExecuteSuccess)
	}
	if err := s.storage.CleanGrayResource(tx, &model.GrayResource{
		Name: model.GetGrayConfigRealseKey(&model.SimpleConfigFileRelease{
			ConfigFileReleaseKey: &model.ConfigFileReleaseKey{
				Namespace:   req.GetNamespace().GetValue(),
				Group:       req.GetGroup().GetValue(),
				Name:        req.GetFileName().GetValue(),
				ReleaseType: model.ReleaseTypeGray,
			},
		}),
	}); err != nil {
		log.Error("[Config][File] stop beta config file release when clean beta rule.", utils.RequestID(ctx), zap.Error(err))
		return api.NewConfigResponse(commonstore.StoreCode2APICode(err))
	}

	if err = s.storage.InactiveConfigFileReleaseTx(tx, betaRelease); err != nil {
		log.Error("[Config][File] stop beta config file release.", utils.RequestID(ctx), zap.Error(err))
		return api.NewConfigResponse(commonstore.StoreCode2APICode(err))
	}
	if err := tx.Commit(); err != nil {
		log.Error("[Config][File] stop config file release when commit tx.", utils.RequestID(ctx), zap.Error(err))
		return api.NewConfigResponse(commonstore.StoreCode2APICode(err))
	}
	s.recordReleaseHistory(ctx, betaRelease, utils.ReleaseTypeCancelGray, utils.ReleaseStatusSuccess, "")
	return api.NewConfigResponse(apimodel.Code_ExecuteSuccess)
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
