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
	"encoding/json"
	"fmt"

	"go.uber.org/zap"

	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/model"
	commontime "github.com/polarismesh/polaris/common/time"
	"github.com/polarismesh/polaris/common/utils"
)

const (
	// Master is
	Master = "master"
	// Service is
	Service = "service"
	// Namespace namespace string
	Namespace = "namespace"
	// ID id
	ID = "id"
	// Version version
	Version = "version"
)

var (
	// MasterCircuitBreakers master circuit breakers
	MasterCircuitBreakers = map[string]bool{
		"id":         true,
		"namespace":  true,
		"name":       true,
		"owner":      true,
		"business":   true,
		"department": true,
		"offset":     true,
		"limit":      true,
	}

	// ReleaseCircuitBreakers release circuit breakers
	ReleaseCircuitBreakers = map[string]bool{
		"id":      true, // 必填参数
		"version": true,
		"offset":  true,
		"limit":   true,
	}

	// ServiceParams service params
	ServiceParams = map[string]bool{
		Service:   true,
		Namespace: true,
	}
)

// CreateCircuitBreakers 批量创建熔断规则
func (s *Server) CreateCircuitBreakers(ctx context.Context, req []*api.CircuitBreaker) *api.BatchWriteResponse {
	if checkErr := checkBatchCircuitBreakers(req); checkErr != nil {
		return checkErr
	}

	resps := api.NewBatchWriteResponse(api.ExecuteSuccess)
	for _, circuitBreaker := range req {
		resp := s.CreateCircuitBreaker(ctx, circuitBreaker)
		resps.Collect(resp)
	}
	return api.FormatBatchWriteResponse(resps)
}

// CreateCircuitBreaker 创建单个熔断规则
func (s *Server) CreateCircuitBreaker(ctx context.Context, req *api.CircuitBreaker) *api.Response {
	requestID := utils.ParseRequestID(ctx)

	// 参数校验并生成规则id
	id, resp := checkCreateCircuitBreaker(req)
	if resp != nil {
		return resp
	}

	// 生成version
	version := Master

	// 检查熔断规则是否存在
	circuitBreaker, err := s.storage.GetCircuitBreaker(id, version)
	if err != nil {
		log.Error(err.Error(), utils.ZapRequestID(requestID))
		return api.NewCircuitBreakerResponse(api.StoreLayerException, req)
	}
	if circuitBreaker != nil {
		req.Id = utils.NewStringValue(id)
		return api.NewCircuitBreakerResponse(api.ExistedResource, req)
	}

	// 构造底层数据结构
	token := utils.NewUUID()
	var data *model.CircuitBreaker
	data, err = api2CircuitBreaker(req, id, token, version)
	if err != nil {
		log.Error(err.Error(), utils.ZapRequestID(requestID))
		return api.NewCircuitBreakerResponse(api.ParseCircuitBreakerException, req)
	}

	// 执行存储层操作
	if err := s.storage.CreateCircuitBreaker(data); err != nil {
		log.Error(err.Error(), utils.ZapRequestID(requestID))
		return wrapperCircuitBreakerStoreResponse(req, err)
	}

	msg := fmt.Sprintf("create circuit breaker: id=%v, version=%v, name=%v, namespace=%v",
		data.ID, data.Version, data.Name, data.Namespace)
	log.Info(msg, utils.ZapRequestID(requestID))

	// todo 记录操作记录

	// 返回请求结果
	req.Id = utils.NewStringValue(data.ID)
	req.Token = utils.NewStringValue(data.Token)
	req.Version = utils.NewStringValue(Master)

	return api.NewCircuitBreakerResponse(api.ExecuteSuccess, req)
}

// CreateCircuitBreakerVersions 批量创建熔断规则版本
func (s *Server) CreateCircuitBreakerVersions(ctx context.Context, req []*api.CircuitBreaker) *api.BatchWriteResponse {
	if checkErr := checkBatchCircuitBreakers(req); checkErr != nil {
		return checkErr
	}

	resps := api.NewBatchWriteResponse(api.ExecuteSuccess)
	for _, circuitBreaker := range req {
		resp := s.CreateCircuitBreakerVersion(ctx, circuitBreaker)
		resps.Collect(resp)
	}
	return api.FormatBatchWriteResponse(resps)
}

