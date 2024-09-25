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
	"fmt"

	"github.com/golang/protobuf/ptypes/wrappers"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"
	"go.uber.org/zap"

	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/model"
	commonstore "github.com/polarismesh/polaris/common/store"
	commontime "github.com/polarismesh/polaris/common/time"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/store"
)

var (
	// AliasFilterAttributes filer attrs alias
	AliasFilterAttributes = map[string]bool{
		"alias":           true,
		"alias_namespace": true,
		"namespace":       true,
		"service":         true,
		"owner":           true,
		"offset":          true,
		"limit":           true,
	}
)

// CreateServiceAlias 创建服务别名
func (s *Server) CreateServiceAlias(ctx context.Context, req *apiservice.ServiceAlias) *apiservice.Response {
	tx, err := s.storage.CreateTransaction()
	if err != nil {
		log.Error(err.Error(), utils.RequestID(ctx))
		return api.NewServiceAliasResponse(commonstore.StoreCode2APICode(err), req)
	}
	defer func() { _ = tx.Commit() }()

	service, response, done := s.checkPointServiceAlias(ctx, tx, req)
	if done {
		return response
	}

	// 检查是否存在同名的alias
	if req.GetAlias().GetValue() != "" {
		oldAlias, getErr := s.storage.GetService(req.GetAlias().GetValue(),
			req.GetAliasNamespace().GetValue())
		if getErr != nil {
			log.Error(getErr.Error(), utils.RequestID(ctx))
			return api.NewServiceAliasResponse(commonstore.StoreCode2APICode(err), req)
		}
		if oldAlias != nil {
			return api.NewServiceAliasResponse(apimodel.Code_ExistedResource, req)
		}
	}

	// 构建别名的信息，这里包括了创建SID
	input, resp := s.createServiceAliasModel(req, service.ID)
	if resp != nil {
		return resp
	}
	if err := s.storage.AddService(input); err != nil {
		log.Error(err.Error(), utils.RequestID(ctx))
		return api.NewServiceAliasResponse(commonstore.StoreCode2APICode(err), req)
	}

	log.Info(fmt.Sprintf("create service alias, service(%s, %s), alias(%s, %s)",
		req.Service.Value, req.Namespace.Value, input.Name, input.Namespace), utils.RequestID(ctx))
	out := &apiservice.ServiceAlias{
		Service:        req.Service,
		Namespace:      req.Namespace,
		Alias:          req.Alias,
		AliasNamespace: req.AliasNamespace,
		ServiceToken:   &wrappers.StringValue{Value: input.Token},
	}
	if out.GetAlias().GetValue() == "" {
		out.Alias = utils.NewStringValue(input.Name)
	}
	record := &apiservice.Service{Name: out.Alias, Namespace: out.AliasNamespace}
	s.RecordHistory(ctx, serviceRecordEntry(ctx, record, input, model.OCreate))
	return api.NewServiceAliasResponse(apimodel.Code_ExecuteSuccess, out)
}

func (s *Server) checkPointServiceAlias(ctx context.Context,
	tx store.Transaction, req *apiservice.ServiceAlias) (*model.Service, *apiservice.Response, bool) {
	// 检查指向服务是否存在以及是否为别名
	service, err := tx.LockService(req.GetService().GetValue(), req.GetNamespace().GetValue())
	if err != nil {
		log.Error(err.Error(), utils.RequestID(ctx))
		return nil, api.NewServiceAliasResponse(commonstore.StoreCode2APICode(err), req), true
	}
	if service == nil {
		return nil, api.NewServiceAliasResponse(apimodel.Code_NotFoundService, req), true
	}
	// 检查该服务是否已经是一个别名服务，不允许再为别名创建别名
	if service.IsAlias() {
		return nil, api.NewServiceAliasResponse(apimodel.Code_NotAllowCreateAliasForAlias, req), true
	}
	return service, nil, false
}

// DeleteServiceAlias 删除服务别名
//
//	需要带上源服务name，namespace，token
//	另外一种删除别名的方式，是直接调用删除服务的接口，也是可行的
func (s *Server) DeleteServiceAlias(ctx context.Context, req *apiservice.ServiceAlias) *apiservice.Response {
	rid := utils.ParseRequestID(ctx)
	alias, err := s.storage.GetService(req.GetAlias().GetValue(),
		req.GetAliasNamespace().GetValue())
	if err != nil {
		log.Error(err.Error(), utils.ZapRequestID(rid))
		return api.NewServiceAliasResponse(commonstore.StoreCode2APICode(err), req)
	}
	if alias == nil {
		return api.NewServiceAliasResponse(apimodel.Code_NotFoundServiceAlias, req)
	}

	// 直接删除alias
	if err := s.storage.DeleteServiceAlias(req.GetAlias().GetValue(),
		req.GetAliasNamespace().GetValue()); err != nil {
		log.Error(err.Error(), utils.ZapRequestID(rid))
		return api.NewServiceAliasResponse(commonstore.StoreCode2APICode(err), req)
	}

	return api.NewServiceAliasResponse(apimodel.Code_ExecuteSuccess, req)
}

