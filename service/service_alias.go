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
	"errors"
	"fmt"

	"github.com/golang/protobuf/ptypes/wrappers"
	"go.uber.org/zap"

	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/model"
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
func (s *Server) CreateServiceAlias(ctx context.Context, req *api.ServiceAlias) *api.Response {
	if resp := checkCreateServiceAliasReq(ctx, req); resp != nil {
		return resp
	}

	rid := utils.ParseRequestID(ctx)
	tx, err := s.storage.CreateTransaction()
	if err != nil {
		log.Error(err.Error(), utils.ZapRequestID(rid))
		return api.NewServiceAliasResponse(api.StoreLayerException, req)
	}
	defer func() { _ = tx.Commit() }()

	service, response, done := s.checkPointServiceAlias(tx, req, rid)
	if done {
		return response
	}

	// 检查是否存在同名的alias
	if req.GetAlias().GetValue() != "" {
		oldAlias, getErr := s.storage.GetService(req.GetAlias().GetValue(),
			req.GetAliasNamespace().GetValue())
		if getErr != nil {
			log.Error(getErr.Error(), utils.ZapRequestID(rid))
			return api.NewServiceAliasResponse(api.StoreLayerException, req)
		}
		if oldAlias != nil {
			return api.NewServiceAliasResponse(api.ExistedResource, req)
		}
	}

	// 构建别名的信息，这里包括了创建SID
	input, resp := s.createServiceAliasModel(req, service.ID)
	if resp != nil {
		return resp
	}
	if err := s.storage.AddService(input); err != nil {
		log.Error(err.Error(), utils.ZapRequestID(rid))
		return api.NewServiceAliasResponse(api.StoreLayerException, req)
	}

	log.Info(fmt.Sprintf("create service alias, service(%s, %s), alias(%s, %s)",
		req.Service.Value, req.Namespace.Value, input.Name, input.Namespace), utils.ZapRequestID(rid))
	out := &api.ServiceAlias{
		Service:        req.Service,
		Namespace:      req.Namespace,
		Alias:          req.Alias,
		AliasNamespace: req.AliasNamespace,
		ServiceToken:   &wrappers.StringValue{Value: input.Token},
	}
	if out.GetAlias().GetValue() == "" {
		out.Alias = utils.NewStringValue(input.Name)
	}
	record := &api.Service{Name: out.Alias, Namespace: out.AliasNamespace}
	s.RecordHistory(serviceRecordEntry(ctx, record, input, model.OCreate))
	return api.NewServiceAliasResponse(api.ExecuteSuccess, out)
}

func (s *Server) checkPointServiceAlias(tx store.Transaction, req *api.ServiceAlias, rid string) (*model.Service, *api.Response, bool) {
	// 检查指向服务是否存在以及是否为别名
	service, err := tx.LockService(req.GetService().GetValue(), req.GetNamespace().GetValue())
	if err != nil {
		log.Error(err.Error(), utils.ZapRequestID(rid))
		return nil, api.NewServiceAliasResponse(api.StoreLayerException, req), true
	}
	if service == nil {
		return nil, api.NewServiceAliasResponse(api.NotFoundService, req), true
	}
	// 检查该服务是否已经是一个别名服务，不允许再为别名创建别名
	if service.IsAlias() {
		return nil, api.NewServiceAliasResponse(api.NotAllowCreateAliasForAlias, req), true
	}
	return service, nil, false
}

// DeleteServiceAlias 删除服务别名
//
//	需要带上源服务name，namespace，token
//	另外一种删除别名的方式，是直接调用删除服务的接口，也是可行的
func (s *Server) DeleteServiceAlias(ctx context.Context, req *api.ServiceAlias) *api.Response {
	if resp := checkDeleteServiceAliasReq(ctx, req); resp != nil {
		return resp
	}
	rid := utils.ParseRequestID(ctx)
	alias, err := s.storage.GetService(req.GetAlias().GetValue(),
		req.GetAliasNamespace().GetValue())
	if err != nil {
		log.Error(err.Error(), utils.ZapRequestID(rid))
		return api.NewServiceAliasResponse(api.StoreLayerException, req)
	}
	if alias == nil {
		return api.NewServiceAliasResponse(api.NotFoundServiceAlias, req)
	}

	// 直接删除alias
	if err := s.storage.DeleteServiceAlias(req.GetAlias().GetValue(),
		req.GetAliasNamespace().GetValue()); err != nil {
		log.Error(err.Error(), utils.ZapRequestID(rid))
		return api.NewServiceAliasResponse(api.StoreLayerException, req)
	}

	return api.NewServiceAliasResponse(api.ExecuteSuccess, req)
}