// CreateCircuitBreakerVersion 创建单个熔断规则版本
func (s *Server) CreateCircuitBreakerVersion(ctx context.Context, req *api.CircuitBreaker) *api.Response {
	requestID := utils.ParseRequestID(ctx)

	// 参数检查
	id, resp := checkReviseCircuitBreaker(ctx, req)
	if resp != nil {
		return resp
	}

	// 判断version是否为master
	if req.GetVersion().GetValue() == Master {
		return api.NewCircuitBreakerResponse(api.InvalidCircuitBreakerVersion, req)
	}

	// 判断规则的master版本是否存在并鉴权
	circuitBreaker, resp := s.checkCircuitBreakerValid(ctx, req, id, Master)
	if resp != nil {
		if resp.GetCode().GetValue() == api.NotFoundCircuitBreaker {
			return api.NewCircuitBreakerResponse(api.NotFoundMasterConfig, req)
		}
		return resp
	}

	// 判断此版本是否存在
	tagCircuitBreaker, err := s.storage.GetCircuitBreaker(id, req.GetVersion().GetValue())
	if err != nil {
		log.Error(err.Error(), utils.ZapRequestID(requestID))
		return api.NewCircuitBreakerResponse(api.StoreLayerException, req)
	}
	if tagCircuitBreaker != nil {
		return api.NewCircuitBreakerResponse(api.ExistedResource, req)
	}

	// 构造底层数据结构
	newReq := &api.CircuitBreaker{
		Id:         utils.NewStringValue(circuitBreaker.ID),
		Version:    req.GetVersion(),
		Name:       utils.NewStringValue(circuitBreaker.Name),
		Namespace:  utils.NewStringValue(circuitBreaker.Namespace),
		Inbounds:   req.GetInbounds(),
		Outbounds:  req.GetOutbounds(),
		Token:      req.GetToken(),
		Owners:     utils.NewStringValue(circuitBreaker.Owner),
		Comment:    utils.NewStringValue(circuitBreaker.Comment),
		Business:   utils.NewStringValue(circuitBreaker.Business),
		Department: utils.NewStringValue(circuitBreaker.Department),
	}

	var data *model.CircuitBreaker
	data, err = api2CircuitBreaker(newReq, circuitBreaker.ID, circuitBreaker.Token, req.GetVersion().GetValue())
	if err != nil {
		log.Error(err.Error(), utils.ZapRequestID(requestID))
		return api.NewCircuitBreakerResponse(api.ParseCircuitBreakerException, req)
	}

	// 执行存储层操作
	if err := s.storage.TagCircuitBreaker(data); err != nil {
		log.Error(err.Error(), utils.ZapRequestID(requestID))
		return wrapperCircuitBreakerStoreResponse(req, err)
	}

	msg := fmt.Sprintf("tag circuit breaker: id=%v, version=%v, name=%v, namespace=%v",
		data.ID, data.Version, data.Name, data.Namespace)
	log.Info(msg, utils.ZapRequestID(requestID))

	// todo 记录操作记录

	return api.NewCircuitBreakerResponse(api.ExecuteSuccess, req)
}

// DeleteCircuitBreakers 批量删除熔断规则
func (s *Server) DeleteCircuitBreakers(ctx context.Context, req []*api.CircuitBreaker) *api.BatchWriteResponse {
	if checkErr := checkBatchCircuitBreakers(req); checkErr != nil {
		return checkErr
	}

	resps := api.NewBatchWriteResponse(api.ExecuteSuccess)
	for _, circuitBreaker := range req {
		resp := s.DeleteCircuitBreaker(ctx, circuitBreaker)
		resps.Collect(resp)
	}
	return api.FormatBatchWriteResponse(resps)
}

// DeleteCircuitBreaker 删除单个熔断规则
func (s *Server) DeleteCircuitBreaker(ctx context.Context, req *api.CircuitBreaker) *api.Response {
	requestID := utils.ParseRequestID(ctx)

	// 参数校验
	id, resp := checkReviseCircuitBreaker(ctx, req)
	if resp != nil {
		return resp
	}

	// 检查熔断规则是否存在并鉴权
	if _, resp := s.checkCircuitBreakerValid(ctx, req, id, req.GetVersion().GetValue()); resp != nil {
		if resp.GetCode().GetValue() == api.NotFoundCircuitBreaker {
			return api.NewCircuitBreakerResponse(api.ExecuteSuccess, req)
		}
		return resp
	}

	if req.GetVersion().GetValue() == Master {
		return s.deleteMasterCircuitBreaker(requestID, id, req)
	}

	return s.deleteTagCircuitBreaker(requestID, id, req)
}

// deleteMasterCircuitBreaker 删除master熔断规则
func (s *Server) deleteMasterCircuitBreaker(requestID string, id string, req *api.CircuitBreaker) *api.Response {
	// 检查规则是否有绑定服务
	relations, err := s.storage.GetCircuitBreakerMasterRelation(id)
	if err != nil {
		log.Error(err.Error(), utils.ZapRequestID(requestID))
		return api.NewCircuitBreakerResponse(api.StoreLayerException, req)
	}
	if len(relations) > 0 {
		log.Errorf("the number of services bound to the circuit breaker(id=%s, version=%s) is %d",
			id, req.GetVersion().GetValue(), len(relations))
		return api.NewCircuitBreakerResponse(api.ExistReleasedConfig, req)
	}

	// 执行存储层操作
	if err := s.storage.DeleteMasterCircuitBreaker(id); err != nil {
		log.Error(err.Error(), utils.ZapRequestID(requestID))
		return wrapperCircuitBreakerStoreResponse(req, err)
	}

	msg := fmt.Sprintf("delete master circuit breaker: id=%v", id)
	log.Info(msg, utils.ZapRequestID(requestID))

	// todo 操作记录

	return api.NewCircuitBreakerResponse(api.ExecuteSuccess, req)
}

/**
 * @brief 删除熔断规则版本
 */
func (s *Server) deleteTagCircuitBreaker(requestID string, id string, req *api.CircuitBreaker) *api.Response {
	// 检查规则是否有绑定服务
	relation, err := s.storage.GetCircuitBreakerRelation(id, req.GetVersion().GetValue())
	if err != nil {
		log.Error(err.Error(), utils.ZapRequestID(requestID))
		return api.NewCircuitBreakerResponse(api.StoreLayerException, req)
	}
	if len(relation) > 0 {
		log.Errorf("the number of services bound to the circuit breaker(id=%s, version=%s) is %d",
			id, req.GetVersion().GetValue(), len(relation))
		return api.NewCircuitBreakerResponse(api.ExistReleasedConfig, req)
	}

	// 执行存储层操作
	if err := s.storage.DeleteTagCircuitBreaker(id, req.GetVersion().GetValue()); err != nil {
		log.Error(err.Error(), utils.ZapRequestID(requestID))
		return wrapperCircuitBreakerStoreResponse(req, err)
	}

	msg := fmt.Sprintf("delete circuit breaker version: id=%v, version=%v", id, req.GetVersion().GetValue())
	log.Info(msg, utils.ZapRequestID(requestID))

	// todo 操作记录

	return api.NewCircuitBreakerResponse(api.ExecuteSuccess, req)
}

