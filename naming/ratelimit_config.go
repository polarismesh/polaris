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

package naming

import (
	"context"
	"encoding/json"
	"fmt"
	time2 "github.com/polarismesh/polaris-server/common/time"
	"time"

	"github.com/golang/protobuf/ptypes"
	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/common/utils"
)

var (
	// RateLimitFilters rate limit filters
	RateLimitFilters = map[string]bool{
		"service":   true,
		"namespace": true,
		"labels":    true,
		"offset":    true,
		"limit":     true,
	}
)

/**
 * CreateRateLimits 批量创建限流规则
 */
func (s *Server) CreateRateLimits(ctx context.Context, request []*api.Rule) *api.BatchWriteResponse {
	if err := checkBatchRateLimits(request); err != nil {
		return err
	}

	responses := api.NewBatchWriteResponse(api.ExecuteSuccess)
	for _, rateLimit := range request {
		response := s.CreateRateLimit(ctx, rateLimit)
		responses.Collect(response)
	}
	return api.FormatBatchWriteResponse(responses)
}

/**
 * CreateRateLimit 创建限流规则
 */
func (s *Server) CreateRateLimit(ctx context.Context, req *api.Rule) *api.Response {
	requestID := ParseRequestID(ctx)
	platformID := ParsePlatformID(ctx)

	// 参数校验
	if resp := checkRateLimitParams(req); resp != nil {
		return resp
	}

	if resp := checkRateLimitRuleParams(requestID, req); resp != nil {
		return resp
	}

	tx, err := s.storage.CreateTransaction()
	if err != nil {
		log.Error(err.Error(), ZapRequestID(requestID), ZapPlatformID(platformID))
		return api.NewRateLimitResponse(api.StoreLayerException, req)
	}
	defer func() {
		_ = tx.Commit()
	}()

	// 锁住服务，防止服务被删除
	service, err := tx.RLockService(req.GetService().GetValue(), req.GetNamespace().GetValue())
	if err != nil {
		log.Error(err.Error(), ZapRequestID(requestID), ZapPlatformID(platformID))
		return api.NewRateLimitResponse(api.StoreLayerException, req)
	}
	if service == nil {
		return api.NewRateLimitResponse(api.NotFoundService, req)
	}
	if service.IsAlias() {
		return api.NewRateLimitResponse(api.NotAllowAliasCreateRateLimit, req)
	}
	if err := s.verifyRateLimitAuth(ctx, service, req); err != nil {
		return err
	}

	clusterID := ""

	// 构造底层数据结构
	data, err := api2RateLimit(service.ID, clusterID, req)
	if err != nil {
		log.Error(err.Error(), ZapRequestID(requestID))
		return api.NewRateLimitResponse(api.ParseRateLimitException, req)
	}
	data.ID = NewUUID()

	// 存储层操作
	if err := s.storage.CreateRateLimit(data); err != nil {
		log.Error(err.Error(), ZapRequestID(requestID), ZapPlatformID(platformID))
		return wrapperRateLimitStoreResponse(req, err)
	}

	msg := fmt.Sprintf("create rate limit rule: id=%v, namespace=%v, service=%v, labels=%v",
		data.ID, req.GetNamespace().GetValue(), req.GetService().GetValue(), data.Labels)
	log.Info(msg, ZapRequestID(requestID), ZapPlatformID(platformID))

	s.RecordHistory(rateLimitRecordEntry(ctx, req.GetNamespace().GetValue(), req.GetService().GetValue(),
		data, model.OCreate))

	req.Id = utils.NewStringValue(data.ID)
	return api.NewRateLimitResponse(api.ExecuteSuccess, req)
}

/**
 * DeleteRateLimits 批量删除限流规则
 */
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

/**
 * DeleteRateLimit 删除单个限流规则
 */
