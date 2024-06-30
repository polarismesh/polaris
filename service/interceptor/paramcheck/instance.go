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
	"strconv"
	"strings"

	"github.com/golang/protobuf/ptypes/wrappers"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	"github.com/polarismesh/specification/source/go/api/v1/service_manage"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"
	"google.golang.org/protobuf/types/known/wrapperspb"

	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/log"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/plugin"
)

var (
	// InstanceFilterAttributes 查询实例支持的过滤字段
	InstanceFilterAttributes = map[string]bool{
		"id":            true, // 实例ID
		"service":       true, // 服务name
		"namespace":     true, // 服务namespace
		"host":          true,
		"port":          true,
		"keys":          true,
		"values":        true,
		"protocol":      true,
		"version":       true,
		"health_status": true,
		"healthy":       true, // health_status, healthy都有，以healthy为准
		"isolate":       true,
		"weight":        true,
		"logic_set":     true,
		"cmdb_region":   true,
		"cmdb_zone":     true,
		"cmdb_idc":      true,
		"priority":      true,
		"offset":        true,
		"limit":         true,
	}
)

// CreateInstances implements service.DiscoverServer.
func (svr *Server) CreateInstances(ctx context.Context,
	reqs []*service_manage.Instance) *service_manage.BatchWriteResponse {
	if checkError := checkBatchInstance(reqs); checkError != nil {
		return checkError
	}
	batchRsp := api.NewBatchWriteResponse(apimodel.Code_ExecuteSuccess)
	for i := range reqs {
		req := reqs[i]
		instanceID, checkError := checkCreateInstance(req)
		if checkError != nil {
			api.Collect(batchRsp, checkError)
			continue
		}
		// Restricted Instance frequently registered
		if ok := svr.allowInstanceAccess(instanceID); !ok {
			log.Error("create instance not allowed to access: exceed ratelimit",
				utils.RequestID(ctx), utils.ZapInstanceID(instanceID))
			api.Collect(batchRsp, api.NewInstanceResponse(apimodel.Code_InstanceTooManyRequests, req))
			continue
		}
		req.Id = wrapperspb.String(instanceID)
		reqs[i] = req
	}
	if !api.IsSuccess(batchRsp) {
		return batchRsp
	}

	return svr.nextSvr.CreateInstances(ctx, reqs)
}

// DeleteInstances implements service.DiscoverServer.
func (svr *Server) DeleteInstances(ctx context.Context,
	reqs []*service_manage.Instance) *service_manage.BatchWriteResponse {
	if checkError := checkBatchInstance(reqs); checkError != nil {
		return checkError
	}
	batchRsp := api.NewBatchWriteResponse(apimodel.Code_ExecuteSuccess)
	for i := range reqs {
		req := reqs[i]
		instanceID, checkError := checkReviseInstance(req)
		if checkError != nil {
			api.Collect(batchRsp, checkError)
			continue
		}
		// Restricted Instance frequently registered
		if ok := svr.allowInstanceAccess(instanceID); !ok {
			log.Error("delete instance is not allow access", utils.RequestID(ctx))
			api.Collect(batchRsp, api.NewInstanceResponse(apimodel.Code_InstanceTooManyRequests, req))
			continue
		}
		req.Id = wrapperspb.String(instanceID)
		reqs[i] = req
	}
	if !api.IsSuccess(batchRsp) {
		return batchRsp
	}
	return svr.nextSvr.DeleteInstances(ctx, reqs)
}

// DeleteInstancesByHost implements service.DiscoverServer.
func (svr *Server) DeleteInstancesByHost(ctx context.Context,
	reqs []*service_manage.Instance) *service_manage.BatchWriteResponse {
	if checkError := checkBatchInstance(reqs); checkError != nil {
		return checkError
	}
	batchRsp := api.NewBatchWriteResponse(apimodel.Code_ExecuteSuccess)
	for i := range reqs {
		// 参数校验
		if err := checkInstanceByHost(reqs[i]); err != nil {
			api.Collect(batchRsp, err)
			continue
		}
	}
	if !api.IsSuccess(batchRsp) {
		return batchRsp
	}

	return svr.nextSvr.DeleteInstancesByHost(ctx, reqs)
}

// GetInstanceLabels implements service.DiscoverServer.
func (svr *Server) GetInstanceLabels(ctx context.Context,
	query map[string]string) *service_manage.Response {
	return svr.nextSvr.GetInstanceLabels(ctx, query)
}