// UpdateCircuitBreakers 批量修改熔断规则
func (s *Server) UpdateCircuitBreakers(ctx context.Context, req []*api.CircuitBreaker) *api.BatchWriteResponse {
	if checkErr := checkBatchCircuitBreakers(req); checkErr != nil {
		return checkErr
	}

	resps := api.NewBatchWriteResponse(api.ExecuteSuccess)
	for _, circuitBreaker := range req {
		resp := s.UpdateCircuitBreaker(ctx, circuitBreaker)
		resps.Collect(resp)
	}
	return api.FormatBatchWriteResponse(resps)
}

// UpdateCircuitBreaker 修改单个熔断规则
func (s *Server) UpdateCircuitBreaker(ctx context.Context, req *api.CircuitBreaker) *api.Response {
	requestID := utils.ParseRequestID(ctx)

	// 基础参数校验
	id, resp := checkReviseCircuitBreaker(ctx, req)
	if resp != nil {
		return resp
	}
	// 只允许修改master规则
	if req.GetVersion().GetValue() != Master {
		return api.NewCircuitBreakerResponse(api.InvalidCircuitBreakerVersion, req)
	}

	// 检查熔断规则是否存在并鉴权
	circuitBreaker, resp := s.checkCircuitBreakerValid(ctx, req, id, req.GetVersion().GetValue())
	if resp != nil {
		return resp
	}

	// 修改
	err, needUpdate := s.updateCircuitBreakerAttribute(req, circuitBreaker)
	if err != nil {
		return err
	}
	// 判断是否需要更新
	if !needUpdate {
		log.Info("update circuit breaker data no change, no need update",
			utils.ZapRequestID(requestID), zap.String("circuit breaker", req.String()))
		return api.NewCircuitBreakerResponse(api.NoNeedUpdate, req)
	}

	// 执行存储层操作
	if err := s.storage.UpdateCircuitBreaker(circuitBreaker); err != nil {
		log.Error(err.Error(), utils.ZapRequestID(requestID))
		return wrapperCircuitBreakerStoreResponse(req, err)
	}

	msg := fmt.Sprintf("update circuit breaker: id=%v, version=%v, name=%v, namespace=%v",
		circuitBreaker.ID, circuitBreaker.Version, circuitBreaker.Name, circuitBreaker.Namespace)
	log.Info(msg, utils.ZapRequestID(requestID))

	// todo 记录操作记录

	return api.NewCircuitBreakerResponse(api.ExecuteSuccess, req)
}

/**
 * @brief 修改规则属性
 */
func (s *Server) updateCircuitBreakerAttribute(req *api.CircuitBreaker, circuitBreaker *model.CircuitBreaker) (
	*api.Response, bool) {
	var needUpdate bool
	if req.GetOwners() != nil {
		if req.GetOwners().GetValue() == "" {
			return api.NewCircuitBreakerResponse(api.InvalidCircuitBreakerOwners, req), needUpdate
		}
		if req.GetOwners().GetValue() != circuitBreaker.Owner {
			circuitBreaker.Owner = req.GetOwners().GetValue()
			needUpdate = true
		}
	}

	if req.GetBusiness() != nil && req.GetBusiness().GetValue() != circuitBreaker.Business {
		circuitBreaker.Business = req.GetBusiness().GetValue()
		needUpdate = true
	}

	if req.GetDepartment() != nil && req.GetDepartment().GetValue() != circuitBreaker.Department {
		circuitBreaker.Department = req.GetDepartment().GetValue()
		needUpdate = true
	}

	if req.GetComment() != nil && req.GetComment().GetValue() != circuitBreaker.Comment {
		circuitBreaker.Comment = req.GetComment().GetValue()
		needUpdate = true
	}

	inbounds, outbounds, err := marshalCircuitBreakerRule(req.GetInbounds(), req.GetOutbounds())
	if err != nil {
		return api.NewCircuitBreakerResponse(api.ParseCircuitBreakerException, req), needUpdate
	}

	if req.GetInbounds() != nil && inbounds != circuitBreaker.Inbounds {
		circuitBreaker.Inbounds = inbounds
		needUpdate = true
	}

	if req.GetOutbounds() != nil && outbounds != circuitBreaker.Outbounds {
		circuitBreaker.Outbounds = outbounds
		needUpdate = true
	}

	if needUpdate {
		circuitBreaker.Revision = utils.NewUUID()
	}

	return nil, needUpdate
}

// ReleaseCircuitBreakers 批量发布熔断规则
func (s *Server) ReleaseCircuitBreakers(ctx context.Context, req []*api.ConfigRelease) *api.BatchWriteResponse {
	if checkErr := checkBatchConfigRelease(req); checkErr != nil {
		return checkErr
	}

	resp := api.NewBatchWriteResponse(api.ExecuteSuccess)
	for _, configRelease := range req {
		resp.Collect(s.ReleaseCircuitBreaker(ctx, configRelease))
	}
	return api.FormatBatchWriteResponse(resp)
}