func checkBatchAlias(req []*api.ServiceAlias) *api.BatchWriteResponse {
	if len(req) == 0 {
		return api.NewBatchWriteResponse(api.EmptyRequest)
	}

	if len(req) > MaxBatchSize {
		return api.NewBatchWriteResponse(api.BatchSizeOverLimit)
	}

	return nil
}

// DeleteServiceAliases 删除服务别名列表
func (s *Server) DeleteServiceAliases(ctx context.Context, req []*api.ServiceAlias) *api.BatchWriteResponse {
	if checkError := checkBatchAlias(req); checkError != nil {
		return checkError
	}

	responses := api.NewBatchWriteResponse(api.ExecuteSuccess)
	for _, alias := range req {
		response := s.DeleteServiceAlias(ctx, alias)
		responses.Collect(response)
	}

	return api.FormatBatchWriteResponse(responses)
}

// UpdateServiceAlias 修改服务别名
func (s *Server) UpdateServiceAlias(ctx context.Context, req *api.ServiceAlias) *api.Response {
	rid := utils.ParseRequestID(ctx)

	// 检查请求参数
	if resp := checkReviseServiceAliasReq(ctx, req); resp != nil {
		return resp
	}

	// 检查别名负责人
	// if err := checkResourceOwners(req.GetOwners()); err != nil {
	//	return api.NewServiceAliasResponse(api.InvalidServiceAliasOwners, req)
	// }

	// 检查服务别名是否存在
	alias, err := s.storage.GetService(req.GetAlias().GetValue(), req.GetAliasNamespace().GetValue())
	if err != nil {
		log.Error(err.Error(), utils.ZapRequestID(rid))
		return api.NewServiceAliasResponse(api.StoreLayerException, req)
	}
	if alias == nil {
		return api.NewServiceAliasResponse(api.NotFoundServiceAlias, req)
	}

	// 检查将要指向的服务是否存在
	service, err := s.storage.GetService(req.GetService().GetValue(), req.GetNamespace().GetValue())
	if err != nil {
		log.Error(err.Error(), utils.ZapRequestID(rid))
		return api.NewServiceAliasResponse(api.StoreLayerException, req)
	}
	if service == nil {
		return api.NewServiceAliasResponse(api.NotFoundService, req)
	}
	// 检查该服务是否已经是一个别名服务，不允许再为别名创建别名
	if service.IsAlias() {
		return api.NewServiceAliasResponse(api.NotAllowCreateAliasForAlias, req)
	}

	// 判断是否需要修改
	resp, needUpdate, needUpdateOwner := s.updateServiceAliasAttribute(req, alias, service.ID)
	if resp != nil {
		return resp
	}

	if !needUpdate {
		log.Info("update service alias data no change, no need update", utils.ZapRequestID(rid),
			zap.String("service alias", req.String()))
		return api.NewServiceAliasResponse(api.NoNeedUpdate, req)
	}

	// 执行存储层操作
	if err := s.storage.UpdateServiceAlias(alias, needUpdateOwner); err != nil {
		log.Error(err.Error(), utils.ZapRequestID(rid))
		return wrapperServiceAliasResponse(req, err)
	}

	log.Info(fmt.Sprintf("update service alias, service(%s, %s), alias(%s)",
		req.GetService().GetValue(), req.GetNamespace().GetValue(), req.GetAlias().GetValue()), utils.ZapRequestID(rid))

	record := &api.Service{Name: req.Alias, Namespace: req.Namespace}
	s.RecordHistory(serviceRecordEntry(ctx, record, alias, model.OUpdate))

	return api.NewServiceAliasResponse(api.ExecuteSuccess, req)
}

