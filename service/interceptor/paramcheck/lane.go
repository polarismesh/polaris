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

	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"
	apitraffic "github.com/polarismesh/specification/source/go/api/v1/traffic_manage"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/wrapperspb"

	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/log"
	"github.com/polarismesh/polaris/common/utils"
)

var (
	laneGroupSearchAttributes = map[string]struct{}{
		"id":          {},
		"name":        {},
		"offset":      {},
		"brief":       {},
		"limit":       {},
		"order_type":  {},
		"order_field": {},
	}
)

// CreateLaneGroups 批量创建泳道组
func (svr *Server) CreateLaneGroups(ctx context.Context, reqs []*apitraffic.LaneGroup) *apiservice.BatchWriteResponse {
	if err := checkBatchLaneGroupRules(reqs); err != nil {
		return err
	}
	batchRsp := api.NewBatchWriteResponse(apimodel.Code_ExecuteSuccess)
	for i := range reqs {
		rsp := checkLaneGroupParam(reqs[i], false)
		api.Collect(batchRsp, rsp)
	}

	if !api.IsSuccess(batchRsp) {
		return batchRsp
	}
	return svr.nextSvr.CreateLaneGroups(ctx, reqs)
}

// UpdateLaneGroups 批量更新泳道组
func (svr *Server) UpdateLaneGroups(ctx context.Context, reqs []*apitraffic.LaneGroup) *apiservice.BatchWriteResponse {
	if err := checkBatchLaneGroupRules(reqs); err != nil {
		return err
	}
	batchRsp := api.NewBatchWriteResponse(apimodel.Code_ExecuteSuccess)
	for i := range reqs {
		rsp := checkLaneGroupParam(reqs[i], true)
		api.Collect(batchRsp, rsp)
	}

	if !api.IsSuccess(batchRsp) {
		return batchRsp
	}
	return svr.nextSvr.UpdateLaneGroups(ctx, reqs)
}

// DeleteLaneGroups 批量删除泳道组
func (svr *Server) DeleteLaneGroups(ctx context.Context, reqs []*apitraffic.LaneGroup) *apiservice.BatchWriteResponse {
	if err := checkBatchLaneGroupRules(reqs); err != nil {
		return err
	}
	return svr.nextSvr.DeleteLaneGroups(ctx, reqs)
}

// GetLaneGroups 查询泳道组列表
func (svr *Server) GetLaneGroups(ctx context.Context, filter map[string]string) *apiservice.BatchQueryResponse {
	offset, limit, err := utils.ParseOffsetAndLimit(filter)
	if err != nil {
		return api.NewBatchQueryResponseWithMsg(apimodel.Code_BadRequest, err.Error())
	}

	for k := range filter {
		if _, ok := laneGroupSearchAttributes[k]; !ok {
			log.Error("[Server][LaneGroup][Query] not allowed", zap.String("attribute", k), utils.RequestID(ctx))
			return api.NewBatchQueryResponseWithMsg(apimodel.Code_InvalidParameter, k+" is not allowed")
		}
		if filter[k] == "" {
			delete(filter, k)
		}
	}

	if _, ok := filter["order_field"]; !ok {
		filter["order_field"] = "mtime"
	}
	if _, ok := filter["order_type"]; !ok {
		filter["order_type"] = "desc"
	}

	filter["offset"] = strconv.FormatUint(uint64(offset), 10)
	filter["limit"] = strconv.FormatUint(uint64(limit), 10)

	return svr.nextSvr.GetLaneGroups(ctx, filter)
}

func checkBatchLaneGroupRules(req []*apitraffic.LaneGroup) *apiservice.BatchWriteResponse {
	if len(req) == 0 {
		return api.NewBatchWriteResponse(apimodel.Code_EmptyRequest)
	}

	if len(req) > utils.MaxBatchSize {
		return api.NewBatchWriteResponse(apimodel.Code_BatchSizeOverLimit)
	}
	return nil
}

func checkLaneGroupParam(req *apitraffic.LaneGroup, update bool) *apiservice.Response {
	if len(req.GetName()) >= utils.MaxRuleName {
		return api.NewResponseWithMsg(apimodel.Code_InvalidParameter, "lane_group name size must be <= 64")
	}
	if err := utils.CheckResourceName(wrapperspb.String(req.GetName())); err != nil {
		return api.NewResponseWithMsg(apimodel.Code_InvalidParameter, err.Error())
	}
	if len(req.Rules) > utils.MaxBatchSize {
		return api.NewResponseWithMsg(apimodel.Code_InvalidParameter, "lane_rule size must be <= 100")
	}
	for i := range req.Rules {
		rule := req.Rules[i]
		if err := utils.CheckResourceName(wrapperspb.String(rule.GetName())); err != nil {
			return api.NewResponseWithMsg(apimodel.Code_InvalidParameter, err.Error())
		}
		if len(rule.GetName()) >= utils.MaxRuleName {
			return api.NewResponseWithMsg(apimodel.Code_InvalidParameter, "lane_rule name size must be <= 64")
		}
	}

	if update {
		if req.GetId() == "" {
			return api.NewResponseWithMsg(apimodel.Code_InvalidParameter, "lane_group id is empty")
		}
	}
	return nil
}