// ReleaseCircuitBreaker 发布单个熔断规则
func (s *Server) ReleaseCircuitBreaker(ctx context.Context, req *api.ConfigRelease) *api.Response {
	requestID := utils.ParseRequestID(ctx)

	// 参数校验
	ruleID, resp := checkReleaseCircuitBreaker(req)
	if resp != nil {
		return resp
	}

	// 检查规则所属命名空间和服务所属命名空间是否一致
	if req.GetService().GetNamespace().GetValue() != req.GetCircuitBreaker().GetNamespace().GetValue() {
		return api.NewConfigResponse(api.NotAllowDifferentNamespaceBindRule, req)
	}

	// 检查服务是否可用并鉴权
	service, resp := s.checkService(ctx, req)
	if resp != nil {
		return resp
	}

	// 检查此版本规则是否存在
	ruleVersion := req.GetCircuitBreaker().GetVersion().GetValue()
	tagCircuitBreaker, err := s.storage.GetCircuitBreaker(ruleID, ruleVersion)
	if err != nil {
		log.Error(err.Error(), utils.ZapRequestID(requestID))
		return api.NewConfigResponse(api.StoreLayerException, req)
	}
	if tagCircuitBreaker == nil {
		return api.NewConfigResponse(api.NotFoundTagConfig, req)
	}

	// 检查服务绑定的熔断规则是否存在以及是否为此规则
	serviceName := req.GetService().GetName().GetValue()
	namespaceName := req.GetService().GetNamespace().GetValue()
	rule, err := s.storage.GetCircuitBreakersByService(serviceName, namespaceName)
	if err != nil {
		log.Error(err.Error(), utils.ZapRequestID(requestID))
		return api.NewConfigResponse(api.StoreLayerException, req)
	}
	if rule != nil && rule.ID == ruleID && rule.Version == ruleVersion {
		return api.NewConfigResponse(api.ExistedResource, req)
	}

	// 构造底层数据结构
	data := api2CircuitBreakerRelation(service.ID, ruleID, ruleVersion)

	// 执行存储层操作
	if err := s.storage.ReleaseCircuitBreaker(data); err != nil {
		log.Error(err.Error(), utils.ZapRequestID(requestID))
		return wrapperConfigStoreResponse(req, err)
	}

	msg := fmt.Sprintf("release circuit breaker: ruleID=%s, ruleVersion=%s, namespace=%s, service=%s",
		ruleID, ruleVersion, service.Namespace, service.Name)
	log.Info(msg, utils.ZapRequestID(requestID))

	// todo 操作记录

	return api.NewConfigResponse(api.ExecuteSuccess, req)
}

// UnBindCircuitBreakers 批量解绑熔断规则
func (s *Server) UnBindCircuitBreakers(ctx context.Context, req []*api.ConfigRelease) *api.BatchWriteResponse {
	if checkErr := checkBatchConfigRelease(req); checkErr != nil {
		return checkErr
	}
	resps := api.NewBatchWriteResponse(api.ExecuteSuccess)
	for _, configRelease := range req {
		resp := s.UnBindCircuitBreaker(ctx, configRelease)
		resps.Collect(resp)
	}
	return api.FormatBatchWriteResponse(resps)
}

// UnBindCircuitBreaker 解绑单个熔断规则
func (s *Server) UnBindCircuitBreaker(ctx context.Context, req *api.ConfigRelease) *api.Response {
	requestID := utils.ParseRequestID(ctx)

	// 参数校验
	ruleID, resp := checkReleaseCircuitBreaker(req)
	if resp != nil {
		return resp
	}

	// 检查服务是否可用并鉴权
	service, resp := s.checkService(ctx, req)
	if resp != nil {
		return resp
	}

	// 检查此版本的规则是否存在
	ruleVersion := req.GetCircuitBreaker().GetVersion().GetValue()
	tagCircuitBreaker, err := s.storage.GetCircuitBreaker(ruleID, ruleVersion)
	if err != nil {
		log.Error(err.Error(), utils.ZapRequestID(requestID))
		return api.NewConfigResponse(api.StoreLayerException, req)
	}
	if tagCircuitBreaker == nil {
		return api.NewConfigResponse(api.NotFoundTagConfig, req)
	}

	// 检查服务绑定的熔断规则是否存在以及是否为此规则
	serviceName := req.GetService().GetName().GetValue()
	namespaceName := req.GetService().GetNamespace().GetValue()
	rule, err := s.storage.GetCircuitBreakersByService(serviceName, namespaceName)
	if err != nil {
		log.Error(err.Error(), utils.ZapRequestID(requestID))
		return api.NewConfigResponse(api.StoreLayerException, req)
	}
	if rule == nil || rule.ID != ruleID || rule.Version != ruleVersion {
		return api.NewConfigResponse(api.ExecuteSuccess, req)
	}

	// 执行存储层操作
	if err := s.storage.UnbindCircuitBreaker(service.ID, ruleID, ruleVersion); err != nil {
		log.Error(err.Error(), utils.ZapRequestID(requestID))
		return wrapperConfigStoreResponse(req, err)
	}

	msg := fmt.Sprintf("unbind circuit breaker: ruleID=%s, ruleVersion=%s, namespace=%s, service=%s",
		ruleID, ruleVersion, service.Namespace, service.Name)
	log.Info(msg, utils.ZapRequestID(requestID))

	// todo 操作记录

	return api.NewConfigResponse(api.ExecuteSuccess, req)
}

// GetCircuitBreaker 根据id和version查询熔断规则
func (s *Server) GetCircuitBreaker(ctx context.Context, query map[string]string) *api.BatchQueryResponse {
	// 必填参数：id和version
	if _, ok := query[ID]; !ok {
		log.Errorf("params %s is not in querying circuit breaker", ID)
		return api.NewBatchQueryResponse(api.InvalidParameter)
	}
	if _, ok := query[Version]; !ok {
		log.Errorf("params %s is not in querying circuit breaker", Version)
		return api.NewBatchQueryResponse(api.InvalidParameter)
	}

	circuitBreaker, err := s.storage.GetCircuitBreaker(query[ID], query[Version])
	if err != nil {
		log.Errorf("get circuit breaker  err: %s", err.Error())
		return api.NewBatchQueryResponse(api.StoreLayerException)
	}

	var breaker *api.CircuitBreaker
	breaker, err = circuitBreaker2API(circuitBreaker)
	if err != nil {
		log.Errorf("get circuit breaker err: %s", err.Error())
		return api.NewBatchQueryResponse(api.ParseCircuitBreakerException)
	}

	resp := api.NewBatchQueryResponse(api.ExecuteSuccess)

	if breaker == nil {
		resp.Amount = utils.NewUInt32Value(0)
		resp.Size = utils.NewUInt32Value(0)
		resp.ConfigWithServices = []*api.ConfigWithService{}
		return resp
	}

	configWithService := &api.ConfigWithService{
		CircuitBreaker: breaker,
	}

	resp.Amount = utils.NewUInt32Value(1)
	resp.Size = utils.NewUInt32Value(1)
	resp.ConfigWithServices = make([]*api.ConfigWithService, 0, 1)
	resp.ConfigWithServices = append(resp.ConfigWithServices, configWithService)
	return resp
}

