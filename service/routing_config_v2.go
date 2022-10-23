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
	"go.uber.org/zap"

	"github.com/polarismesh/polaris/cache"
	apiv1 "github.com/polarismesh/polaris/common/api/v1"
	apiv2 "github.com/polarismesh/polaris/common/api/v2"
	"github.com/polarismesh/polaris/common/model"
	v2 "github.com/polarismesh/polaris/common/model/v2"
	routingcommon "github.com/polarismesh/polaris/common/routing"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/store"
)

var (
	// RoutingConfigV2FilterAttrs router config filter attrs
	RoutingConfigV2FilterAttrs = map[string]bool{
		"id":             true,
		"name":           true,
		"service":        true,
		"namespace":      true,
		"source_service": true,
		"enable":         true,
		"offset":         true,
		"limit":          true,
		"order_field":    true,
		"order_type":     true,
	}
)

// CreateRoutingConfigsV2 批量创建路由配置
func (s *Server) CreateRoutingConfigsV2(ctx context.Context, req []*apiv2.Routing) *apiv2.BatchWriteResponse {
	if err := checkBatchRoutingConfigV2(req); err != nil {
		return err
	}

	resp := apiv2.NewBatchWriteResponse(apiv1.ExecuteSuccess)
	for _, entry := range req {
		resp.Collect(s.createRoutingConfigV2(ctx, entry))
	}

	return apiv2.FormatBatchWriteResponse(resp)
}

// createRoutingConfigV2 创建一个路由配置
func (s *Server) createRoutingConfigV2(ctx context.Context, req *apiv2.Routing) *apiv2.Response {
	if resp := checkRoutingConfigV2(req); resp != nil {
		return resp
	}

	// 构造底层数据结构，并且写入store
	conf, err := api2RoutingConfigV2(req)
	if err != nil {
		log.Error("[Routing][V2] parse routing config v2 from request for create",
			utils.ZapRequestIDByCtx(ctx), zap.Error(err))
		return apiv2.NewResponse(apiv1.ExecuteException)
	}

	if err := s.storage.CreateRoutingConfigV2(conf); err != nil {
		log.Error("[Routing][V2] create routing config v2 store layer",
			utils.ZapRequestIDByCtx(ctx), zap.Error(err))
		return apiv2.NewResponse(apiv1.StoreLayerException)
	}

	s.RecordHistory(routingV2RecordEntry(ctx, req, conf, model.OCreate))

	req.Id = conf.ID
	return apiv2.NewRoutingResponse(apiv1.ExecuteSuccess, req)
}

// DeleteRoutingConfigsV2 批量删除路由配置
func (s *Server) DeleteRoutingConfigsV2(ctx context.Context, req []*apiv2.Routing) *apiv2.BatchWriteResponse {
	if err := checkBatchRoutingConfigV2(req); err != nil {
		return err
	}

	out := apiv2.NewBatchWriteResponse(apiv1.ExecuteSuccess)
	for _, entry := range req {
		resp := s.deleteRoutingConfigV2(ctx, entry)
		out.Collect(resp)
	}

	return apiv2.FormatBatchWriteResponse(out)
}

// DeleteRoutingConfigV2 删除一个路由配置
func (s *Server) deleteRoutingConfigV2(ctx context.Context, req *apiv2.Routing) *apiv2.Response {
	if resp := checkRoutingConfigIDV2(req); resp != nil {
		return resp
	}

	// 判断当前的路由规则是否只是从 v1 版本中的内存中转换过来的
	if _, ok := s.Cache().RoutingConfig().IsConvertFromV1(req.Id); ok {
		resp := s.transferV1toV2OnModify(ctx, req)
		if resp.GetCode() != apiv1.ExecuteSuccess {
			return resp
		}
	}

	if err := s.storage.DeleteRoutingConfigV2(req.Id); err != nil {
		log.Error("[Routing][V2] delete routing config v2 store layer",
			utils.ZapRequestIDByCtx(ctx), zap.Error(err))
		return apiv2.NewResponse(apiv1.StoreLayerException)
	}

	s.RecordHistory(routingV2RecordEntry(ctx, req, nil, model.ODelete))
	return apiv2.NewRoutingResponse(apiv1.ExecuteSuccess, req)
}

