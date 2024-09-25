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
	"go.uber.org/zap"

	cachetypes "github.com/polarismesh/polaris/cache/api"
	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/model"
	commonstore "github.com/polarismesh/polaris/common/store"
	commontime "github.com/polarismesh/polaris/common/time"
	"github.com/polarismesh/polaris/common/utils"
)

// CreateFaultDetectRules Create a FaultDetect rule
func (s *Server) CreateFaultDetectRules(
	ctx context.Context, request []*apifault.FaultDetectRule) *apiservice.BatchWriteResponse {
	responses := api.NewBatchWriteResponse(apimodel.Code_ExecuteSuccess)
	for _, cbRule := range request {
		response := s.createFaultDetectRule(ctx, cbRule)
		api.Collect(responses, response)
	}
	return api.FormatBatchWriteResponse(responses)
}

// DeleteFaultDetectRules Delete current Fault Detect rules
func (s *Server) DeleteFaultDetectRules(
	ctx context.Context, request []*apifault.FaultDetectRule) *apiservice.BatchWriteResponse {

	responses := api.NewBatchWriteResponse(apimodel.Code_ExecuteSuccess)
	for _, cbRule := range request {
		response := s.deleteFaultDetectRule(ctx, cbRule)
		api.Collect(responses, response)
	}
	return api.FormatBatchWriteResponse(responses)
}

// UpdateFaultDetectRules Modify the FaultDetect rule
func (s *Server) UpdateFaultDetectRules(
	ctx context.Context, request []*apifault.FaultDetectRule) *apiservice.BatchWriteResponse {

	responses := api.NewBatchWriteResponse(apimodel.Code_ExecuteSuccess)
	for _, cbRule := range request {
		response := s.updateFaultDetectRule(ctx, cbRule)
		api.Collect(responses, response)
	}
	return api.FormatBatchWriteResponse(responses)
}

func faultDetectRuleRecordEntry(ctx context.Context, req *apifault.FaultDetectRule, md *model.FaultDetectRule,
	opt model.OperationType) *model.RecordEntry {
	marshaler := jsonpb.Marshaler{}
	detail, _ := marshaler.MarshalToString(req)
	entry := &model.RecordEntry{
		ResourceType:  model.RFaultDetectRule,
		ResourceName:  fmt.Sprintf("%s(%s)", md.Name, md.ID),
		Namespace:     req.GetNamespace(),
		OperationType: opt,
		Operator:      utils.ParseOperator(ctx),
		Detail:        detail,
		HappenTime:    time.Now(),
	}
	return entry
}

// createFaultDetectRule Create a FaultDetect rule
func (s *Server) createFaultDetectRule(ctx context.Context, request *apifault.FaultDetectRule) *apiservice.Response {
	data, err := api2FaultDetectRule(request)
	if err != nil {
		log.Error(err.Error(), utils.RequestID(ctx))
		return api.NewResponse(apimodel.Code_ParseException)
	}
	exists, err := s.storage.HasFaultDetectRuleByName(data.Name, data.Namespace)
	if err != nil {
		log.Error(err.Error(), utils.RequestID(ctx))
		return api.NewResponseWithMsg(commonstore.StoreCode2APICode(err), err.Error())
	}
	if exists {
		return api.NewResponse(apimodel.Code_FaultDetectRuleExisted)
	}
	data.ID = utils.NewUUID()

	// 存储层操作
	if err := s.storage.CreateFaultDetectRule(data); err != nil {
		log.Error(err.Error(), utils.RequestID(ctx))
		return api.NewResponseWithMsg(commonstore.StoreCode2APICode(err), err.Error())
	}

	msg := fmt.Sprintf("create fault detect rule: id=%v, name=%v, namespace=%v",
		data.ID, request.GetName(), request.GetNamespace())
	log.Info(msg, utils.RequestID(ctx))

	s.RecordHistory(ctx, faultDetectRuleRecordEntry(ctx, request, data, model.OCreate))

	request.Id = data.ID
	return api.NewAnyDataResponse(apimodel.Code_ExecuteSuccess, request)
}