// GetCircuitBreakerVersions 根据id查询熔断规则所有版本
func (s *Server) GetCircuitBreakerVersions(ctx context.Context, query map[string]string) *api.BatchQueryResponse {
	// 必填参数：id
	if _, ok := query[ID]; !ok {
		log.Errorf("params %s is not in querying circuit breaker", ID)
		return api.NewBatchQueryResponse(api.InvalidParameter)
	}

	versions, err := s.storage.GetCircuitBreakerVersions(query[ID])
	if err != nil {
		log.Errorf("get circuit breaker versions err: %s", err.Error())
		return api.NewBatchQueryResponse(api.StoreLayerException)
	}

	nums := len(versions)

	resp := api.NewBatchQueryResponse(api.ExecuteSuccess)
	resp.Amount = utils.NewUInt32Value(uint32(nums))
	resp.Size = utils.NewUInt32Value(uint32(nums))
	resp.ConfigWithServices = make([]*api.ConfigWithService, 0, nums)
	for _, version := range versions {
		config := ruleIDAndVersion2API(query[ID], version)
		resp.ConfigWithServices = append(resp.ConfigWithServices, config)
	}
	return resp
}

// GetMasterCircuitBreakers 查询master熔断规则
func (s *Server) GetMasterCircuitBreakers(ctx context.Context, query map[string]string) *api.BatchQueryResponse {
	for key := range query {
		if _, ok := MasterCircuitBreakers[key]; !ok {
			log.Errorf("params %s is not allowed in querying master circuit breakers", key)
			return api.NewBatchQueryResponse(api.InvalidParameter)
		}
	}

	// 处理offset和limit
	offset, limit, err := utils.ParseOffsetAndLimit(query)
	if err != nil {
		return api.NewBatchQueryResponse(api.InvalidParameter)
	}

	c, err := s.storage.ListMasterCircuitBreakers(query, offset, limit)
	if err != nil {
		log.Errorf("get master circuit breakers err: %s", err.Error())
		return api.NewBatchQueryResponse(api.StoreLayerException)
	}

	return genCircuitBreakersResult(c)
}

// GetReleaseCircuitBreakers 根据规则id查询已发布规则
func (s *Server) GetReleaseCircuitBreakers(ctx context.Context, query map[string]string) *api.BatchQueryResponse {
	// 必须参数：id
	if _, ok := query[ID]; !ok {
		log.Errorf("params %s is not in querying release circuit breakers", ID)
		return api.NewBatchQueryResponse(api.InvalidParameter)
	}

	for key := range query {
		if _, ok := ReleaseCircuitBreakers[key]; !ok {
			log.Errorf("params %s is not allowed in querying release circuit breakers", key)
			return api.NewBatchQueryResponse(api.InvalidParameter)
		}
	}

	// id转化为rule_id
	if ruleID, ok := query[ID]; ok {
		query["rule_id"] = ruleID
		delete(query, ID)
	}

	if ruleVersion, ok := query[Version]; ok {
		query["rule_version"] = ruleVersion
		delete(query, Version)
	}

	// 处理offset和limit
	offset, limit, err := utils.ParseOffsetAndLimit(query)
	if err != nil {
		return api.NewBatchQueryResponse(api.InvalidParameter)
	}

	c, err := s.storage.ListReleaseCircuitBreakers(query, offset, limit)
	if err != nil {
		log.Errorf("get release circuit breakers err: %s", err.Error())
		return api.NewBatchQueryResponse(api.StoreLayerException)
	}

	return genCircuitBreakersResult(c)
}

// genCircuitBreakersResult 生成返回查询熔断规则的数据
func genCircuitBreakersResult(c *model.CircuitBreakerDetail) *api.BatchQueryResponse {
	resp := api.NewBatchQueryResponse(api.ExecuteSuccess)
	resp.Amount = utils.NewUInt32Value(c.Total)
	resp.Size = utils.NewUInt32Value(uint32(len(c.CircuitBreakerInfos)))
	resp.ConfigWithServices = make([]*api.ConfigWithService, 0, len(c.CircuitBreakerInfos))
	for _, item := range c.CircuitBreakerInfos {
		info, err := circuitBreaker2ConsoleAPI(item)
		if err != nil {
			log.Errorf("get circuit breakers err: %s", err.Error())
			return api.NewBatchQueryResponse(api.ParseCircuitBreakerException)
		}
		resp.ConfigWithServices = append(resp.ConfigWithServices, info)
	}
	return resp
}