// UpdateRoutingConfigsV2 批量更新路由配置
func (s *Server) UpdateRoutingConfigsV2(ctx context.Context, req []*apiv2.Routing) *apiv2.BatchWriteResponse {
	if err := checkBatchRoutingConfigV2(req); err != nil {
		return err
	}

	out := apiv2.NewBatchWriteResponse(apiv1.ExecuteSuccess)
	for _, entry := range req {
		resp := s.updateRoutingConfigV2(ctx, entry)
		out.Collect(resp)
	}

	return apiv2.FormatBatchWriteResponse(out)
}

// updateRoutingConfigV2 更新单个路由配置
func (s *Server) updateRoutingConfigV2(ctx context.Context, req *apiv2.Routing) *apiv2.Response {
	// 如果当前待修改的 v2 路由规则，其实是从 v1 规则在 cache 中转换而来的，则需要先做以下几个步骤
	// step 1: 将 v1 规则真实的转为 v2 规则
	// step 2: 将本次要修改的 v2 规则，在 v1 规则中的 inBound 或者 outBound 找到对应的 route，设置其规则 ID
	// step 3: 进行存储持久化
	// 判断当前的路由规则是否只是从 v1 版本中的内存中转换过来的
	if _, ok := s.Cache().RoutingConfig().IsConvertFromV1(req.Id); ok {
		resp := s.transferV1toV2OnModify(ctx, req)
		if resp.GetCode() != apiv1.ExecuteSuccess {
			return resp
		}
	}

	if resp := checkUpdateRoutingConfigV2(req); resp != nil {
		return resp
	}

	// 检查路由配置是否存在
	conf, err := s.storage.GetRoutingConfigV2WithID(req.Id)
	if err != nil {
		log.Error("[Routing][V2] get routing config v2 store layer",
			utils.ZapRequestIDByCtx(ctx), zap.Error(err))
		return apiv2.NewResponse(apiv1.StoreLayerException)
	}
	if conf == nil {
		return apiv2.NewResponse(apiv1.NotFoundRouting)
	}

	// 作为一个整体进行Update，所有参数都要传递
	reqModel, err := api2RoutingConfigV2(req)
	reqModel.Revision = utils.NewV2Revision()
	if err != nil {
		log.Error("[Routing][V2] parse routing config v2 from request for update",
			utils.ZapRequestIDByCtx(ctx), zap.Error(err))
		return apiv2.NewResponse(apiv1.ExecuteException)
	}

	if err := s.storage.UpdateRoutingConfigV2(reqModel); err != nil {
		log.Error("[Routing][V2] update routing config v2 store layer",
			utils.ZapRequestIDByCtx(ctx), zap.Error(err))
		return apiv2.NewResponse(apiv1.StoreLayerException)
	}

	s.RecordHistory(routingV2RecordEntry(ctx, req, reqModel, model.OUpdate))
	return apiv2.NewResponse(apiv1.ExecuteSuccess)
}

// GetRoutingConfigsV2 提供给OSS的查询路由配置的接口
func (s *Server) GetRoutingConfigsV2(ctx context.Context, query map[string]string) *apiv2.BatchQueryResponse {
	args, presp := parseRoutingArgs(query, ctx)
	if presp != nil {
		return apiv2.NewBatchQueryResponse(presp.GetCode())
	}

	total, ret, err := s.Cache().RoutingConfig().GetRoutingConfigsV2(args)
	if err != nil {
		log.Error("[Routing][V2] query routing list from cache", utils.ZapRequestIDByCtx(ctx), zap.Error(err))
		return apiv2.NewBatchQueryResponse(apiv1.ExecuteException)
	}

	data, err := marshalRoutingV2toAnySlice(ret)
	if err != nil {
		log.Error("[Routing][V2] marshal routing list to anypb.Any list",
			utils.ZapRequestIDByCtx(ctx), zap.Error(err))
		return apiv2.NewBatchQueryResponse(apiv1.ExecuteException)
	}

	resp := apiv2.NewBatchQueryResponse(apiv1.ExecuteSuccess)
	resp.Amount = total
	resp.Size = uint32(len(ret))
	resp.Data = data
	return resp
}

// EnableRoutings batch enable routing rules
func (s *Server) EnableRoutings(ctx context.Context, req []*apiv2.Routing) *apiv2.BatchWriteResponse {
	out := apiv2.NewBatchWriteResponse(apiv1.ExecuteSuccess)
	for _, entry := range req {
		resp := s.enableRoutings(ctx, entry)
		out.Collect(resp)
	}

	return apiv2.FormatBatchWriteResponse(out)
}

