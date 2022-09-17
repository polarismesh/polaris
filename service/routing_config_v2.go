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
	"github.com/polarismesh/polaris-server/cache"
	api "github.com/polarismesh/polaris-server/common/api/v1"
	apiv1 "github.com/polarismesh/polaris-server/common/api/v1"
	apiv2 "github.com/polarismesh/polaris-server/common/api/v2"
	"github.com/polarismesh/polaris-server/common/model"
	v2 "github.com/polarismesh/polaris-server/common/model/v2"
	"github.com/polarismesh/polaris-server/common/utils"
	"github.com/polarismesh/polaris-server/store"
	"go.uber.org/zap"
)

var (
	// RoutingConfigFilterAttrs router config filter attrs
	RoutingConfigV2FilterAttrs = map[string]bool{
		"name":           true,
		"namespace":      true,
		"source_service": true,
		"enable":         true,
		"offset":         true,
		"limit":          true,
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
		log.Error("[Service][Routing] parse routing config v2 from request for create",
			utils.ZapRequestIDByCtx(ctx), zap.Error(err))
		return apiv2.NewResponse(api.ExecuteException)
	}

	if err := s.storage.CreateRoutingConfigV2(conf); err != nil {
		log.Error("[Service][Routing] create routing config v2 store layer",
			utils.ZapRequestIDByCtx(ctx), zap.Error(err))
		return apiv2.NewResponse(api.StoreLayerException)
	}

	s.RecordHistory(routingV2RecordEntry(ctx, req, conf, model.OCreate))
	return apiv2.NewRoutingResponse(api.ExecuteSuccess, req)
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

	if err := s.storage.DeleteRoutingConfigV2(req.Id); err != nil {
		log.Error("[Service][Routing] delete routing config v2 store layer",
			utils.ZapRequestIDByCtx(ctx), zap.Error(err))
		return apiv2.NewResponse(api.StoreLayerException)
	}

	s.RecordHistory(routingV2RecordEntry(ctx, req, nil, model.ODelete))
	return apiv2.NewRoutingResponse(api.ExecuteSuccess, req)
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
	// 先判断是不是从 v1 过来的修改操作
	extendInfo := req.GetExtendInfo()
	if _, ok := extendInfo[v2.V1RuleIDKey]; ok {
		resp := s.updateRoutingConfigV2FromV1(ctx, req)
		if resp.GetCode() != apiv1.ExecuteSuccess {
			return resp
		}
	}

	// 检查路由配置是否存在
	conf, err := s.storage.GetRoutingConfigV2WithID(req.Id)
	if err != nil {
		log.Error("[Service][Routing] get routing config v2 store layer",
			utils.ZapRequestIDByCtx(ctx), zap.Error(err))
		return apiv2.NewResponse(api.StoreLayerException)
	}
	if conf == nil {
		return apiv2.NewResponse(api.NotFoundRouting)
	}

	// 作为一个整体进行Update，所有参数都要传递
	reqModel, err := api2RoutingConfigV2(req)
	if err != nil {
		log.Error("[Service][Routing] parse routing config v2 from request for update",
			utils.ZapRequestIDByCtx(ctx), zap.Error(err))
		return apiv2.NewResponse(api.ExecuteException)
	}

	if err := s.storage.UpdateRoutingConfigV2(reqModel); err != nil {
		log.Error("[Service][Routing] update routing config v2 store layer",
			utils.ZapRequestIDByCtx(ctx), zap.Error(err))
		return apiv2.NewResponse(apiv1.StoreLayerException)
	}

	s.RecordHistory(routingV2RecordEntry(ctx, req, reqModel, model.OUpdate))
	return apiv2.NewResponse(api.ExecuteSuccess)
}

