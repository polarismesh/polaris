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
	"time"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/ptypes/wrappers"
	apifault "github.com/polarismesh/specification/source/go/api/v1/fault_tolerance"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"

	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/model"
	commonstore "github.com/polarismesh/polaris/common/store"
	commontime "github.com/polarismesh/polaris/common/time"
	"github.com/polarismesh/polaris/common/utils"
)

// CreateCircuitBreakerRules Create a CircuitBreaker rule
func (s *Server) CreateCircuitBreakerRules(
	ctx context.Context, request []*apifault.CircuitBreakerRule) *apiservice.BatchWriteResponse {
	responses := api.NewBatchWriteResponse(apimodel.Code_ExecuteSuccess)
	for _, cbRule := range request {
		response := s.createCircuitBreakerRule(ctx, cbRule)
		api.Collect(responses, response)
	}
	return api.FormatBatchWriteResponse(responses)
}

// CreateCircuitBreakerRule Create a CircuitBreaker rule
func (s *Server) createCircuitBreakerRule(
	ctx context.Context, request *apifault.CircuitBreakerRule) *apiservice.Response {
	requestID := utils.ParseRequestID(ctx)
	if resp := checkCircuitBreakerRuleParams(request, false, true); resp != nil {
		return resp
	}

	// 构造底层数据结构
	data, err := api2CircuitBreakerRule(request)
	if err != nil {
		log.Error(err.Error(), utils.ZapRequestID(requestID))
		return api.NewResponse(apimodel.Code_ParseCircuitBreakerException)
	}
	exists, err := s.storage.HasCircuitBreakerRuleByName(data.Name, data.Namespace)
	if err != nil {
		log.Error(err.Error(), utils.ZapRequestID(requestID))
		return api.NewResponseWithMsg(commonstore.StoreCode2APICode(err), err.Error())
	}
	if exists {
		return api.NewResponse(apimodel.Code_ServiceExistedCircuitBreakers)
	}
	data.ID = utils.NewUUID()

	// 存储层操作
	if err := s.storage.CreateCircuitBreakerRule(data); err != nil {
		log.Error(err.Error(), utils.ZapRequestID(requestID))
		return api.NewResponseWithMsg(commonstore.StoreCode2APICode(err), err.Error())
	}

	msg := fmt.Sprintf("create circuitBreaker rule: id=%v, name=%v, namespace=%v",
		data.ID, request.GetName(), request.GetNamespace())
	log.Info(msg, utils.ZapRequestID(requestID))

	s.RecordHistory(ctx, circuitBreakerRuleRecordEntry(ctx, request, data, model.OCreate))

	request.Id = data.ID
	return api.NewAnyDataResponse(apimodel.Code_ExecuteSuccess, request)
}

func checkCircuitBreakerRuleParams(
	req *apifault.CircuitBreakerRule, idRequired bool, nameRequired bool) *apiservice.Response {
	if req == nil {
		return api.NewResponse(apimodel.Code_EmptyRequest)
	}
	if resp := checkCircuitBreakerRuleParamsDbLen(req); nil != resp {
		return resp
	}
	if nameRequired && len(req.GetName()) == 0 {
		return api.NewResponse(apimodel.Code_InvalidCircuitBreakerName)
	}
	if idRequired && len(req.GetId()) == 0 {
		return api.NewResponse(apimodel.Code_InvalidCircuitBreakerID)
	}
	return nil
}

