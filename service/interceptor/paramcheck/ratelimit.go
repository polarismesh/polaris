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
	"time"

	"github.com/golang/protobuf/ptypes"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	"github.com/polarismesh/specification/source/go/api/v1/service_manage"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"
	"github.com/polarismesh/specification/source/go/api/v1/traffic_manage"
	apitraffic "github.com/polarismesh/specification/source/go/api/v1/traffic_manage"

	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/log"
	"github.com/polarismesh/polaris/common/utils"
)

// CreateRateLimits implements service.DiscoverServer.
func (svr *Server) CreateRateLimits(ctx context.Context,
	reqs []*traffic_manage.Rule) *service_manage.BatchWriteResponse {
	if err := checkBatchRateLimits(reqs); err != nil {
		return err
	}

	batchRsp := api.NewBatchWriteResponse(apimodel.Code_ExecuteSuccess)
	for i := range reqs {
		// 参数校验
		// 参数校验
		if resp := checkRateLimitParams(reqs[i]); resp != nil {
			api.Collect(batchRsp, resp)
			continue
		}
		if resp := checkRateLimitRuleParams(ctx, reqs[i]); resp != nil {
			api.Collect(batchRsp, resp)
			continue
		}
	}
	if !api.IsSuccess(batchRsp) {
		return batchRsp
	}

	return svr.nextSvr.CreateRateLimits(ctx, reqs)
}

// DeleteRateLimits implements service.DiscoverServer.
func (svr *Server) DeleteRateLimits(ctx context.Context, reqs []*traffic_manage.Rule) *service_manage.BatchWriteResponse {
	if err := checkBatchRateLimits(reqs); err != nil {
		return err
	}
	batchRsp := api.NewBatchWriteResponse(apimodel.Code_ExecuteSuccess)
	for i := range reqs {
		// 参数校验
		resp := checkRevisedRateLimitParams(reqs[i])
		api.Collect(batchRsp, resp)
	}
	if !api.IsSuccess(batchRsp) {
		return batchRsp
	}
	return svr.nextSvr.DeleteRateLimits(ctx, reqs)
}

// EnableRateLimits implements service.DiscoverServer.
func (svr *Server) EnableRateLimits(ctx context.Context,
	reqs []*traffic_manage.Rule) *service_manage.BatchWriteResponse {
	if err := checkBatchRateLimits(reqs); err != nil {
		return err
	}
	batchRsp := api.NewBatchWriteResponse(apimodel.Code_ExecuteSuccess)
	for i := range reqs {
		// 参数校验
		resp := checkRevisedRateLimitParams(reqs[i])
		api.Collect(batchRsp, resp)
	}
	if !api.IsSuccess(batchRsp) {
		return batchRsp
	}
	return svr.nextSvr.EnableRateLimits(ctx, reqs)
}

// GetRateLimits implements service.DiscoverServer.
func (svr *Server) GetRateLimits(ctx context.Context,
	query map[string]string) *service_manage.BatchQueryResponse {
	return svr.nextSvr.GetRateLimits(ctx, query)
}

// UpdateRateLimits implements service.DiscoverServer.
func (svr *Server) UpdateRateLimits(ctx context.Context, reqs []*traffic_manage.Rule) *service_manage.BatchWriteResponse {
	if err := checkBatchRateLimits(reqs); err != nil {
		return err
	}
	batchRsp := api.NewBatchWriteResponse(apimodel.Code_ExecuteSuccess)
	for i := range reqs {
		// 参数校验
		if resp := checkRevisedRateLimitParams(reqs[i]); resp != nil {
			api.Collect(batchRsp, resp)
			continue
		}
		if resp := checkRateLimitRuleParams(ctx, reqs[i]); resp != nil {
			api.Collect(batchRsp, resp)
			continue
		}
		if resp := checkRateLimitParamsDbLen(reqs[i]); resp != nil {
			api.Collect(batchRsp, resp)
			continue
		}
	}
	if !api.IsSuccess(batchRsp) {
		return batchRsp
	}

	return svr.nextSvr.UpdateRateLimits(ctx, reqs)
}

// checkBatchRateLimits 检查批量请求的限流规则
func checkBatchRateLimits(req []*apitraffic.Rule) *apiservice.BatchWriteResponse {
	if len(req) == 0 {
		return api.NewBatchWriteResponse(apimodel.Code_EmptyRequest)
	}

	if len(req) > utils.MaxBatchSize {
		return api.NewBatchWriteResponse(apimodel.Code_BatchSizeOverLimit)
	}

	return nil
}

// checkRateLimitParams 检查限流规则基础参数
func checkRateLimitParams(req *apitraffic.Rule) *apiservice.Response {
	if req == nil {
		return api.NewRateLimitResponse(apimodel.Code_EmptyRequest, req)
	}
	if err := utils.CheckResourceName(req.GetNamespace()); err != nil {
		return api.NewRateLimitResponse(apimodel.Code_InvalidNamespaceName, req)
	}
	if err := utils.CheckResourceName(req.GetService()); err != nil {
		return api.NewRateLimitResponse(apimodel.Code_InvalidServiceName, req)
	}
	if resp := checkRateLimitParamsDbLen(req); nil != resp {
		return resp
	}
	return nil
}

// checkRateLimitParams 检查限流规则基础参数
func checkRateLimitParamsDbLen(req *apitraffic.Rule) *apiservice.Response {
	if err := utils.CheckDbStrFieldLen(req.GetService(), utils.MaxDbServiceNameLength); err != nil {
		return api.NewRateLimitResponse(apimodel.Code_InvalidServiceName, req)
	}
	if err := utils.CheckDbStrFieldLen(req.GetNamespace(), utils.MaxDbServiceNamespaceLength); err != nil {
		return api.NewRateLimitResponse(apimodel.Code_InvalidNamespaceName, req)
	}
	if err := utils.CheckDbStrFieldLen(req.GetName(), utils.MaxDbRateLimitName); err != nil {
		return api.NewRateLimitResponse(apimodel.Code_InvalidRateLimitName, req)
	}
	return nil
}

// checkRateLimitRuleParams 检查限流规则其他参数
func checkRateLimitRuleParams(ctx context.Context, req *apitraffic.Rule) *apiservice.Response {
	// 检查amounts是否有重复周期
	amounts := req.GetAmounts()
	durations := make(map[time.Duration]bool)
	for _, amount := range amounts {
		d := amount.GetValidDuration()
		duration, err := ptypes.Duration(d)
		if err != nil {
			log.Error(err.Error(), utils.RequestID(ctx))
			return api.NewRateLimitResponse(apimodel.Code_InvalidRateLimitAmounts, req)
		}
		durations[duration] = true
	}
	if len(amounts) != len(durations) {
		return api.NewRateLimitResponse(apimodel.Code_InvalidRateLimitAmounts, req)
	}
	return nil
}

// checkRevisedRateLimitParams 检查修改/删除限流规则基础参数
func checkRevisedRateLimitParams(req *apitraffic.Rule) *apiservice.Response {
	if req == nil {
		return api.NewRateLimitResponse(apimodel.Code_EmptyRequest, req)
	}
	if req.GetId().GetValue() == "" {
		return api.NewRateLimitResponse(apimodel.Code_InvalidRateLimitID, req)
	}
	return nil
}