// GetCircuitBreakerByService 根据服务查询绑定熔断规则
func (s *Server) GetCircuitBreakerByService(ctx context.Context, query map[string]string) *api.BatchQueryResponse {
	// 必须参数：service和namespace
	for key := range ServiceParams {
		if _, ok := query[key]; !ok {
			log.Errorf("params %s is not in querying circuit breakers by service", key)
			return api.NewBatchQueryResponse(api.InvalidParameter)
		}
	}

	circuitBreaker, err := s.storage.GetCircuitBreakersByService(query[Service], query[Namespace])
	if err != nil {
		log.Errorf("get circuit breaker by service err: %s", err.Error())
		return api.NewBatchQueryResponse(api.StoreLayerException)
	}

	breaker, err := circuitBreaker2API(circuitBreaker)
	if err != nil {
		log.Errorf("get circuit breaker to api err: %s", err.Error())
		return api.NewBatchQueryResponse(api.ParseCircuitBreakerException)
	}

	resp := api.NewBatchQueryResponse(api.ExecuteSuccess)

	if breaker == nil {
		resp.Amount = utils.NewUInt32Value(0)
		resp.Size = utils.NewUInt32Value(0)
		resp.ConfigWithServices = []*api.ConfigWithService{}
		return resp
	}

	configWithService := &api.ConfigWithService{
		CircuitBreaker: breaker,
	}

	resp.Amount = utils.NewUInt32Value(1)
	resp.Size = utils.NewUInt32Value(1)
	resp.ConfigWithServices = make([]*api.ConfigWithService, 0, 1)
	resp.ConfigWithServices = append(resp.ConfigWithServices, configWithService)

	return resp
}

// GetCircuitBreakerToken 查询熔断规则的token
func (s *Server) GetCircuitBreakerToken(ctx context.Context, req *api.CircuitBreaker) *api.Response {
	id, resp := checkReviseCircuitBreaker(ctx, req)
	if resp != nil {
		return resp
	}

	circuitBreaker, resp := s.checkCircuitBreakerValid(ctx, req, id, Master)
	if resp != nil {
		return resp
	}

	out := api.NewResponse(api.ExecuteSuccess)
	out.CircuitBreaker = &api.CircuitBreaker{
		Id:        utils.NewStringValue(id),
		Name:      utils.NewStringValue(circuitBreaker.Name),
		Namespace: utils.NewStringValue(circuitBreaker.Namespace),
		Token:     utils.NewStringValue(circuitBreaker.Token),
	}
	return out
}

// checkBatchCircuitBreakers 检查熔断规则批量请求
func checkBatchCircuitBreakers(req []*api.CircuitBreaker) *api.BatchWriteResponse {
	if len(req) == 0 {
		return api.NewBatchWriteResponse(api.EmptyRequest)
	}

	if len(req) > MaxBatchSize {
		return api.NewBatchWriteResponse(api.BatchSizeOverLimit)
	}

	return nil
}

// checkBatchConfigRelease 检查规则发布批量请求
func checkBatchConfigRelease(req []*api.ConfigRelease) *api.BatchWriteResponse {
	if len(req) == 0 {
		return api.NewBatchWriteResponse(api.EmptyRequest)
	}

	if len(req) > MaxBatchSize {
		return api.NewBatchWriteResponse(api.BatchSizeOverLimit)
	}

	return nil
}

// checkCreateCircuitBreaker 检查创建熔断规则参数
func checkCreateCircuitBreaker(req *api.CircuitBreaker) (string, *api.Response) {
	if req == nil {
		return "", api.NewCircuitBreakerResponse(api.EmptyRequest, req)
	}
	// 检查负责人
	if err := checkResourceOwners(req.GetOwners()); err != nil {
		return "", api.NewCircuitBreakerResponse(api.InvalidCircuitBreakerOwners, req)
	}
	// 检查字段长度是否大于DB中对应字段长
	err, notOk := CheckDbCircuitBreakerFieldLen(req)
	if notOk {
		return "", err
	}
	return checkRuleTwoTuple(req)
}

// checkReviseCircuitBreaker 检查修改/删除/创建熔断规则参数
func checkReviseCircuitBreaker(ctx context.Context, req *api.CircuitBreaker) (string, *api.Response) {
	if req == nil {
		return "", api.NewCircuitBreakerResponse(api.EmptyRequest, req)
	}
	// 检查规则version
	if err := checkResourceName(req.GetVersion()); err != nil {
		return "", api.NewCircuitBreakerResponse(api.InvalidCircuitBreakerVersion, req)
	}
	// 检查规则token
	if token := parseCircuitBreakerToken(ctx, req); token == "" {
		return "", api.NewCircuitBreakerResponse(api.InvalidCircuitBreakerToken, req)
	}
	// 检查规则id
	if req.GetId() != nil {
		if req.GetId().GetValue() == "" {
			return "", api.NewCircuitBreakerResponse(api.InvalidCircuitBreakerID, req)
		}
		return req.GetId().GetValue(), nil
	}
	// 检查字段长度是否大于DB中对应字段长
	err, notOk := CheckDbCircuitBreakerFieldLen(req)
	if notOk {
		return "", err
	}
	return checkRuleTwoTuple(req)
}

// checkReleaseCircuitBreaker 检查发布、解绑熔断规则参数
func checkReleaseCircuitBreaker(req *api.ConfigRelease) (string, *api.Response) {
	if req == nil {
		return "", api.NewConfigResponse(api.EmptyRequest, req)
	}
	// 检查命名空间
	if err := checkResourceName(req.GetService().GetNamespace()); err != nil {
		return "", api.NewConfigResponse(api.InvalidNamespaceName, req)
	}
	if err := utils.CheckDbStrFieldLen(req.GetService().GetNamespace(), MaxDbServiceNamespaceLength); err != nil {
		return "", api.NewConfigResponse(api.InvalidNamespaceName, req)
	}
	// 检查服务名
	if err := checkResourceName(req.GetService().GetName()); err != nil {
		return "", api.NewConfigResponse(api.InvalidServiceName, req)
	}
	if err := utils.CheckDbStrFieldLen(req.GetService().GetName(), MaxDbServiceNameLength); err != nil {
		return "", api.NewConfigResponse(api.InvalidServiceName, req)
	}
	if err := utils.CheckDbStrFieldLen(req.GetService().GetToken(), MaxDbServiceToken); err != nil {
		return "", api.NewConfigResponse(api.InvalidServiceToken, req)
	}
	// 检查规则version
	if err := checkResourceName(req.GetCircuitBreaker().GetVersion()); err != nil {
		return "", api.NewConfigResponse(api.InvalidCircuitBreakerVersion, req)
	}
	if err := utils.CheckDbStrFieldLen(req.GetCircuitBreaker().GetVersion(), MaxDbCircuitbreakerVersion); err != nil {
		return "", api.NewConfigResponse(api.InvalidCircuitBreakerVersion, req)
	}
	// 判断version是否为master
	if req.GetCircuitBreaker().GetVersion().GetValue() == Master {
		return "", api.NewConfigResponse(api.InvalidCircuitBreakerVersion, req)
	}
	// 规则name和规则namespace必填
	return checkRuleTwoTuple(req.GetCircuitBreaker())
}