// GetRoutingConfigsV2 提供给OSS的查询路由配置的接口
func (s *Server) GetRoutingConfigsV2(ctx context.Context, query map[string]string) *apiv2.BatchQueryResponse {
	// 先处理offset和limit
	offset, limit, err := ParseOffsetAndLimit(query)
	if err != nil {
		return apiv2.NewBatchQueryResponse(api.InvalidParameter)
	}

	// 处理剩余的参数
	filter := make(map[string]string)
	for key, value := range query {
		if _, ok := RoutingConfigV2FilterAttrs[key]; !ok {
			log.Errorf("[Server][Routing][Query] attribute(%s) is not allowed", key)
			return apiv2.NewBatchQueryResponse(api.InvalidParameter)
		}
		filter[key] = value
	}
	// service -- > name 这个特殊处理一下
	if service, ok := filter["service"]; ok {
		filter["name"] = service
		delete(filter, "service")
	}

	total, ret, err := s.Cache().RoutingConfig().GetRoutingConfigsV2(parseRoutingArgs(filter, offset, limit, ctx))
	if err != nil {
		log.Error("[Server][Routing] query routing list from cache", utils.ZapRequestIDByCtx(ctx), zap.Error(err))
		return apiv2.NewBatchQueryResponse(api.ExecuteException)
	}

	data, err := marshalRoutingV2toAnySlice(ret)
	if err != nil {
		log.Error("[Server][Routing] marshal routing list to anypb.Any list",
			utils.ZapRequestIDByCtx(ctx), zap.Error(err))
		return apiv2.NewBatchQueryResponse(api.ExecuteException)
	}

	resp := apiv2.NewBatchQueryResponse(api.ExecuteSuccess)
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
	// 检查路由配置是否存在
	conf, err := s.storage.GetRoutingConfigV2WithID(req.Id)
	if err != nil {
		log.Error("[Service][Routing] get routing config v2 store layer",
			utils.ZapRequestIDByCtx(ctx), zap.Error(err))
		return apiv2.NewResponse(api.StoreLayerException)
	}
	if conf == nil {
		return apiv2.NewResponse(api.NotFoundRouting)
	}

	conf.Enable = req.GetEnable()

	if err := s.storage.EnableRouting(conf); err != nil {
		log.Error("[Service][Routing] enable routing config v2 store layer",
			utils.ZapRequestIDByCtx(ctx), zap.Error(err))
		return apiv2.NewResponse(apiv1.StoreLayerException)
	}

	s.RecordHistory(routingV2RecordEntry(ctx, req, conf, model.OUpdate))
	return apiv2.NewResponse(api.ExecuteSuccess)
}

// parseServiceArgs 解析服务的查询条件
func parseRoutingArgs(filter map[string]string, offset, limit uint32, ctx context.Context) *cache.RoutingArgs {
	res := &cache.RoutingArgs{
		Filter:    filter,
		Namespace: filter["namespace"],
		Offset:    offset,
		Limit:     limit,
	}
	var ok bool
	if res.Name, ok = filter["name"]; ok && store.IsWildName(res.Name) {
		log.Infof("[Service][Routing][Query] fuzzy search with name %s", res.Name)
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
	return res
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
		return apiv2.NewRoutingResponse(api.EmptyRequest, req)
	}

	if err := checkResourceName(utils.NewStringValue(req.GetName())); err != nil {
		return apiv2.NewRoutingResponse(api.InvalidServiceName, req)
	}

	// if err := checkResourceName(utils.NewStringValue(req.GetNamespace())); err != nil {
	// 	return apiv2.NewRoutingResponse(api.InvalidNamespaceName, req)
	// }

	if err := CheckDbStrFieldLen(utils.NewStringValue(req.GetNamespace()),
		MaxDbServiceNamespaceLength); err != nil {
		return apiv2.NewRoutingResponse(api.InvalidNamespaceName, req)
	}

	return nil
}

// api2RoutingConfig 把API参数转换为内部的数据结构
func api2RoutingConfigV2(req *apiv2.Routing) (*v2.RoutingConfig, error) {

	out := &v2.RoutingConfig{
		ID:       req.Id,
		Revision: req.Revision,
		Valid:    true,
	}

	if out.ID == "" {
		out.ID = utils.NewRoutingV2UUID()
	}
	if out.Revision == "" {
		out.Revision = utils.NewV2Revision()
	}

	if err := out.ParseFromAPI(req); err != nil {
		return nil, err
	}

	return out, nil
}

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