// updateFaultDetectRule Update a FaultDetect rule
func (s *Server) updateFaultDetectRule(ctx context.Context, request *apifault.FaultDetectRule) *apiservice.Response {
	fdRuleId := &apifault.FaultDetectRule{Id: request.GetId()}
	fdRule, err := api2FaultDetectRule(request)
	if err != nil {
		log.Error(err.Error(), utils.RequestID(ctx))
		return api.NewAnyDataResponse(apimodel.Code_ParseException, fdRuleId)
	}
	fdRule.ID = request.GetId()
	exists, err := s.storage.HasFaultDetectRuleByNameExcludeId(fdRule.Name, fdRule.Namespace, fdRule.ID)
	if err != nil {
		log.Error(err.Error(), utils.RequestID(ctx))
		return api.NewResponseWithMsg(commonstore.StoreCode2APICode(err), err.Error())
	}
	if exists {
		return api.NewAnyDataResponse(apimodel.Code_FaultDetectRuleExisted, fdRuleId)
	}
	if err := s.storage.UpdateFaultDetectRule(fdRule); err != nil {
		log.Error(err.Error(), utils.RequestID(ctx))
		return storeError2AnyResponse(err, fdRuleId)
	}

	msg := fmt.Sprintf("update fault detect rule: id=%v, name=%v, namespace=%v",
		request.GetId(), request.GetName(), request.GetNamespace())
	log.Info(msg, utils.RequestID(ctx))

	s.RecordHistory(ctx, faultDetectRuleRecordEntry(ctx, request, fdRule, model.OUpdate))
	return api.NewAnyDataResponse(apimodel.Code_ExecuteSuccess, fdRuleId)
}

// deleteFaultDetectRule Delete a FaultDetect rule
func (s *Server) deleteFaultDetectRule(ctx context.Context, request *apifault.FaultDetectRule) *apiservice.Response {
	requestID := utils.ParseRequestID(ctx)
	resp := s.checkFaultDetectRuleExists(request.GetId(), requestID)
	if resp != nil {
		if resp.GetCode().GetValue() == uint32(apimodel.Code_NotFoundResource) {
			resp.Code = &wrappers.UInt32Value{Value: uint32(apimodel.Code_ExecuteSuccess)}
		}
		return resp
	}
	cbRuleId := &apifault.FaultDetectRule{Id: request.GetId()}
	err := s.storage.DeleteFaultDetectRule(request.GetId())
	if err != nil {
		log.Error(err.Error(), utils.ZapRequestID(requestID))
		return api.NewAnyDataResponse(apimodel.Code_ParseException, cbRuleId)
	}
	msg := fmt.Sprintf("delete fault detect rule: id=%v, name=%v, namespace=%v",
		request.GetId(), request.GetName(), request.GetNamespace())
	log.Info(msg, utils.ZapRequestID(requestID))

	cbRule := &model.FaultDetectRule{ID: request.GetId(), Name: request.GetName(), Namespace: request.GetNamespace()}
	s.RecordHistory(ctx, faultDetectRuleRecordEntry(ctx, request, cbRule, model.ODelete))
	return api.NewAnyDataResponse(apimodel.Code_ExecuteSuccess, cbRuleId)
}

func (s *Server) checkFaultDetectRuleExists(id, requestID string) *apiservice.Response {
	exists, err := s.storage.HasFaultDetectRule(id)
	if err != nil {
		log.Error(err.Error(), utils.ZapRequestID(requestID))
		return api.NewResponse(commonstore.StoreCode2APICode(err))
	}
	if !exists {
		return api.NewResponse(apimodel.Code_NotFoundResource)
	}
	return nil
}

func (s *Server) GetFaultDetectRules(ctx context.Context, query map[string]string) *apiservice.BatchQueryResponse {
	offset, limit, _ := utils.ParseOffsetAndLimit(query)
	total, cbRules, err := s.caches.FaultDetector().Query(ctx, &cachetypes.FaultDetectArgs{
		ID:               query["id"],
		Name:             query["name"],
		Namespace:        query["namespace"],
		Service:          query["service"],
		ServiceNamespace: query["serviceNamespace"],
		DstNamespace:     query["dstNamespace"],
		DstService:       query["dstService"],
		DstMethod:        query["dstMethod"],
		Offset:           offset,
		Limit:            limit,
	})
	if err != nil {
		log.Errorf("get fault detect rules store err: %s", err.Error())
		return api.NewBatchQueryResponse(commonstore.StoreCode2APICode(err))
	}
	out := api.NewBatchQueryResponse(apimodel.Code_ExecuteSuccess)
	out.Amount = utils.NewUInt32Value(total)
	out.Size = utils.NewUInt32Value(uint32(len(cbRules)))
	for _, cbRule := range cbRules {
		cbRuleProto, err := faultDetectRule2api(cbRule)
		if nil != err {
			log.Error("marshal circuitbreaker rule fail", utils.RequestID(ctx), zap.Error(err))
			continue
		}
		if nil == cbRuleProto {
			continue
		}
		err = api.AddAnyDataIntoBatchQuery(out, cbRuleProto)
		if nil != err {
			log.Error("add circuitbreaker rule as any data fail", utils.RequestID(ctx), zap.Error(err))
			continue
		}
	}
	return out
}

