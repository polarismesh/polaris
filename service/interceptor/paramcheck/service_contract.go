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

	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	"github.com/polarismesh/specification/source/go/api/v1/service_manage"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"

	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/utils"
)

// CreateServiceContracts implements service.DiscoverServer.
func (svr *Server) CreateServiceContracts(ctx context.Context,
	req []*service_manage.ServiceContract) *service_manage.BatchWriteResponse {
	if rsp := checkBatchContractRules(req); rsp != nil {
		return rsp
	}
	batchRsp := api.NewBatchWriteResponse(apimodel.Code_ExecuteSuccess)
	for i := range req {
		rsp := checkBaseServiceContract(req[i])
		api.Collect(batchRsp, rsp)
	}
	if !api.IsSuccess(batchRsp) {
		return batchRsp
	}

	return svr.nextSvr.CreateServiceContracts(ctx, req)
}

// DeleteServiceContracts implements service.DiscoverServer.
func (svr *Server) DeleteServiceContracts(ctx context.Context,
	req []*service_manage.ServiceContract) *service_manage.BatchWriteResponse {
	if rsp := checkBatchContractRules(req); rsp != nil {
		return rsp
	}
	return svr.nextSvr.DeleteServiceContracts(ctx, req)
}

// GetServiceContractVersions implements service.DiscoverServer.
func (svr *Server) GetServiceContractVersions(ctx context.Context,
	filter map[string]string) *service_manage.BatchQueryResponse {
	return svr.nextSvr.GetServiceContractVersions(ctx, filter)
}

// GetServiceContracts implements service.DiscoverServer.
func (svr *Server) GetServiceContracts(ctx context.Context,
	query map[string]string) *service_manage.BatchQueryResponse {
	return svr.nextSvr.GetServiceContracts(ctx, query)
}

// CreateServiceContractInterfaces implements service.DiscoverServer.
func (svr *Server) CreateServiceContractInterfaces(ctx context.Context,
	contract *service_manage.ServiceContract, source service_manage.InterfaceDescriptor_Source) *service_manage.Response {
	if errRsp := checkOperationServiceContractInterface(contract); errRsp != nil {
		return errRsp
	}
	return svr.nextSvr.CreateServiceContractInterfaces(ctx, contract, source)
}

// AppendServiceContractInterfaces implements service.DiscoverServer.
func (svr *Server) AppendServiceContractInterfaces(ctx context.Context,
	contract *service_manage.ServiceContract,
	source service_manage.InterfaceDescriptor_Source) *service_manage.Response {
	if errRsp := checkOperationServiceContractInterface(contract); errRsp != nil {
		return errRsp
	}
	return svr.nextSvr.AppendServiceContractInterfaces(ctx, contract, source)
}

// DeleteServiceContractInterfaces implements service.DiscoverServer.
func (svr *Server) DeleteServiceContractInterfaces(ctx context.Context,
	contract *service_manage.ServiceContract) *service_manage.Response {
	if errRsp := checkOperationServiceContractInterface(contract); errRsp != nil {
		return errRsp
	}
	return svr.nextSvr.DeleteServiceContractInterfaces(ctx, contract)
}

func checkBaseServiceContract(req *apiservice.ServiceContract) *apiservice.Response {
	if err := utils.CheckResourceName(utils.NewStringValue(req.GetNamespace())); err != nil {
		return api.NewResponse(apimodel.Code_InvalidNamespaceName)
	}
	if req.GetName() == "" {
		return api.NewResponseWithMsg(apimodel.Code_BadRequest, "invalid service_contract name")
	}
	if req.GetProtocol() == "" {
		return api.NewResponseWithMsg(apimodel.Code_BadRequest, "invalid service_contract protocol")
	}
	return nil
}

func checkOperationServiceContractInterface(contract *apiservice.ServiceContract) *apiservice.Response {
	if contract.Id != "" {
		return nil
	}
	if err := checkBaseServiceContract(contract); err != nil {
		return err
	}
	id, errRsp := utils.CheckContractTetrad(contract)
	if errRsp != nil {
		return errRsp
	}
	contract.Id = id
	return nil
}

func checkBatchContractRules(req []*service_manage.ServiceContract) *apiservice.BatchWriteResponse {
	if len(req) == 0 {
		return api.NewBatchWriteResponse(apimodel.Code_EmptyRequest)
	}

	if len(req) > utils.MaxBatchSize {
		return api.NewBatchWriteResponse(apimodel.Code_BatchSizeOverLimit)
	}
	return nil
}