func checkCircuitBreakerRuleParamsDbLen(req *apifault.CircuitBreakerRule) *apiservice.Response {
	if err := utils.CheckDbRawStrFieldLen(
		req.RuleMatcher.GetSource().GetService(), MaxDbServiceNameLength); err != nil {
		return api.NewResponse(apimodel.Code_InvalidServiceName)
	}
	if err := utils.CheckDbRawStrFieldLen(
		req.RuleMatcher.GetSource().GetNamespace(), MaxDbServiceNamespaceLength); err != nil {
		return api.NewResponse(apimodel.Code_InvalidNamespaceName)
	}
	if err := utils.CheckDbRawStrFieldLen(req.GetName(), MaxRuleName); err != nil {
		return api.NewResponse(apimodel.Code_InvalidCircuitBreakerName)
	}
	if err := utils.CheckDbRawStrFieldLen(req.GetNamespace(), MaxDbServiceNamespaceLength); err != nil {
		return api.NewResponse(apimodel.Code_InvalidNamespaceName)
	}
	if err := utils.CheckDbRawStrFieldLen(req.GetDescription(), MaxCommentLength); err != nil {
		return api.NewResponse(apimodel.Code_InvalidServiceComment)
	}
	return nil
}

func circuitBreakerRuleRecordEntry(ctx context.Context, req *apifault.CircuitBreakerRule, md *model.CircuitBreakerRule,
	opt model.OperationType) *model.RecordEntry {
	marshaler := jsonpb.Marshaler{}
	detail, _ := marshaler.MarshalToString(req)
	entry := &model.RecordEntry{
		ResourceType:  model.RCircuitBreakerRule,
		ResourceName:  fmt.Sprintf("%s(%s)", md.Name, md.ID),
		Namespace:     req.GetNamespace(),
		OperationType: opt,
		Operator:      utils.ParseOperator(ctx),
		Detail:        detail,
		HappenTime:    time.Now(),
	}
	return entry
}

var (
	// CircuitBreakerRuleFilters filter circuitbreaker rule query parameters
	CircuitBreakerRuleFilters = map[string]bool{
		"brief":            true,
		"offset":           true,
		"limit":            true,
		"id":               true,
		"name":             true,
		"namespace":        true,
		"enable":           true,
		"level":            true,
		"service":          true,
		"serviceNamespace": true,
		"srcService":       true,
		"srcNamespace":     true,
		"dstService":       true,
		"dstNamespace":     true,
		"dstMethod":        true,
		"description":      true,
	}
)

// DeleteCircuitBreakerRules Delete current CircuitBreaker rules
func (s *Server) DeleteCircuitBreakerRules(
	ctx context.Context, request []*apifault.CircuitBreakerRule) *apiservice.BatchWriteResponse {
	responses := api.NewBatchWriteResponse(apimodel.Code_ExecuteSuccess)
	for _, entry := range request {
		resp := s.deleteCircuitBreakerRule(ctx, entry)
		api.Collect(responses, resp)
	}
	return api.FormatBatchWriteResponse(responses)
}

// deleteCircuitBreakerRule delete current CircuitBreaker rule
func (s *Server) deleteCircuitBreakerRule(
	ctx context.Context, request *apifault.CircuitBreakerRule) *apiservice.Response {
	requestID := utils.ParseRequestID(ctx)
	if resp := checkCircuitBreakerRuleParams(request, true, false); resp != nil {
		return resp
	}
	resp := s.checkCircuitBreakerRuleExists(request.GetId(), requestID)
	if resp != nil {
		if resp.GetCode().GetValue() == uint32(apimodel.Code_NotFoundCircuitBreaker) {
			resp.Code = &wrappers.UInt32Value{Value: uint32(apimodel.Code_ExecuteSuccess)}
		}
		return resp
	}
	cbRuleId := &apifault.CircuitBreakerRule{Id: request.GetId()}
	err := s.storage.DeleteCircuitBreakerRule(request.GetId())
	if err != nil {
		log.Error(err.Error(), utils.ZapRequestID(requestID))
		return api.NewAnyDataResponse(apimodel.Code_ParseCircuitBreakerException, cbRuleId)
	}
	msg := fmt.Sprintf("delete circuitbreaker rule: id=%v, name=%v, namespace=%v",
		request.GetId(), request.GetName(), request.GetNamespace())
	log.Info(msg, utils.ZapRequestID(requestID))

	cbRule := &model.CircuitBreakerRule{
		ID: request.GetId(), Name: request.GetName(), Namespace: request.GetNamespace()}
	s.RecordHistory(ctx, circuitBreakerRuleRecordEntry(ctx, request, cbRule, model.ODelete))
	return api.NewAnyDataResponse(apimodel.Code_ExecuteSuccess, cbRuleId)
}

