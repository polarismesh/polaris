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
	"time"

	"github.com/gogo/protobuf/jsonpb"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/anypb"

	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/store"
	commonstore "github.com/polarismesh/polaris/common/store"
	commontime "github.com/polarismesh/polaris/common/time"
	"github.com/polarismesh/polaris/common/utils"
)

var (
	contractSearchFilters = map[string]string{
		"id":             "id",
		"namespace":      "namespace",
		"service":        "service",
		"name":           "type",
		"type":           "type",
		"protocol":       "protocol",
		"version":        "version",
		"brief":          "brief",
		"offset":         "offset",
		"limit":          "limit",
		"interface_name": "interface_name",
		"interface_path": "interface_path",
	}
)

func (s *Server) CreateServiceContracts(ctx context.Context,
	req []*apiservice.ServiceContract) *apiservice.BatchWriteResponse {

	responses := api.NewBatchWriteResponse(apimodel.Code_ExecuteSuccess)
	for i := range req {
		response := s.CreateServiceContract(ctx, req[i])
		api.Collect(responses, response)
	}
	return api.FormatBatchWriteResponse(responses)
}

func (s *Server) CreateServiceContract(ctx context.Context, contract *apiservice.ServiceContract) *apiservice.Response {
	contractId := contract.GetId()
	if contractId == "" {
		tmpId, errRsp := utils.CheckContractTetrad(contract)
		if errRsp != nil {
			return errRsp
		}
		contractId = tmpId
	}

	existContract, err := s.storage.GetServiceContract(contractId)
	if err != nil {
		log.Error("[Service][Contract] get service_contract from store when create", utils.RequestID(ctx),
			zap.Error(err))
		return api.NewAnyDataResponse(store.StoreCode2APICode(err), contract)
	}
	if existContract != nil {
		if existContract.Content == contract.Content {
			return api.NewAnyDataResponse(apimodel.Code_NoNeedUpdate, nil)
		}
		existContract.Content = contract.Content
		existContract.Revision = utils.NewUUID()
		if err := s.storage.UpdateServiceContract(existContract.ServiceContract); err != nil {
			log.Error("[Service][Contract] do update to store", utils.RequestID(ctx), zap.Error(err))
			return api.NewAnyDataResponse(store.StoreCode2APICode(err), contract)
		}
		s.RecordHistory(ctx, serviceContractRecordEntry(ctx, contract, &model.EnrichServiceContract{
			ServiceContract: existContract.ServiceContract,
		}, model.OUpdate))
		return api.NewAnyDataResponse(apimodel.Code_ExecuteSuccess, nil)
	}

	saveData := &model.ServiceContract{
		ID:        contractId,
		Type:      utils.DefaultString(contract.GetType(), contract.GetName()),
		Namespace: contract.GetNamespace(),
		Service:   contract.GetService(),
		Protocol:  contract.GetProtocol(),
		Version:   contract.GetVersion(),
		Revision:  utils.NewUUID(),
		Content:   contract.GetContent(),
	}

	if err := s.storage.CreateServiceContract(saveData); err != nil {
		log.Error("[Service][Contract] do save to store", utils.RequestID(ctx), zap.Error(err))
		return api.NewAnyDataResponse(store.StoreCode2APICode(err), nil)
	}
	s.RecordHistory(ctx, serviceContractRecordEntry(ctx, contract, &model.EnrichServiceContract{
		ServiceContract: saveData,
	}, model.OCreate))
	return api.NewAnyDataResponse(apimodel.Code_ExecuteSuccess, nil)
}

