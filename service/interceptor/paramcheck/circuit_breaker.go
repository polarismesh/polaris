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
	"github.com/polarismesh/polaris/common/utils"
)

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

// GetMasterCircuitBreakers implements service.DiscoverServer.
// Deprecated: not support from 1.14.x
func (svr *Server) GetMasterCircuitBreakers(ctx context.Context,
	query map[string]string) *service_manage.BatchQueryResponse {
	return svr.nextSvr.GetMasterCircuitBreakers(ctx, query)
}

// GetReleaseCircuitBreakers implements service.DiscoverServer.
// Deprecated: not support from 1.14.x
func (svr *Server) GetReleaseCircuitBreakers(ctx context.Context,
	query map[string]string) *service_manage.BatchQueryResponse {
	return svr.nextSvr.GetReleaseCircuitBreakers(ctx, query)
}

// GetCircuitBreaker implements service.DiscoverServer.
// Deprecated: not support from 1.14.x
func (svr *Server) GetCircuitBreaker(ctx context.Context,
	query map[string]string) *service_manage.BatchQueryResponse {
	return svr.nextSvr.GetCircuitBreaker(ctx, query)
}

// GetCircuitBreakerByService implements service.DiscoverServer.
// Deprecated: not support from 1.14.x
func (svr *Server) GetCircuitBreakerByService(ctx context.Context,
	query map[string]string) *service_manage.BatchQueryResponse {
	return svr.nextSvr.GetCircuitBreakerByService(ctx, query)
}

// DeleteCircuitBreakers implements service.DiscoverServer.
// Deprecated: not support from 1.14.x
func (svr *Server) DeleteCircuitBreakers(ctx context.Context,
	req []*fault_tolerance.CircuitBreaker) *service_manage.BatchWriteResponse {
	return svr.nextSvr.DeleteCircuitBreakers(ctx, req)
}

// GetCircuitBreakerToken implements service.DiscoverServer.
// Deprecated: not support from 1.14.x
func (svr *Server) GetCircuitBreakerToken(ctx context.Context,
	req *fault_tolerance.CircuitBreaker) *service_manage.Response {
	return svr.nextSvr.GetCircuitBreakerToken(ctx, req)
}

// GetCircuitBreakerVersions implements service.DiscoverServer.
// Deprecated: not support from 1.14.x
func (svr *Server) GetCircuitBreakerVersions(ctx context.Context,
	query map[string]string) *service_manage.BatchQueryResponse {
	return svr.nextSvr.GetCircuitBreakerVersions(ctx, query)
}

// ReleaseCircuitBreakers implements service.DiscoverServer.
// Deprecated: not support from 1.14.x
func (svr *Server) ReleaseCircuitBreakers(ctx context.Context, req []*service_manage.ConfigRelease) *service_manage.BatchWriteResponse {
	return svr.nextSvr.ReleaseCircuitBreakers(ctx, req)
}

// UnBindCircuitBreakers implements service.DiscoverServer.
// Deprecated: not support from 1.14.x
func (svr *Server) UnBindCircuitBreakers(ctx context.Context, req []*service_manage.ConfigRelease) *service_manage.BatchWriteResponse {
	return svr.nextSvr.UnBindCircuitBreakers(ctx, req)
}

// CreateCircuitBreakerVersions implements service.DiscoverServer.
// Deprecated: not support from 1.14.x
func (svr *Server) CreateCircuitBreakerVersions(ctx context.Context,
	req []*fault_tolerance.CircuitBreaker) *service_manage.BatchWriteResponse {
	return svr.nextSvr.CreateCircuitBreakerVersions(ctx, req)
}

// CreateCircuitBreakers implements service.DiscoverServer.
// Deprecated: not support from 1.14.x
func (svr *Server) CreateCircuitBreakers(ctx context.Context,
	req []*fault_tolerance.CircuitBreaker) *service_manage.BatchWriteResponse {
	return svr.nextSvr.CreateCircuitBreakers(ctx, req)
}

// UpdateCircuitBreakers implements service.DiscoverServer.
// Deprecated: not support from 1.14.x
func (svr *Server) UpdateCircuitBreakers(ctx context.Context, req []*fault_tolerance.CircuitBreaker) *service_manage.BatchWriteResponse {
	return svr.nextSvr.UpdateCircuitBreakers(ctx, req)
}

// ------------- 这里开始接口实现才是正式有效的 -------------