func (s *Server) enableRoutings(ctx context.Context, req *apiv2.Routing) *apiv2.Response {
	if resp := checkRoutingConfigIDV2(req); resp != nil {
		return resp
	}

	// 判断当前的路由规则是否只是从 v1 版本中的内存中转换过来的
	if _, ok := s.Cache().RoutingConfig().IsConvertFromV1(req.Id); ok {
		resp := s.transferV1toV2OnModify(ctx, req)
		if resp.GetCode() != apiv1.ExecuteSuccess {
			return resp
		}
	}

	// 检查路由配置是否存在
	conf, err := s.storage.GetRoutingConfigV2WithID(req.Id)
	if err != nil {
		log.Error("[Routing][V2] get routing config v2 store layer",
			utils.ZapRequestIDByCtx(ctx), zap.Error(err))
		return apiv2.NewResponse(apiv1.StoreLayerException)
	}
	if conf == nil {
		return apiv2.NewResponse(apiv1.NotFoundRouting)
	}

	conf.Enable = req.GetEnable()
	conf.Revision = utils.NewV2Revision()

	if err := s.storage.EnableRouting(conf); err != nil {
		log.Error("[Routing][V2] enable routing config v2 store layer",
			utils.ZapRequestIDByCtx(ctx), zap.Error(err))
		return apiv2.NewResponse(apiv1.StoreLayerException)
	}

	s.RecordHistory(routingV2RecordEntry(ctx, req, conf, model.OUpdate))
	return apiv2.NewResponse(apiv1.ExecuteSuccess)
}