func (s *Server) GetServiceContracts(ctx context.Context, query map[string]string) *apiservice.BatchQueryResponse {

	out := api.NewBatchQueryResponse(apimodel.Code_ExecuteSuccess)
	out.Amount = utils.NewUInt32Value(0)
	out.Size = utils.NewUInt32Value(0)

	searchFilters := map[string]string{}
	for k, v := range query {
		newK, ok := contractSearchFilters[k]
		if !ok {
			continue
		}
		if v == "" {
			continue
		}
		searchFilters[newK] = v
	}
	offset, limit, err := utils.ParseOffsetAndLimit(searchFilters)
	if err != nil {
		out = api.NewBatchQueryResponseWithMsg(apimodel.Code_InvalidParameter, err.Error())
		return out
	}

	totalCount, ret, err := s.storage.GetServiceContracts(ctx, searchFilters, offset, limit)
	if err != nil {
		out = api.NewBatchQueryResponseWithMsg(commonstore.StoreCode2APICode(err), err.Error())
		return out
	}
	for _, item := range ret {
		methods := make([]*apiservice.InterfaceDescriptor, 0)
		for _, methodItem := range item.Interfaces {
			methods = append(methods, &apiservice.InterfaceDescriptor{
				Id:       methodItem.ID,
				Method:   methodItem.Method,
				Path:     methodItem.Path,
				Content:  methodItem.Content,
				Revision: methodItem.Revision,
				Source:   methodItem.Source,
				Ctime:    commontime.Time2String(methodItem.CreateTime),
				Mtime:    commontime.Time2String(methodItem.ModifyTime),
			})
		}

		status := "Offline"
		if svc := s.caches.Service().GetServiceByName(item.Service, item.Namespace); svc != nil {
			insCount := s.caches.Instance().GetInstancesCountByServiceID(svc.ID)
			if versionCount, ok := insCount.VersionCounts[item.Version]; ok {
				if versionCount.HealthyInstanceCount > 0 {
					status = "Online"
				}
			}
		}

		contract := &apiservice.ServiceContract{
			Id:         item.ID,
			Name:       item.Type,
			Type:       item.Type,
			Namespace:  item.Namespace,
			Service:    item.Service,
			Protocol:   item.Protocol,
			Version:    item.Version,
			Revision:   item.Revision,
			Content:    item.Content,
			Interfaces: methods,
			Status:     status,
			Ctime:      commontime.Time2String(item.CreateTime),
			Mtime:      commontime.Time2String(item.ModifyTime),
		}
		if addErr := api.AddAnyDataIntoBatchQuery(out, contract); addErr != nil {
			log.Error("[Service][Contract] add service_contract as any data fail",
				utils.RequestID(ctx), zap.Error(err))
			continue
		}
	}

	out.Amount = utils.NewUInt32Value(totalCount)
	out.Size = utils.NewUInt32Value(uint32(len(ret)))
	return out
}

// DeleteServiceContracts 删除服务契约（包含详情）
func (s *Server) DeleteServiceContracts(ctx context.Context,
	req []*apiservice.ServiceContract) *apiservice.BatchWriteResponse {

	responses := api.NewBatchWriteResponse(apimodel.Code_ExecuteSuccess)
	for i := range req {
		response := s.DeleteServiceContract(ctx, req[i])
		api.Collect(responses, response)
	}
	return api.FormatBatchWriteResponse(responses)
}

// DeleteServiceContract 删除服务契约（包含详情）
func (s *Server) DeleteServiceContract(ctx context.Context,
	contract *apiservice.ServiceContract) *apiservice.Response {

	if contract.Id == "" {
		id, errRsp := utils.CheckContractTetrad(contract)
		if errRsp != nil {
			return errRsp
		}
		contract.Id = id
	}

	saveData, err := s.storage.GetServiceContract(contract.Id)
	if err != nil {
		log.Error("[Service][Contract] get save service_contract when delete", utils.RequestID(ctx), zap.Error(err))
		return api.NewAnyDataResponse(store.StoreCode2APICode(err), nil)
	}
	if saveData == nil {
		return api.NewResponse(apimodel.Code_ExecuteSuccess)
	}

	deleteData := &model.ServiceContract{
		ID:        contract.Id,
		Type:      utils.DefaultString(contract.Type, contract.Name),
		Namespace: contract.Namespace,
		Service:   contract.Service,
		Protocol:  contract.Protocol,
		Version:   contract.Version,
	}

	if createErr := s.storage.DeleteServiceContract(deleteData); createErr != nil {
		log.Error("[Service][Contract] do delete from store", utils.RequestID(ctx), zap.Error(err))
		return api.NewAnyDataResponse(store.StoreCode2APICode(err), nil)
	}
	s.RecordHistory(ctx, serviceContractRecordEntry(ctx, contract, &model.EnrichServiceContract{
		ServiceContract: deleteData,
	}, model.ODelete))
	return api.NewAnyDataResponse(apimodel.Code_ExecuteSuccess, nil)
}

