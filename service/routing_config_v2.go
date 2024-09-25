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
	"strconv"

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/golang/protobuf/ptypes/wrappers"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"
	apitraffic "github.com/polarismesh/specification/source/go/api/v1/traffic_manage"
	"go.uber.org/zap"

	cachetypes "github.com/polarismesh/polaris/cache/api"
	apiv1 "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/model"
	commonstore "github.com/polarismesh/polaris/common/store"
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

// CreateRoutingConfigsV2 Create a routing configuration
func (s *Server) CreateRoutingConfigsV2(
	ctx context.Context, req []*apitraffic.RouteRule) *apiservice.BatchWriteResponse {
	if err := checkBatchRoutingConfigV2(req); err != nil {
		return err
	}

	resp := apiv1.NewBatchWriteResponse(apimodel.Code_ExecuteSuccess)
	for _, entry := range req {
		apiv1.Collect(resp, s.createRoutingConfigV2(ctx, entry))
	}

	return apiv1.FormatBatchWriteResponse(resp)
}

// createRoutingConfigV2 Create a routing configuration
func (s *Server) createRoutingConfigV2(ctx context.Context, req *apitraffic.RouteRule) *apiservice.Response {
	if resp := checkRoutingConfigV2(req); resp != nil {
		return resp
	}

	conf, err := Api2RoutingConfigV2(req)
	if err != nil {
		log.Error("[Routing][V2] parse routing config v2 from request for create",
			utils.RequestID(ctx), zap.Error(err))
		return apiv1.NewResponse(apimodel.Code_ExecuteException)
	}

	if err := s.storage.CreateRoutingConfigV2(conf); err != nil {
		log.Error("[Routing][V2] create routing config v2 store layer",
			utils.RequestID(ctx), zap.Error(err))
		return apiv1.NewResponse(commonstore.StoreCode2APICode(err))
	}

	s.RecordHistory(ctx, routingV2RecordEntry(ctx, req, conf, model.OCreate))

	req.Id = conf.ID
	return apiv1.NewRouterResponse(apimodel.Code_ExecuteSuccess, req)
}

// DeleteRoutingConfigsV2 Batch delete routing configuration
func (s *Server) DeleteRoutingConfigsV2(
	ctx context.Context, req []*apitraffic.RouteRule) *apiservice.BatchWriteResponse {
	if err := checkBatchRoutingConfigV2(req); err != nil {
		return err
	}

	out := apiv1.NewBatchWriteResponse(apimodel.Code_ExecuteSuccess)
	for _, entry := range req {
		resp := s.deleteRoutingConfigV2(ctx, entry)
		apiv1.Collect(out, resp)
	}

	return apiv1.FormatBatchWriteResponse(out)
}

// DeleteRoutingConfigV2 Delete a routing configuration
func (s *Server) deleteRoutingConfigV2(ctx context.Context, req *apitraffic.RouteRule) *apiservice.Response {
	if resp := checkRoutingConfigIDV2(req); resp != nil {
		return resp
	}

	// Determine whether the current routing rules are only converted from the memory transmission in the V1 version
	if _, ok := s.Cache().RoutingConfig().IsConvertFromV1(req.Id); ok {
		resp := s.transferV1toV2OnModify(ctx, req)
		if resp.GetCode().GetValue() != uint32(apimodel.Code_ExecuteSuccess) {
			return resp
		}
	}

	if err := s.storage.DeleteRoutingConfigV2(req.Id); err != nil {
		log.Error("[Routing][V2] delete routing config v2 store layer",
			utils.RequestID(ctx), zap.Error(err))
		return apiv1.NewResponse(commonstore.StoreCode2APICode(err))
	}

	s.RecordHistory(ctx, routingV2RecordEntry(ctx, req, &model.RouterConfig{
		ID:   req.GetId(),
		Name: req.GetName(),
	}, model.ODelete))
	return apiv1.NewRouterResponse(apimodel.Code_ExecuteSuccess, req)
}

