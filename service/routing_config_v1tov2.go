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

	api "github.com/polarismesh/polaris-server/common/api/v1"
	apiv1 "github.com/polarismesh/polaris-server/common/api/v1"
	apiv2 "github.com/polarismesh/polaris-server/common/api/v2"
	"github.com/polarismesh/polaris-server/common/model"
	v2 "github.com/polarismesh/polaris-server/common/model/v2"
	routingcommon "github.com/polarismesh/polaris-server/common/routing"
	"github.com/polarismesh/polaris-server/common/utils"
	"go.uber.org/zap"
)

// createRoutingConfigV1toV2 这里需要兼容 v1 版本的创建路由规则动作，将 v1 的数据转为 v2 进行存储
func (s *Server) createRoutingConfigV1toV2(ctx context.Context, req *apiv1.Routing) *apiv1.Response {
	saveDatas, resp := batchBuildV2Routings(req)
	if resp != nil {
		return resp
	}

	serviceTx, err := s.storage.CreateTransaction()
	if err != nil {
		log.Error(err.Error(), utils.ZapRequestIDByCtx(ctx))
		return api.NewRoutingResponse(api.StoreLayerException, req)
	}
	defer func() { _ = serviceTx.Commit() }()

	serviceName := req.GetService().GetValue()
	namespaceName := req.GetNamespace().GetValue()
	svc, err := serviceTx.RLockService(serviceName, namespaceName)
	if err != nil {
		log.Error("[Service][Routing] get read lock for service", zap.String("service", serviceName),
			zap.String("namespace", namespaceName), utils.ZapRequestIDByCtx(ctx), zap.Error(err))
		return api.NewRoutingResponse(api.StoreLayerException, req)
	}

	tx, err := s.storage.StartTx()
	if err != nil {
		log.Error("[Service][Routing] create routing v2 from v1 open tx",
			utils.ZapRequestIDByCtx(ctx), zap.Error(err))
		return apiv1.NewResponse(apiv1.StoreLayerException)
	}

	defer tx.Rollback()

	if err := s.storage.DeleteRoutingConfigTx(tx, svc.ID); err != nil {
		log.Error("[Service][Routing] clean routing v from store",
			utils.ZapRequestIDByCtx(ctx), zap.Error(err))
		return apiv1.NewResponse(apiv1.StoreLayerException)
	}

	for i := range saveDatas {
		item := saveDatas[i]
		data := &v2.RoutingConfig{}
		if err := data.ParseFromAPI(item); err != nil {
			return apiv1.NewResponse(apiv1.ExecuteException)
		}
		if err := s.storage.CreateRoutingConfigV2Tx(tx, data); err != nil {
			log.Error("[Service][Routing] create routing v2 from v1 into store",
				utils.ZapRequestIDByCtx(ctx), zap.Error(err))
			return apiv1.NewResponse(apiv1.StoreLayerException)
		}
		s.RecordHistory(routingV2RecordEntry(ctx, item, data, model.OCreate))
	}

	if err := tx.Commit(); err != nil {
		log.Error("[Service][Routing] create routing v2 from v1 commit",
			utils.ZapRequestIDByCtx(ctx), zap.Error(err))
		return apiv1.NewResponse(apiv1.ExecuteException)
	}

	return apiv1.NewResponse(apiv1.ExecuteSuccess)
}

// updateRoutingConfigV1toV2 这里需要兼容 v1 版本的更新路由规则动作，将 v1 的数据转为 v2 进行存储
func (s *Server) updateRoutingConfigV1toV2(ctx context.Context, req *apiv1.Routing) *apiv1.Response {
	saveDatas, resp := batchBuildV2Routings(req)
	if resp != nil {
		return resp
	}

	tx, err := s.storage.StartTx()
	if err != nil {
		log.Error("[Service][Routing] create routing v2 from v1 open tx",
			utils.ZapRequestIDByCtx(ctx), zap.Error(err))
		return apiv1.NewResponse(apiv1.StoreLayerException)
	}

	defer tx.Rollback()

	for i := range saveDatas {
		item := saveDatas[i]
		if item.Id == "" {
			data := &v2.RoutingConfig{
				ID:       utils.NewRoutingV2UUID(),
				Revision: utils.NewV2Revision(),
			}
			if err := data.ParseFromAPI(item); err != nil {
				return apiv1.NewResponse(apiv1.ExecuteException)
			}
			if err := s.storage.CreateRoutingConfigV2Tx(tx, data); err != nil {
				log.Error("[Service][Routing] create routing v2 from v1 into store",
					utils.ZapRequestIDByCtx(ctx), zap.Error(err))
				return apiv1.NewResponse(apiv1.StoreLayerException)
			}
			s.RecordHistory(routingV2RecordEntry(ctx, item, data, model.OCreate))
		} else {
			old, err := s.storage.GetRoutingConfigV2WithIDTx(tx, item.Id)
			if err != nil {
				log.Error("[Service][Routing] get routing v1 from store",
					utils.ZapRequestIDByCtx(ctx), zap.Error(err))
				return apiv1.NewResponse(apiv1.StoreLayerException)
			}
			if old == nil {
				return apiv1.NewResponse(apiv1.NotFoundRouting)
			}

			reqModel, err := api2RoutingConfigV2(item)
			if err != nil {
				log.Error("[Service][Routing] parse routing config v2 from request for update",
					utils.ZapRequestIDByCtx(ctx), zap.Error(err))
				return apiv1.NewResponse(api.ExecuteException)
			}

			if err := s.storage.UpdateRoutingConfigV2Tx(tx, reqModel); err != nil {
				log.Error("[Service][Routing] update routing config v2 store layer",
					utils.ZapRequestIDByCtx(ctx), zap.Error(err))
				return apiv1.NewResponse(apiv1.StoreLayerException)
			}
			s.RecordHistory(routingV2RecordEntry(ctx, item, reqModel, model.OUpdate))
		}
	}

	if err := tx.Commit(); err != nil {
		log.Error("[Service][Routing] create routing v2 from v1 commit",
			utils.ZapRequestIDByCtx(ctx), zap.Error(err))
		return apiv1.NewResponse(apiv1.ExecuteException)
	}

	return apiv1.NewResponse(apiv1.ExecuteSuccess)
}

