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
	"errors"
	"strings"

	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	"github.com/polarismesh/specification/source/go/api/v1/service_manage"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"

	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/log"
	"github.com/polarismesh/polaris/common/utils"
)

var (
	serviceFilter           = 1 // 过滤服务的
	instanceFilter          = 2 // 过滤实例的
	serviceMetaFilter       = 3 // 过滤service Metadata的
	instanceMetaFilter      = 4 // 过滤instance Metadata的
	ServiceFilterAttributes = map[string]int{
		"name":        serviceFilter,
		"namespace":   serviceFilter,
		"business":    serviceFilter,
		"department":  serviceFilter,
		"cmdb_mod1":   serviceFilter,
		"cmdb_mod2":   serviceFilter,
		"cmdb_mod3":   serviceFilter,
		"owner":       serviceFilter,
		"offset":      serviceFilter,
		"limit":       serviceFilter,
		"platform_id": serviceFilter,
		// 只返回存在健康实例的服务列表
		"only_exist_health_instance": serviceFilter,
		"host":                       instanceFilter,
		"port":                       instanceFilter,
		"keys":                       serviceMetaFilter,
		"values":                     serviceMetaFilter,
		"instance_keys":              instanceMetaFilter,
		"instance_values":            instanceMetaFilter,
	}
)

// CreateServices implements service.DiscoverServer.
func (svr *Server) CreateServices(ctx context.Context,
	req []*service_manage.Service) *service_manage.BatchWriteResponse {
	if checkError := checkBatchService(req); checkError != nil {
		return checkError
	}
	batchRsp := api.NewBatchWriteResponse(apimodel.Code_ExecuteSuccess)
	for i := range req {
		rsp := checkCreateService(req[i])
		api.Collect(batchRsp, rsp)
	}
	if !api.IsSuccess(batchRsp) {
		return batchRsp
	}
	return svr.nextSvr.CreateServices(ctx, req)
}

// DeleteServices implements service.DiscoverServer.
func (svr *Server) DeleteServices(ctx context.Context,
	req []*service_manage.Service) *service_manage.BatchWriteResponse {
	if checkError := checkBatchService(req); checkError != nil {
		return checkError
	}
	batchRsp := api.NewBatchWriteResponse(apimodel.Code_ExecuteSuccess)
	for i := range req {
		rsp := checkReviseService(req[i])
		api.Collect(batchRsp, rsp)
	}
	if !api.IsSuccess(batchRsp) {
		return batchRsp
	}
	return svr.nextSvr.DeleteServices(ctx, req)
}

// UpdateServices implements service.DiscoverServer.
func (svr *Server) UpdateServices(ctx context.Context, req []*service_manage.Service) *service_manage.BatchWriteResponse {
	if checkError := checkBatchService(req); checkError != nil {
		return checkError
	}
	batchRsp := api.NewBatchWriteResponse(apimodel.Code_ExecuteSuccess)
	for i := range req {
		rsp := checkReviseService(req[i])
		// 待更新的参数检查
		if err := checkMetadata(req[i].GetMetadata()); err != nil {
			rsp = api.NewServiceResponse(apimodel.Code_InvalidMetadata, req[i])
		}
		api.Collect(batchRsp, rsp)
	}
	if !api.IsSuccess(batchRsp) {
		return batchRsp
	}
	return svr.nextSvr.UpdateServices(ctx, req)
}

// GetAllServices implements service.DiscoverServer.
func (svr *Server) GetAllServices(ctx context.Context,
	query map[string]string) *service_manage.BatchQueryResponse {
	return svr.nextSvr.GetAllServices(ctx, query)
}

// GetServiceOwner implements service.DiscoverServer.
func (svr *Server) GetServiceOwner(ctx context.Context,
	req []*service_manage.Service) *service_manage.BatchQueryResponse {
	if err := checkBatchReadService(req); err != nil {
		return err
	}
	return svr.nextSvr.GetServiceOwner(ctx, req)
}

// GetServiceToken implements service.DiscoverServer.
func (svr *Server) GetServiceToken(ctx context.Context, req *service_manage.Service) *service_manage.Response {
	// 校验参数合法性
	if resp := checkReviseService(req); resp != nil {
		return resp
	}
	return svr.nextSvr.GetServiceToken(ctx, req)
}