// UpdateRoutingConfigsV2 Batch update routing configuration
func (s *Server) UpdateRoutingConfigsV2(
	ctx context.Context, req []*apitraffic.RouteRule) *apiservice.BatchWriteResponse {
	if err := checkBatchRoutingConfigV2(req); err != nil {
		return err
	}

	out := apiv1.NewBatchWriteResponse(apimodel.Code_ExecuteSuccess)
	for _, entry := range req {
		resp := s.updateRoutingConfigV2(ctx, entry)
		apiv1.Collect(out, resp)
	}

	return apiv1.FormatBatchWriteResponse(out)
}

// updateRoutingConfigV2 Update a single routing configuration
func (s *Server) updateRoutingConfigV2(ctx context.Context, req *apitraffic.RouteRule) *apiservice.Response {
	// If V2 routing rules to be modified are from the V1 rule in the cache, need to do the following steps first
	// step 1: Turn the V1 rule to the real V2 rule
	// step 2: Find the corresponding route to the V2 rules to be modified in the V1 rules, set their rules ID
	// step 3: Store persistence
	if _, ok := s.Cache().RoutingConfig().IsConvertFromV1(req.Id); ok {
		resp := s.transferV1toV2OnModify(ctx, req)
		if resp.GetCode().GetValue() != uint32(apimodel.Code_ExecuteSuccess) {
			return resp
		}
	}

	if resp := checkUpdateRoutingConfigV2(req); resp != nil {
		return resp
	}

	// Check whether the routing configuration exists
	conf, err := s.storage.GetRoutingConfigV2WithID(req.Id)
	if err != nil {
		log.Error("[Routing][V2] get routing config v2 store layer",
			utils.RequestID(ctx), zap.Error(err))
		return apiv1.NewResponse(commonstore.StoreCode2APICode(err))
	}
	if conf == nil {
		return apiv1.NewResponse(apimodel.Code_NotFoundRouting)
	}

	reqModel, err := Api2RoutingConfigV2(req)
	reqModel.Revision = utils.NewV2Revision()
	if err != nil {
		log.Error("[Routing][V2] parse routing config v2 from request for update",
			utils.RequestID(ctx), zap.Error(err))
		return apiv1.NewResponse(apimodel.Code_ExecuteException)
	}

	if err := s.storage.UpdateRoutingConfigV2(reqModel); err != nil {
		log.Error("[Routing][V2] update routing config v2 store layer",
			utils.RequestID(ctx), zap.Error(err))
		return apiv1.NewResponse(commonstore.StoreCode2APICode(err))
	}

	s.RecordHistory(ctx, routingV2RecordEntry(ctx, req, reqModel, model.OUpdate))
	return apiv1.NewResponse(apimodel.Code_ExecuteSuccess)
}

// QueryRoutingConfigsV2 The interface of the query configuration to the OSS
func (s *Server) QueryRoutingConfigsV2(ctx context.Context, query map[string]string) *apiservice.BatchQueryResponse {
	args, presp := parseRoutingArgs(query, ctx)
	if presp != nil {
		return apiv1.NewBatchQueryResponse(apimodel.Code(presp.GetCode().GetValue()))
	}

	total, ret, err := s.Cache().RoutingConfig().QueryRoutingConfigsV2(ctx, args)
	if err != nil {
		log.Error("[Routing][V2] query routing list from cache", utils.RequestID(ctx), zap.Error(err))
		return apiv1.NewBatchQueryResponse(apimodel.Code_ExecuteException)
	}

	routers, err := marshalRoutingV2toAnySlice(ret)
	if err != nil {
		log.Error("[Routing][V2] marshal routing list to anypb.Any list",
			utils.RequestID(ctx), zap.Error(err))
		return apiv1.NewBatchQueryResponse(apimodel.Code_ExecuteException)
	}

	resp := apiv1.NewBatchQueryResponse(apimodel.Code_ExecuteSuccess)
	resp.Amount = &wrappers.UInt32Value{Value: total}
	resp.Size = &wrappers.UInt32Value{Value: uint32(len(ret))}
	resp.Data = routers
	return resp
}

