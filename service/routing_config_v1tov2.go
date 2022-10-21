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

	apiv1 "github.com/polarismesh/polaris/common/api/v1"
	apiv2 "github.com/polarismesh/polaris/common/api/v2"
	"github.com/polarismesh/polaris/common/model"
	v2 "github.com/polarismesh/polaris/common/model/v2"
	routingcommon "github.com/polarismesh/polaris/common/routing"
	"github.com/polarismesh/polaris/common/utils"
)

// createRoutingConfigV1toV2 这里需要兼容 v1 版本的创建路由规则动作，将 v1 的数据转为 v2 进行存储
func (s *Server) createRoutingConfigV1toV2(ctx context.Context, req *apiv1.Routing) *apiv1.Response {
	if resp := checkRoutingConfig(req); resp != nil {
		return resp
	}

	serviceTx, err := s.storage.CreateTransaction()
	if err != nil {
		log.Error(err.Error(), utils.ZapRequestIDByCtx(ctx))
		return apiv1.NewRoutingResponse(apiv1.StoreLayerException, req)
	}
	// 释放对于服务的锁
	defer serviceTx.Commit()

	serviceName := req.GetService().GetValue()
	namespaceName := req.GetNamespace().GetValue()
	svc, err := serviceTx.RLockService(serviceName, namespaceName)
	if err != nil {
		log.Error("[Service][Routing] get read lock for service", zap.String("service", serviceName),
			zap.String("namespace", namespaceName), utils.ZapRequestIDByCtx(ctx), zap.Error(err))
		return apiv1.NewRoutingResponse(apiv1.StoreLayerException, req)
	}
	if svc == nil {
		return apiv1.NewRoutingResponse(apiv1.NotFoundService, req)
	}
	if svc.IsAlias() {
		return apiv1.NewRoutingResponse(apiv1.NotAllowAliasCreateRouting, req)
	}

	inDatas, outDatas, resp := batchBuildV2Routings(req)
	if resp != nil {
		return resp
	}

	if resp := s.saveRoutingV1toV2(ctx, svc.ID, inDatas, outDatas); resp.GetCode().GetValue() != apiv1.ExecuteSuccess {
		return resp
	}

	return apiv1.NewRoutingResponse(apiv1.ExecuteSuccess, req)
}

// updateRoutingConfigV1toV2 这里需要兼容 v1 版本的更新路由规则动作，将 v1 的数据转为 v2 进行存储
// 一旦将 v1 规则转为 v2 规则，那么原本的 v1 规则将从存储中移除
func (s *Server) updateRoutingConfigV1toV2(ctx context.Context, req *apiv1.Routing) *apiv1.Response {
	svc, resp := s.routingConfigCommonCheck(ctx, req)
	if resp != nil {
		return resp
	}

	serviceTx, err := s.storage.CreateTransaction()
	if err != nil {
		log.Error(err.Error(), utils.ZapRequestIDByCtx(ctx))
		return apiv1.NewRoutingResponse(apiv1.StoreLayerException, req)
	}
	// 释放对于服务的锁
	defer serviceTx.Commit()

	// 需要禁止对 v1 规则的并发修改
	_, err = serviceTx.LockService(svc.Name, svc.Namespace)
	if err != nil {
		log.Error("[Service][Routing] get service x-lock", zap.String("service", svc.Name),
			zap.String("namespace", svc.Namespace), utils.ZapRequestIDByCtx(ctx), zap.Error(err))
		return apiv1.NewRoutingResponse(apiv1.StoreLayerException, req)
	}

	// 检查路由配置是否存在
	conf, err := s.storage.GetRoutingConfigWithService(svc.Name, svc.Namespace)
	if err != nil {
		log.Error(err.Error(), utils.ZapRequestIDByCtx(ctx))
		return apiv1.NewRoutingResponse(apiv1.StoreLayerException, req)
	}
	if conf == nil {
		return apiv1.NewRoutingResponse(apiv1.NotFoundRouting, req)
	}

	inDatas, outDatas, resp := batchBuildV2Routings(req)
	if resp != nil {
		return resp
	}

	if resp := s.saveRoutingV1toV2(ctx, svc.ID, inDatas, outDatas); resp.GetCode().GetValue() != apiv1.ExecuteSuccess {
		return resp
	}

	return apiv1.NewRoutingResponse(apiv1.ExecuteSuccess, req)
}

