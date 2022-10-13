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

	"go.uber.org/zap"

	api "github.com/polarismesh/polaris/common/api/v1"
	apiv2 "github.com/polarismesh/polaris/common/api/v2"
	"github.com/polarismesh/polaris/common/utils"
)

// GetRoutingConfigWithCache User Client Get Service Routing Configuration Information
func (s *Server) GetRoutingConfigV2WithCache(ctx context.Context, req *apiv2.Service) *apiv2.DiscoverResponse {
	if s.caches == nil {
		return apiv2.NewDiscoverRoutingResponse(api.ClientAPINotOpen, req)
	}
	if req == nil {
		return apiv2.NewDiscoverRoutingResponse(api.EmptyRequest, req)
	}

	if req.GetName() == "" {
		return apiv2.NewDiscoverRoutingResponse(api.InvalidServiceName, req)
	}
	if req.GetNamespace() == "" {
		return apiv2.NewDiscoverRoutingResponse(api.InvalidNamespaceName, req)
	}

	resp := apiv2.NewDiscoverRoutingResponse(api.ExecuteSuccess, nil)
	resp.Service = &apiv2.Service{
		Name:      req.GetName(),
		Namespace: req.GetNamespace(),
	}

	// 先从缓存获取ServiceID，这里返回的是源服务
	svc := s.getServiceCache(req.GetName(), req.GetNamespace())
	if svc == nil {
		return apiv2.NewDiscoverRoutingResponse(api.NotFoundService, req)
	}
	out, err := s.caches.RoutingConfig().GetRoutingConfigV2(svc.ID, svc.Name, svc.Namespace)
	if err != nil {
		log.Error("[Server][Service][Routing] discover routing v2", utils.ZapRequestIDByCtx(ctx), zap.Error(err))
		return apiv2.NewDiscoverRoutingResponse(api.ExecuteException, req)
	}

	if out == nil {
		return resp
	}

	// // 获取路由数据，并对比revision
	// if out.GetRevision() == req.GetRevision() {
	// 	return apiv2.NewDiscoverRoutingResponse(api.DataNoChange, req)
	// }

	// 数据不一致，发生了改变
	// 数据格式转换，service只需要返回二元组与routing的revision
	resp.Service.Revision = utils.NewV2Revision()
	resp.Routings = out
	return resp
}

// GetCircuitBreakerWithCache Fuse configuration information for obtaining services for clients
func (s *Server) GetCircuitBreakerV2WithCache(ctx context.Context, req *apiv2.Service) *apiv2.DiscoverResponse {
	return apiv2.NewDiscoverRoutingResponse(api.NotFoundService, req)
}
