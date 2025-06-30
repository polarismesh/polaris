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
	"strings"
	"time"

	"github.com/gogo/protobuf/jsonpb"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/anypb"

	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/model"
	commonstore "github.com/polarismesh/polaris/common/store"
	commontime "github.com/polarismesh/polaris/common/time"
	"github.com/polarismesh/polaris/common/utils"
)

var (
	contractSearchFilters = map[string]string{
		"id":          "id",
		"namespace":   "namespace",
		"service":     "service",
		"name":        "type",
		"type":        "type",
		"protocol":    "protocol",
		"version":     "version",
		"brief":       "brief",
		"offset":      "offset",
		"limit":       "limit",
		"order_field": "order_field",
		"order_type":  "order_type",
	}

	interfaceSearchFilters = map[string]string{
		"id":          "id",
		"namespace":   "namespace",
		"service":     "service",
		"name":        "type",
		"type":        "type",
		"protocol":    "protocol",
		"version":     "version",
		"path":        "path",
		"method":      "method",
		"offset":      "offset",
		"limit":       "limit",
		"order_field": "order_field",
		"order_type":  "order_type",
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
	if errRsp := checkBaseServiceContract(contract); errRsp != nil {
		return errRsp
	}
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
		return api.NewServiceContractResponse(commonstore.StoreCode2APICode(err), nil)
	}
	if existContract != nil {
		var needUpdate = false
		if existContract.Content != contract.Content {
			existContract.Content = contract.Content
			if existContract.ContentDigest, err = utils.BuildSha1Digest(existContract.Content); err != nil {
				log.Error("[Service][Contract] do build content digest for update contract", utils.RequestID(ctx),
					zap.Error(err))
				return api.NewServiceContractResponse(commonstore.StoreCode2APICode(err), nil)
			}
			needUpdate = true
		}
		if utils.NeedUpdateMetadata(existContract.Metadata, contract.Metadata) {
			existContract.Metadata = contract.Metadata
			if existContract.MetadataStr, err = utils.ConvertMetadataToStringValue(existContract.Metadata); err != nil {
				log.Error("[Service][Contract] do serialize metadata for update contract", utils.RequestID(ctx),
					zap.Error(err))
				return api.NewServiceContractResponse(commonstore.StoreCode2APICode(err), nil)
			}
			needUpdate = true
		}
		if !needUpdate {
			return api.NewServiceContractResponse(apimodel.Code_NoNeedUpdate, nil)
		}
		existContract.Revision = utils.NewUUID()
		if err := s.storage.UpdateServiceContract(existContract.ServiceContract); err != nil {
			log.Error("[Service][Contract] do update to store", utils.RequestID(ctx), zap.Error(err))
			return api.NewServiceContractResponse(commonstore.StoreCode2APICode(err), nil)
		}
		s.RecordHistory(ctx, serviceContractRecordEntry(ctx, contract, &model.EnrichServiceContract{
			ServiceContract: existContract.ServiceContract,
		}, model.OUpdate))
		return api.NewServiceContractResponse(apimodel.Code_ExecuteSuccess, &apiservice.ServiceContract{Id: contractId})
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
		Metadata:  contract.Metadata,
	}
	if saveData.ContentDigest, err = utils.BuildSha1Digest(saveData.Content); err != nil {
		log.Error("[Service][Contract] do build content digest for create contract", utils.RequestID(ctx), zap.Error(err))
		return api.NewServiceContractResponse(commonstore.StoreCode2APICode(err), nil)
	}
	if saveData.MetadataStr, err = utils.ConvertMetadataToStringValue(saveData.Metadata); err != nil {
		log.Error("[Service][Contract] do serialize metadata for create contract", utils.RequestID(ctx), zap.Error(err))
		return api.NewServiceContractResponse(commonstore.StoreCode2APICode(err), nil)
	}

	if err := s.storage.CreateServiceContract(saveData); err != nil {
		log.Error("[Service][Contract] do save to store", utils.RequestID(ctx), zap.Error(err))
		return api.NewServiceContractResponse(commonstore.StoreCode2APICode(err), nil)
	}
	s.RecordHistory(ctx, serviceContractRecordEntry(ctx, contract, &model.EnrichServiceContract{
		ServiceContract: saveData,
	}, model.OCreate))
	return api.NewServiceContractResponse(apimodel.Code_ExecuteSuccess, &apiservice.ServiceContract{Id: contractId})
}