func (s *Server) DeleteRateLimit(ctx context.Context, req *api.Rule) *api.Response {
	requestID := ParseRequestID(ctx)
	platformID := ParsePlatformID(ctx)

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

	// 鉴权
	service, resp := s.checkRateLimitValid(ctx, rateLimit.ServiceID, req)
	if resp != nil {
		return resp
	}

	// 生成新的revision
	rateLimit.Revision = NewUUID()

	// 存储层操作
	if err := s.storage.DeleteRateLimit(rateLimit); err != nil {
		log.Error(err.Error(), ZapRequestID(requestID), ZapPlatformID(platformID))
		return wrapperRateLimitStoreResponse(req, err)
	}

	msg := fmt.Sprintf("delete rate limit rule: id=%v, namespace=%v, service=%v, labels=%v",
		rateLimit.ID, service.Namespace, service.Name, rateLimit.Labels)
	log.Info(msg, ZapRequestID(requestID), ZapPlatformID(platformID))

	s.RecordHistory(rateLimitRecordEntry(ctx, service.Namespace, service.Name, rateLimit, model.ODelete))
	return api.NewRateLimitResponse(api.ExecuteSuccess, req)
}

/**
 * UpdateRateLimits 批量更新限流规则
 */
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

/**
 * UpdateRateLimit 更新限流规则
 */
func (s *Server) UpdateRateLimit(ctx context.Context, req *api.Rule) *api.Response {
	requestID := ParseRequestID(ctx)
	platformID := ParsePlatformID(ctx)

	// 参数校验
	if resp := checkRevisedRateLimitParams(req); resp != nil {
		return resp
	}

	if resp := checkRateLimitRuleParams(requestID, req); resp != nil {
		return resp
	}

	// 检查限流规则是否存在
	data, resp := s.checkRateLimitExisted(req.GetId().GetValue(), requestID, req)
	if resp != nil {
		return resp
	}

	// 鉴权
	service, resp := s.checkRateLimitValid(ctx, data.ServiceID, req)
	if resp != nil {
		return resp
	}

	clusterID := ""

	// 构造底层数据结构
	rateLimit, err := api2RateLimit(service.ID, clusterID, req)
	if err != nil {
		log.Error(err.Error(), ZapRequestID(requestID), ZapPlatformID(platformID))
		return api.NewRateLimitResponse(api.ParseRateLimitException, req)
	}
	rateLimit.ID = data.ID

	if err := s.storage.UpdateRateLimit(rateLimit); err != nil {
		log.Error(err.Error(), ZapRequestID(requestID), ZapPlatformID(platformID))
		return wrapperRateLimitStoreResponse(req, err)
	}

	msg := fmt.Sprintf("update rate limit: id=%v, namespace=%v, service=%v, labels=%v",
		rateLimit.ID, service.Namespace, service.Name, rateLimit.Labels)
	log.Info(msg, ZapRequestID(requestID), ZapPlatformID(platformID))

	s.RecordHistory(rateLimitRecordEntry(ctx, service.Namespace, service.Name, rateLimit, model.OUpdate))
	return api.NewRateLimitResponse(api.ExecuteSuccess, req)
}

/**
 * GetRateLimits 查询限流规则
 */
func (s *Server) GetRateLimits(query map[string]string) *api.BatchQueryResponse {
	for key := range query {
		if _, ok := RateLimitFilters[key]; !ok {
			log.Errorf("params %s is not allowed in querying rate limits", key)
			return api.NewBatchQueryResponse(api.InvalidParameter)
		}
	}

	// service转化为name
	if serviceName, ok := query["service"]; ok {
		query["name"] = serviceName
		delete(query, "service")
	}

	// 处理offset和limit
	offset, limit, err := ParseOffsetAndLimit(query)
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
		limit, err := rateLimit2api(item.ServiceName, item.NamespaceName, item.RateLimit)
		if err != nil {
			log.Errorf("get rate limits convert err: %s", err.Error())
			return api.NewBatchQueryResponse(api.ParseRateLimitException)
		}
		out.RateLimits = append(out.RateLimits, limit)
	}

	return out
}

/**
 * @brief 检查批量请求
 */
func checkBatchRateLimits(req []*api.Rule) *api.BatchWriteResponse {
	if len(req) == 0 {
		return api.NewBatchWriteResponse(api.EmptyRequest)
	}

	if len(req) > MaxBatchSize {
		return api.NewBatchWriteResponse(api.BatchSizeOverLimit)
	}

	return nil
}

