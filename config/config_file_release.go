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

	"github.com/polarismesh/polaris/cache"
	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/model"
	commonstore "github.com/polarismesh/polaris/common/store"
	commontime "github.com/polarismesh/polaris/common/time"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/store"
)

// PublishConfigFile 发布配置文件
func (s *Server) PublishConfigFile(
	ctx context.Context, configFileRelease *apiconfig.ConfigFileRelease) *apiconfig.ConfigResponse {
	namespace := configFileRelease.Namespace.GetValue()
	group := configFileRelease.Group.GetValue()
	fileName := configFileRelease.FileName.GetValue()

	if err := CheckFileName(utils.NewStringValue(fileName)); err != nil {
		return api.NewConfigResponse(apimodel.Code_InvalidConfigFileName)
	}
	if err := CheckResourceName(utils.NewStringValue(namespace)); err != nil {
		return api.NewConfigResponse(apimodel.Code_InvalidNamespaceName)
	}
	if err := CheckResourceName(utils.NewStringValue(group)); err != nil {
		return api.NewConfigResponse(apimodel.Code_InvalidConfigFileGroupName)
	}
	if !s.checkNamespaceExisted(namespace) {
		return api.NewConfigResponse(apimodel.Code_NotFoundNamespace)
	}

	tx, ctx, err := s.StartTxAndSetToContext(ctx)
	if err != nil {
		return api.NewConfigResponse(commonstore.StoreCode2APICode(err))
	}
	defer func() {
		_ = tx.Rollback()
	}()

	data, resp := s.handlePublishConfigFile(ctx, tx, configFileRelease)
	if resp.GetCode().GetValue() != uint32(apimodel.Code_ExecuteSuccess) {
		_ = tx.Rollback()
		if data != nil {
			s.recordReleaseFail(ctx, utils.ReleaseTypeNormal, data, err)
		}
		return resp
	}

	if err := tx.Commit(); err != nil {
		s.recordReleaseFail(ctx, utils.ReleaseTypeNormal, data, err)
		return api.NewConfigFileReleaseResponse(commonstore.StoreCode2APICode(err), nil)
	}
	s.recordReleaseHistory(ctx, data, utils.ReleaseTypeNormal, utils.ReleaseStatusSuccess, "")
	return resp
}

