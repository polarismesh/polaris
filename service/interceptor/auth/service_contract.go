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

package service_auth

import (
	"context"

	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"

	api "github.com/polarismesh/polaris/common/api/v1"
	authcommon "github.com/polarismesh/polaris/common/model/auth"
	"github.com/polarismesh/polaris/common/utils"
)

// CreateServiceContracts .
func (svr *Server) CreateServiceContracts(ctx context.Context,
	req []*apiservice.ServiceContract) *apiservice.BatchWriteResponse {
	services := make([]*apiservice.Service, 0, len(req))
	for i := range req {
		services = append(services, &apiservice.Service{
			Namespace: utils.NewStringValue(req[i].Namespace),
			Name:      utils.NewStringValue(req[i].Service),
		})
	}

	authCtx := svr.collectServiceAuthContext(ctx, services, authcommon.Create, authcommon.CreateServiceContracts)
	if _, err := svr.policySvr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewBatchWriteResponse(authcommon.ConvertToErrCode(err))
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)
	return svr.nextSvr.CreateServiceContracts(ctx, req)
}

// GetServiceContracts .
func (svr *Server) GetServiceContracts(ctx context.Context,
	query map[string]string) *apiservice.BatchQueryResponse {
	authCtx := svr.collectServiceAuthContext(ctx, nil, authcommon.Read, authcommon.DescribeServiceContracts)
	if _, err := svr.policySvr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewBatchQueryResponse(authcommon.ConvertToErrCode(err))
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)
	return svr.nextSvr.GetServiceContracts(ctx, query)
}

// GetServiceContractVersions .
func (svr *Server) GetServiceContractVersions(ctx context.Context,
	filter map[string]string) *apiservice.BatchQueryResponse {

	authCtx := svr.collectServiceAuthContext(ctx, nil, authcommon.Read, authcommon.DescribeServiceContractVersions)
	if _, err := svr.policySvr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewBatchQueryResponse(authcommon.ConvertToErrCode(err))
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)
	return svr.nextSvr.GetServiceContractVersions(ctx, filter)
}

// DeleteServiceContracts .
func (svr *Server) DeleteServiceContracts(ctx context.Context,
	req []*apiservice.ServiceContract) *apiservice.BatchWriteResponse {
	services := make([]*apiservice.Service, 0, len(req))
	for i := range req {
		services = append(services, &apiservice.Service{
			Namespace: utils.NewStringValue(req[i].Namespace),
			Name:      utils.NewStringValue(req[i].Service),
		})
	}

	authCtx := svr.collectServiceAuthContext(ctx, services, authcommon.Delete, authcommon.DeleteServiceContracts)
	if _, err := svr.policySvr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewBatchWriteResponse(authcommon.ConvertToErrCode(err))
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)
	return svr.nextSvr.DeleteServiceContracts(ctx, req)
}

// CreateServiceContractInterfaces .
func (svr *Server) CreateServiceContractInterfaces(ctx context.Context, contract *apiservice.ServiceContract,
	source apiservice.InterfaceDescriptor_Source) *apiservice.Response {
	authCtx := svr.collectServiceAuthContext(ctx, []*apiservice.Service{
		{
			Namespace: utils.NewStringValue(contract.Namespace),
			Name:      utils.NewStringValue(contract.Service),
		},
	}, authcommon.Modify, authcommon.CreateServiceContractInterfaces)
	if _, err := svr.policySvr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewResponse(authcommon.ConvertToErrCode(err))
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)
	return svr.nextSvr.CreateServiceContractInterfaces(ctx, contract, source)
}

// AppendServiceContractInterfaces .
func (svr *Server) AppendServiceContractInterfaces(ctx context.Context,
	contract *apiservice.ServiceContract, source apiservice.InterfaceDescriptor_Source) *apiservice.Response {
	authCtx := svr.collectServiceAuthContext(ctx, []*apiservice.Service{
		{
			Namespace: utils.NewStringValue(contract.Namespace),
			Name:      utils.NewStringValue(contract.Service),
		},
	}, authcommon.Modify, authcommon.AppendServiceContractInterfaces)
	if _, err := svr.policySvr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewResponse(authcommon.ConvertToErrCode(err))
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)
	return svr.nextSvr.AppendServiceContractInterfaces(ctx, contract, source)
}

// DeleteServiceContractInterfaces .
func (svr *Server) DeleteServiceContractInterfaces(ctx context.Context,
	contract *apiservice.ServiceContract) *apiservice.Response {
	authCtx := svr.collectServiceAuthContext(ctx, []*apiservice.Service{
		{
			Namespace: utils.NewStringValue(contract.Namespace),
			Name:      utils.NewStringValue(contract.Service),
		},
	}, authcommon.Modify, authcommon.DeleteServiceContractInterfaces)
	if _, err := svr.policySvr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewResponse(authcommon.ConvertToErrCode(err))
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)
	return svr.nextSvr.DeleteServiceContractInterfaces(ctx, contract)
}