func checkBatchAlias(req []*apiservice.ServiceAlias) *apiservice.BatchWriteResponse {
	if len(req) == 0 {
		return api.NewBatchWriteResponse(apimodel.Code_EmptyRequest)
	}

	if len(req) > MaxBatchSize {
		return api.NewBatchWriteResponse(apimodel.Code_BatchSizeOverLimit)
	}

	return nil
}

// DeleteServiceAliases 删除服务别名列表
func (s *Server) DeleteServiceAliases(
	ctx context.Context, req []*apiservice.ServiceAlias) *apiservice.BatchWriteResponse {
	if checkError := checkBatchAlias(req); checkError != nil {
		return checkError
	}

	responses := api.NewBatchWriteResponse(apimodel.Code_ExecuteSuccess)
	for _, alias := range req {
		response := s.DeleteServiceAlias(ctx, alias)
		api.Collect(responses, response)
	}

	return api.FormatBatchWriteResponse(responses)
}

// UpdateServiceAlias 修改服务别名
func (s *Server) UpdateServiceAlias(ctx context.Context, req *apiservice.ServiceAlias) *apiservice.Response {
	// 检查服务别名是否存在
	alias, err := s.storage.GetService(req.GetAlias().GetValue(), req.GetAliasNamespace().GetValue())
	if err != nil {
		log.Error(err.Error(), utils.RequestID(ctx))
		return api.NewServiceAliasResponse(commonstore.StoreCode2APICode(err), req)
	}
	if alias == nil {
		return api.NewServiceAliasResponse(apimodel.Code_NotFoundServiceAlias, req)
	}

	// 检查将要指向的服务是否存在
	service, err := s.storage.GetService(req.GetService().GetValue(), req.GetNamespace().GetValue())
	if err != nil {
		log.Error(err.Error(), utils.RequestID(ctx))
		return api.NewServiceAliasResponse(commonstore.StoreCode2APICode(err), req)
	}
	if service == nil {
		return api.NewServiceAliasResponse(apimodel.Code_NotFoundService, req)
	}
	// 检查该服务是否已经是一个别名服务，不允许再为别名创建别名
	if service.IsAlias() {
		return api.NewServiceAliasResponse(apimodel.Code_NotAllowCreateAliasForAlias, req)
	}

	// 判断是否需要修改
	resp, needUpdate, needUpdateOwner := s.updateServiceAliasAttribute(req, alias, service.ID)
	if resp != nil {
		return resp
	}

	if !needUpdate {
		log.Info("update service alias data no change, no need update", utils.RequestID(ctx),
			zap.String("service alias", req.String()))
		return api.NewServiceAliasResponse(apimodel.Code_NoNeedUpdate, req)
	}

	// 执行存储层操作
	if err := s.storage.UpdateServiceAlias(alias, needUpdateOwner); err != nil {
		log.Error(err.Error(), utils.RequestID(ctx))
		return wrapperServiceAliasResponse(req, err)
	}

	log.Info(fmt.Sprintf("update service alias, service(%s, %s), alias(%s)",
		req.GetService().GetValue(), req.GetNamespace().GetValue(), req.GetAlias().GetValue()), utils.RequestID(ctx))

	record := &apiservice.Service{Name: req.Alias, Namespace: req.Namespace}
	s.RecordHistory(ctx, serviceRecordEntry(ctx, record, alias, model.OUpdate))

	return api.NewServiceAliasResponse(apimodel.Code_ExecuteSuccess, req)
}