// PublishConfigFile 发布配置文件
func (s *Server) handlePublishConfigFile(ctx context.Context, tx store.Tx,
	req *apiconfig.ConfigFileRelease) (*model.ConfigFileRelease, *apiconfig.ConfigResponse) {
	namespace := req.Namespace.GetValue()
	group := req.Group.GetValue()
	fileName := req.FileName.GetValue()

	// 获取待发布的 configFile 信息
	toPublishFile, err := s.storage.GetConfigFileTx(tx, namespace, group, fileName)
	if err != nil {
		log.Error("[Config][Service] get config file error.", utils.RequestID(ctx),
			zap.String("namespace", namespace), zap.String("group", group), zap.String("fileName", fileName),
			zap.Error(err))
		s.recordReleaseFail(ctx, utils.ReleaseTypeNormal, model.ToConfigFileReleaseStore(req), err)
		return nil, api.NewConfigResponse(commonstore.StoreCode2APICode(err))
	}
	if toPublishFile == nil {
		return nil, api.NewConfigResponse(apimodel.Code_NotFoundResource)
	}
	if releaseName := req.GetName().GetValue(); releaseName == "" {
		req.Name = utils.NewStringValue(GenReleaseName("", fileName))
	}

	fileRelease := &model.ConfigFileRelease{
		SimpleConfigFileRelease: &model.SimpleConfigFileRelease{
			ConfigFileReleaseKey: &model.ConfigFileReleaseKey{
				Name:      req.GetName().GetValue(),
				Namespace: namespace,
				Group:     group,
				FileName:  fileName,
			},
			Format:   toPublishFile.Format,
			Metadata: toPublishFile.Metadata,
			Comment:  req.Comment.GetValue(),
			Md5:      CalMd5(toPublishFile.Content),
			CreateBy: utils.ParseUserName(ctx),
			ModifyBy: utils.ParseUserName(ctx),
		},
		Content: toPublishFile.Content,
	}
	if err := s.storage.CreateConfigFileReleaseTx(tx, fileRelease); err != nil {
		log.Error("[Config][Service] update config file release error.",
			utils.RequestID(ctx), zap.String("namespace", namespace),
			zap.String("group", group), zap.String("fileName", fileName), zap.Error(err))
		return fileRelease, api.NewConfigResponse(commonstore.StoreCode2APICode(err))
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
	if errCode, errMsg := checkBaseReleaseParam(req); errCode != apimodel.Code_ExecuteSuccess {
		return api.NewConfigResponseWithInfo(errCode, errMsg)
	}
	ret, err := s.storage.GetConfigFileRelease(&model.ConfigFileReleaseKey{
		Namespace: namespace,
		Group:     group,
		FileName:  fileName,
		Name:      releaseName,
	})
	if err != nil {
		log.Error("[Config][Service]get config file release error.", utils.RequestID(ctx),
			utils.ZapNamespace(namespace), utils.ZapGroup(group), utils.ZapFileName(fileName), zap.Error(err))
		return api.NewConfigResponse(commonstore.StoreCode2APICode(err))
	}
	if ret == nil {
		return api.NewConfigResponse(apimodel.Code_NotFoundResource)
	}
	ret, err = s.chains.AfterGetFileRelease(ctx, ret)
	if err != nil {
		out := api.NewConfigResponse(apimodel.Code_ExecuteException)
		out.Info = utils.NewStringValue(err.Error())
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

	if errCode, errMsg := checkBaseReleaseParam(req); errCode != apimodel.Code_ExecuteSuccess {
		return api.NewConfigResponseWithInfo(errCode, errMsg)
	}
	release := &model.ConfigFileRelease{
		SimpleConfigFileRelease: &model.SimpleConfigFileRelease{
			ConfigFileReleaseKey: &model.ConfigFileReleaseKey{
				Name:      req.GetName().GetValue(),
				Namespace: req.GetNamespace().GetValue(),
				Group:     req.GetGroup().GetValue(),
				FileName:  req.GetFileName().GetValue(),
			},
		},
	}
	if err := s.storage.DeleteConfigFileRelease(release.ConfigFileReleaseKey); err != nil {
		log.Error("[Config][Service] delete config file release error.",
			utils.RequestID(ctx), zap.String("namespace", req.GetNamespace().GetValue()),
			zap.String("group", req.GetGroup().GetValue()), zap.String("fileName", req.GetFileName().GetValue()),
			zap.Error(err))

		s.recordReleaseFail(ctx, utils.ReleaseTypeDelete, release, err)
		return api.NewConfigResponse(commonstore.StoreCode2APICode(err))
	}

	s.recordReleaseSuccess(ctx, utils.ReleaseTypeDelete, release)
	s.RecordHistory(ctx, configFileReleaseRecordEntry(ctx, req, release, model.ODelete))
	return api.NewConfigResponse(apimodel.Code_ExecuteSuccess)
}

func (s *Server) GetConfigFileReleaseVersions(ctx context.Context,
	filters map[string]string) *apiconfig.ConfigBatchQueryResponse {

	namespace := filters["namespace"]
	group := filters["group"]
	fileName := filters["file_name"]
	if namespace == "" {
		return api.NewConfigBatchQueryResponseWithInfo(apimodel.Code_BadRequest, "invalid namespace")
	}
	if group == "" {
		return api.NewConfigBatchQueryResponseWithInfo(apimodel.Code_BadRequest, "invalid config group")
	}
	if fileName == "" {
		return api.NewConfigBatchQueryResponseWithInfo(apimodel.Code_BadRequest, "invalid config file name")
	}
	args := cache.ConfigReleaseArgs{
		BaseConfigArgs: cache.BaseConfigArgs{
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
		if _, ok := availableSearch["config_file_release"][k]; ok {
			searchFilters[k] = v
		}
	}

	offset, limit, err := utils.ParseOffsetAndLimit(filter)
	if err != nil {
		return api.NewConfigBatchQueryResponseWithInfo(apimodel.Code_BadRequest, err.Error())
	}

	args := cache.ConfigReleaseArgs{
		BaseConfigArgs: cache.BaseConfigArgs{
			Namespace:  searchFilters["namespace"],
			Group:      searchFilters["group"],
			Offset:     offset,
			Limit:      limit,
			OrderField: searchFilters["order_field"],
			OrderType:  searchFilters["order_type"],
		},
		FileName:   searchFilters["file_name"],
		OnlyActive: strings.Compare(searchFilters["only_active"], "true") == 0,
	}
	return s.handleDescribeConfigFileReleases(ctx, args)
}

func (s *Server) handleDescribeConfigFileReleases(ctx context.Context,
	args cache.ConfigReleaseArgs) *apiconfig.ConfigBatchQueryResponse {

	total, simpleReleases, err := s.fileCache.QueryReleases(&args)
	if err != nil {
		return api.NewConfigBatchQueryResponseWithInfo(apimodel.Code_ExecuteException, err.Error())
	}
	ret := make([]*apiconfig.ConfigFileRelease, 0, len(simpleReleases))
	for i := range simpleReleases {
		item := simpleReleases[i]
		ret = append(ret, &apiconfig.ConfigFileRelease{
			Id:         utils.NewUInt64Value(item.Id),
			Name:       utils.NewStringValue(item.Name),
			Namespace:  utils.NewStringValue(item.Namespace),
			Group:      utils.NewStringValue(item.Group),
			FileName:   utils.NewStringValue(item.FileName),
			Version:    utils.NewUInt64Value(item.Version),
			Active:     utils.NewBoolValue(item.Active),
			CreateTime: utils.NewStringValue(commontime.Time2String(item.CreateTime)),
			ModifyTime: utils.NewStringValue(commontime.Time2String(item.ModifyTime)),
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
			chs[index] <- s.handleRollbackConfigFileRelease(ctx, ins)
		}(i, instance)
	}

	for _, ch := range chs {
		resp := <-ch
		api.ConfigCollect(responses, resp)
	}
	return responses
}

// handleRollbackConfigFileRelease 回滚配置
func (s *Server) handleRollbackConfigFileRelease(ctx context.Context,
	req *apiconfig.ConfigFileRelease) *apiconfig.ConfigResponse {
	if errCode, errMsg := checkBaseReleaseParam(req); errCode != apimodel.Code_ExecuteSuccess {
		return api.NewConfigResponseWithInfo(errCode, errMsg)
	}
	data := &model.ConfigFileRelease{
		SimpleConfigFileRelease: &model.SimpleConfigFileRelease{
			ConfigFileReleaseKey: &model.ConfigFileReleaseKey{
				Name:      req.GetName().GetValue(),
				Namespace: req.GetNamespace().GetValue(),
				Group:     req.GetGroup().GetValue(),
				FileName:  req.GetFileName().GetValue(),
			},
		},
	}

	targetRelease, err := s.storage.GetConfigFileRelease(data.ConfigFileReleaseKey)
	if err != nil {
		return api.NewConfigResponse(commonstore.StoreCode2APICode(err))
	}
	if targetRelease == nil {
		return api.NewConfigResponse(apimodel.Code_NotFoundResource)
	}

	if err := s.storage.ActiveConfigFileRelease(data); err != nil {
		log.Error("[Config][Service] rollback config file release error.",
			utils.RequestID(ctx), zap.String("namespace", req.GetNamespace().GetValue()),
			zap.String("group", req.GetGroup().GetValue()),
			zap.String("fileName", req.GetFileName().GetValue()), zap.Error(err))

		s.recordReleaseFail(ctx, utils.ReleaseTypeRollback, data, err)
		return api.NewConfigResponse(commonstore.StoreCode2APICode(err))
	}

	s.recordReleaseSuccess(ctx, utils.ReleaseTypeRollback, data)
	s.RecordHistory(ctx, configFileReleaseRecordEntry(ctx, req, data, model.ORollback))
	return api.NewConfigResponse(apimodel.Code_ExecuteSuccess)
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

func checkBaseReleaseParam(req *apiconfig.ConfigFileRelease) (apimodel.Code, string) {
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
	if releaseName == "" {
		return apimodel.Code_BadRequest, "invalid config release name"
	}
	return apimodel.Code_ExecuteSuccess, ""
}