/**
 * @brief 检查限流规则是否允许修改/删除
 */
func (s *Server) checkRateLimitValid(ctx context.Context, serviceID string, req *api.Rule) (
	*model.Service, *api.Response) {
	requestID := ParseRequestID(ctx)

	service, err := s.storage.GetServiceByID(serviceID)
	if err != nil {
		log.Error(err.Error(), ZapRequestID(requestID))
		return nil, api.NewRateLimitResponse(api.StoreLayerException, req)
	}

	if err := s.verifyRateLimitAuth(ctx, service, req); err != nil {
		return nil, err
	}
	return service, nil
}

/**
 * @brief 检查限流规则基础参数
 */
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
	if err := CheckDbStrFieldLen(req.GetService(), MaxDbServiceNameLength); err != nil {
		return api.NewRateLimitResponse(api.InvalidServiceName, req)
	}
	if err := CheckDbStrFieldLen(req.GetNamespace(), MaxDbServiceNamespaceLength); err != nil {
		return api.NewRateLimitResponse(api.InvalidNamespaceName, req)
	}
	if err := CheckDbStrFieldLen(req.GetServiceToken(), MaxDbServiceToken); err != nil {
		return api.NewRateLimitResponse(api.InvalidServiceToken, req)
	}
	return nil
}

/*
 * @brief 检查限流规则其他参数
 */
func checkRateLimitRuleParams(requestID string, req *api.Rule) *api.Response {
	// 检查业务维度标签
	if req.GetLabels() == nil {
		req.Labels = map[string]*api.MatchString{}
	}
	// 检查amounts是否有重复周期
	amounts := req.GetAmounts()
	durations := make(map[time.Duration]bool)
	for _, amount := range amounts {
		d := amount.GetValidDuration()
		duration, err := ptypes.Duration(d)
		if err != nil {
			log.Error(err.Error(), ZapRequestID(requestID))
			return api.NewRateLimitResponse(api.InvalidRateLimitAmounts, req)
		}
		durations[duration] = true
	}
	if len(amounts) != len(durations) {
		return api.NewRateLimitResponse(api.InvalidRateLimitAmounts, req)
	}
	return nil
}

/**
 * @brief 检查修改/删除限流规则基础参数
 */
func checkRevisedRateLimitParams(req *api.Rule) *api.Response {
	if req == nil {
		return api.NewRateLimitResponse(api.EmptyRequest, req)
	}
	if req.GetId().GetValue() == "" {
		return api.NewRateLimitResponse(api.InvalidRateLimitID, req)
	}
	return nil
}

/**
 * @brief 检查限流规则是否存在
 */
func (s *Server) checkRateLimitExisted(id, requestID string, req *api.Rule) (*model.RateLimit, *api.Response) {
	rateLimit, err := s.storage.GetRateLimitWithID(id)
	if err != nil {
		log.Error(err.Error(), ZapRequestID(requestID))
		return nil, api.NewRateLimitResponse(api.StoreLayerException, req)
	}
	if rateLimit == nil {
		return nil, api.NewRateLimitResponse(api.NotFoundRateLimit, req)
	}
	return rateLimit, nil
}

/**
 * @brief 获取限流规则请求的token信息
 */
func parseRateLimitReqToken(ctx context.Context, req *api.Rule) string {
	if reqToken := req.GetServiceToken().GetValue(); reqToken != "" {
		return reqToken
	}

	return ParseToken(ctx)
}

/**
 * @brief 限流鉴权
 */
func (s *Server) verifyRateLimitAuth(ctx context.Context, service *model.Service, req *api.Rule) *api.Response {
	// 使用平台id及token鉴权
	if ok := s.verifyAuthByPlatform(ctx, service.PlatformID); !ok {
		// 检查token是否存在
		token := parseRateLimitReqToken(ctx, req)
		if !s.authority.VerifyToken(token) {
			return api.NewRateLimitResponse(api.InvalidServiceToken, req)
		}

		// 检查token是否ok
		if ok := s.authority.VerifyService(service.Token, token); !ok {
			return api.NewRateLimitResponse(api.Unauthorized, req)
		}
	}

	return nil
}

