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

	"github.com/polarismesh/specification/source/go/api/v1/fault_tolerance"
	"github.com/polarismesh/specification/source/go/api/v1/service_manage"
)

// DeleteFaultDetectRules implements service.DiscoverServer.
func (svr *Server) DeleteFaultDetectRules(ctx context.Context,
	request []*fault_tolerance.FaultDetectRule) *service_manage.BatchWriteResponse {
	return svr.nextSvr.DeleteFaultDetectRules(ctx, request)
}

// GetFaultDetectRules implements service.DiscoverServer.
func (svr *Server) GetFaultDetectRules(ctx context.Context,
	query map[string]string) *service_manage.BatchQueryResponse {
	return svr.nextSvr.GetFaultDetectRules(ctx, query)
}

// CreateFaultDetectRules implements service.DiscoverServer.
func (svr *Server) CreateFaultDetectRules(ctx context.Context,
	request []*fault_tolerance.FaultDetectRule) *service_manage.BatchWriteResponse {
	return svr.nextSvr.CreateFaultDetectRules(ctx, request)
}

// UpdateFaultDetectRules implements service.DiscoverServer.
func (svr *Server) UpdateFaultDetectRules(ctx context.Context, request []*fault_tolerance.FaultDetectRule) *service_manage.BatchWriteResponse {
	return svr.nextSvr.UpdateFaultDetectRules(ctx, request)
}