func (s *Server) GetServiceContracts(ctx context.Context, query map[string]string) *apiservice.BatchQueryResponse {

	out := api.NewBatchQueryResponse(apimodel.Code_ExecuteSuccess)
	out.Amount = utils.NewUInt32Value(0)
	out.Size = utils.NewUInt32Value(0)

	var isBrief = false
	if bValue, ok := query[briefSearch]; ok && strings.ToLower(bValue) == "true" {
		isBrief = true
		delete(query, briefSearch)
	}
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

	if _, ok := searchFilters["order_field"]; !ok {
		searchFilters["order_field"] = "mtime"
	}
	if _, ok := searchFilters["order_type"]; !ok {
		searchFilters["order_type"] = "desc"
	}

	totalCount, ret, err := s.storage.GetServiceContracts(ctx, searchFilters, offset, limit)
	if err != nil {
		out = api.NewBatchQueryResponseWithMsg(commonstore.StoreCode2APICode(err), err.Error())
		return out
	}
	for _, item := range ret {
		methods := make([]*apiservice.InterfaceDescriptor, 0)
		for _, methodItem := range item.Interfaces {
			intf := &apiservice.InterfaceDescriptor{
				Id:            methodItem.ID,
				Method:        methodItem.Method,
				Name:          methodItem.Type,
				Type:          methodItem.Type,
				Path:          methodItem.Path,
				Content:       methodItem.Content,
				ContentDigest: methodItem.ContentDigest,
				Revision:      methodItem.Revision,
				Source:        methodItem.Source,
				Ctime:         commontime.Time2String(methodItem.CreateTime),
				Mtime:         commontime.Time2String(methodItem.ModifyTime),
			}
			if isBrief {
				intf.Content = ""
			}
			methods = append(methods, intf)
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
			Id:            item.ID,
			Name:          item.Type,
			Type:          item.Type,
			Namespace:     item.Namespace,
			Service:       item.Service,
			Protocol:      item.Protocol,
			Version:       item.Version,
			Revision:      item.Revision,
			Content:       item.Content,
			ContentDigest: item.ContentDigest,
			Metadata:      item.Metadata,
			Interfaces:    methods,
			Status:        status,
			Ctime:         commontime.Time2String(item.CreateTime),
			Mtime:         commontime.Time2String(item.ModifyTime),
		}
		if isBrief {
			contract.Content = ""
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
		return api.NewServiceContractResponse(commonstore.StoreCode2APICode(err), nil)
	}
	if saveData == nil {
		return api.NewServiceContractResponse(apimodel.Code_ExecuteSuccess, nil)
	}

	deleteData := &model.ServiceContract{
		ID:        contract.Id,
		Type:      utils.DefaultString(contract.GetType(), contract.GetName()),
		Namespace: contract.Namespace,
		Service:   contract.Service,
		Protocol:  contract.Protocol,
		Version:   contract.Version,
	}

	if createErr := s.storage.DeleteServiceContract(deleteData); createErr != nil {
		log.Error("[Service][Contract] do delete from store", utils.RequestID(ctx), zap.Error(err))
		return api.NewServiceContractResponse(commonstore.StoreCode2APICode(err), nil)
	}
	s.RecordHistory(ctx, serviceContractRecordEntry(ctx, contract, &model.EnrichServiceContract{
		ServiceContract: deleteData,
	}, model.ODelete))
	return api.NewServiceContractResponse(apimodel.Code_ExecuteSuccess, &apiservice.ServiceContract{Id: contract.Id})
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
		return api.NewBatchQueryResponse(commonstore.StoreCode2APICode(err))
	}
	resp := api.NewBatchQueryResponse(apimodel.Code_ExecuteSuccess)
	resp.Data = make([]*anypb.Any, 0, len(ret))
	for i := range ret {
		item := ret[i]
		if err := api.AddAnyDataIntoBatchQuery(resp, &apiservice.ServiceContract{
			Id:            item.ID,
			Name:          item.Type,
			Type:          item.Type,
			Namespace:     item.Namespace,
			Service:       item.Service,
			Version:       item.Version,
			Protocol:      item.Protocol,
			Metadata:      item.Metadata,
			ContentDigest: item.ContentDigest,
			Ctime:         commontime.Time2String(item.CreateTime),
			Mtime:         commontime.Time2String(item.ModifyTime),
			Revision:      item.Revision,
		}); err != nil {
			log.Error("[Service][Contract] list all versions fail", utils.RequestID(ctx), zap.String("namespace", namespace),
				zap.String("service", serviceName), zap.Error(err))
			return api.NewBatchQueryResponse(apimodel.Code_ExecuteException)
		}
	}
	resp.Amount = utils.NewUInt32Value(uint32(len(ret)))
	resp.Size = utils.NewUInt32Value(uint32(len(ret)))
	return resp
}

