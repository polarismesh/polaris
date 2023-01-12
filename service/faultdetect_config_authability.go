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

	apifault "github.com/polarismesh/specification/source/go/api/v1/fault_tolerance"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"
)

func (svr *serverAuthAbility) CreateFaultDetectRules(
	ctx context.Context, request []*apifault.FaultDetectRule) *apiservice.BatchWriteResponse {
	return svr.targetServer.CreateFaultDetectRules(ctx, request)
}

func (svr *serverAuthAbility) DeleteFaultDetectRules(
	ctx context.Context, request []*apifault.FaultDetectRule) *apiservice.BatchWriteResponse {
	return svr.targetServer.DeleteFaultDetectRules(ctx, request)
}

func (svr *serverAuthAbility) UpdateFaultDetectRules(
	ctx context.Context, request []*apifault.FaultDetectRule) *apiservice.BatchWriteResponse {
	return svr.targetServer.UpdateFaultDetectRules(ctx, request)
}

func (svr *serverAuthAbility) GetFaultDetectRules(
	ctx context.Context, query map[string]string) *apiservice.BatchQueryResponse {
	return svr.targetServer.GetFaultDetectRules(ctx, query)
}

func (svr *serverAuthAbility) GetFaultDetectWithCache(
	ctx context.Context, req *apiservice.Service) *apiservice.DiscoverResponse {
	return svr.targetServer.GetFaultDetectWithCache(ctx, req)
}