// GetInstances implements service.DiscoverServer.
func (svr *Server) GetInstances(ctx context.Context,
	query map[string]string) *service_manage.BatchQueryResponse {

	// 不允许全量查询服务实例
	if len(query) == 0 {
		return api.NewBatchQueryResponse(apimodel.Code_EmptyQueryParameter)
	}

	var metaFilter map[string]string
	metaKey, metaKeyAvail := query["keys"]
	metaValue, metaValueAvail := query["values"]
	if metaKeyAvail != metaValueAvail {
		return api.NewBatchQueryResponseWithMsg(
			apimodel.Code_InvalidQueryInsParameter, "instance metadata key and value must be both provided")
	}
	if metaKeyAvail {
		metaFilter = map[string]string{}
		keys := strings.Split(metaKey, ",")
		values := strings.Split(metaValue, ",")
		if len(keys) == len(values) {
			for i := range keys {
				metaFilter[keys[i]] = values[i]
			}
		} else {
			return api.NewBatchQueryResponseWithMsg(
				apimodel.Code_InvalidQueryInsParameter, "instance metadata key and value length are different")
		}
	}

	// 以healthy为准
	_, lhs := query["health_status"]
	_, rhs := query["healthy"]
	if lhs && rhs {
		delete(query, "health_status")
	}

	for key, value := range query {
		if _, ok := InstanceFilterAttributes[key]; !ok {
			log.Errorf("[Server][Instance][Query] attribute(%s) is not allowed", key)
			return api.NewBatchQueryResponseWithMsg(
				apimodel.Code_InvalidParameter, key+" is not allowed")
		}

		if value == "" {
			log.Errorf("[Server][Instance][Query] attribute(%s: %s) is not allowed empty", key, value)
			return api.NewBatchQueryResponseWithMsg(
				apimodel.Code_InvalidParameter, "the value for "+key+" is empty")
		}
	}

	offset, limit, err := utils.ParseOffsetAndLimit(query)
	if err != nil {
		return api.NewBatchQueryResponse(apimodel.Code_InvalidParameter)
	}
	query["offset"] = strconv.FormatUint(uint64(offset), 10)
	query["limit"] = strconv.FormatUint(uint64(limit), 10)
	return svr.nextSvr.GetInstances(ctx, query)
}

// GetInstancesCount implements service.DiscoverServer.
func (svr *Server) GetInstancesCount(ctx context.Context) *service_manage.BatchQueryResponse {
	return svr.nextSvr.GetInstancesCount(ctx)
}

// UpdateInstances implements service.DiscoverServer.
func (svr *Server) UpdateInstances(ctx context.Context, reqs []*service_manage.Instance) *service_manage.BatchWriteResponse {
	if checkError := checkBatchInstance(reqs); checkError != nil {
		return checkError
	}
	batchRsp := api.NewBatchWriteResponse(apimodel.Code_ExecuteSuccess)
	for i := range reqs {
		if err := checkMetadata(reqs[i].GetMetadata()); err != nil {
			api.Collect(batchRsp, api.NewInstanceResponse(apimodel.Code_InvalidMetadata, reqs[i]))
			continue
		}
		// 参数检查
		instanceID, checkError := checkReviseInstance(reqs[i])
		if checkError != nil {
			api.Collect(batchRsp, checkError)
			continue
		}
		reqs[i].Id = wrapperspb.String(instanceID)
	}
	if !api.IsSuccess(batchRsp) {
		return batchRsp
	}
	return svr.nextSvr.UpdateInstances(ctx, reqs)
}

// UpdateInstancesIsolate implements service.DiscoverServer.
func (svr *Server) UpdateInstancesIsolate(ctx context.Context, reqs []*service_manage.Instance) *service_manage.BatchWriteResponse {
	if checkError := checkBatchInstance(reqs); checkError != nil {
		return checkError
	}
	batchRsp := api.NewBatchWriteResponse(apimodel.Code_ExecuteSuccess)
	for i := range reqs {
		// 参数校验
		if err := checkInstanceByHost(reqs[i]); err != nil {
			api.Collect(batchRsp, err)
			continue
		}
	}
	return svr.nextSvr.UpdateInstancesIsolate(ctx, reqs)
}

/*
 * @brief 检查批量请求
 */
func checkBatchInstance(req []*apiservice.Instance) *apiservice.BatchWriteResponse {
	if len(req) == 0 {
		return api.NewBatchWriteResponse(apimodel.Code_EmptyRequest)
	}

	if len(req) > utils.MaxBatchSize {
		return api.NewBatchWriteResponse(apimodel.Code_BatchSizeOverLimit)
	}

	return nil
}

/*
 * @brief 检查创建服务实例请求参数
 */