// GetCircuitBreakerRules implements service.DiscoverServer.
func (svr *Server) GetCircuitBreakerRules(ctx context.Context,
	query map[string]string) *service_manage.BatchQueryResponse {

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

	searchFilter["offset"] = strconv.FormatUint(uint64(offset), 10)
	searchFilter["limit"] = strconv.FormatUint(uint64(limit), 10)

	return svr.nextSvr.GetCircuitBreakerRules(ctx, searchFilter)
}

// DeleteCircuitBreakerRules implements service.DiscoverServer.
func (svr *Server) DeleteCircuitBreakerRules(ctx context.Context,
	reqs []*fault_tolerance.CircuitBreakerRule) *service_manage.BatchWriteResponse {
	if err := checkBatchCircuitBreakerRules(reqs); err != nil {
		return err
	}
	batchRsp := api.NewBatchWriteResponse(apimodel.Code_ExecuteSuccess)
	for i := range reqs {
		rsp := checkCircuitBreakerRuleParams(reqs[i], true, false)
		api.Collect(batchRsp, rsp)
	}
	if !api.IsSuccess(batchRsp) {
		return batchRsp
	}
	return svr.nextSvr.DeleteCircuitBreakerRules(ctx, reqs)
}

// EnableCircuitBreakerRules implements service.DiscoverServer.
func (svr *Server) EnableCircuitBreakerRules(ctx context.Context,
	request []*fault_tolerance.CircuitBreakerRule) *service_manage.BatchWriteResponse {
	if err := checkBatchCircuitBreakerRules(request); err != nil {
		return err
	}
	return svr.nextSvr.EnableCircuitBreakerRules(ctx, request)
}

// CreateCircuitBreakerRules implements service.DiscoverServer.
func (svr *Server) CreateCircuitBreakerRules(ctx context.Context,
	reqs []*fault_tolerance.CircuitBreakerRule) *service_manage.BatchWriteResponse {
	if err := checkBatchCircuitBreakerRules(reqs); err != nil {
		return err
	}

	batchRsp := api.NewBatchWriteResponse(apimodel.Code_ExecuteSuccess)
	for i := range reqs {
		rsp := checkCircuitBreakerRuleParams(reqs[i], false, true)
		api.Collect(batchRsp, rsp)
	}
	if !api.IsSuccess(batchRsp) {
		return batchRsp
	}
	return svr.nextSvr.CreateCircuitBreakerRules(ctx, reqs)
}

// UpdateCircuitBreakerRules implements service.DiscoverServer.
func (svr *Server) UpdateCircuitBreakerRules(ctx context.Context,
	reqs []*fault_tolerance.CircuitBreakerRule) *service_manage.BatchWriteResponse {
	if err := checkBatchCircuitBreakerRules(reqs); err != nil {
		return err
	}
	batchRsp := api.NewBatchWriteResponse(apimodel.Code_ExecuteSuccess)
	for i := range reqs {
		rsp := checkCircuitBreakerRuleParams(reqs[i], true, true)
		api.Collect(batchRsp, rsp)
	}
	if !api.IsSuccess(batchRsp) {
		return batchRsp
	}
	return svr.nextSvr.UpdateCircuitBreakerRules(ctx, reqs)
}

func checkBatchCircuitBreakerRules(req []*apifault.CircuitBreakerRule) *apiservice.BatchWriteResponse {
	if len(req) == 0 {
		return api.NewBatchWriteResponse(apimodel.Code_EmptyRequest)
	}

	if len(req) > utils.MaxBatchSize {
		return api.NewBatchWriteResponse(apimodel.Code_BatchSizeOverLimit)
	}
	return nil
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
		req.RuleMatcher.GetSource().GetService(), utils.MaxDbServiceNameLength); err != nil {
		return api.NewResponse(apimodel.Code_InvalidServiceName)
	}
	if err := utils.CheckDbRawStrFieldLen(
		req.RuleMatcher.GetSource().GetNamespace(), utils.MaxDbServiceNamespaceLength); err != nil {
		return api.NewResponse(apimodel.Code_InvalidNamespaceName)
	}
	if err := utils.CheckDbRawStrFieldLen(req.GetName(), utils.MaxRuleName); err != nil {
		return api.NewResponse(apimodel.Code_InvalidCircuitBreakerName)
	}
	if err := utils.CheckDbRawStrFieldLen(req.GetNamespace(), utils.MaxDbServiceNamespaceLength); err != nil {
		return api.NewResponse(apimodel.Code_InvalidNamespaceName)
	}
	if err := utils.CheckDbRawStrFieldLen(req.GetDescription(), utils.MaxCommentLength); err != nil {
		return api.NewResponse(apimodel.Code_InvalidServiceComment)
	}
	return nil
}