// GetServiceAliases 查找服务别名
func (s *Server) GetServiceAliases(ctx context.Context, query map[string]string) *api.BatchQueryResponse {
	// 先处理offset和limit
	offset, limit, err := utils.ParseOffsetAndLimit(query)
	if err != nil {
		return api.NewBatchQueryResponse(api.InvalidParameter)
	}

	// 处理剩余的参数
	filter := make(map[string]string)
	for key, value := range query {
		if _, ok := AliasFilterAttributes[key]; !ok {
			log.Errorf("[Server][Alias][Query] attribute(%s) is not allowed", key)
			return api.NewBatchQueryResponse(api.InvalidParameter)
		}
		filter[key] = value
	}

	total, aliases, err := s.storage.GetServiceAliases(filter, offset, limit)
	if err != nil {
		log.Errorf("[Server][Alias] get aliases err: %s", err.Error())
		return api.NewBatchQueryResponse(api.StoreLayerException)
	}

	resp := api.NewBatchQueryResponse(api.ExecuteSuccess)
	resp.Amount = utils.NewUInt32Value(total)
	resp.Size = utils.NewUInt32Value(uint32(len(aliases)))
	resp.Aliases = make([]*api.ServiceAlias, 0, len(aliases))
	for _, entry := range aliases {
		item := &api.ServiceAlias{
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

// checkCreateServiceAliasReq 检查别名请求
func checkCreateServiceAliasReq(ctx context.Context, req *api.ServiceAlias) *api.Response {
	response, done := preCheckAlias(req)
	if done {
		return response
	}
	// 检查字段长度是否大于DB中对应字段长
	err, notOk := CheckDbServiceAliasFieldLen(req)
	if notOk {
		return err
	}
	return nil
}

func preCheckAlias(req *api.ServiceAlias) (*api.Response, bool) {
	if req == nil {
		return api.NewServiceAliasResponse(api.EmptyRequest, req), true
	}

	if err := checkResourceName(req.GetService()); err != nil {
		return api.NewServiceAliasResponse(api.InvalidServiceName, req), true
	}

	if err := checkResourceName(req.GetNamespace()); err != nil {
		return api.NewServiceAliasResponse(api.InvalidNamespaceName, req), true
	}

	if err := checkResourceName(req.GetAliasNamespace()); err != nil {
		return api.NewServiceAliasResponse(api.InvalidNamespaceWithAlias, req), true
	}

	// 默认类型，需要检查alias是否为空
	if req.GetType() == api.AliasType_DEFAULT {
		if err := checkResourceName(req.GetAlias()); err != nil {
			return api.NewServiceAliasResponse(api.InvalidServiceAlias, req), true
		}
	}
	return nil, false
}

// checkReviseServiceAliasReq 检查删除、修改别名请求
func checkReviseServiceAliasReq(ctx context.Context, req *api.ServiceAlias) *api.Response {
	resp := checkDeleteServiceAliasReq(ctx, req)
	if resp != nil {
		return resp
	}
	// 检查服务名
	if err := checkResourceName(req.GetService()); err != nil {
		return api.NewServiceAliasResponse(api.InvalidServiceName, req)
	}

	// 检查命名空间
	if err := checkResourceName(req.GetNamespace()); err != nil {
		return api.NewServiceAliasResponse(api.InvalidNamespaceName, req)
	}
	return nil
}

// checkDeleteServiceAliasReq 检查删除、修改别名请求
func checkDeleteServiceAliasReq(ctx context.Context, req *api.ServiceAlias) *api.Response {
	if req == nil {
		return api.NewServiceAliasResponse(api.EmptyRequest, req)
	}

	// 检查服务别名
	if err := checkResourceName(req.GetAlias()); err != nil {
		return api.NewServiceAliasResponse(api.InvalidServiceAlias, req)
	}

	// 检查服务别名命名空间
	if err := checkResourceName(req.GetAliasNamespace()); err != nil {
		return api.NewServiceAliasResponse(api.InvalidNamespaceWithAlias, req)
	}

	// 检查字段长度是否大于DB中对应字段长
	err, notOk := CheckDbServiceAliasFieldLen(req)
	if notOk {
		return err
	}

	return nil
}

// updateServiceAliasAttribute 修改服务别名属性
func (s *Server) updateServiceAliasAttribute(req *api.ServiceAlias, alias *model.Service, serviceID string) (
	*api.Response, bool, bool) {
	var (
		needUpdate      bool
		needUpdateOwner bool
	)

	// 获取当前指向服务
	service, err := s.storage.GetServiceByID(alias.Reference)
	if err != nil {
		return api.NewServiceAliasResponse(api.StoreLayerException, req), needUpdate, needUpdateOwner
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
func (s *Server) createServiceAliasModel(req *api.ServiceAlias, svcId string) (
	*model.Service, *api.Response) {
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
	if req.GetType() == api.AliasType_CL5SID {
		layoutID, ok := Namespace2SidLayoutID[req.GetAliasNamespace().GetValue()]
		if !ok {
			log.Errorf("[Server][Alias] namespace(%s) not allow to create sid alias",
				req.GetNamespace().GetValue())
			return nil, api.NewServiceAliasResponse(api.InvalidNamespaceWithAlias, req)
		}
		sid, err := s.storage.GenNextL5Sid(layoutID)
		if err != nil {
			log.Errorf("[Server] gen next l5 sid err: %s", err.Error())
			return nil, api.NewServiceAliasResponse(api.StoreLayerException, req)
		}
		out.Name = sid
	}

	return out, nil
}

// getSourceServiceToken 根据Reference获取源服务的token
func (s *Server) getSourceServiceToken(refer string) (string, uint32, error) {
	if refer == "" {
		return "", 0, nil
	}
	service, err := s.storage.GetServiceByID(refer)
	if err != nil {
		return "", api.StoreLayerException, err
	}
	if service == nil {
		return "", api.NotFoundSourceService, errors.New("not found source service")
	}

	return service.Token, 0, nil
}

// wrapperServiceAliasResponse wrapper service alias error
func wrapperServiceAliasResponse(alias *api.ServiceAlias, err error) *api.Response {
	resp := storeError2Response(err)
	if resp == nil {
		return nil
	}

	resp.Alias = alias
	return resp
}

// CheckDbServiceAliasFieldLen 检查DB中service表对应的入参字段合法性
func CheckDbServiceAliasFieldLen(req *api.ServiceAlias) (*api.Response, bool) {
	if err := utils.CheckDbStrFieldLen(req.GetService(), MaxDbServiceNameLength); err != nil {
		return api.NewServiceAliasResponse(api.InvalidServiceName, req), true
	}
	if err := utils.CheckDbStrFieldLen(req.GetNamespace(), MaxDbServiceNamespaceLength); err != nil {
		return api.NewServiceAliasResponse(api.InvalidNamespaceName, req), true
	}
	if err := utils.CheckDbStrFieldLen(req.GetAlias(), MaxDbServiceNameLength); err != nil {
		return api.NewServiceAliasResponse(api.InvalidServiceAlias, req), true
	}
	if err := utils.CheckDbStrFieldLen(req.GetAliasNamespace(), MaxDbServiceNamespaceLength); err != nil {
		return api.NewServiceAliasResponse(api.InvalidNamespaceWithAlias, req), true
	}
	if err := utils.CheckDbStrFieldLen(req.GetComment(), MaxDbServiceCommentLength); err != nil {
		return api.NewServiceAliasResponse(api.InvalidServiceAliasComment, req), true
	}
	if err := utils.CheckDbStrFieldLen(req.GetOwners(), MaxDbServiceOwnerLength); err != nil {
		return api.NewServiceAliasResponse(api.InvalidServiceAliasOwners, req), true
	}
	return nil, false
}