func checkCreateInstance(req *apiservice.Instance) (string, *apiservice.Response) {
	if req == nil {
		return "", api.NewInstanceResponse(apimodel.Code_EmptyRequest, req)
	}

	if err := checkMetadata(req.GetMetadata()); err != nil {
		return "", api.NewInstanceResponse(apimodel.Code_InvalidMetadata, req)
	}

	// 检查字段长度是否大于DB中对应字段长
	err, notOk := CheckDbInstanceFieldLen(req)
	if notOk {
		return "", err
	}

	return utils.CheckInstanceTetrad(req)
}

/*
 * @brief 检查删除/修改服务实例请求参数
 */
func checkReviseInstance(req *apiservice.Instance) (string, *apiservice.Response) {
	if req == nil {
		return "", api.NewInstanceResponse(apimodel.Code_EmptyRequest, req)
	}

	if req.GetId() != nil {
		if req.GetId().GetValue() == "" {
			return "", api.NewInstanceResponse(apimodel.Code_InvalidInstanceID, req)
		}
		return req.GetId().GetValue(), nil
	}

	// 检查字段长度是否大于DB中对应字段长
	err, notOk := CheckDbInstanceFieldLen(req)
	if notOk {
		return "", err
	}

	return utils.CheckInstanceTetrad(req)
}

// CheckDbInstanceFieldLen 检查DB中service表对应的入参字段合法性
func CheckDbInstanceFieldLen(req *apiservice.Instance) (*apiservice.Response, bool) {
	if err := utils.CheckDbStrFieldLen(req.GetService(), utils.MaxDbServiceNameLength); err != nil {
		return api.NewInstanceResponse(apimodel.Code_InvalidServiceName, req), true
	}
	if err := utils.CheckDbStrFieldLen(req.GetNamespace(), utils.MaxDbServiceNamespaceLength); err != nil {
		return api.NewInstanceResponse(apimodel.Code_InvalidNamespaceName, req), true
	}
	if err := utils.CheckDbStrFieldLen(req.GetHost(), utils.MaxDbInsHostLength); err != nil {
		return api.NewInstanceResponse(apimodel.Code_InvalidInstanceHost, req), true
	}
	if err := utils.CheckDbStrFieldLen(req.GetProtocol(), utils.MaxDbInsProtocolLength); err != nil {
		return api.NewInstanceResponse(apimodel.Code_InvalidInstanceProtocol, req), true
	}
	if err := utils.CheckDbStrFieldLen(req.GetVersion(), utils.MaxDbInsVersionLength); err != nil {
		return api.NewInstanceResponse(apimodel.Code_InvalidInstanceVersion, req), true
	}
	if err := utils.CheckDbStrFieldLen(req.GetLogicSet(), utils.MaxDbInsLogicSetLength); err != nil {
		return api.NewInstanceResponse(apimodel.Code_InvalidInstanceLogicSet, req), true
	}
	if err := utils.CheckDbMetaDataFieldLen(req.GetMetadata()); err != nil {
		return api.NewInstanceResponse(apimodel.Code_InvalidMetadata, req), true
	}
	if req.GetPort().GetValue() > 65535 {
		return api.NewInstanceResponse(apimodel.Code_InvalidInstancePort, req), true
	}

	if req.GetWeight().GetValue() > 65535 {
		return api.NewInstanceResponse(apimodel.Code_InvalidParameter, req), true
	}
	return nil, false
}

// 实例访问限流
func (s *Server) allowInstanceAccess(instanceID string) bool {
	if s.ratelimit == nil {
		return true
	}

	return s.ratelimit.Allow(plugin.InstanceRatelimit, instanceID)
}

/**
 * @brief 根据ip隔离和删除服务实例的参数检查
 */
func checkInstanceByHost(req *apiservice.Instance) *apiservice.Response {
	if req == nil {
		return api.NewInstanceResponse(apimodel.Code_EmptyRequest, req)
	}
	if err := utils.CheckResourceName(req.GetService()); err != nil {
		return api.NewInstanceResponse(apimodel.Code_InvalidServiceName, req)
	}
	if err := utils.CheckResourceName(req.GetNamespace()); err != nil {
		return api.NewInstanceResponse(apimodel.Code_InvalidNamespaceName, req)
	}
	if err := checkInstanceHost(req.GetHost()); err != nil {
		return api.NewInstanceResponse(apimodel.Code_InvalidInstanceHost, req)
	}
	return nil
}

// checkInstanceHost 检查服务实例Host
func checkInstanceHost(host *wrappers.StringValue) error {
	if host == nil {
		return errors.New(utils.NilErrString)
	}

	if host.GetValue() == "" {
		return errors.New(utils.EmptyErrString)
	}

	return nil
}