// CreateServiceContractInterfaces 添加服务契约详情
func (s *Server) CreateServiceContractInterfaces(ctx context.Context,
	contract *apiservice.ServiceContract, source apiservice.InterfaceDescriptor_Source) *apiservice.Response {

	if errRsp := checkOperationServiceContractInterface(contract); errRsp != nil {
		return errRsp
	}
	contract.Type = utils.DefaultString(contract.GetType(), contract.GetName())
	createData := &model.EnrichServiceContract{
		ServiceContract: &model.ServiceContract{
			ID:       contract.Id,
			Revision: utils.NewUUID(),
		},
		Interfaces: make([]*model.InterfaceDescriptor, 0, len(contract.Interfaces)),
	}
	retContract := &apiservice.ServiceContract{Id: contract.Id}
	var err error
	interfaces := make(map[string]*model.InterfaceDescriptor, len(contract.Interfaces))
	for _, item := range contract.Interfaces {
		interfaceId, errRsp := utils.CheckContractInterfaceTetrad(createData.ID, source, item)
		if errRsp != nil {
			log.Error("[Service][Contract] check service_contract interface id", utils.RequestID(ctx),
				zap.String("err", errRsp.GetInfo().GetValue()))
			return errRsp
		}
		interfaceDescriptor := &model.InterfaceDescriptor{
			ID:         interfaceId,
			ContractID: contract.Id,
			Namespace:  contract.Namespace,
			Service:    contract.Service,
			Protocol:   contract.Protocol,
			Version:    contract.Version,
			Type:       contract.GetType(),
			Method:     item.Method,
			Path:       item.Path,
			Content:    item.Content,
			Source:     source,
			Revision:   utils.NewUUID(),
		}
		if interfaceDescriptor.ContentDigest, err = utils.BuildSha1Digest(interfaceDescriptor.Content); err != nil {
			log.Error("[Service][Contract] do build content digest for create interface descriptor", utils.RequestID(ctx), zap.Error(err))
			return api.NewServiceContractResponse(commonstore.StoreCode2APICode(err), nil)
		}
		interfaces[interfaceId] = interfaceDescriptor
		createData.Interfaces = append(createData.Interfaces, interfaceDescriptor)
		retContract.Interfaces = append(retContract.Interfaces, &apiservice.InterfaceDescriptor{Id: interfaceId})
	}

	// 比较是否需要更新
	saveData, err := s.storage.GetServiceContract(contract.Id)
	if err != nil {
		log.Error("[Service][Contract] get save service_contract when add interfaces", utils.RequestID(ctx), zap.Error(err))
		return api.NewServiceContractResponse(commonstore.StoreCode2APICode(err), nil)
	}
	var needUpdate = false
	if saveData != nil {
		if len(saveData.Interfaces) != len(interfaces) {
			needUpdate = true
		} else {
			for _, localInterface := range saveData.Interfaces {
				if remoteInterface, ok := interfaces[localInterface.ID]; ok {
					if localInterface.Type != contract.Type {
						needUpdate = true
						break
					}
					if localInterface.Content != remoteInterface.Content {
						needUpdate = true
						break
					}
					if len(localInterface.ContentDigest) == 0 && len(localInterface.Content) > 0 {
						// 老版本接口，没有写入digest，这里需要覆盖写入
						needUpdate = true
						break
					}
				} else {
					needUpdate = true
					break
				}
			}
		}
	}

	if !needUpdate {
		return api.NewServiceContractResponse(apimodel.Code_NoNeedUpdate, nil)
	}
	if err := s.storage.AddServiceContractInterfaces(createData); err != nil {
		log.Error("[Service][Contract] full replace service_contract interfaces", utils.RequestID(ctx), zap.Error(err))
		return api.NewServiceContractResponse(commonstore.StoreCode2APICode(err), nil)
	}
	s.RecordHistory(ctx, serviceContractRecordEntry(ctx, contract, createData, model.OUpdate))
	return api.NewServiceContractResponse(apimodel.Code_ExecuteSuccess, retContract)
}

