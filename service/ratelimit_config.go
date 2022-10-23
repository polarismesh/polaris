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

	"github.com/golang/protobuf/ptypes"

	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/model"
	commontime "github.com/polarismesh/polaris/common/time"
	"github.com/polarismesh/polaris/common/utils"
)

var (
	// RateLimitFilters rate limit filters
	RateLimitFilters = map[string]bool{
		"id":        true,
		"name":      true,
		"service":   true,
		"namespace": true,
		"brief":     true,
		"method":    true,
		"labels":    true,
		"disable":   true,
		"offset":    true,
		"limit":     true,
	}
)

// CreateRateLimits 批量创建限流规则
func (s *Server) CreateRateLimits(ctx context.Context, request []*api.Rule) *api.BatchWriteResponse {
	if err := checkBatchRateLimits(request); err != nil {
		return err
	}

	responses := api.NewBatchWriteResponse(api.ExecuteSuccess)
	for _, rateLimit := range request {
		var response *api.Response
		// create service if absent
		code, svcId, err := s.createServiceIfAbsent(ctx, rateLimit)

		if err != nil {
			log.Errorf("[Service]fail to create ratelimit config, err: %v", err)
			response = api.NewRateLimitResponse(code, rateLimit)
		} else {
			response = s.CreateRateLimit(ctx, rateLimit, svcId)
		}
		responses.Collect(response)
	}
	return api.FormatBatchWriteResponse(responses)
}