func marshalFaultDetectRule(req *apifault.FaultDetectRule) (string, error) {
	r := &apifault.FaultDetectRule{
		TargetService: req.TargetService,
		Interval:      req.Interval,
		Timeout:       req.Timeout,
		Port:          req.Port,
		Protocol:      req.Protocol,
		HttpConfig:    req.HttpConfig,
		TcpConfig:     req.TcpConfig,
		UdpConfig:     req.UdpConfig,
	}
	rule, err := json.Marshal(r)
	if err != nil {
		return "", err
	}
	return string(rule), nil
}

// api2FaultDetectRule 把API参数转化为内部数据结构
func api2FaultDetectRule(req *apifault.FaultDetectRule) (*model.FaultDetectRule, error) {
	rule, err := marshalFaultDetectRule(req)
	if err != nil {
		return nil, err
	}

	out := &model.FaultDetectRule{
		Name:         req.GetName(),
		Namespace:    req.GetNamespace(),
		Description:  req.GetDescription(),
		DstService:   req.GetTargetService().GetService(),
		DstNamespace: req.GetTargetService().GetNamespace(),
		DstMethod:    req.GetTargetService().GetMethod().GetValue().GetValue(),
		Rule:         rule,
		Revision:     utils.NewUUID(),
	}
	if out.Namespace == "" {
		out.Namespace = DefaultNamespace
	}
	return out, nil
}

func faultDetectRule2api(fdRule *model.FaultDetectRule) (*apifault.FaultDetectRule, error) {
	if fdRule == nil {
		return nil, nil
	}
	fdRule.Proto = &apifault.FaultDetectRule{}
	if len(fdRule.Rule) > 0 {
		if err := json.Unmarshal([]byte(fdRule.Rule), fdRule.Proto); err != nil {
			return nil, err
		}
	} else {
		// brief search, to display the services in list result
		fdRule.Proto.TargetService = &apifault.FaultDetectRule_DestinationService{
			Service:   fdRule.DstService,
			Namespace: fdRule.DstNamespace,
			Method:    &apimodel.MatchString{Value: &wrappers.StringValue{Value: fdRule.DstMethod}},
		}
	}
	fdRule.Proto.Id = fdRule.ID
	fdRule.Proto.Name = fdRule.Name
	fdRule.Proto.Namespace = fdRule.Namespace
	fdRule.Proto.Description = fdRule.Description
	fdRule.Proto.Revision = fdRule.Revision
	fdRule.Proto.Ctime = commontime.Time2String(fdRule.CreateTime)
	fdRule.Proto.Mtime = commontime.Time2String(fdRule.ModifyTime)
	return fdRule.Proto, nil
}

// faultDetectRule2ClientAPI 把内部数据结构转化为客户端API参数
func faultDetectRule2ClientAPI(req *model.ServiceWithFaultDetectRules) (*apifault.FaultDetector, error) {
	if req == nil {
		return nil, nil
	}

	out := &apifault.FaultDetector{}
	out.Revision = req.Revision
	out.Rules = make([]*apifault.FaultDetectRule, 0, req.CountFaultDetectRules())
	var iterateErr error
	req.IterateFaultDetectRules(func(rule *model.FaultDetectRule) {
		cbRule, err := faultDetectRule2api(rule)
		if err != nil {
			iterateErr = err
			return
		}
		out.Rules = append(out.Rules, cbRule)
	})
	if nil != iterateErr {
		return nil, iterateErr
	}
	return out, nil
}