// transferV1toV2OnModify 在针对 v2 规则进行启用或者禁止时，需要将 v1 规则转为 v2 规则并执行持久化存储
func (s *Server) transferV1toV2OnModify(ctx context.Context, req *apiv2.Routing) *apiv2.Response {
	svcId, _ := s.Cache().RoutingConfig().IsConvertFromV1(req.Id)
	v1conf, err := s.storage.GetRoutingConfigWithID(svcId)
	if err != nil {
		log.Error("[Routing][V2] get routing config v1 store layer",
			utils.ZapRequestIDByCtx(ctx), zap.Error(err))
		return apiv2.NewResponse(apiv1.StoreLayerException)
	}
	if v1conf != nil {
		svc := s.Cache().Service().GetServiceByID(svcId)
		if svc == nil {
			_svc, err := s.storage.GetServiceByID(svcId)
			if err != nil {
				return nil
			}
			svc = _svc
		}
		if svc == nil {
			log.Error("[Routing][V2] convert routing config v1 to v2 find svc",
				utils.ZapRequestIDByCtx(ctx), zap.Error(err))
			return apiv2.NewResponse(apiv1.NotFoundService)
		}

		// 这里需要将 apiModel 的 id 全部生成一遍 extendInfo 信息
		inV2, outV2, err := routingcommon.ConvertRoutingV1ToExtendV2(svc.Name, svc.Namespace, v1conf)
		if err != nil {
			log.Error("[Routing][V2] convert routing config v1 to v2",
				utils.ZapRequestIDByCtx(ctx), zap.Error(err))
			return apiv2.NewResponse(apiv1.ExecuteException)
		}

		formatApi := func(rules []*v2.ExtendRoutingConfig) ([]*apiv2.Routing, *apiv2.Response) {
			ret := make([]*apiv2.Routing, 0, len(rules))
			for i := range rules {
				item, err := rules[i].ToApi()
				if err != nil {
					log.Error("[Routing][V2] convert routing config v1 to v2, format v2 to api",
						utils.ZapRequestIDByCtx(ctx), zap.Error(err))
					return nil, apiv2.NewResponse(apiv1.ExecuteException)
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
			return apiv2.NewResponse(resp.GetCode().GetValue())
		}
	}

	return apiv2.NewResponse(apiv1.ExecuteSuccess)
}

// parseServiceArgs 解析服务的查询条件
func parseRoutingArgs(query map[string]string, ctx context.Context) (*cache.RoutingArgs, *apiv2.Response) {
	// 先处理offset和limit
	offset, limit, err := utils.ParseOffsetAndLimit(query)
	if err != nil {
		return nil, apiv2.NewResponse(apiv1.InvalidParameter)
	}

	filter := make(map[string]string)
	for key, value := range query {
		if _, ok := RoutingConfigV2FilterAttrs[key]; !ok {
			log.Errorf("[Routing][V2][Query] attribute(%s) is not allowed", key)
			return nil, apiv2.NewResponse(apiv1.InvalidParameter)
		}
		filter[key] = value
	}

	res := &cache.RoutingArgs{
		Filter:     filter,
		ID:         filter["id"],
		Namespace:  filter["namespace"],
		Service:    filter["service"],
		OrderField: filter["order_field"],
		OrderType:  filter["order_type"],
		Offset:     offset,
		Limit:      limit,
	}
	var ok bool
	if res.Name, ok = filter["name"]; ok && store.IsWildName(res.Name) {
		log.Infof("[Routing][V2][Query] fuzzy search with name %s", res.Name)
		res.FuzzyName = true
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

// checkBatchRoutingConfig 检查批量请求
func checkBatchRoutingConfigV2(req []*apiv2.Routing) *apiv2.BatchWriteResponse {
	if len(req) == 0 {
		return apiv2.NewBatchWriteResponse(apiv1.EmptyRequest)
	}

	if len(req) > MaxBatchSize {
		return apiv2.NewBatchWriteResponse(apiv1.BatchSizeOverLimit)
	}

	return nil
}

// checkRoutingConfig 检查路由配置基础参数有效性
func checkRoutingConfigV2(req *apiv2.Routing) *apiv2.Response {
	if req == nil {
		return apiv2.NewRoutingResponse(apiv1.EmptyRequest, req)
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

// checkUpdateRoutingConfigV2 检查路由配置基础参数有效性
func checkUpdateRoutingConfigV2(req *apiv2.Routing) *apiv2.Response {
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

func checkRoutingNameAndNamespace(req *apiv2.Routing) *apiv2.Response {
	if err := utils.CheckDbStrFieldLen(utils.NewStringValue(req.GetName()), MaxDbRoutingName); err != nil {
		return apiv2.NewRoutingResponse(apiv1.InvalidRoutingName, req)
	}

	if err := utils.CheckDbStrFieldLen(utils.NewStringValue(req.GetNamespace()),
		MaxDbServiceNamespaceLength); err != nil {
		return apiv2.NewRoutingResponse(apiv1.InvalidNamespaceName, req)
	}

	return nil
}

func checkRoutingConfigIDV2(req *apiv2.Routing) *apiv2.Response {
	if req == nil {
		return apiv2.NewRoutingResponse(apiv1.EmptyRequest, req)
	}

	if req.Id == "" {
		return apiv2.NewResponse(apiv1.InvalidRoutingID)
	}

	return nil
}

func checkRoutingConfigPriorityV2(req *apiv2.Routing) *apiv2.Response {
	if req == nil {
		return apiv2.NewRoutingResponse(apiv1.EmptyRequest, req)
	}

	if req.Priority < 0 || req.Priority > 10 {
		return apiv2.NewResponse(apiv1.InvalidRoutingPriority)
	}

	return nil
}

func checkRoutingPolicyV2(req *apiv2.Routing) *apiv2.Response {
	if req == nil {
		return apiv2.NewRoutingResponse(apiv1.EmptyRequest, req)
	}

	if req.GetRoutingPolicy() != apiv2.RoutingPolicy_RulePolicy {
		return apiv2.NewRoutingResponse(apiv1.InvalidRoutingPolicy, req)
	}

	// 自动根据 policy 补充 @type 属性
	if req.RoutingConfig.TypeUrl == "" {
		if req.GetRoutingPolicy() == apiv2.RoutingPolicy_RulePolicy {
			req.RoutingConfig.TypeUrl = v2.RuleRoutingTypeUrl
		}
		if req.GetRoutingPolicy() == apiv2.RoutingPolicy_MetadataPolicy {
			req.RoutingConfig.TypeUrl = v2.MetaRoutingTypeUrl
		}
	}

	return nil
}

// api2RoutingConfig 把API参数转换为内部的数据结构
func api2RoutingConfigV2(req *apiv2.Routing) (*v2.RoutingConfig, error) {
	out := &v2.RoutingConfig{
		Valid: true,
	}

	if req.Id == "" {
		req.Id = utils.NewRoutingV2UUID()
	}
	if req.Revision == "" {
		req.Revision = utils.NewV2Revision()
	}

	if err := out.ParseFromAPI(req); err != nil {
		return nil, err
	}
	return out, nil
}

// marshalRoutingV2toAnySlice 转换为 []*anypb.Any 数组
func marshalRoutingV2toAnySlice(routings []*v2.ExtendRoutingConfig) ([]*any.Any, error) {
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