// AppendServiceContractInterfaces 追加服务契约详情
func (s *Server) AppendServiceContractInterfaces(ctx context.Context,
	contract *apiservice.ServiceContract, source apiservice.InterfaceDescriptor_Source) *apiservice.Response {

	if errRsp := checkOperationServiceContractInterface(contract); errRsp != nil {
		return errRsp
	}
	contract.Type = utils.DefaultString(contract.GetType(), contract.GetName())
	saveData, err := s.storage.GetServiceContract(contract.Id)
	if err != nil {
		log.Error("[Service][Contract] get save service_contract when append interfaces", utils.RequestID(ctx), zap.Error(err))
		return api.NewServiceContractResponse(commonstore.StoreCode2APICode(err), nil)
	}
	if saveData == nil {
		return api.NewServiceContractResponse(apimodel.Code_NotFoundResource, nil)
	}

	appendData := &model.EnrichServiceContract{
		ServiceContract: &model.ServiceContract{
			ID:       contract.Id,
			Revision: utils.NewUUID(),
		},
		Interfaces: make([]*model.InterfaceDescriptor, 0, len(contract.Interfaces)),
	}
	retContract := &apiservice.ServiceContract{Id: contract.Id}
	for _, item := range contract.Interfaces {
		interfaceId, errRsp := utils.CheckContractInterfaceTetrad(appendData.ID, source, item)
		if errRsp != nil {
			log.Error("[Service][Contract] check service_contract interface id", utils.RequestID(ctx),
				zap.String("err", errRsp.GetInfo().GetValue()))
			return errRsp
		}
		interfaceDescriptor := &model.InterfaceDescriptor{
			ID:         interfaceId,
			ContractID: contract.Id,
			Namespace:  contract.Namespace,
			Service:    contract.Service,
			Protocol:   contract.Protocol,
			Version:    contract.Version,
			Type:       contract.GetType(),
			Method:     item.Method,
			Path:       item.Path,
			Content:    item.Content,
			Source:     source,
			Revision:   utils.NewUUID(),
		}
		if interfaceDescriptor.ContentDigest, err = utils.BuildSha1Digest(interfaceDescriptor.Content); err != nil {
			log.Error("[Service][Contract] do build content digest for create interface descriptor", utils.RequestID(ctx), zap.Error(err))
			return api.NewServiceContractResponse(commonstore.StoreCode2APICode(err), nil)
		}
		appendData.Interfaces = append(appendData.Interfaces, interfaceDescriptor)
		retContract.Interfaces = append(retContract.Interfaces, &apiservice.InterfaceDescriptor{Id: interfaceId})

	}
	if err := s.storage.AppendServiceContractInterfaces(appendData); err != nil {
		log.Error("[Service][Contract] append service_contract interfaces", utils.RequestID(ctx), zap.Error(err))
		return api.NewServiceContractResponse(commonstore.StoreCode2APICode(err), nil)
	}
	s.RecordHistory(ctx, serviceContractRecordEntry(ctx, contract, appendData, model.OUpdate))
	return api.NewServiceContractResponse(apimodel.Code_ExecuteSuccess, retContract)
}

