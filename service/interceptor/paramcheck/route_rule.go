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
	"github.com/polarismesh/specification/source/go/api/v1/service_manage"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"
	"github.com/polarismesh/specification/source/go/api/v1/traffic_manage"
	apitraffic "github.com/polarismesh/specification/source/go/api/v1/traffic_manage"

	apiv1 "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/log"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
)

var (
	// RoutingConfigV2FilterAttrs router config filter attrs
	RoutingConfigV2FilterAttrs = map[string]bool{
		"id":                    true,
		"name":                  true,
		"service":               true,
		"namespace":             true,
		"source_service":        true,
		"destination_service":   true,
		"source_namespace":      true,
		"destination_namespace": true,
		"enable":                true,
		"offset":                true,
		"limit":                 true,
		"order_field":           true,
		"order_type":            true,
	}
)

// UpdateRoutingConfigs implements service.DiscoverServer.
// Deprecated: not support from 1.19.x
func (svr *Server) UpdateRoutingConfigs(ctx context.Context, req []*traffic_manage.Routing) *service_manage.BatchWriteResponse {
	return svr.nextSvr.UpdateRoutingConfigs(ctx, req)
}

// GetRoutingConfigs implements service.DiscoverServer.
// Deprecated: not support from 1.19.x
func (svr *Server) GetRoutingConfigs(ctx context.Context,
	query map[string]string) *service_manage.BatchQueryResponse {
	return svr.nextSvr.GetRoutingConfigs(ctx, query)
}

// CreateRoutingConfigs implements service.DiscoverServer.
// Deprecated: not support from 1.19.x
func (svr *Server) CreateRoutingConfigs(ctx context.Context,
	req []*traffic_manage.Routing) *service_manage.BatchWriteResponse {
	return svr.nextSvr.CreateRoutingConfigs(ctx, req)
}

// DeleteRoutingConfigs implements service.DiscoverServer.
// Deprecated: not support from 1.19.x
func (svr *Server) DeleteRoutingConfigs(ctx context.Context,
	req []*traffic_manage.Routing) *service_manage.BatchWriteResponse {
	return svr.nextSvr.DeleteRoutingConfigs(ctx, req)
}

// CreateRoutingConfigsV2 implements service.DiscoverServer.
func (svr *Server) CreateRoutingConfigsV2(ctx context.Context,
	req []*traffic_manage.RouteRule) *service_manage.BatchWriteResponse {
	if err := checkBatchRoutingConfigV2(req); err != nil {
		return err
	}
	batchRsp := apiv1.NewBatchWriteResponse(apimodel.Code_ExecuteSuccess)
	for _, item := range req {
		if resp := checkRoutingConfigV2(item); resp != nil {
			apiv1.Collect(batchRsp, resp)
		}
	}
	if !apiv1.IsSuccess(batchRsp) {
		return batchRsp
	}
	return svr.nextSvr.CreateRoutingConfigsV2(ctx, req)
}

// EnableRoutings implements service.DiscoverServer.
func (svr *Server) EnableRoutings(ctx context.Context,
	req []*traffic_manage.RouteRule) *service_manage.BatchWriteResponse {
	if err := checkBatchRoutingConfigV2(req); err != nil {
		return err
	}
	batchRsp := apiv1.NewBatchWriteResponse(apimodel.Code_ExecuteSuccess)
	for _, item := range req {
		if resp := checkRoutingConfigIDV2(item); resp != nil {
			apiv1.Collect(batchRsp, resp)
		}
	}
	if !apiv1.IsSuccess(batchRsp) {
		return batchRsp
	}
	return svr.nextSvr.EnableRoutings(ctx, req)
}

// UpdateRoutingConfigsV2 implements service.DiscoverServer.
func (svr *Server) UpdateRoutingConfigsV2(ctx context.Context,
	req []*traffic_manage.RouteRule) *service_manage.BatchWriteResponse {
	if err := checkBatchRoutingConfigV2(req); err != nil {
		return err
	}
	batchRsp := apiv1.NewBatchWriteResponse(apimodel.Code_ExecuteSuccess)
	for _, item := range req {
		if resp := checkUpdateRoutingConfigV2(item); resp != nil {
			apiv1.Collect(batchRsp, resp)
		}
	}
	if !apiv1.IsSuccess(batchRsp) {
		return batchRsp
	}
	return svr.nextSvr.UpdateRoutingConfigsV2(ctx, req)
}

// QueryRoutingConfigsV2 implements service.DiscoverServer.
func (svr *Server) QueryRoutingConfigsV2(ctx context.Context,
	query map[string]string) *service_manage.BatchQueryResponse {

	offset, limit, err := utils.ParseOffsetAndLimit(query)
	if err != nil {
		return apiv1.NewBatchQueryResponse(apimodel.Code_InvalidParameter)
	}

	filter := make(map[string]string)
	for key, value := range query {
		if _, ok := RoutingConfigV2FilterAttrs[key]; !ok {
			log.Errorf("[Routing][V2][Query] attribute(%s) is not allowed", key)
			return apiv1.NewBatchQueryResponse(apimodel.Code_InvalidParameter)
		}
		filter[key] = value
	}
	filter["offset"] = strconv.FormatUint(uint64(offset), 10)
	filter["limit"] = strconv.FormatUint(uint64(limit), 10)

	return svr.nextSvr.QueryRoutingConfigsV2(ctx, filter)
}