func (s *Server) GetServiceContractVersions(ctx context.Context, filter map[string]string) *apiservice.BatchQueryResponse {
	serviceName := filter["service"]
	namespace := filter["namespace"]
	if namespace == "" {
		return api.NewBatchQueryResponseWithMsg(apimodel.Code_InvalidParameter, "namespace is empty")
	}

	ret, err := s.storage.ListVersions(ctx, serviceName, namespace)
	if err != nil {
		log.Error("[Service][Contract] list save service_contract versions", utils.RequestID(ctx), zap.Error(err))
		return api.NewBatchQueryResponse(store.StoreCode2APICode(err))
	}

	resp := api.NewBatchQueryResponse(apimodel.Code_ExecuteSuccess)
	resp.Data = make([]*anypb.Any, 0, len(ret))
	for i := range ret {
		item := ret[i]
		if err := api.AddAnyDataIntoBatchQuery(resp, &apiservice.ServiceContract{
			Id:        item.ID,
			Name:      item.Type,
			Type:      item.Type,
			Namespace: item.Namespace,
			Service:   item.Service,
			Version:   item.Version,
			Protocol:  item.Protocol,
			Revision:  utils.NewUUID(),
			Ctime:     commontime.Time2String(item.CreateTime),
			Mtime:     commontime.Time2String(item.ModifyTime),
		}); err != nil {
			log.Error("[Service][Contract] list all versions fail", utils.RequestID(ctx), zap.String("namespace", namespace),
				zap.String("service", serviceName), zap.Error(err))
			return api.NewBatchQueryResponse(apimodel.Code_ExecuteException)
		}
	}
	resp.Amount = utils.NewUInt32Value(uint32(len(ret)))
	return resp
}

// CreateServiceContractInterfaces 添加服务契约详情
func (s *Server) CreateServiceContractInterfaces(ctx context.Context,
	contract *apiservice.ServiceContract, source apiservice.InterfaceDescriptor_Source) *apiservice.Response {

	if contract.Id == "" {
		id, errRsp := utils.CheckContractTetrad(contract)
		if errRsp != nil {
			return errRsp
		}
		contract.Id = id
	}

	createData := &model.EnrichServiceContract{
		ServiceContract: &model.ServiceContract{
			ID:       contract.Id,
			Revision: utils.NewUUID(),
		},
		Interfaces: make([]*model.InterfaceDescriptor, 0, len(contract.Interfaces)),
	}
	for _, item := range contract.Interfaces {
		interfaceId, errRsp := utils.CheckContractInterfaceTetrad(createData.ID, source, item)
		if errRsp != nil {
			log.Error("[Service][Contract] check service_contract interface id", utils.RequestID(ctx),
				zap.String("err", errRsp.GetInfo().GetValue()))
			return errRsp
		}
		createData.Interfaces = append(createData.Interfaces, &model.InterfaceDescriptor{
			ID:         interfaceId,
			ContractID: contract.Id,
			Type:       utils.DefaultString(item.Type, item.Name),
			Method:     item.Method,
			Path:       item.Path,
			Content:    item.Content,
			Source:     source,
			Revision:   utils.NewUUID(),
		})
	}

	if err := s.storage.AddServiceContractInterfaces(createData); err != nil {
		log.Error("[Service][Contract] full replace service_contract interfaces", utils.RequestID(ctx), zap.Error(err))
		return api.NewAnyDataResponse(store.StoreCode2APICode(err), nil)
	}
	s.RecordHistory(ctx, serviceContractRecordEntry(ctx, contract, createData, model.OUpdate))
	return api.NewAnyDataResponse(apimodel.Code_ExecuteSuccess, nil)
}