// GetServices implements service.DiscoverServer.
func (svr *Server) GetServices(ctx context.Context, query map[string]string) *service_manage.BatchQueryResponse {
	var (
		inputInstMetaKeys, inputInstMetaValues string
	)
	for key, value := range query {
		typ, ok := ServiceFilterAttributes[key]
		if !ok {
			log.Errorf("[Server][Service][Query] attribute(%s) it not allowed", key)
			return api.NewBatchQueryResponseWithMsg(apimodel.Code_InvalidParameter, key+" is not allowed")
		}
		// 元数据value允许为空
		if key != "values" && value == "" {
			log.Errorf("[Server][Service][Query] attribute(%s: %s) is not allowed empty", key, value)
			return api.NewBatchQueryResponseWithMsg(
				apimodel.Code_InvalidParameter, "the value for "+key+" is empty")
		}
		switch {
		case typ == instanceMetaFilter:
			if key == "instance_keys" {
				inputInstMetaKeys = value
			} else {
				inputInstMetaValues = value
			}
		}
	}

	if inputInstMetaKeys != "" {
		instMetaKeys := strings.Split(inputInstMetaKeys, ",")
		instMetaValues := strings.Split(inputInstMetaValues, ",")
		if len(instMetaKeys) != len(instMetaValues) {
			log.Errorf("[Server][Service][Query] length of instance meta %s and %s should be equal",
				inputInstMetaKeys, inputInstMetaValues)
			return api.NewBatchQueryResponseWithMsg(apimodel.Code_InvalidParameter,
				" length of instance_keys and instance_values are not equal")
		}
	}

	return svr.nextSvr.GetServices(ctx, query)
}

// GetServicesCount implements service.DiscoverServer.
func (svr *Server) GetServicesCount(ctx context.Context) *service_manage.BatchQueryResponse {
	return svr.nextSvr.GetServicesCount(ctx)
}

// UpdateServiceToken implements service.DiscoverServer.
func (svr *Server) UpdateServiceToken(ctx context.Context, req *service_manage.Service) *service_manage.Response {
	// 校验参数合法性
	if resp := checkReviseService(req); resp != nil {
		return resp
	}
	return svr.nextSvr.UpdateServiceToken(ctx, req)
}

// checkBatchService检查批量请求
func checkBatchService(req []*apiservice.Service) *apiservice.BatchWriteResponse {
	if len(req) == 0 {
		return api.NewBatchWriteResponse(apimodel.Code_EmptyRequest)
	}

	if len(req) > utils.MaxBatchSize {
		return api.NewBatchWriteResponse(apimodel.Code_BatchSizeOverLimit)
	}

	return nil
}

// checkBatchReadService 检查批量读请求
func checkBatchReadService(req []*apiservice.Service) *apiservice.BatchQueryResponse {
	if len(req) == 0 {
		return api.NewBatchQueryResponse(apimodel.Code_EmptyRequest)
	}

	if len(req) > utils.MaxBatchSize {
		return api.NewBatchQueryResponse(apimodel.Code_BatchSizeOverLimit)
	}

	return nil
}

// checkCreateService 检查创建服务请求参数
func checkCreateService(req *apiservice.Service) *apiservice.Response {
	if req == nil {
		return api.NewServiceResponse(apimodel.Code_EmptyRequest, req)
	}

	if err := utils.CheckResourceName(req.GetName()); err != nil {
		return api.NewServiceResponse(apimodel.Code_InvalidServiceName, req)
	}

	if err := utils.CheckResourceName(req.GetNamespace()); err != nil {
		return api.NewServiceResponse(apimodel.Code_InvalidNamespaceName, req)
	}

	if err := checkMetadata(req.GetMetadata()); err != nil {
		return api.NewServiceResponse(apimodel.Code_InvalidMetadata, req)
	}

	// 检查字段长度是否大于DB中对应字段长
	err, notOk := CheckDbServiceFieldLen(req)
	if notOk {
		return err
	}

	return nil
}

// checkReviseService 检查删除/修改/服务token的服务请求参数
func checkReviseService(req *apiservice.Service) *apiservice.Response {
	if req == nil {
		return api.NewServiceResponse(apimodel.Code_EmptyRequest, req)
	}

	if err := utils.CheckResourceName(req.GetName()); err != nil {
		return api.NewServiceResponse(apimodel.Code_InvalidServiceName, req)
	}

	if err := utils.CheckResourceName(req.GetNamespace()); err != nil {
		return api.NewServiceResponse(apimodel.Code_InvalidNamespaceName, req)
	}

	// 检查字段长度是否大于DB中对应字段长
	err, notOk := CheckDbServiceFieldLen(req)
	if notOk {
		return err
	}

	return nil
}

