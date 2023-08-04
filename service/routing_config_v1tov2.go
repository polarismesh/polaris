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

	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"
	apitraffic "github.com/polarismesh/specification/source/go/api/v1/traffic_manage"
	"go.uber.org/zap"

	apiv1 "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/model"
	commonstore "github.com/polarismesh/polaris/common/store"
	"github.com/polarismesh/polaris/common/utils"
)

// createRoutingConfigV1toV2 Compatible with V1 version of the creation routing rules, convert V1 to V2 for storage
func (s *Server) createRoutingConfigV1toV2(ctx context.Context, req *apitraffic.Routing) *apiservice.Response {
	if resp := checkRoutingConfig(req); resp != nil {
		return resp
	}

	serviceName := req.GetService().GetValue()
	namespaceName := req.GetNamespace().GetValue()
	svc, errResp := s.loadService(namespaceName, serviceName)
	if errResp != nil {
		log.Error("[Service][Routing] get read lock for service", zap.String("service", serviceName),
			zap.String("namespace", namespaceName), utils.RequestID(ctx), zap.Any("err", errResp))
		return apiv1.NewRoutingResponse(apimodel.Code(errResp.GetCode().GetValue()), req)
	}
	if svc == nil {
		return apiv1.NewRoutingResponse(apimodel.Code_NotFoundService, req)
	}
	if svc.IsAlias() {
		return apiv1.NewRoutingResponse(apimodel.Code_NotAllowAliasCreateRouting, req)
	}

	inDatas, outDatas, resp := batchBuildV2Routings(req)
	if resp != nil {
		return resp
	}

	resp = s.saveRoutingV1toV2(ctx, svc.ID, inDatas, outDatas)
	if resp.GetCode().GetValue() != uint32(apimodel.Code_ExecuteSuccess) {
		return resp
	}

	return apiv1.NewRoutingResponse(apimodel.Code_ExecuteSuccess, req)
}

// updateRoutingConfigV1toV2 Compatible with V1 version update routing rules, convert the data of V1 to V2 for storage
// Once the V1 rule is converted to V2 rules, the original V1 rules will be removed from storage
func (s *Server) updateRoutingConfigV1toV2(ctx context.Context, req *apitraffic.Routing) *apiservice.Response {
	svc, resp := s.routingConfigCommonCheck(ctx, req)
	if resp != nil {
		return resp
	}

	serviceTx, err := s.storage.CreateTransaction()
	if err != nil {
		log.Error(err.Error(), utils.RequestID(ctx))
		return apiv1.NewRoutingResponse(commonstore.StoreCode2APICode(err), req)
	}
	// Release the lock for the service
	defer func() {
		_ = serviceTx.Commit()
	}()

	// Need to prohibit the concurrent modification of the V1 rules
	if _, err = serviceTx.LockService(svc.Name, svc.Namespace); err != nil {
		log.Error("[Service][Routing] get service x-lock", zap.String("service", svc.Name),
			zap.String("namespace", svc.Namespace), utils.RequestID(ctx), zap.Error(err))
		return apiv1.NewRoutingResponse(commonstore.StoreCode2APICode(err), req)
	}

	conf, err := s.storage.GetRoutingConfigWithService(svc.Name, svc.Namespace)
	if err != nil {
		log.Error(err.Error(), utils.RequestID(ctx))
		return apiv1.NewRoutingResponse(commonstore.StoreCode2APICode(err), req)
	}
	if conf == nil {
		return apiv1.NewRoutingResponse(apimodel.Code_NotFoundRouting, req)
	}

	inDatas, outDatas, resp := batchBuildV2Routings(req)
	if resp != nil {
		return resp
	}

	if resp := s.saveRoutingV1toV2(ctx, svc.ID, inDatas, outDatas); resp.GetCode().GetValue() != uint32(
		apimodel.Code_ExecuteSuccess) {
		return resp
	}

	return apiv1.NewRoutingResponse(apimodel.Code_ExecuteSuccess, req)
}

// saveRoutingV1toV2 Convert the V1 rules of the target to V2 rule
func (s *Server) saveRoutingV1toV2(ctx context.Context, svcId string,
	inRules, outRules []*apitraffic.RouteRule) *apiservice.Response {
	tx, err := s.storage.StartTx()
	if err != nil {
		log.Error("[Service][Routing] create routing v2 from v1 open tx",
			utils.RequestID(ctx), zap.Error(err))
		return apiv1.NewResponse(commonstore.StoreCode2APICode(err))
	}
	defer func() {
		_ = tx.Rollback()
	}()

	// Need to delete the routing rules of V1 first
	if err := s.storage.DeleteRoutingConfigTx(tx, svcId); err != nil {
		log.Error("[Service][Routing] clean routing v1 from store",
			utils.RequestID(ctx), zap.Error(err))
		return apiv1.NewResponse(commonstore.StoreCode2APICode(err))
	}

	saveOperation := func(routings []*apitraffic.RouteRule) *apiservice.Response {
		priorityMax := 0
		for i := range routings {
			item := routings[i]
			if item.Id == "" {
				item.Id = utils.NewRoutingV2UUID()
			}
			item.Revision = utils.NewV2Revision()
			data := &model.RouterConfig{}
			if err := data.ParseRouteRuleFromAPI(item); err != nil {
				return apiv1.NewResponse(apimodel.Code_ExecuteException)
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
					utils.RequestID(ctx), zap.Error(err))
				return apiv1.NewResponse(commonstore.StoreCode2APICode(err))
			}
			s.RecordHistory(ctx, routingV2RecordEntry(ctx, item, data, model.OCreate))
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
			utils.RequestID(ctx), zap.Error(err))
		return apiv1.NewResponse(apimodel.Code_ExecuteException)
	}

	return apiv1.NewResponse(apimodel.Code_ExecuteSuccess)
}

func batchBuildV2Routings(
	req *apitraffic.Routing) ([]*apitraffic.RouteRule, []*apitraffic.RouteRule, *apiservice.Response) {
	inBounds := req.GetInbounds()
	outBounds := req.GetOutbounds()
	inRoutings := make([]*apitraffic.RouteRule, 0, len(inBounds))
	outRoutings := make([]*apitraffic.RouteRule, 0, len(outBounds))
	for i := range inBounds {
		routing, err := model.BuildV2RoutingFromV1Route(req, inBounds[i])
		if err != nil {
			return nil, nil, apiv1.NewResponse(apimodel.Code_ExecuteException)
		}
		routing.Name = req.GetNamespace().GetValue() + "." + req.GetService().GetValue()
		inRoutings = append(inRoutings, routing)
	}

	for i := range outBounds {
		routing, err := model.BuildV2RoutingFromV1Route(req, outBounds[i])
		if err != nil {
			return nil, nil, apiv1.NewResponse(apimodel.Code_ExecuteException)
		}
		outRoutings = append(outRoutings, routing)
	}

	return inRoutings, outRoutings, nil
}