// EnableCircuitBreakerRules Enable the CircuitBreaker rule
func (s *Server) EnableCircuitBreakerRules(
	ctx context.Context, request []*apifault.CircuitBreakerRule) *apiservice.BatchWriteResponse {
	responses := api.NewBatchWriteResponse(apimodel.Code_ExecuteSuccess)
	for _, entry := range request {
		resp := s.enableCircuitBreakerRule(ctx, entry)
		api.Collect(responses, resp)
	}
	return api.FormatBatchWriteResponse(responses)
}

func (s *Server) enableCircuitBreakerRule(
	ctx context.Context, request *apifault.CircuitBreakerRule) *apiservice.Response {
	requestID := utils.ParseRequestID(ctx)
	if resp := checkCircuitBreakerRuleParams(request, true, false); resp != nil {
		return resp
	}
	resp := s.checkCircuitBreakerRuleExists(request.GetId(), requestID)
	if resp != nil {
		return resp
	}
	cbRuleId := &apifault.CircuitBreakerRule{Id: request.GetId()}
	cbRule := &model.CircuitBreakerRule{
		ID:        request.GetId(),
		Namespace: request.GetNamespace(),
		Name:      request.GetName(),
		Enable:    request.GetEnable(),
		Revision:  utils.NewUUID(),
	}
	if err := s.storage.EnableCircuitBreakerRule(cbRule); err != nil {
		log.Error(err.Error(), utils.ZapRequestID(requestID))
		return storeError2AnyResponse(err, cbRuleId)
	}

	msg := fmt.Sprintf("enable circuitbreaker rule: id=%v, name=%v, namespace=%v",
		request.GetId(), request.GetName(), request.GetNamespace())
	log.Info(msg, utils.ZapRequestID(requestID))

	s.RecordHistory(ctx, circuitBreakerRuleRecordEntry(ctx, request, cbRule, model.OUpdate))
	return api.NewAnyDataResponse(apimodel.Code_ExecuteSuccess, cbRuleId)
}

// UpdateCircuitBreakerRules Modify the CircuitBreaker rule
func (s *Server) UpdateCircuitBreakerRules(
	ctx context.Context, request []*apifault.CircuitBreakerRule) *apiservice.BatchWriteResponse {
	responses := api.NewBatchWriteResponse(apimodel.Code_ExecuteSuccess)
	for _, entry := range request {
		response := s.updateCircuitBreakerRule(ctx, entry)
		api.Collect(responses, response)
	}
	return api.FormatBatchWriteResponse(responses)
}

func (s *Server) updateCircuitBreakerRule(
	ctx context.Context, request *apifault.CircuitBreakerRule) *apiservice.Response {
	requestID := utils.ParseRequestID(ctx)
	if resp := checkCircuitBreakerRuleParams(request, true, true); resp != nil {
		return resp
	}
	resp := s.checkCircuitBreakerRuleExists(request.GetId(), requestID)
	if resp != nil {
		return resp
	}
	cbRuleId := &apifault.CircuitBreakerRule{Id: request.GetId()}
	cbRule, err := api2CircuitBreakerRule(request)
	if err != nil {
		log.Error(err.Error(), utils.ZapRequestID(requestID))
		return api.NewAnyDataResponse(apimodel.Code_ParseCircuitBreakerException, cbRuleId)
	}
	cbRule.ID = request.GetId()
	exists, err := s.storage.HasCircuitBreakerRuleByNameExcludeId(cbRule.Name, cbRule.Namespace, cbRule.ID)
	if err != nil {
		log.Error(err.Error(), utils.ZapRequestID(requestID))
		return api.NewResponseWithMsg(commonstore.StoreCode2APICode(err), err.Error())
	}
	if exists {
		return api.NewResponse(apimodel.Code_ServiceExistedCircuitBreakers)
	}
	if err := s.storage.UpdateCircuitBreakerRule(cbRule); err != nil {
		log.Error(err.Error(), utils.ZapRequestID(requestID))
		return storeError2AnyResponse(err, cbRuleId)
	}

	msg := fmt.Sprintf("update circuitbreaker rule: id=%v, name=%v, namespace=%v",
		request.GetId(), request.GetName(), request.GetNamespace())
	log.Info(msg, utils.ZapRequestID(requestID))

	s.RecordHistory(ctx, circuitBreakerRuleRecordEntry(ctx, request, cbRule, model.OUpdate))
	return api.NewAnyDataResponse(apimodel.Code_ExecuteSuccess, cbRuleId)
}