// DeleteRoutingConfigsV2 implements service.DiscoverServer.
func (svr *Server) DeleteRoutingConfigsV2(ctx context.Context,
	req []*traffic_manage.RouteRule) *service_manage.BatchWriteResponse {
	if err := checkBatchRoutingConfigV2(req); err != nil {
		return err
	}
	batchRsp := apiv1.NewBatchWriteResponse(apimodel.Code_ExecuteSuccess)
	for _, item := range req {
		if resp := checkRoutingConfigIDV2(item); resp != nil {
			apiv1.Collect(batchRsp, resp)
		}
	}
	if !apiv1.IsSuccess(batchRsp) {
		return batchRsp
	}
	return svr.nextSvr.DeleteRoutingConfigsV2(ctx, req)
}

// checkBatchRoutingConfig Check batch request
func checkBatchRoutingConfigV2(req []*apitraffic.RouteRule) *apiservice.BatchWriteResponse {
	if len(req) == 0 {
		return apiv1.NewBatchWriteResponse(apimodel.Code_EmptyRequest)
	}

	if len(req) > utils.MaxBatchSize {
		return apiv1.NewBatchWriteResponse(apimodel.Code_BatchSizeOverLimit)
	}

	return nil
}

// checkRoutingConfig Check the validity of the basic parameter of the routing configuration
func checkRoutingConfigV2(req *apitraffic.RouteRule) *apiservice.Response {
	if req == nil {
		return apiv1.NewRouterResponse(apimodel.Code_EmptyRequest, req)
	}

	if err := checkRoutingNameAndNamespace(req); err != nil {
		return err
	}

	if err := checkRoutingConfigPriorityV2(req); err != nil {
		return err
	}

	if err := checkRoutingPolicyV2(req); err != nil {
		return err
	}

	return nil
}

// checkUpdateRoutingConfigV2 Check the validity of the basic parameter of the routing configuration
func checkUpdateRoutingConfigV2(req *apitraffic.RouteRule) *apiservice.Response {
	if resp := checkRoutingConfigIDV2(req); resp != nil {
		return resp
	}

	if err := checkRoutingNameAndNamespace(req); err != nil {
		return err
	}

	if err := checkRoutingConfigPriorityV2(req); err != nil {
		return err
	}

	if err := checkRoutingPolicyV2(req); err != nil {
		return err
	}

	return nil
}

func checkRoutingNameAndNamespace(req *apitraffic.RouteRule) *apiservice.Response {
	if err := utils.CheckDbStrFieldLen(utils.NewStringValue(req.GetName()), utils.MaxRuleName); err != nil {
		return apiv1.NewRouterResponse(apimodel.Code_InvalidRoutingName, req)
	}

	if err := utils.CheckDbStrFieldLen(utils.NewStringValue(req.GetNamespace()),
		utils.MaxDbServiceNamespaceLength); err != nil {
		return apiv1.NewRouterResponse(apimodel.Code_InvalidNamespaceName, req)
	}

	return nil
}

func checkRoutingConfigIDV2(req *apitraffic.RouteRule) *apiservice.Response {
	if req == nil {
		return apiv1.NewRouterResponse(apimodel.Code_EmptyRequest, req)
	}

	if req.Id == "" {
		return apiv1.NewResponse(apimodel.Code_InvalidRoutingID)
	}

	return nil
}

func checkRoutingConfigPriorityV2(req *apitraffic.RouteRule) *apiservice.Response {
	if req == nil {
		return apiv1.NewRouterResponse(apimodel.Code_EmptyRequest, req)
	}

	if req.Priority > 10 {
		return apiv1.NewResponse(apimodel.Code_InvalidRoutingPriority)
	}

	return nil
}

func checkRoutingPolicyV2(req *apitraffic.RouteRule) *apiservice.Response {
	if req == nil {
		return apiv1.NewRouterResponse(apimodel.Code_EmptyRequest, req)
	}

	if req.GetRoutingPolicy() != apitraffic.RoutingPolicy_RulePolicy {
		return apiv1.NewRouterResponse(apimodel.Code_InvalidRoutingPolicy, req)
	}

	// Automatically supplement @Type attribute according to Policy
	if req.RoutingConfig.TypeUrl == "" {
		if req.GetRoutingPolicy() == apitraffic.RoutingPolicy_RulePolicy {
			req.RoutingConfig.TypeUrl = model.RuleRoutingTypeUrl
		}
		if req.GetRoutingPolicy() == apitraffic.RoutingPolicy_MetadataPolicy {
			req.RoutingConfig.TypeUrl = model.MetaRoutingTypeUrl
		}
		if req.GetRoutingPolicy() == apitraffic.RoutingPolicy_NearbyPolicy {
			req.RoutingConfig.TypeUrl = model.NearbyRoutingTypeUrl
		}
	}

	return nil
}
