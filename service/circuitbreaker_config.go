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
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"

	api "github.com/polarismesh/polaris/common/api/v1"
)

// CreateCircuitBreakers 批量创建熔断规则
func (s *Server) CreateCircuitBreakers(
	ctx context.Context, req []*apifault.CircuitBreaker) *apiservice.BatchWriteResponse {
	resps := api.NewBatchWriteResponseWithMsg(apimodel.Code_BadRequest, "API is Deprecated")
	return resps
}

// CreateCircuitBreakerVersions 批量创建熔断规则版本
func (s *Server) CreateCircuitBreakerVersions(
	ctx context.Context, req []*apifault.CircuitBreaker) *apiservice.BatchWriteResponse {
	resps := api.NewBatchWriteResponseWithMsg(apimodel.Code_BadRequest, "API is Deprecated")
	return resps
}

// DeleteCircuitBreakers 批量删除熔断规则
func (s *Server) DeleteCircuitBreakers(
	ctx context.Context, req []*apifault.CircuitBreaker) *apiservice.BatchWriteResponse {
	resps := api.NewBatchWriteResponseWithMsg(apimodel.Code_BadRequest, "API is Deprecated")
	return resps
}

// UpdateCircuitBreakers 批量修改熔断规则
func (s *Server) UpdateCircuitBreakers(
	ctx context.Context, req []*apifault.CircuitBreaker) *apiservice.BatchWriteResponse {
	resps := api.NewBatchWriteResponseWithMsg(apimodel.Code_BadRequest, "API is Deprecated")
	return resps
}

// ReleaseCircuitBreakers 批量发布熔断规则
func (s *Server) ReleaseCircuitBreakers(
	ctx context.Context, req []*apiservice.ConfigRelease) *apiservice.BatchWriteResponse {
	resps := api.NewBatchWriteResponseWithMsg(apimodel.Code_BadRequest, "API is Deprecated")
	return resps
}

// UnBindCircuitBreakers 批量解绑熔断规则
func (s *Server) UnBindCircuitBreakers(
	ctx context.Context, req []*apiservice.ConfigRelease) *apiservice.BatchWriteResponse {
	resps := api.NewBatchWriteResponseWithMsg(apimodel.Code_BadRequest, "API is Deprecated")
	return resps
}

// GetCircuitBreaker 根据id和version查询熔断规则
func (s *Server) GetCircuitBreaker(ctx context.Context, query map[string]string) *apiservice.BatchQueryResponse {
	resps := api.NewBatchQueryResponseWithMsg(apimodel.Code_BadRequest, "API is Deprecated")
	return resps
}

// GetCircuitBreakerVersions 根据id查询熔断规则所有版本
func (s *Server) GetCircuitBreakerVersions(
	ctx context.Context, query map[string]string) *apiservice.BatchQueryResponse {
	resps := api.NewBatchQueryResponseWithMsg(apimodel.Code_BadRequest, "API is Deprecated")
	return resps
}

// GetMasterCircuitBreakers 查询master熔断规则
func (s *Server) GetMasterCircuitBreakers(
	ctx context.Context, query map[string]string) *apiservice.BatchQueryResponse {
	resps := api.NewBatchQueryResponseWithMsg(apimodel.Code_BadRequest, "API is Deprecated")
	return resps
}

// GetReleaseCircuitBreakers 根据规则id查询已发布规则
func (s *Server) GetReleaseCircuitBreakers(
	ctx context.Context, query map[string]string) *apiservice.BatchQueryResponse {
	resps := api.NewBatchQueryResponseWithMsg(apimodel.Code_BadRequest, "API is Deprecated")
	return resps
}

// GetCircuitBreakerByService 根据服务查询绑定熔断规则
func (s *Server) GetCircuitBreakerByService(
	ctx context.Context, query map[string]string) *apiservice.BatchQueryResponse {
	resps := api.NewBatchQueryResponseWithMsg(apimodel.Code_BadRequest, "API is Deprecated")
	return resps
}

// GetCircuitBreakerToken 查询熔断规则的token
func (s *Server) GetCircuitBreakerToken(ctx context.Context, req *apifault.CircuitBreaker) *apiservice.Response {
	resps := api.NewResponseWithMsg(apimodel.Code_BadRequest, "API is Deprecated")
	return resps
}
