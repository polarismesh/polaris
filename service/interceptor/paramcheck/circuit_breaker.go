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
	apifault "github.com/polarismesh/specification/source/go/api/v1/fault_tolerance"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	"github.com/polarismesh/specification/source/go/api/v1/service_manage"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"

	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/utils"
)

// GetMasterCircuitBreakers implements service.DiscoverServer.
func (svr *Server) GetMasterCircuitBreakers(ctx context.Context,
	query map[string]string) *service_manage.BatchQueryResponse {
	return svr.nextSvr.GetMasterCircuitBreakers(ctx, query)
}

// GetReleaseCircuitBreakers implements service.DiscoverServer.
func (svr *Server) GetReleaseCircuitBreakers(ctx context.Context,
	query map[string]string) *service_manage.BatchQueryResponse {
	return svr.nextSvr.GetReleaseCircuitBreakers(ctx, query)
}

// GetCircuitBreaker implements service.DiscoverServer.
func (svr *Server) GetCircuitBreaker(ctx context.Context,
	query map[string]string) *service_manage.BatchQueryResponse {
	return svr.nextSvr.GetCircuitBreaker(ctx, query)
}

// GetCircuitBreakerByService implements service.DiscoverServer.
func (svr *Server) GetCircuitBreakerByService(ctx context.Context,
	query map[string]string) *service_manage.BatchQueryResponse {
	return svr.nextSvr.GetCircuitBreakerByService(ctx, query)
}

// DeleteCircuitBreakers implements service.DiscoverServer.
func (svr *Server) DeleteCircuitBreakers(ctx context.Context,
	req []*fault_tolerance.CircuitBreaker) *service_manage.BatchWriteResponse {
	return svr.nextSvr.DeleteCircuitBreakers(ctx, req)
}

// GetCircuitBreakerToken implements service.DiscoverServer.
func (svr *Server) GetCircuitBreakerToken(ctx context.Context,
	req *fault_tolerance.CircuitBreaker) *service_manage.Response {
	return svr.nextSvr.GetCircuitBreakerToken(ctx, req)
}

// GetCircuitBreakerVersions implements service.DiscoverServer.
func (svr *Server) GetCircuitBreakerVersions(ctx context.Context,
	query map[string]string) *service_manage.BatchQueryResponse {
	return svr.nextSvr.GetCircuitBreakerVersions(ctx, query)
}

// GetCircuitBreakerRules implements service.DiscoverServer.
func (svr *Server) GetCircuitBreakerRules(ctx context.Context,
	query map[string]string) *service_manage.BatchQueryResponse {
	return svr.nextSvr.GetCircuitBreakerRules(ctx, query)
}

// DeleteCircuitBreakerRules implements service.DiscoverServer.
func (svr *Server) DeleteCircuitBreakerRules(ctx context.Context,
	request []*fault_tolerance.CircuitBreakerRule) *service_manage.BatchWriteResponse {
	if err := checkBatchCircuitBreakerRules(request); err != nil {
		return err
	}
	return svr.nextSvr.DeleteCircuitBreakerRules(ctx, request)
}

// EnableCircuitBreakerRules implements service.DiscoverServer.
func (svr *Server) EnableCircuitBreakerRules(ctx context.Context,
	request []*fault_tolerance.CircuitBreakerRule) *service_manage.BatchWriteResponse {
	if err := checkBatchCircuitBreakerRules(request); err != nil {
		return err
	}
	return svr.nextSvr.EnableCircuitBreakerRules(ctx, request)
}

// ReleaseCircuitBreakers implements service.DiscoverServer.
func (svr *Server) ReleaseCircuitBreakers(ctx context.Context, req []*service_manage.ConfigRelease) *service_manage.BatchWriteResponse {
	return svr.nextSvr.ReleaseCircuitBreakers(ctx, req)
}

// UnBindCircuitBreakers implements service.DiscoverServer.
func (svr *Server) UnBindCircuitBreakers(ctx context.Context, req []*service_manage.ConfigRelease) *service_manage.BatchWriteResponse {
	return svr.nextSvr.UnBindCircuitBreakers(ctx, req)
}

// UpdateCircuitBreakerRules implements service.DiscoverServer.
func (svr *Server) UpdateCircuitBreakerRules(ctx context.Context, request []*fault_tolerance.CircuitBreakerRule) *service_manage.BatchWriteResponse {
	if err := checkBatchCircuitBreakerRules(request); err != nil {
		return err
	}
	return svr.nextSvr.UpdateCircuitBreakerRules(ctx, request)
}

// CreateCircuitBreakerRules implements service.DiscoverServer.
func (svr *Server) CreateCircuitBreakerRules(ctx context.Context,
	request []*fault_tolerance.CircuitBreakerRule) *service_manage.BatchWriteResponse {
	if err := checkBatchCircuitBreakerRules(request); err != nil {
		return err
	}
	return svr.nextSvr.CreateCircuitBreakerRules(ctx, request)
}

// CreateCircuitBreakerVersions implements service.DiscoverServer.
func (svr *Server) CreateCircuitBreakerVersions(ctx context.Context,
	req []*fault_tolerance.CircuitBreaker) *service_manage.BatchWriteResponse {
	return svr.nextSvr.CreateCircuitBreakerVersions(ctx, req)
}

// CreateCircuitBreakers implements service.DiscoverServer.
func (svr *Server) CreateCircuitBreakers(ctx context.Context,
	req []*fault_tolerance.CircuitBreaker) *service_manage.BatchWriteResponse {
	return svr.nextSvr.CreateCircuitBreakers(ctx, req)
}

// UpdateCircuitBreakers implements service.DiscoverServer.
func (svr *Server) UpdateCircuitBreakers(ctx context.Context, req []*fault_tolerance.CircuitBreaker) *service_manage.BatchWriteResponse {
	return svr.nextSvr.UpdateCircuitBreakers(ctx, req)
}

func checkBatchCircuitBreakerRules(req []*apifault.CircuitBreakerRule) *apiservice.BatchWriteResponse {
	if len(req) == 0 {
		return api.NewBatchWriteResponse(apimodel.Code_EmptyRequest)
	}

	if len(req) > utils.MaxBatchSize {
		return api.NewBatchWriteResponse(apimodel.Code_BatchSizeOverLimit)
	}
	return nil
}