// checkRuleTwoTuple 根据规则name和规则namespace计算ID
func checkRuleTwoTuple(req *api.CircuitBreaker) (string, *api.Response) {
	// 检查规则name
	if err := checkResourceName(req.GetName()); err != nil {
		return "", api.NewCircuitBreakerResponse(api.InvalidCircuitBreakerName, req)
	}
	if err := utils.CheckDbStrFieldLen(req.GetName(), MaxDbCircuitbreakerName); err != nil {
		return "", api.NewCircuitBreakerResponse(api.InvalidCircuitBreakerName, req)
	}
	// 检查规则namespace
	if err := checkResourceName(req.GetNamespace()); err != nil {
		return "", api.NewCircuitBreakerResponse(api.InvalidCircuitBreakerNamespace, req)
	}
	if err := utils.CheckDbStrFieldLen(req.GetNamespace(), MaxDbCircuitbreakerNamespace); err != nil {
		return "", api.NewCircuitBreakerResponse(api.InvalidCircuitBreakerNamespace, req)
	}
	return utils.CalculateRuleID(req.GetName().GetValue(), req.GetNamespace().GetValue()), nil
}

// checkCircuitBreakerValid 修改/删除/发布熔断规则的公共检查
func (s *Server) checkCircuitBreakerValid(ctx context.Context, req *api.CircuitBreaker, id, version string) (
	*model.CircuitBreaker, *api.Response) {
	requestID := utils.ParseRequestID(ctx)

	// 检查熔断规则是否存在
	circuitBreaker, err := s.storage.GetCircuitBreaker(id, version)
	if err != nil {
		log.Error(err.Error(), utils.ZapRequestID(requestID))
		return nil, api.NewCircuitBreakerResponse(api.StoreLayerException, req)
	}
	if circuitBreaker == nil {
		return nil, api.NewCircuitBreakerResponse(api.NotFoundCircuitBreaker, req)
	}

	return circuitBreaker, nil
}

// checkService 判断服务是否可用并鉴权
func (s *Server) checkService(ctx context.Context, req *api.ConfigRelease) (*model.Service, *api.Response) {
	requestID := utils.ParseRequestID(ctx)
	serviceName := req.GetService().GetName().GetValue()
	namespaceName := req.GetService().GetNamespace().GetValue()

	service, err := s.storage.GetService(serviceName, namespaceName)
	if err != nil {
		log.Error(err.Error(), utils.ZapRequestID(requestID))
		return nil, api.NewConfigResponse(api.StoreLayerException, req)
	}
	if service == nil {
		return nil, api.NewConfigResponse(api.NotFoundService, req)
	}
	if service.IsAlias() {
		return nil, api.NewConfigResponse(api.NotAllowAliasBindRule, req)
	}

	return service, nil
}

// parseCircuitBreakerToken 获取熔断规则的token信息
func parseCircuitBreakerToken(ctx context.Context, req *api.CircuitBreaker) string {
	if token := req.GetToken().GetValue(); token != "" {
		return token
	}

	return utils.ParseToken(ctx)
}

// api2CircuitBreaker 创建存储层熔断规则模型
func api2CircuitBreaker(req *api.CircuitBreaker, id, token, version string) (*model.CircuitBreaker, error) {
	inbounds, outbounds, err := marshalCircuitBreakerRule(req.GetInbounds(), req.GetOutbounds())
	if err != nil {
		return nil, err
	}

	circuitBreaker := &model.CircuitBreaker{
		ID:         id,
		Version:    version,
		Name:       req.GetName().GetValue(),
		Namespace:  req.GetNamespace().GetValue(),
		Business:   req.GetBusiness().GetValue(),
		Department: req.GetDepartment().GetValue(),
		Comment:    req.GetComment().GetValue(),
		Inbounds:   inbounds,
		Outbounds:  outbounds,
		Token:      token,
		Owner:      req.GetOwners().GetValue(),
		Revision:   utils.NewUUID(),
	}

	return circuitBreaker, nil
}

// api2CircuitBreakerRelation 创建存储层熔断规则关系模型
func api2CircuitBreakerRelation(serviceID, ruleID, ruleVersion string) *model.CircuitBreakerRelation {
	circuitBreakerRelation := &model.CircuitBreakerRelation{
		ServiceID:   serviceID,
		RuleID:      ruleID,
		RuleVersion: ruleVersion,
	}

	return circuitBreakerRelation
}

// ruleIDAndVersion2API 返回规则id和version
func ruleIDAndVersion2API(id, version string) *api.ConfigWithService {
	out := &api.ConfigWithService{}

	rule := &api.CircuitBreaker{
		Id:      utils.NewStringValue(id),
		Version: utils.NewStringValue(version),
	}

	out.CircuitBreaker = rule
	return out
}