func (s *Server) checkCircuitBreakerRuleExists(id, requestID string) *apiservice.Response {
	exists, err := s.storage.HasCircuitBreakerRule(id)
	if err != nil {
		log.Error(err.Error(), utils.ZapRequestID(requestID))
		return api.NewResponse(commonstore.StoreCode2APICode(err))
	}
	if !exists {
		return api.NewResponse(apimodel.Code_NotFoundCircuitBreaker)
	}
	return nil
}

// GetCircuitBreakerRules Query CircuitBreaker rules
func (s *Server) GetCircuitBreakerRules(ctx context.Context, query map[string]string) *apiservice.BatchQueryResponse {
	offset, limit, err := utils.ParseOffsetAndLimit(query)
	if err != nil {
		return api.NewBatchQueryResponse(apimodel.Code_InvalidParameter)
	}
	searchFilter := make(map[string]string, len(query))
	for key, value := range query {
		if _, ok := CircuitBreakerRuleFilters[key]; !ok {
			log.Errorf("params %s is not allowed in querying circuitbreaker rule", key)
			return api.NewBatchQueryResponse(apimodel.Code_InvalidParameter)
		}
		if value == "" {
			continue
		}
		searchFilter[key] = value
	}
	total, cbRules, err := s.storage.GetCircuitBreakerRules(searchFilter, offset, limit)
	if err != nil {
		log.Errorf("get circuitbreaker rules store err: %s", err.Error())
		return api.NewBatchQueryResponse(commonstore.StoreCode2APICode(err))
	}
	out := api.NewBatchQueryResponse(apimodel.Code_ExecuteSuccess)
	out.Amount = utils.NewUInt32Value(total)
	out.Size = utils.NewUInt32Value(uint32(len(cbRules)))
	for _, cbRule := range cbRules {
		cbRuleProto, err := circuitBreakerRule2api(cbRule)
		if nil != err {
			log.Errorf("marshal circuitbreaker rule fail: %v", err)
			continue
		}
		if nil == cbRuleProto {
			continue
		}
		err = api.AddAnyDataIntoBatchQuery(out, cbRuleProto)
		if nil != err {
			log.Errorf("add circuitbreaker rule as any data fail: %v", err)
			continue
		}
	}
	return out
}

func marshalCircuitBreakerRuleV2(req *apifault.CircuitBreakerRule) (string, error) {
	r := &apifault.CircuitBreakerRule{
		RuleMatcher:        req.RuleMatcher,
		ErrorConditions:    req.ErrorConditions,
		TriggerCondition:   req.TriggerCondition,
		MaxEjectionPercent: req.MaxEjectionPercent,
		RecoverCondition:   req.RecoverCondition,
		FaultDetectConfig:  req.FaultDetectConfig,
		FallbackConfig:     req.FallbackConfig,
	}
	rule, err := json.Marshal(r)
	if err != nil {
		return "", err
	}
	return string(rule), nil
}