/**
 * @brief 把API参数转化为内部数据结构
 */
func api2RateLimit(serviceID string, clusterID string, req *api.Rule) (*model.RateLimit, error) {
	labels, err := marshalRateLimitLabels(req.GetLabels())
	if err != nil {
		return nil, err
	}
	rule, err := marshalRateLimitRules(req)
	if err != nil {
		return nil, err
	}

	out := &model.RateLimit{
		ServiceID: serviceID,
		ClusterID: clusterID,
		Labels:    labels,
		Priority:  req.GetPriority().GetValue(),
		Rule:      rule,
		Revision:  NewUUID(),
	}
	return out, nil
}

/**
 * @brief 把内部数据结构转化为API参数
 */
func rateLimit2api(service string, namespace string, rateLimit *model.RateLimit) (
	*api.Rule, error) {
	if rateLimit == nil {
		return nil, nil
	}

	rule := &api.Rule{}

	// 反序列化rule
	if err := json.Unmarshal([]byte(rateLimit.Rule), rule); err != nil {
		return nil, err
	}

	// 反序列化labels
	labels := make(map[string]*api.MatchString)
	if err := json.Unmarshal([]byte(rateLimit.Labels), &labels); err != nil {
		return nil, err
	}

	// 暂时不返回Cluster
	rule.Id = utils.NewStringValue(rateLimit.ID)
	rule.Service = utils.NewStringValue(service)
	rule.Namespace = utils.NewStringValue(namespace)
	rule.Priority = utils.NewUInt32Value(rateLimit.Priority)
	rule.Labels = labels
	rule.Ctime = utils.NewStringValue(time2.Time2String(rateLimit.CreateTime))
	rule.Mtime = utils.NewStringValue(time2.Time2String(rateLimit.ModifyTime))
	rule.Revision = utils.NewStringValue(rateLimit.Revision)

	return rule, nil
}

/**
 * @brief 格式化限流规则labels
 */
func marshalRateLimitLabels(l map[string]*api.MatchString) (string, error) {
	labels, err := json.Marshal(l)
	if err != nil {
		return "", err
	}
	return string(labels), nil
}

/**
 * @brief 序列化限流规则具体内容
 */
func marshalRateLimitRules(req *api.Rule) (string, error) {
	r := &api.Rule{
		Subset:       req.GetSubset(),
		Resource:     req.GetResource(),
		Type:         req.GetType(),
		Amounts:      req.GetAmounts(),
		Action:       req.GetAction(),
		Disable:      req.GetDisable(),
		Report:       req.GetReport(),
		Adjuster:     req.GetAdjuster(),
		RegexCombine: req.GetRegexCombine(),
		AmountMode:   req.GetAmountMode(),
		Failover:     req.GetFailover(),
		Cluster:      req.GetCluster(),
	}

	rule, err := json.Marshal(r)
	if err != nil {
		return "", err
	}
	return string(rule), nil
}

/**
 * @brief 构建rateLimit的记录entry
 */
func rateLimitRecordEntry(ctx context.Context, namespace string, service string, md *model.RateLimit,
	opt model.OperationType) *model.RecordEntry {
	entry := &model.RecordEntry{
		ResourceType:  model.RRateLimit,
		OperationType: opt,
		Namespace:     namespace,
		Service:       service,
		Operator:      ParseOperator(ctx),
		CreateTime:    time.Now(),
	}

	if md != nil {
		entry.Context = fmt.Sprintf("id:%s,label:%s,priority:%d,rule:%s,revision:%s",
			md.ID, md.Labels, md.Priority, md.Rule, md.Revision)
	}
	return entry
}

/**
 * @brief 封装路由存储层错误
 */
func wrapperRateLimitStoreResponse(rule *api.Rule, err error) *api.Response {
	resp := storeError2Response(err)
	if resp == nil {
		return nil
	}
	resp.RateLimit = rule
	return resp
}