// AppendServiceContractInterfaces 追加服务契约详情
func (s *Server) AppendServiceContractInterfaces(ctx context.Context,
	contract *apiservice.ServiceContract, source apiservice.InterfaceDescriptor_Source) *apiservice.Response {

	if contract.Id == "" {
		id, errRsp := utils.CheckContractTetrad(contract)
		if errRsp != nil {
			return errRsp
		}
		contract.Id = id
	}

	saveData, err := s.storage.GetServiceContract(contract.Id)
	if err != nil {
		log.Error("[Service][Contract] get save service_contract when append interfaces", utils.RequestID(ctx), zap.Error(err))
		return api.NewAnyDataResponse(store.StoreCode2APICode(err), nil)
	}
	if saveData == nil {
		return api.NewResponse(apimodel.Code_NotFoundResource)
	}

	appendData := &model.EnrichServiceContract{
		ServiceContract: &model.ServiceContract{
			ID:       contract.Id,
			Revision: utils.NewUUID(),
		},
		Interfaces: make([]*model.InterfaceDescriptor, 0, len(contract.Interfaces)),
	}

	for _, item := range contract.Interfaces {
		interfaceId, errRsp := utils.CheckContractInterfaceTetrad(appendData.ID, source, item)
		if errRsp != nil {
			log.Error("[Service][Contract] check service_contract interface id", utils.RequestID(ctx),
				zap.String("err", errRsp.GetInfo().GetValue()))
			return errRsp
		}
		appendData.Interfaces = append(appendData.Interfaces, &model.InterfaceDescriptor{
			ID:         interfaceId,
			ContractID: contract.Id,
			Type:       utils.DefaultString(item.Type, item.Name),
			Method:     item.Method,
			Path:       item.Path,
			Content:    item.Content,
			Source:     source,
			Revision:   utils.NewUUID(),
		})
	}
	if err := s.storage.AppendServiceContractInterfaces(appendData); err != nil {
		log.Error("[Service][Contract] append service_contract interfaces", utils.RequestID(ctx), zap.Error(err))
		return api.NewAnyDataResponse(store.StoreCode2APICode(err), nil)
	}
	s.RecordHistory(ctx, serviceContractRecordEntry(ctx, contract, appendData, model.OUpdate))
	return api.NewAnyDataResponse(apimodel.Code_ExecuteSuccess, nil)
}

// DeleteServiceContractInterfaces 删除服务契约详情
func (s *Server) DeleteServiceContractInterfaces(ctx context.Context,
	contract *apiservice.ServiceContract) *apiservice.Response {

	if contract.Id == "" {
		id, errRsp := utils.CheckContractTetrad(contract)
		if errRsp != nil {
			return errRsp
		}
		contract.Id = id
	}

	saveData, err := s.storage.GetServiceContract(contract.Id)
	if err != nil {
		log.Error("[Service][Contract] get save service_contract when delete interfaces", utils.RequestID(ctx), zap.Error(err))
		return api.NewAnyDataResponse(store.StoreCode2APICode(err), nil)
	}
	if saveData == nil {
		return api.NewResponse(apimodel.Code_NotFoundResource)
	}

	deleteData := &model.EnrichServiceContract{
		ServiceContract: &model.ServiceContract{
			ID:       saveData.ID,
			Revision: utils.NewUUID(),
		},
		Interfaces: make([]*model.InterfaceDescriptor, 0, len(contract.Interfaces)),
	}

	for _, item := range contract.Interfaces {
		interfaceId, errRsp := utils.CheckContractInterfaceTetrad(deleteData.ID, apiservice.InterfaceDescriptor_Manual, item)
		if errRsp != nil {
			log.Error("[Service][Contract] check service_contract interface id", utils.RequestID(ctx),
				zap.String("err", errRsp.GetInfo().GetValue()))
			return errRsp
		}
		deleteData.Interfaces = append(deleteData.Interfaces, &model.InterfaceDescriptor{
			ID:         interfaceId,
			ContractID: contract.Id,
			Method:     item.Method,
			Type:       utils.DefaultString(item.Type, item.Name),
			Path:       item.Path,
		})
	}
	if err := s.storage.DeleteServiceContractInterfaces(deleteData); err != nil {
		log.Error("[Service][Contract] delete service_contract interfaces", utils.RequestID(ctx), zap.Error(err))
		return api.NewAnyDataResponse(store.StoreCode2APICode(err), nil)
	}
	s.RecordHistory(ctx, serviceContractRecordEntry(ctx, contract, deleteData, model.ODelete))
	return api.NewAnyDataResponse(apimodel.Code_ExecuteSuccess, nil)
}

// serviceContractRecordEntry 生成服务的记录entry
func serviceContractRecordEntry(ctx context.Context, req *apiservice.ServiceContract, data *model.EnrichServiceContract,
	operationType model.OperationType) *model.RecordEntry {

	marshaler := jsonpb.Marshaler{}
	detail, _ := marshaler.MarshalToString(req)

	entry := &model.RecordEntry{
		ResourceType:  model.RServiceContract,
		ResourceName:  data.GetResourceName(),
		Namespace:     req.GetNamespace(),
		OperationType: operationType,
		Operator:      utils.ParseOperator(ctx),
		Detail:        detail,
		HappenTime:    time.Now(),
	}

	return entry
}
