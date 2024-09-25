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

package paramcheck

import (
	"context"
	"strconv"

	"github.com/polarismesh/specification/source/go/api/v1/fault_tolerance"
	apifault "github.com/polarismesh/specification/source/go/api/v1/fault_tolerance"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	"github.com/polarismesh/specification/source/go/api/v1/service_manage"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"

	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/log"
	commonstore "github.com/polarismesh/polaris/common/store"
	"github.com/polarismesh/polaris/common/utils"
)

var (
	// FaultDetectRuleFilters filter fault detect rule query parameters
	FaultDetectRuleFilters = map[string]bool{
		"brief":            true,
		"offset":           true,
		"limit":            true,
		"id":               true,
		"name":             true,
		"namespace":        true,
		"service":          true,
		"serviceNamespace": true,
		"dstService":       true,
		"dstNamespace":     true,
		"dstMethod":        true,
		"description":      true,
	}
)

// DeleteFaultDetectRules implements service.DiscoverServer.
func (svr *Server) DeleteFaultDetectRules(ctx context.Context,
	request []*fault_tolerance.FaultDetectRule) *service_manage.BatchWriteResponse {

	if checkErr := checkBatchFaultDetectRules(request); checkErr != nil {
		return checkErr
	}

	batchRsp := api.NewBatchWriteResponse(apimodel.Code_ExecuteSuccess)
	for _, cbRule := range request {
		if resp := checkFaultDetectRuleParams(cbRule, false, true); resp != nil {
			api.Collect(batchRsp, resp)
			continue
		}
	}

	if !api.IsSuccess(batchRsp) {
		return batchRsp
	}

	return svr.nextSvr.DeleteFaultDetectRules(ctx, request)
}

// GetFaultDetectRules implements service.DiscoverServer.
func (svr *Server) GetFaultDetectRules(ctx context.Context,
	query map[string]string) *service_manage.BatchQueryResponse {

	for key := range query {
		if _, ok := FaultDetectRuleFilters[key]; !ok {
			log.Errorf("params %s is not allowed in querying fault detect rule", key)
			return api.NewBatchQueryResponse(apimodel.Code_InvalidParameter)
		}
	}
	offset, limit, err := utils.ParseOffsetAndLimit(query)
	if err != nil {
		return api.NewBatchQueryResponse(apimodel.Code_InvalidParameter)
	}

	query["offset"] = strconv.FormatUint(uint64(offset), 10)
	query["limit"] = strconv.FormatUint(uint64(limit), 10)

	return svr.nextSvr.GetFaultDetectRules(ctx, query)
}

// CreateFaultDetectRules implements service.DiscoverServer.
func (svr *Server) CreateFaultDetectRules(ctx context.Context,
	request []*fault_tolerance.FaultDetectRule) *service_manage.BatchWriteResponse {

	if checkErr := checkBatchFaultDetectRules(request); checkErr != nil {
		return checkErr
	}

	batchRsp := api.NewBatchWriteResponse(apimodel.Code_ExecuteSuccess)
	for _, cbRule := range request {
		if resp := checkFaultDetectRuleParams(cbRule, false, true); resp != nil {
			api.Collect(batchRsp, resp)
			continue
		}
	}

	if !api.IsSuccess(batchRsp) {
		return batchRsp
	}

	return svr.nextSvr.CreateFaultDetectRules(ctx, request)
}

// UpdateFaultDetectRules implements service.DiscoverServer.
func (svr *Server) UpdateFaultDetectRules(ctx context.Context, request []*fault_tolerance.FaultDetectRule) *service_manage.BatchWriteResponse {
	if checkErr := checkBatchFaultDetectRules(request); checkErr != nil {
		return checkErr
	}

	batchRsp := api.NewBatchWriteResponse(apimodel.Code_ExecuteSuccess)
	for _, cbRule := range request {
		if resp := checkFaultDetectRuleParams(cbRule, false, true); resp != nil {
			api.Collect(batchRsp, resp)
			continue
		}
		if resp := svr.checkFaultDetectRuleExists(ctx, cbRule.GetId()); resp != nil {
			api.Collect(batchRsp, resp)
			continue
		}
	}

	if !api.IsSuccess(batchRsp) {
		return batchRsp
	}

	return svr.nextSvr.UpdateFaultDetectRules(ctx, request)
}

func (svr *Server) checkFaultDetectRuleExists(ctx context.Context, id string) *apiservice.Response {
	exists, err := svr.storage.HasFaultDetectRule(id)
	if err != nil {
		log.Error(err.Error(), utils.RequestID(ctx))
		return api.NewResponse(commonstore.StoreCode2APICode(err))
	}
	if !exists {
		return api.NewResponse(apimodel.Code_NotFoundResource)
	}
	return nil
}

func checkBatchFaultDetectRules(req []*apifault.FaultDetectRule) *apiservice.BatchWriteResponse {
	if len(req) == 0 {
		return api.NewBatchWriteResponse(apimodel.Code_EmptyRequest)
	}

	if len(req) > utils.MaxBatchSize {
		return api.NewBatchWriteResponse(apimodel.Code_BatchSizeOverLimit)
	}

	return nil
}

func checkFaultDetectRuleParams(
	req *apifault.FaultDetectRule, idRequired bool, nameRequired bool) *apiservice.Response {
	if req == nil {
		return api.NewResponse(apimodel.Code_EmptyRequest)
	}
	if resp := checkFaultDetectRuleParamsDbLen(req); nil != resp {
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

func checkFaultDetectRuleParamsDbLen(req *apifault.FaultDetectRule) *apiservice.Response {
	if err := utils.CheckDbRawStrFieldLen(req.GetTargetService().GetService(), utils.MaxDbServiceNameLength); err != nil {
		return api.NewResponse(apimodel.Code_InvalidServiceName)
	}
	if err := utils.CheckDbRawStrFieldLen(
		req.GetTargetService().GetNamespace(), utils.MaxDbServiceNamespaceLength); err != nil {
		return api.NewResponse(apimodel.Code_InvalidNamespaceName)
	}
	if err := utils.CheckDbRawStrFieldLen(req.GetName(), utils.MaxRuleName); err != nil {
		return api.NewResponse(apimodel.Code_InvalidRateLimitName)
	}
	if err := utils.CheckDbRawStrFieldLen(req.GetNamespace(), utils.MaxDbServiceNamespaceLength); err != nil {
		return api.NewResponse(apimodel.Code_InvalidNamespaceName)
	}
	if err := utils.CheckDbRawStrFieldLen(req.GetDescription(), utils.MaxCommentLength); err != nil {
		return api.NewResponse(apimodel.Code_InvalidServiceComment)
	}
	return nil
}