// CheckDbServiceFieldLen 检查DB中service表对应的入参字段合法性
func CheckDbServiceFieldLen(req *apiservice.Service) (*apiservice.Response, bool) {
	if err := utils.CheckDbStrFieldLen(req.GetName(), utils.MaxDbServiceNameLength); err != nil {
		return api.NewServiceResponse(apimodel.Code_InvalidServiceName, req), true
	}
	if err := utils.CheckDbStrFieldLen(req.GetNamespace(), utils.MaxDbServiceNamespaceLength); err != nil {
		return api.NewServiceResponse(apimodel.Code_InvalidNamespaceName, req), true
	}
	if err := utils.CheckDbMetaDataFieldLen(req.GetMetadata()); err != nil {
		return api.NewServiceResponse(apimodel.Code_InvalidMetadata, req), true
	}
	if err := utils.CheckDbStrFieldLen(req.GetPorts(), utils.MaxDbServicePortsLength); err != nil {
		return api.NewServiceResponse(apimodel.Code_InvalidServicePorts, req), true
	}
	if err := utils.CheckDbStrFieldLen(req.GetBusiness(), utils.MaxDbServiceBusinessLength); err != nil {
		return api.NewServiceResponse(apimodel.Code_InvalidServiceBusiness, req), true
	}
	if err := utils.CheckDbStrFieldLen(req.GetDepartment(), utils.MaxDbServiceDeptLength); err != nil {
		return api.NewServiceResponse(apimodel.Code_InvalidServiceDepartment, req), true
	}
	if err := utils.CheckDbStrFieldLen(req.GetCmdbMod1(), utils.MaxDbServiceCMDBLength); err != nil {
		return api.NewServiceResponse(apimodel.Code_InvalidServiceCMDB, req), true
	}
	if err := utils.CheckDbStrFieldLen(req.GetCmdbMod2(), utils.MaxDbServiceCMDBLength); err != nil {
		return api.NewServiceResponse(apimodel.Code_InvalidServiceCMDB, req), true
	}
	if err := utils.CheckDbStrFieldLen(req.GetCmdbMod3(), utils.MaxDbServiceCMDBLength); err != nil {
		return api.NewServiceResponse(apimodel.Code_InvalidServiceCMDB, req), true
	}
	if err := utils.CheckDbStrFieldLen(req.GetComment(), utils.MaxDbServiceCommentLength); err != nil {
		return api.NewServiceResponse(apimodel.Code_InvalidServiceComment, req), true
	}
	if err := utils.CheckDbStrFieldLen(req.GetOwners(), utils.MaxDbServiceOwnerLength); err != nil {
		return api.NewServiceResponse(apimodel.Code_InvalidServiceOwners, req), true
	}
	if err := utils.CheckDbStrFieldLen(req.GetToken(), utils.MaxDbServiceToken); err != nil {
		return api.NewServiceResponse(apimodel.Code_InvalidServiceToken, req), true
	}
	if err := utils.CheckDbStrFieldLen(req.GetPlatformId(), utils.MaxPlatformIDLength); err != nil {
		return api.NewServiceResponse(apimodel.Code_InvalidPlatformID, req), true
	}
	return nil, false
}

// checkMetadata 检查metadata的个数; 最大是64个
// key/value是否符合要求
func checkMetadata(meta map[string]string) error {
	if meta == nil {
		return nil
	}

	if len(meta) > utils.MaxMetadataLength {
		return errors.New("metadata is too long")
	}

	/*regStr := "^[0-9A-Za-z-._*]+$"
	  matchFunc := func(str string) error {
	  	if str == "" {
	  		return nil
	  	}
	  	ok, err := regexp.MatchString(regStr, str)
	  	if err != nil {
	  		log.Errorf("regexp match string(%s) err: %s", str, err.Error())
	  		return err
	  	}
	  	if !ok {
	  		log.Errorf("metadata string(%s) contains invalid character", str)
	  		return errors.New("contain invalid character")
	  	}
	  	return nil
	  }
	  for key, value := range meta {
	  	if err := matchFunc(key); err != nil {
	  		return err
	  	}
	  	if err := matchFunc(value); err != nil {
	  		return err
	  	}
	  }*/

	return nil
}