// EnableRoutings batch enable routing rules
func (s *Server) EnableRoutings(ctx context.Context, req []*apitraffic.RouteRule) *apiservice.BatchWriteResponse {
	out := apiv1.NewBatchWriteResponse(apimodel.Code_ExecuteSuccess)
	for _, entry := range req {
		resp := s.enableRoutings(ctx, entry)
		apiv1.Collect(out, resp)
	}

	return apiv1.FormatBatchWriteResponse(out)
}

func (s *Server) enableRoutings(ctx context.Context, req *apitraffic.RouteRule) *apiservice.Response {
	if resp := checkRoutingConfigIDV2(req); resp != nil {
		return resp
	}

	if _, ok := s.Cache().RoutingConfig().IsConvertFromV1(req.Id); ok {
		resp := s.transferV1toV2OnModify(ctx, req)
		if resp.GetCode().GetValue() != uint32(apimodel.Code_ExecuteSuccess) {
			return resp
		}
	}

	conf, err := s.storage.GetRoutingConfigV2WithID(req.Id)
	if err != nil {
		log.Error("[Routing][V2] get routing config v2 store layer",
			utils.RequestID(ctx), zap.Error(err))
		return apiv1.NewResponse(commonstore.StoreCode2APICode(err))
	}
	if conf == nil {
		return apiv1.NewResponse(apimodel.Code_NotFoundRouting)
	}

	conf.Enable = req.GetEnable()
	conf.Revision = utils.NewV2Revision()

	if err := s.storage.EnableRouting(conf); err != nil {
		log.Error("[Routing][V2] enable routing config v2 store layer",
			utils.RequestID(ctx), zap.Error(err))
		return apiv1.NewResponse(commonstore.StoreCode2APICode(err))
	}

	s.RecordHistory(ctx, routingV2RecordEntry(ctx, req, conf, model.OUpdate))
	return apiv1.NewResponse(apimodel.Code_ExecuteSuccess)
}

// transferV1toV2OnModify When enabled or prohibited for the V2 rules, the V1 rules need to be converted to V2 rules
// and execute persistent storage
func (s *Server) transferV1toV2OnModify(ctx context.Context, req *apitraffic.RouteRule) *apiservice.Response {
	svcId, _ := s.Cache().RoutingConfig().IsConvertFromV1(req.Id)
	v1conf, err := s.storage.GetRoutingConfigWithID(svcId)
	if err != nil {
		log.Error("[Routing][V2] get routing config v1 store layer",
			utils.RequestID(ctx), zap.Error(err))
		return apiv1.NewResponse(commonstore.StoreCode2APICode(err))
	}
	if v1conf != nil {
		svc, err := s.loadServiceByID(svcId)
		if svc == nil {
			log.Error("[Routing][V2] convert routing config v1 to v2 find svc",
				utils.RequestID(ctx), zap.Error(err))
			return apiv1.NewResponse(apimodel.Code_NotFoundService)
		}

		inV2, outV2, err := model.ConvertRoutingV1ToExtendV2(svc.Name, svc.Namespace, v1conf)
		if err != nil {
			log.Error("[Routing][V2] convert routing config v1 to v2",
				utils.RequestID(ctx), zap.Error(err))
			return apiv1.NewResponse(apimodel.Code_ExecuteException)
		}

		formatApi := func(rules []*model.ExtendRouterConfig) ([]*apitraffic.RouteRule, *apiservice.Response) {
			ret := make([]*apitraffic.RouteRule, 0, len(rules))
			for i := range rules {
				item, err := rules[i].ToApi()
				if err != nil {
					log.Error("[Routing][V2] convert routing config v1 to v2, format v2 to api",
						utils.RequestID(ctx), zap.Error(err))
					return nil, apiv1.NewResponse(apimodel.Code_ExecuteException)
				}
				ret = append(ret, item)
			}

			return ret, nil
		}

		inDatas, resp := formatApi(inV2)
		if resp != nil {
			return resp
		}
		outDatas, resp := formatApi(outV2)
		if resp != nil {
			return resp
		}

		if resp := s.saveRoutingV1toV2(ctx, svcId, inDatas, outDatas); resp.GetCode().GetValue() != apiv1.ExecuteSuccess {
			return apiv1.NewResponse(apimodel.Code(resp.GetCode().GetValue()))
		}
	}

	return apiv1.NewResponse(apimodel.Code_ExecuteSuccess)
}