// saveRoutingV1toV2 将目标的 v1 规则转为 v2 规则
func (s *Server) saveRoutingV1toV2(ctx context.Context, svcId string,
	inRules, outRules []*apiv2.Routing) *apiv1.Response {

	tx, err := s.storage.StartTx()
	if err != nil {
		log.Error("[Service][Routing] create routing v2 from v1 open tx",
			utils.ZapRequestIDByCtx(ctx), zap.Error(err))
		return apiv1.NewResponse(apiv1.StoreLayerException)
	}
	defer tx.Rollback()

	// 这里需要删除掉 v1 的路由规则
	if err := s.storage.DeleteRoutingConfigTx(tx, svcId); err != nil {
		log.Error("[Service][Routing] clean routing v1 from store",
			utils.ZapRequestIDByCtx(ctx), zap.Error(err))
		return apiv1.NewResponse(apiv1.StoreLayerException)
	}

	saveOperation := func(routings []*apiv2.Routing) *apiv1.Response {
		priorityMax := 0
		for i := range routings {
			item := routings[i]
			if item.Id == "" {
				item.Id = utils.NewRoutingV2UUID()
			}
			item.Revision = utils.NewV2Revision()
			data := &v2.RoutingConfig{}
			if err := data.ParseFromAPI(item); err != nil {
				return apiv1.NewResponse(apiv1.ExecuteException)
			}

			data.Valid = true
			data.Enable = true
			if priorityMax > 10 {
				priorityMax = 10
			}

			data.Priority = uint32(priorityMax)
			priorityMax++

			if err := s.storage.CreateRoutingConfigV2Tx(tx, data); err != nil {
				log.Error("[Routing][V2] create routing v2 from v1 into store",
					utils.ZapRequestIDByCtx(ctx), zap.Error(err))
				return apiv1.NewResponse(apiv1.StoreLayerException)
			}
			s.RecordHistory(routingV2RecordEntry(ctx, item, data, model.OCreate))
		}

		return nil
	}

	if resp := saveOperation(inRules); resp != nil {
		return resp
	}
	if resp := saveOperation(outRules); resp != nil {
		return resp
	}

	if err := tx.Commit(); err != nil {
		log.Error("[Service][Routing] create routing v2 from v1 commit",
			utils.ZapRequestIDByCtx(ctx), zap.Error(err))
		return apiv1.NewResponse(apiv1.ExecuteException)
	}

	return apiv1.NewResponse(apiv1.ExecuteSuccess)
}

func batchBuildV2Routings(req *apiv1.Routing) ([]*apiv2.Routing, []*apiv2.Routing, *apiv1.Response) {
	inBounds := req.GetInbounds()
	outBounds := req.GetOutbounds()
	inRoutings := make([]*apiv2.Routing, 0, len(inBounds))
	outRoutings := make([]*apiv2.Routing, 0, len(outBounds))
	for i := range inBounds {
		routing, err := routingcommon.BuildV2RoutingFromV1Route(req, inBounds[i])
		if err != nil {
			return nil, nil, apiv1.NewResponse(apiv1.ExecuteException)
		}
		routing.Name = req.GetNamespace().GetValue() + "." + req.GetService().GetValue()
		inRoutings = append(inRoutings, routing)
	}

	for i := range outBounds {
		routing, err := routingcommon.BuildV2RoutingFromV1Route(req, outBounds[i])
		if err != nil {
			return nil, nil, apiv1.NewResponse(apiv1.ExecuteException)
		}
		outRoutings = append(outRoutings, routing)
	}

	return inRoutings, outRoutings, nil
}