// DeleteServiceContractInterfaces 删除服务契约详情
func (s *Server) DeleteServiceContractInterfaces(ctx context.Context,
	contract *apiservice.ServiceContract) *apiservice.Response {

	if errRsp := checkOperationServiceContractInterface(contract); errRsp != nil {
		return errRsp
	}

	saveData, err := s.storage.GetServiceContract(contract.Id)
	if err != nil {
		log.Error("[Service][Contract] get save service_contract when delete interfaces", utils.RequestID(ctx), zap.Error(err))
		return api.NewServiceContractResponse(commonstore.StoreCode2APICode(err), nil)
	}
	if saveData == nil {
		return api.NewServiceContractResponse(apimodel.Code_NotFoundResource, nil)
	}

	deleteData := &model.EnrichServiceContract{
		ServiceContract: &model.ServiceContract{
			ID:       saveData.ID,
			Revision: utils.NewUUID(),
		},
		Interfaces: make([]*model.InterfaceDescriptor, 0, len(contract.Interfaces)),
	}
	retContract := &apiservice.ServiceContract{Id: contract.Id}
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
			Path:       item.Path,
			Type:       utils.DefaultString(item.Type, contract.GetType()),
		})
		retContract.Interfaces = append(retContract.Interfaces, &apiservice.InterfaceDescriptor{Id: interfaceId})
	}
	if err := s.storage.DeleteServiceContractInterfaces(deleteData); err != nil {
		log.Error("[Service][Contract] delete service_contract interfaces", utils.RequestID(ctx), zap.Error(err))
		return api.NewServiceContractResponse(commonstore.StoreCode2APICode(err), nil)
	}
	s.RecordHistory(ctx, serviceContractRecordEntry(ctx, contract, deleteData, model.ODelete))
	return api.NewServiceContractResponse(apimodel.Code_ExecuteSuccess, retContract)
}

const (
	briefSearch = "brief"
)

func (s *Server) GetServiceInterfaces(ctx context.Context, filter map[string]string) *apiservice.BatchQueryResponse {
	out := api.NewBatchQueryResponse(apimodel.Code_ExecuteSuccess)
	out.Amount = utils.NewUInt32Value(0)
	out.Size = utils.NewUInt32Value(0)

	var isBrief = false
	if bValue, ok := filter[briefSearch]; ok && strings.ToLower(bValue) == "true" {
		isBrief = true
		delete(filter, briefSearch)
	}

	searchFilters := map[string]string{}
	for k, v := range filter {
		newK, ok := interfaceSearchFilters[k]
		if !ok {
			log.Error("[Server][Contract][Query] not allowed", zap.String("attribute", k), utils.RequestID(ctx))
			return api.NewBatchQueryResponseWithMsg(apimodel.Code_InvalidParameter, k+" is not allowed")
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

	if _, ok := searchFilters["order_field"]; !ok {
		searchFilters["order_field"] = "mtime"
	}
	if _, ok := searchFilters["order_type"]; !ok {
		searchFilters["order_type"] = "desc"
	}

	total, ret, err := s.storage.GetInterfaceDescriptors(ctx, searchFilters, offset, limit)
	if err != nil {
		log.Error("[Service][Contract] query service_contract interfaces fail", utils.RequestID(ctx), zap.Error(err))
		return api.NewBatchQueryResponse(commonstore.StoreCode2APICode(err))
	}
	out.Amount = utils.NewUInt32Value(total)
	out.Size = utils.NewUInt32Value(uint32(len(ret)))
	for i := range ret {
		if isBrief {
			ret[i].Content = ""
		}
		if err := api.AddAnyDataIntoBatchQuery(out, ret[i].ToSpec()); err != nil {
			log.Error("[Service][Contract] query service_contract interfaces fail", utils.RequestID(ctx), zap.Error(err))
			return api.NewBatchQueryResponse(apimodel.Code_ExecuteException)
		}
	}
	return out
}

func checkOperationServiceContractInterface(contract *apiservice.ServiceContract) *apiservice.Response {
	if contract.Id != "" {
		return nil
	}
	id, errRsp := utils.CheckContractTetrad(contract)
	if errRsp != nil {
		return errRsp
	}
	contract.Id = id
	return nil
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

func checkBaseServiceContract(req *apiservice.ServiceContract) *apiservice.Response {
	if err := utils.CheckResourceName(utils.NewStringValue(req.GetNamespace())); err != nil {
		return api.NewResponse(apimodel.Code_InvalidNamespaceName)
	}
	if req.GetName() == "" && req.GetType() == "" {
		return api.NewResponseWithMsg(apimodel.Code_BadRequest, "invalid service_contract name")
	}
	if req.GetProtocol() == "" {
		return api.NewResponseWithMsg(apimodel.Code_BadRequest, "invalid service_contract protocol")
	}
	return nil
}