// updateRoutingConfigV2FromV1 这里需要兼容 v1 版本的更新路由规则动作，将 v1 的数据转为 v2 进行存储
func (s *Server) updateRoutingConfigV2FromV1(ctx context.Context, req *apiv2.Routing) *apiv2.Response {

	extendInfo := req.GetExtendInfo()
	val, _ := extendInfo[v2.V1RuleIDKey]

	// 如果当前要修改的 v2 路由规则是从 v1 版本转换过来的，需要先做一下几个步骤
	// stpe 1: 现将 v1 规则转换为 v2 规则存储
	// stpe 2: 删除原本的 v1 规则
	// step 3: 更新当前的 v2 路由规则
	v1rule, err := s.storage.GetRoutingConfigWithID(val)
	if err != nil {
		log.Error("[Service][Routing] get routing config v1 store layer",
			utils.ZapRequestIDByCtx(ctx), zap.Error(err))
		return apiv2.NewResponse(apiv1.StoreLayerException)
	}
	svc := s.Cache().Service().GetServiceByID(val)
	if v1rule != nil {
		v1req, err := routingConfig2API(v1rule, svc.Name, svc.Namespace)
		if err != nil {
			log.Error("[Service][Routing] delete routing config v2 store layer",
				utils.ZapRequestIDByCtx(ctx), zap.Error(err))
			return apiv2.NewResponse(apiv1.ExecuteException)
		}

		indexStr, hasIndex := extendInfo[v2.V1RuleRouteIndexKey]
		if hasIndex {
			index, err := strconv.ParseInt(indexStr, 10, 64)
			if err == nil {
				routeType := extendInfo[v2.V1RuleRouteTypeKey]
				if routeType == v2.V1RuleInRoute {
					for i := range v1req.Inbounds {
						if i == int(index) {
							v1req.Inbounds[i].ExtendInfo = map[string]string{
								v2.V2RuleIDKey: req.Id,
							}
						}
					}
				}
				if routeType == v2.V1RuleOutRoute {
					for i := range v1req.Outbounds {
						if i == int(index) {
							v1req.Outbounds[i].ExtendInfo = map[string]string{
								v2.V2RuleIDKey: req.Id,
							}
						}
					}
				}
			}
		}

		resp := s.createRoutingConfigV1toV2(ctx, v1req)
		if resp.GetCode().GetValue() != apiv1.ExecuteSuccess {
			return apiv2.NewResponse(resp.GetCode().GetValue())
		}
	}

	return apiv2.NewResponse(apiv1.ExecuteSuccess)
}

func batchBuildV2Routings(req *apiv1.Routing) ([]*apiv2.Routing, *apiv1.Response) {
	inBounds := req.GetInbounds()
	outBounds := req.GetOutbounds()
	saveDatas := make([]*apiv2.Routing, 0, len(inBounds)+len(outBounds))
	for i := range inBounds {
		routing, err := routingcommon.BuildV2Routing(req, inBounds[i])
		if err != nil {
			return nil, apiv1.NewResponse(apiv1.ExecuteException)
		}
		routing.Name = req.GetNamespace().GetValue() + "." + req.GetService().GetValue()
		saveDatas = append(saveDatas, routing)
	}

	for i := range outBounds {
		routing, err := routingcommon.BuildV2Routing(req, outBounds[i])
		if err != nil {
			return nil, apiv1.NewResponse(apiv1.ExecuteException)
		}
		saveDatas = append(saveDatas, routing)
	}

	return saveDatas, nil
}