// GetServiceAliases 查找服务别名
func (s *Server) GetServiceAliases(ctx context.Context, query map[string]string) *apiservice.BatchQueryResponse {
	// 先处理offset和limit
	offset, limit, err := utils.ParseOffsetAndLimit(query)
	if err != nil {
		return api.NewBatchQueryResponse(apimodel.Code_InvalidParameter)
	}

	// 处理剩余的参数
	filter := make(map[string]string)
	for key, value := range query {
		if _, ok := AliasFilterAttributes[key]; !ok {
			log.Errorf("[Server][Alias][Query] attribute(%s) is not allowed", key)
			return api.NewBatchQueryResponse(apimodel.Code_InvalidParameter)
		}
		filter[key] = value
	}

	total, aliases, err := s.storage.GetServiceAliases(filter, offset, limit)
	if err != nil {
		log.Errorf("[Server][Alias] get aliases err: %s", err.Error())
		return api.NewBatchQueryResponse(commonstore.StoreCode2APICode(err))
	}

	resp := api.NewBatchQueryResponse(apimodel.Code_ExecuteSuccess)
	resp.Amount = utils.NewUInt32Value(total)
	resp.Size = utils.NewUInt32Value(uint32(len(aliases)))
	resp.Aliases = make([]*apiservice.ServiceAlias, 0, len(aliases))
	for _, entry := range aliases {
		item := &apiservice.ServiceAlias{
			Id:             utils.NewStringValue(entry.ID),
			Service:        utils.NewStringValue(entry.Service),
			Namespace:      utils.NewStringValue(entry.Namespace),
			Alias:          utils.NewStringValue(entry.Alias),
			AliasNamespace: utils.NewStringValue(entry.AliasNamespace),
			Owners:         utils.NewStringValue(entry.Owner),
			Comment:        utils.NewStringValue(entry.Comment),
			Ctime:          utils.NewStringValue(commontime.Time2String(entry.CreateTime)),
			Mtime:          utils.NewStringValue(commontime.Time2String(entry.ModifyTime)),
		}
		resp.Aliases = append(resp.Aliases, item)
	}

	return resp
}

// updateServiceAliasAttribute 修改服务别名属性
func (s *Server) updateServiceAliasAttribute(req *apiservice.ServiceAlias, alias *model.Service, serviceID string) (
	*apiservice.Response, bool, bool) {
	var (
		needUpdate      bool
		needUpdateOwner bool
	)

	// 获取当前指向服务
	service, err := s.storage.GetServiceByID(alias.Reference)
	if err != nil {
		return api.NewServiceAliasResponse(commonstore.StoreCode2APICode(err), req), needUpdate, needUpdateOwner
	}

	if service.ID != serviceID {
		alias.Reference = serviceID
		needUpdate = true
	}

	if len(req.GetOwners().GetValue()) > 0 && req.GetOwners().GetValue() != alias.Owner {
		alias.Owner = req.GetOwners().GetValue()
		needUpdate = true
		needUpdateOwner = true
	}

	if req.GetComment() != nil && req.GetComment().GetValue() != alias.Comment {
		alias.Comment = req.GetComment().GetValue()
		needUpdate = true
	}

	if needUpdate {
		alias.Revision = utils.NewUUID()
	}

	return nil, needUpdate, needUpdateOwner
}

// createServiceAliasModel 构建存储结构
func (s *Server) createServiceAliasModel(req *apiservice.ServiceAlias, svcId string) (
	*model.Service, *apiservice.Response) {
	out := &model.Service{
		ID:        utils.NewUUID(),
		Name:      req.GetAlias().GetValue(),
		Namespace: req.GetAliasNamespace().GetValue(),
		Reference: svcId,
		Token:     utils.NewUUID(),
		Owner:     req.GetOwners().GetValue(),
		Comment:   req.GetComment().GetValue(),
		Revision:  utils.NewUUID(),
	}

	// sid类型，则创建SID
	if req.GetType() == apiservice.AliasType_CL5SID {
		layoutID, ok := Namespace2SidLayoutID[req.GetAliasNamespace().GetValue()]
		if !ok {
			log.Errorf("[Server][Alias] namespace(%s) not allow to create sid alias",
				req.GetNamespace().GetValue())
			return nil, api.NewServiceAliasResponse(apimodel.Code_InvalidNamespaceWithAlias, req)
		}
		sid, err := s.storage.GenNextL5Sid(layoutID)
		if err != nil {
			log.Errorf("[Server] gen next l5 sid err: %s", err.Error())
			return nil, api.NewServiceAliasResponse(commonstore.StoreCode2APICode(err), req)
		}
		out.Name = sid
	}

	return out, nil
}

// wrapperServiceAliasResponse wrapper service alias error
func wrapperServiceAliasResponse(alias *apiservice.ServiceAlias, err error) *apiservice.Response {
	if err == nil {
		return nil
	}
	resp := api.NewResponseWithMsg(commonstore.StoreCode2APICode(err), err.Error())
	resp.Alias = alias
	return resp
}