// api2CircuitBreakerRule 把API参数转化为内部数据结构
func api2CircuitBreakerRule(req *apifault.CircuitBreakerRule) (*model.CircuitBreakerRule, error) {
	rule, err := marshalCircuitBreakerRuleV2(req)
	if err != nil {
		return nil, err
	}

	out := &model.CircuitBreakerRule{
		Name:         req.GetName(),
		Namespace:    req.GetNamespace(),
		Description:  req.GetDescription(),
		Level:        int(req.GetLevel()),
		SrcService:   req.GetRuleMatcher().GetSource().GetService(),
		SrcNamespace: req.GetRuleMatcher().GetSource().GetNamespace(),
		DstService:   req.GetRuleMatcher().GetDestination().GetService(),
		DstNamespace: req.GetRuleMatcher().GetDestination().GetNamespace(),
		DstMethod:    req.GetRuleMatcher().GetDestination().GetMethod().GetValue().GetValue(),
		Enable:       req.GetEnable(),
		Rule:         rule,
		Revision:     utils.NewUUID(),
	}
	if out.Namespace == "" {
		out.Namespace = DefaultNamespace
	}
	return out, nil
}

func circuitBreakerRule2api(cbRule *model.CircuitBreakerRule) (*apifault.CircuitBreakerRule, error) {
	if cbRule == nil {
		return nil, nil
	}
	cbRule.Proto = &apifault.CircuitBreakerRule{}
	if len(cbRule.Rule) > 0 {
		if err := json.Unmarshal([]byte(cbRule.Rule), cbRule.Proto); err != nil {
			return nil, err
		}
	} else {
		// brief search, to display the services in list result
		cbRule.Proto.RuleMatcher = &apifault.RuleMatcher{
			Source: &apifault.RuleMatcher_SourceService{
				Service:   cbRule.SrcService,
				Namespace: cbRule.SrcNamespace,
			},
			Destination: &apifault.RuleMatcher_DestinationService{
				Service:   cbRule.DstService,
				Namespace: cbRule.DstNamespace,
				Method:    &apimodel.MatchString{Value: &wrappers.StringValue{Value: cbRule.DstMethod}},
			},
		}
	}
	cbRule.Proto.Id = cbRule.ID
	cbRule.Proto.Name = cbRule.Name
	cbRule.Proto.Namespace = cbRule.Namespace
	cbRule.Proto.Description = cbRule.Description
	cbRule.Proto.Level = apifault.Level(cbRule.Level)
	cbRule.Proto.Enable = cbRule.Enable
	cbRule.Proto.Revision = cbRule.Revision
	cbRule.Proto.Ctime = commontime.Time2String(cbRule.CreateTime)
	cbRule.Proto.Mtime = commontime.Time2String(cbRule.ModifyTime)
	cbRule.Proto.Enable = cbRule.Enable
	if cbRule.EnableTime.Year() > 2000 {
		cbRule.Proto.Etime = commontime.Time2String(cbRule.EnableTime)
	} else {
		cbRule.Proto.Etime = ""
	}
	return cbRule.Proto, nil
}

// circuitBreaker2ClientAPI 把内部数据结构转化为客户端API参数
func circuitBreaker2ClientAPI(
	req *model.ServiceWithCircuitBreakerRules, service string, namespace string) (*apifault.CircuitBreaker, error) {
	if req == nil {
		return nil, nil
	}

	out := &apifault.CircuitBreaker{}
	out.Revision = &wrappers.StringValue{Value: req.Revision}
	out.Rules = make([]*apifault.CircuitBreakerRule, 0, req.CountCircuitBreakerRules())
	var iterateErr error
	req.IterateCircuitBreakerRules(func(rule *model.CircuitBreakerRule) {
		cbRule, err := circuitBreakerRule2api(rule)
		if err != nil {
			iterateErr = err
			return
		}
		out.Rules = append(out.Rules, cbRule)
	})
	if nil != iterateErr {
		return nil, iterateErr
	}

	out.Service = utils.NewStringValue(service)
	out.ServiceNamespace = utils.NewStringValue(namespace)

	return out, nil
}