// parseServiceArgs The query conditions of the analysis service
func parseRoutingArgs(query map[string]string, ctx context.Context) (*cachetypes.RoutingArgs, *apiservice.Response) {
	offset, limit, err := utils.ParseOffsetAndLimit(query)
	if err != nil {
		return nil, apiv1.NewResponse(apimodel.Code_InvalidParameter)
	}

	filter := make(map[string]string)
	for key, value := range query {
		if _, ok := RoutingConfigV2FilterAttrs[key]; !ok {
			log.Errorf("[Routing][V2][Query] attribute(%s) is not allowed", key)
			return nil, apiv1.NewResponse(apimodel.Code_InvalidParameter)
		}
		filter[key] = value
	}

	res := &cachetypes.RoutingArgs{
		Filter:     filter,
		Name:       filter["name"],
		ID:         filter["id"],
		OrderField: filter["order_field"],
		OrderType:  filter["order_type"],
		Offset:     offset,
		Limit:      limit,
	}

	if _, ok := filter["service"]; ok {
		res.Namespace = filter["namespace"]
		res.Service = filter["service"]
	} else {
		res.SourceService = filter["source_service"]
		res.SourceNamespace = filter["source_namespace"]

		res.DestinationService = filter["destination_service"]
		res.DestinationNamespace = filter["destination_namespace"]
	}

	if enableStr, ok := filter["enable"]; ok {
		enable, err := strconv.ParseBool(enableStr)
		if err == nil {
			res.Enable = &enable
		} else {
			log.Error("[Service][Routing][Query] search with routing enable", zap.Error(err))
		}
	}
	log.Infof("[Service][Routing][Query] routing query args: %+v", res)
	return res, nil
}

// checkBatchRoutingConfig Check batch request
func checkBatchRoutingConfigV2(req []*apitraffic.RouteRule) *apiservice.BatchWriteResponse {
	if len(req) == 0 {
		return apiv1.NewBatchWriteResponse(apimodel.Code_EmptyRequest)
	}

	if len(req) > MaxBatchSize {
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
	if err := utils.CheckDbStrFieldLen(utils.NewStringValue(req.GetName()), MaxDbRoutingName); err != nil {
		return apiv1.NewRouterResponse(apimodel.Code_InvalidRoutingName, req)
	}

	if err := utils.CheckDbStrFieldLen(utils.NewStringValue(req.GetNamespace()),
		MaxDbServiceNamespaceLength); err != nil {
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
	}

	return nil
}

// Api2RoutingConfigV2 Convert the API parameter to internal data structure
func Api2RoutingConfigV2(req *apitraffic.RouteRule) (*model.RouterConfig, error) {
	out := &model.RouterConfig{
		Valid: true,
	}

	if req.Id == "" {
		req.Id = utils.NewRoutingV2UUID()
	}
	if req.Revision == "" {
		req.Revision = utils.NewV2Revision()
	}

	if err := out.ParseRouteRuleFromAPI(req); err != nil {
		return nil, err
	}
	return out, nil
}

// marshalRoutingV2toAnySlice Converted to []*anypb.Any array
func marshalRoutingV2toAnySlice(routings []*model.ExtendRouterConfig) ([]*any.Any, error) {
	ret := make([]*any.Any, 0, len(routings))

	for i := range routings {
		entry, err := routings[i].ToApi()
		if err != nil {
			return nil, err
		}
		item, err := ptypes.MarshalAny(entry)
		if err != nil {
			return nil, err
		}

		ret = append(ret, item)
	}

	return ret, nil
}