// CreateRateLimit 创建限流规则
func (s *Server) CreateRateLimit(ctx context.Context, req *api.Rule, svcId string) *api.Response {
	requestID := utils.ParseRequestID(ctx)
	platformID := utils.ParsePlatformID(ctx)

	// 参数校验
	if resp := checkRateLimitParams(req); resp != nil {
		return resp
	}

	if resp := checkRateLimitRuleParams(requestID, req); resp != nil {
		return resp
	}

	tx, err := s.storage.CreateTransaction()
	if err != nil {
		log.Error(err.Error(), utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
		return api.NewRateLimitResponse(api.StoreLayerException, req)
	}
	defer func() {
		_ = tx.Commit()
	}()

	// 构造底层数据结构
	data, err := api2RateLimit(svcId, req, nil)
	if err != nil {
		log.Error(err.Error(), utils.ZapRequestID(requestID))
		return api.NewRateLimitResponse(api.ParseRateLimitException, req)
	}
	data.ID = utils.NewUUID()

	// 存储层操作
	if err := s.storage.CreateRateLimit(data); err != nil {
		log.Error(err.Error(), utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
		return wrapperRateLimitStoreResponse(req, err)
	}

	msg := fmt.Sprintf("create rate limit rule: id=%v, namespace=%v, service=%v, name=%v",
		data.ID, req.GetNamespace().GetValue(), req.GetService().GetValue(), req.GetName().GetValue())
	log.Info(msg, utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))

	s.RecordHistory(rateLimitRecordEntry(ctx, req.GetNamespace().GetValue(), req.GetService().GetValue(),
		data, model.OCreate))

	req.Id = utils.NewStringValue(data.ID)
	return api.NewRateLimitResponse(api.ExecuteSuccess, req)
}

// DeleteRateLimits 批量删除限流规则
func (s *Server) DeleteRateLimits(ctx context.Context, request []*api.Rule) *api.BatchWriteResponse {
	if err := checkBatchRateLimits(request); err != nil {
		return err
	}

	responses := api.NewBatchWriteResponse(api.ExecuteSuccess)
	for _, entry := range request {
		resp := s.DeleteRateLimit(ctx, entry)
		responses.Collect(resp)
	}
	return api.FormatBatchWriteResponse(responses)
}

// DeleteRateLimit 删除单个限流规则
func (s *Server) DeleteRateLimit(ctx context.Context, req *api.Rule) *api.Response {
	requestID := utils.ParseRequestID(ctx)
	platformID := utils.ParsePlatformID(ctx)

	// 参数校验
	if resp := checkRevisedRateLimitParams(req); resp != nil {
		return resp
	}

	// 检查限流规则是否存在
	rateLimit, resp := s.checkRateLimitExisted(req.GetId().GetValue(), requestID, req)
	if resp != nil {
		if resp.GetCode().GetValue() == api.NotFoundRateLimit {
			return api.NewRateLimitResponse(api.ExecuteSuccess, req)
		}
		return resp
	}

	// 生成新的revision
	rateLimit.Revision = utils.NewUUID()

	// 存储层操作
	if err := s.storage.DeleteRateLimit(rateLimit); err != nil {
		log.Error(err.Error(), utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
		return wrapperRateLimitStoreResponse(req, err)
	}

	msg := fmt.Sprintf("delete rate limit rule: id=%v, namespace=%v, service=%v, name=%v",
		rateLimit.ID, req.GetNamespace().GetValue(), req.GetService().GetValue(), rateLimit.Labels)
	log.Info(msg, utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))

	s.RecordHistory(
		rateLimitRecordEntry(ctx, req.GetNamespace().GetValue(), req.GetService().GetValue(), rateLimit, model.ODelete))
	return api.NewRateLimitResponse(api.ExecuteSuccess, req)
}

func (s *Server) EnableRateLimits(ctx context.Context, request []*api.Rule) *api.BatchWriteResponse {
	if err := checkBatchRateLimits(request); err != nil {
		return err
	}
	responses := api.NewBatchWriteResponse(api.ExecuteSuccess)
	for _, entry := range request {
		response := s.EnableRateLimit(ctx, entry)
		responses.Collect(response)
	}
	return api.FormatBatchWriteResponse(responses)
}

// EnableRateLimit 启用限流规则
func (s *Server) EnableRateLimit(ctx context.Context, req *api.Rule) *api.Response {
	requestID := utils.ParseRequestID(ctx)
	platformID := utils.ParsePlatformID(ctx)

	// 参数校验
	if resp := checkRevisedRateLimitParams(req); resp != nil {
		return resp
	}

	// 检查限流规则是否存在
	data, resp := s.checkRateLimitExisted(req.GetId().GetValue(), requestID, req)
	if resp != nil {
		return resp
	}

	// 构造底层数据结构
	rateLimit := &model.RateLimit{}
	rateLimit.ID = data.ID
	rateLimit.ServiceID = data.ServiceID
	rateLimit.Disable = req.GetDisable().GetValue()
	rateLimit.Revision = utils.NewUUID()

	if err := s.storage.EnableRateLimit(rateLimit); err != nil {
		log.Error(err.Error(), utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
		return wrapperRateLimitStoreResponse(req, err)
	}

	msg := fmt.Sprintf("enable rate limit: id=%v, disable=%v",
		rateLimit.ID, rateLimit.Disable)
	log.Info(msg, utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))

	s.RecordHistory(rateLimitRecordEntry(ctx, "", "", rateLimit, model.OUpdate))
	return api.NewRateLimitResponse(api.ExecuteSuccess, req)
}

// UpdateRateLimits 批量更新限流规则
func (s *Server) UpdateRateLimits(ctx context.Context, request []*api.Rule) *api.BatchWriteResponse {
	if err := checkBatchRateLimits(request); err != nil {
		return err
	}

	responses := api.NewBatchWriteResponse(api.ExecuteSuccess)
	for _, entry := range request {
		response := s.UpdateRateLimit(ctx, entry)
		responses.Collect(response)
	}
	return api.FormatBatchWriteResponse(responses)
}

// UpdateRateLimit 更新限流规则
func (s *Server) UpdateRateLimit(ctx context.Context, req *api.Rule) *api.Response {
	requestID := utils.ParseRequestID(ctx)
	platformID := utils.ParsePlatformID(ctx)

	// 参数校验
	if resp := checkRevisedRateLimitParams(req); resp != nil {
		return resp
	}

	if resp := checkRateLimitRuleParams(requestID, req); resp != nil {
		return resp
	}

	if resp := checkRateLimitParamsDbLen(req); resp != nil {
		return resp
	}

	// 检查限流规则是否存在
	data, resp := s.checkRateLimitExisted(req.GetId().GetValue(), requestID, req)
	if resp != nil {
		return resp
	}
	// create service if absent
	code, svcId, err := s.createServiceIfAbsent(ctx, req)
	if err != nil {
		log.Errorf("[Service]fail to create ratelimit config, err: %v", err)
		return api.NewRateLimitResponse(code, req)
	}
	// 构造底层数据结构
	rateLimit, err := api2RateLimit(svcId, req, data)
	if err != nil {
		log.Error(err.Error(), utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
		return api.NewRateLimitResponse(api.ParseRateLimitException, req)
	}
	rateLimit.ID = data.ID
	if err := s.storage.UpdateRateLimit(rateLimit); err != nil {
		log.Error(err.Error(), utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
		return wrapperRateLimitStoreResponse(req, err)
	}

	msg := fmt.Sprintf("update rate limit: id=%v, namespace=%v, service=%v, name=%v",
		rateLimit.ID, req.GetNamespace().GetValue(), req.GetService().GetValue(), rateLimit.Name)
	log.Info(msg, utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))

	s.RecordHistory(rateLimitRecordEntry(ctx, req.GetNamespace().GetValue(), req.GetService().GetValue(), rateLimit, model.OUpdate))
	return api.NewRateLimitResponse(api.ExecuteSuccess, req)
}

// GetRateLimits 查询限流规则
func (s *Server) GetRateLimits(ctx context.Context, query map[string]string) *api.BatchQueryResponse {
	for key := range query {
		if _, ok := RateLimitFilters[key]; !ok {
			log.Errorf("params %s is not allowed in querying rate limits", key)
			return api.NewBatchQueryResponse(api.InvalidParameter)
		}
	}
	// 处理offset和limit
	offset, limit, err := utils.ParseOffsetAndLimit(query)
	if err != nil {
		return api.NewBatchQueryResponse(api.InvalidParameter)
	}

	total, extendRateLimits, err := s.storage.GetExtendRateLimits(query, offset, limit)
	if err != nil {
		log.Errorf("get rate limits store err: %s", err.Error())
		return api.NewBatchQueryResponse(api.StoreLayerException)
	}

	out := api.NewBatchQueryResponse(api.ExecuteSuccess)
	out.Amount = utils.NewUInt32Value(total)
	out.Size = utils.NewUInt32Value(uint32(len(extendRateLimits)))
	out.RateLimits = make([]*api.Rule, 0, len(extendRateLimits))
	for _, item := range extendRateLimits {
		limit, err := rateLimit2Console(item.ServiceName, item.NamespaceName, item.RateLimit)
		if err != nil {
			log.Errorf("get rate limits convert err: %s", err.Error())
			return api.NewBatchQueryResponse(api.ParseRateLimitException)
		}
		out.RateLimits = append(out.RateLimits, limit)
	}

	return out
}

// checkBatchRateLimits 检查批量请求的限流规则
func checkBatchRateLimits(req []*api.Rule) *api.BatchWriteResponse {
	if len(req) == 0 {
		return api.NewBatchWriteResponse(api.EmptyRequest)
	}

	if len(req) > MaxBatchSize {
		return api.NewBatchWriteResponse(api.BatchSizeOverLimit)
	}

	return nil
}

// checkRateLimitValid 检查限流规则是否允许修改/删除
func (s *Server) checkRateLimitValid(ctx context.Context, serviceID string, req *api.Rule) (
	*model.Service, *api.Response) {
	requestID := utils.ParseRequestID(ctx)

	service, err := s.storage.GetServiceByID(serviceID)
	if err != nil {
		log.Error(err.Error(), utils.ZapRequestID(requestID))
		return nil, api.NewRateLimitResponse(api.StoreLayerException, req)
	}

	return service, nil
}

// checkRateLimitParams 检查限流规则基础参数
func checkRateLimitParams(req *api.Rule) *api.Response {
	if req == nil {
		return api.NewRateLimitResponse(api.EmptyRequest, req)
	}
	if err := checkResourceName(req.GetNamespace()); err != nil {
		return api.NewRateLimitResponse(api.InvalidNamespaceName, req)
	}
	if err := checkResourceName(req.GetService()); err != nil {
		return api.NewRateLimitResponse(api.InvalidServiceName, req)
	}
	if resp := checkRateLimitParamsDbLen(req); nil != resp {
		return resp
	}
	return nil
}

// checkRateLimitParams 检查限流规则基础参数
func checkRateLimitParamsDbLen(req *api.Rule) *api.Response {
	if err := utils.CheckDbStrFieldLen(req.GetService(), MaxDbServiceNameLength); err != nil {
		return api.NewRateLimitResponse(api.InvalidServiceName, req)
	}
	if err := utils.CheckDbStrFieldLen(req.GetNamespace(), MaxDbServiceNamespaceLength); err != nil {
		return api.NewRateLimitResponse(api.InvalidNamespaceName, req)
	}
	if err := utils.CheckDbStrFieldLen(req.GetName(), MaxDbRateLimitName); err != nil {
		return api.NewRateLimitResponse(api.InvalidRateLimitName, req)
	}
	return nil
}

// checkRateLimitRuleParams 检查限流规则其他参数
func checkRateLimitRuleParams(requestID string, req *api.Rule) *api.Response {
	// 检查amounts是否有重复周期
	amounts := req.GetAmounts()
	durations := make(map[time.Duration]bool)
	for _, amount := range amounts {
		d := amount.GetValidDuration()
		duration, err := ptypes.Duration(d)
		if err != nil {
			log.Error(err.Error(), utils.ZapRequestID(requestID))
			return api.NewRateLimitResponse(api.InvalidRateLimitAmounts, req)
		}
		durations[duration] = true
	}
	if len(amounts) != len(durations) {
		return api.NewRateLimitResponse(api.InvalidRateLimitAmounts, req)
	}
	return nil
}

// checkRevisedRateLimitParams 检查修改/删除限流规则基础参数
func checkRevisedRateLimitParams(req *api.Rule) *api.Response {
	if req == nil {
		return api.NewRateLimitResponse(api.EmptyRequest, req)
	}
	if req.GetId().GetValue() == "" {
		return api.NewRateLimitResponse(api.InvalidRateLimitID, req)
	}
	return nil
}

// checkRateLimitExisted 检查限流规则是否存在
func (s *Server) checkRateLimitExisted(id, requestID string, req *api.Rule) (*model.RateLimit, *api.Response) {
	rateLimit, err := s.storage.GetRateLimitWithID(id)
	if err != nil {
		log.Error(err.Error(), utils.ZapRequestID(requestID))
		return nil, api.NewRateLimitResponse(api.StoreLayerException, req)
	}
	if rateLimit == nil {
		return nil, api.NewRateLimitResponse(api.NotFoundRateLimit, req)
	}
	return rateLimit, nil
}

const (
	defaultRuleAction = "REJECT"
)

// api2RateLimit 把API参数转化为内部数据结构
func api2RateLimit(serviceID string, req *api.Rule, old *model.RateLimit) (*model.RateLimit, error) {
	rule, err := marshalRateLimitRules(req)
	if err != nil {
		return nil, err
	}

	labels := req.GetLabels()
	var labelStr []byte
	if len(labels) > 0 {
		labelStr, err = json.Marshal(labels)
	}

	out := &model.RateLimit{
		Name:      req.GetName().GetValue(),
		ServiceID: serviceID,
		Method:    req.GetMethod().GetValue().GetValue(),
		Disable:   req.GetDisable().GetValue(),
		Priority:  req.GetPriority().GetValue(),
		Labels:    string(labelStr),
		Rule:      rule,
		Revision:  utils.NewUUID(),
	}
	return out, nil
}

// rateLimit2api 把内部数据结构转化为API参数
func rateLimit2Console(
	service string, namespace string, rateLimit *model.RateLimit) (*api.Rule, error) {
	if rateLimit == nil {
		return nil, nil
	}
	if len(rateLimit.Rule) > 0 {
		rateLimit.Proto = &api.Rule{}
		// 控制台查询的请求
		if err := json.Unmarshal([]byte(rateLimit.Rule), rateLimit.Proto); err != nil {
			return nil, err
		}
		// 存量标签适配到参数列表
		if err := rateLimit.AdaptLabels(); err != nil {
			return nil, err
		}
	}
	rule := &api.Rule{}
	rule.Id = utils.NewStringValue(rateLimit.ID)
	rule.Name = utils.NewStringValue(rateLimit.Name)
	rule.Service = utils.NewStringValue(service)
	rule.Namespace = utils.NewStringValue(namespace)
	rule.Priority = utils.NewUInt32Value(rateLimit.Priority)
	rule.Ctime = utils.NewStringValue(commontime.Time2String(rateLimit.CreateTime))
	rule.Mtime = utils.NewStringValue(commontime.Time2String(rateLimit.ModifyTime))
	rule.Disable = utils.NewBoolValue(rateLimit.Disable)
	if rateLimit.EnableTime.Year() > 2000 {
		rule.Etime = utils.NewStringValue(commontime.Time2String(rateLimit.EnableTime))
	} else {
		rule.Etime = utils.NewStringValue("")
	}
	rule.Revision = utils.NewStringValue(rateLimit.Revision)
	if nil != rateLimit.Proto {
		copyRateLimitProto(rateLimit, rule)
	} else {
		rule.Method = &api.MatchString{Value: utils.NewStringValue(rateLimit.Method)}
	}
	return rule, nil
}

func populateDefaultRuleValue(rule *api.Rule) {
	if rule.GetAction().GetValue() == "" {
		rule.Action = utils.NewStringValue(defaultRuleAction)
	}
}

func copyRateLimitProto(rateLimit *model.RateLimit, rule *api.Rule) {
	// copy proto values
	rule.Method = rateLimit.Proto.Method
	rule.Arguments = rateLimit.Proto.Arguments
	rule.Labels = rateLimit.Proto.Labels
	rule.Resource = rateLimit.Proto.Resource
	rule.Type = rateLimit.Proto.Type
	rule.Amounts = rateLimit.Proto.Amounts
	rule.RegexCombine = rateLimit.Proto.RegexCombine
	rule.Action = rateLimit.Proto.Action
	rule.Failover = rateLimit.Proto.Failover
	rule.AmountMode = rateLimit.Proto.AmountMode
	rule.Adjuster = rateLimit.Proto.Adjuster
	rule.MaxQueueDelay = rateLimit.Proto.MaxQueueDelay
	populateDefaultRuleValue(rule)
}

// rateLimit2api 把内部数据结构转化为API参数
func rateLimit2Client(
	service string, namespace string, rateLimit *model.RateLimit) (*api.Rule, error) {
	if rateLimit == nil {
		return nil, nil
	}

	rule := &api.Rule{}
	rule.Id = utils.NewStringValue(rateLimit.ID)
	rule.Service = utils.NewStringValue(service)
	rule.Namespace = utils.NewStringValue(namespace)
	rule.Priority = utils.NewUInt32Value(rateLimit.Priority)
	rule.Revision = utils.NewStringValue(rateLimit.Revision)
	copyRateLimitProto(rateLimit, rule)
	return rule, nil
}

// marshalRateLimitRules 序列化限流规则具体内容
func marshalRateLimitRules(req *api.Rule) (string, error) {
	r := &api.Rule{
		Name:          req.GetName(),
		Resource:      req.GetResource(),
		Type:          req.GetType(),
		Amounts:       req.GetAmounts(),
		Action:        req.GetAction(),
		Disable:       req.GetDisable(),
		Report:        req.GetReport(),
		Adjuster:      req.GetAdjuster(),
		RegexCombine:  req.GetRegexCombine(),
		AmountMode:    req.GetAmountMode(),
		Failover:      req.GetFailover(),
		Arguments:     req.GetArguments(),
		Method:        req.GetMethod(),
		MaxQueueDelay: req.GetMaxQueueDelay(),
	}
	rule, err := json.Marshal(r)
	if err != nil {
		return "", err
	}
	return string(rule), nil
}

// rateLimitRecordEntry 构建rateLimit的记录entry
func rateLimitRecordEntry(ctx context.Context, namespace string, service string, md *model.RateLimit,
	opt model.OperationType) *model.RecordEntry {
	entry := &model.RecordEntry{
		ResourceType:  model.RRateLimit,
		OperationType: opt,
		Namespace:     namespace,
		Service:       service,
		Operator:      utils.ParseOperator(ctx),
		CreateTime:    time.Now(),
	}

	if md != nil {
		entry.Context = fmt.Sprintf("id:%s,label:%s,priority:%d,rule:%s,revision:%s",
			md.ID, md.Labels, md.Priority, md.Rule, md.Revision)
	}
	return entry
}

// wrapperRateLimitStoreResponse 封装路由存储层错误
func wrapperRateLimitStoreResponse(rule *api.Rule, err error) *api.Response {
	resp := storeError2Response(err)
	if resp == nil {
		return nil
	}
	resp.RateLimit = rule
	return resp
}