// circuitBreaker2API 把内部数据结构转化为熔断规则API参数
func circuitBreaker2API(req *model.CircuitBreaker) (*api.CircuitBreaker, error) {
	if req == nil {
		return nil, nil
	}

	// token不返回
	out := &api.CircuitBreaker{
		Id:         utils.NewStringValue(req.ID),
		Version:    utils.NewStringValue(req.Version),
		Name:       utils.NewStringValue(req.Name),
		Namespace:  utils.NewStringValue(req.Namespace),
		Owners:     utils.NewStringValue(req.Owner),
		Comment:    utils.NewStringValue(req.Comment),
		Ctime:      utils.NewStringValue(commontime.Time2String(req.CreateTime)),
		Mtime:      utils.NewStringValue(commontime.Time2String(req.ModifyTime)),
		Revision:   utils.NewStringValue(req.Revision),
		Business:   utils.NewStringValue(req.Business),
		Department: utils.NewStringValue(req.Department),
	}

	if req.Inbounds != "" {
		var inBounds []*api.CbRule
		if err := json.Unmarshal([]byte(req.Inbounds), &inBounds); err != nil {
			return nil, err
		}
		out.Inbounds = inBounds
	}
	if req.Outbounds != "" {
		var outBounds []*api.CbRule
		if err := json.Unmarshal([]byte(req.Outbounds), &outBounds); err != nil {
			return nil, err
		}
		out.Outbounds = outBounds
	}
	return out, nil
}

// circuitBreaker2ClientAPI 把内部数据结构转化为客户端API参数
func circuitBreaker2ClientAPI(req *model.ServiceWithCircuitBreaker, service string, namespace string) (
	*api.CircuitBreaker, error) {
	if req == nil {
		return nil, nil
	}

	out, err := circuitBreaker2API(req.CircuitBreaker)
	if err != nil {
		return nil, err
	}

	if out == nil {
		return nil, nil
	}

	out.Service = utils.NewStringValue(service)
	out.ServiceNamespace = utils.NewStringValue(namespace)

	return out, nil
}

// circuitBreaker2ConsoleAPI 把内部数据结构转化为控制台API参数
func circuitBreaker2ConsoleAPI(req *model.CircuitBreakerInfo) (*api.ConfigWithService, error) {
	if req == nil {
		return nil, nil
	}

	out := &api.ConfigWithService{}
	circuitBreaker, err := circuitBreaker2API(req.CircuitBreaker)
	if err != nil {
		return nil, err
	}
	out.CircuitBreaker = circuitBreaker

	if len(req.Services) == 0 {
		return out, nil
	}

	services := make([]*api.Service, 0, len(req.Services))
	for _, item := range req.Services {
		service := serviceRelatedRules2API(item)
		services = append(services, service)
	}

	out.Services = services
	return out, nil
}

// serviceRelatedRules2API 转化服务名和命名空间
func serviceRelatedRules2API(service *model.Service) *api.Service {
	if service == nil {
		return nil
	}

	out := &api.Service{
		Name:      utils.NewStringValue(service.Name),
		Namespace: utils.NewStringValue(service.Namespace),
		Owners:    utils.NewStringValue(service.Owner),
		Ctime:     utils.NewStringValue(commontime.Time2String(service.CreateTime)),
		Mtime:     utils.NewStringValue(commontime.Time2String(service.ModifyTime)),
	}

	return out
}

// marshalCircuitBreakerRule 序列化inbounds和outbounds
func marshalCircuitBreakerRule(in []*api.CbRule, out []*api.CbRule) (string, string, error) {
	inbounds, err := json.Marshal(in)
	if err != nil {
		return "", "", err
	}

	outbounds, err := json.Marshal(out)
	if err != nil {
		return "", "", err
	}

	return string(inbounds), string(outbounds), nil
}

// wrapperCircuitBreakerStoreResponse 封装熔断规则存储层错误
func wrapperCircuitBreakerStoreResponse(circuitBreaker *api.CircuitBreaker, err error) *api.Response {
	resp := storeError2Response(err)
	if resp == nil {
		return nil
	}
	resp.CircuitBreaker = circuitBreaker
	return resp
}

// wrapperConfigStoreResponse 封装熔断规则发布存储层错误
func wrapperConfigStoreResponse(configRelease *api.ConfigRelease, err error) *api.Response {
	resp := storeError2Response(err)
	if resp == nil {
		return nil
	}
	resp.ConfigRelease = configRelease
	return resp
}

// CheckDbCircuitBreakerFieldLen 检查DB中circuitBreaker表对应的入参字段合法性
func CheckDbCircuitBreakerFieldLen(req *api.CircuitBreaker) (*api.Response, bool) {
	if err := utils.CheckDbStrFieldLen(req.GetName(), MaxDbCircuitbreakerName); err != nil {
		return api.NewCircuitBreakerResponse(api.InvalidCircuitBreakerName, req), true
	}
	if err := utils.CheckDbStrFieldLen(req.GetNamespace(), MaxDbCircuitbreakerNamespace); err != nil {
		return api.NewCircuitBreakerResponse(api.InvalidCircuitBreakerNamespace, req), true
	}
	if err := utils.CheckDbStrFieldLen(req.GetBusiness(), MaxDbCircuitbreakerBusiness); err != nil {
		return api.NewCircuitBreakerResponse(api.InvalidCircuitBreakerBusiness, req), true
	}
	if err := utils.CheckDbStrFieldLen(req.GetDepartment(), MaxDbCircuitbreakerDepartment); err != nil {
		return api.NewCircuitBreakerResponse(api.InvalidCircuitBreakerDepartment, req), true
	}
	if err := utils.CheckDbStrFieldLen(req.GetComment(), MaxDbCircuitbreakerComment); err != nil {
		return api.NewCircuitBreakerResponse(api.InvalidCircuitBreakerComment, req), true
	}
	if err := utils.CheckDbStrFieldLen(req.GetOwners(), MaxDbCircuitbreakerOwner); err != nil {
		return api.NewCircuitBreakerResponse(api.InvalidCircuitBreakerOwners, req), true
	}

	return nil, false
}
