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
	"encoding/json"
	"fmt"
	"time"

	"github.com/gogo/protobuf/jsonpb"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"
	apitraffic "github.com/polarismesh/specification/source/go/api/v1/traffic_manage"

	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/model"
	commonstore "github.com/polarismesh/polaris/common/store"
	commontime "github.com/polarismesh/polaris/common/time"
	"github.com/polarismesh/polaris/common/utils"
)

var (
	// RoutingConfigFilterAttrs router config filter attrs
	RoutingConfigFilterAttrs = map[string]bool{
		"service":   true,
		"namespace": true,
		"offset":    true,
		"limit":     true,
	}
)

// CreateRoutingConfigs Create a routing configuration
func (s *Server) CreateRoutingConfigs(ctx context.Context, req []*apitraffic.Routing) *apiservice.BatchWriteResponse {
	if err := checkBatchRoutingConfig(req); err != nil {
		return err
	}

	resp := api.NewBatchWriteResponse(apimodel.Code_ExecuteSuccess)
	for _, entry := range req {
		api.Collect(resp, s.createRoutingConfigV1toV2(ctx, entry))
	}

	return api.FormatBatchWriteResponse(resp)
}

// CreateRoutingConfig Create a routing configuration, Creating route configuration requires locking
// services to prevent the service from being deleted
// Deprecated: This method is ready to abandon
func (s *Server) CreateRoutingConfig(ctx context.Context, req *apitraffic.Routing) *apiservice.Response {
	rid := utils.ParseRequestID(ctx)
	pid := utils.ParsePlatformID(ctx)
	if resp := checkRoutingConfig(req); resp != nil {
		return resp
	}

	serviceName := req.GetService().GetValue()
	namespaceName := req.GetNamespace().GetValue()
	service, errResp := s.loadService(namespaceName, serviceName)
	if errResp != nil {
		log.Error(errResp.GetInfo().GetValue(), utils.ZapRequestID(rid), utils.ZapPlatformID(pid))
		return api.NewRoutingResponse(apimodel.Code(errResp.GetCode().GetValue()), req)
	}
	if service == nil {
		return api.NewRoutingResponse(apimodel.Code_NotFoundService, req)
	}
	if service.IsAlias() {
		return api.NewRoutingResponse(apimodel.Code_NotAllowAliasCreateRouting, req)
	}

	routingConfig, err := s.storage.GetRoutingConfigWithService(service.Name, service.Namespace)
	if err != nil {
		log.Error(err.Error(), utils.ZapRequestID(rid), utils.ZapPlatformID(pid))
		return api.NewRoutingResponse(commonstore.StoreCode2APICode(err), req)
	}
	if routingConfig != nil {
		return api.NewRoutingResponse(apimodel.Code_ExistedResource, req)
	}

	conf, err := api2RoutingConfig(service.ID, req)
	if err != nil {
		log.Error(err.Error(), utils.ZapRequestID(rid), utils.ZapPlatformID(pid))
		return api.NewRoutingResponse(apimodel.Code_ExecuteException, req)
	}
	if err := s.storage.CreateRoutingConfig(conf); err != nil {
		log.Error(err.Error(), utils.ZapRequestID(rid), utils.ZapPlatformID(pid))
		return wrapperRoutingStoreResponse(req, err)
	}

	s.RecordHistory(ctx, routingRecordEntry(ctx, req, service, conf, model.OCreate))
	return api.NewRoutingResponse(apimodel.Code_ExecuteSuccess, req)
}

// DeleteRoutingConfigs Batch delete routing configuration
func (s *Server) DeleteRoutingConfigs(ctx context.Context, req []*apitraffic.Routing) *apiservice.BatchWriteResponse {
	if err := checkBatchRoutingConfig(req); err != nil {
		return err
	}

	out := api.NewBatchWriteResponse(apimodel.Code_ExecuteSuccess)
	for _, entry := range req {
		resp := s.DeleteRoutingConfig(ctx, entry)
		api.Collect(out, resp)
	}

	return api.FormatBatchWriteResponse(out)
}

// DeleteRoutingConfig Delete a routing configuration
// Deprecated: This method is ready to abandon
func (s *Server) DeleteRoutingConfig(ctx context.Context, req *apitraffic.Routing) *apiservice.Response {
	rid := utils.ParseRequestID(ctx)
	pid := utils.ParsePlatformID(ctx)
	service, resp := s.routingConfigCommonCheck(ctx, req)
	if resp != nil {
		return resp
	}

	if err := s.storage.DeleteRoutingConfig(service.ID); err != nil {
		log.Error(err.Error(), utils.ZapRequestID(rid), utils.ZapPlatformID(pid))
		return wrapperRoutingStoreResponse(req, err)
	}

	s.RecordHistory(ctx, routingRecordEntry(ctx, req, service, nil, model.ODelete))
	return api.NewResponse(apimodel.Code_ExecuteSuccess)
}

// UpdateRoutingConfigs Batch update routing configuration
func (s *Server) UpdateRoutingConfigs(ctx context.Context, req []*apitraffic.Routing) *apiservice.BatchWriteResponse {
	if err := checkBatchRoutingConfig(req); err != nil {
		return err
	}

	out := api.NewBatchWriteResponse(apimodel.Code_ExecuteSuccess)
	for _, entry := range req {
		resp := s.updateRoutingConfigV1toV2(ctx, entry)
		api.Collect(out, resp)
	}

	return api.FormatBatchWriteResponse(out)
}

// UpdateRoutingConfig Update a routing configuration
// Deprecated: 该方法准备舍弃
func (s *Server) UpdateRoutingConfig(ctx context.Context, req *apitraffic.Routing) *apiservice.Response {
	rid := utils.ParseRequestID(ctx)
	pid := utils.ParsePlatformID(ctx)
	service, resp := s.routingConfigCommonCheck(ctx, req)
	if resp != nil {
		return resp
	}

	conf, err := s.storage.GetRoutingConfigWithService(service.Name, service.Namespace)
	if err != nil {
		log.Error(err.Error(), utils.ZapRequestID(rid), utils.ZapPlatformID(pid))
		return api.NewRoutingResponse(commonstore.StoreCode2APICode(err), req)
	}
	if conf == nil {
		return api.NewRoutingResponse(apimodel.Code_NotFoundRouting, req)
	}

	reqModel, err := api2RoutingConfig(service.ID, req)
	if err != nil {
		log.Error(err.Error(), utils.ZapRequestID(rid), utils.ZapPlatformID(pid))
		return api.NewRoutingResponse(apimodel.Code_ParseRoutingException, req)
	}

	if err := s.storage.UpdateRoutingConfig(reqModel); err != nil {
		log.Error(err.Error(), utils.ZapRequestID(rid), utils.ZapPlatformID(pid))
		return wrapperRoutingStoreResponse(req, err)
	}

	s.RecordHistory(ctx, routingRecordEntry(ctx, req, service, reqModel, model.OUpdate))
	return api.NewRoutingResponse(apimodel.Code_ExecuteSuccess, req)
}

// GetRoutingConfigs Get the routing configuration in batches, and provide the interface of
// the query routing configuration to the OSS
// Deprecated: This method is ready to abandon
func (s *Server) GetRoutingConfigs(ctx context.Context, query map[string]string) *apiservice.BatchQueryResponse {
	rid := utils.ParseRequestID(ctx)
	pid := utils.ParsePlatformID(ctx)

	offset, limit, err := utils.ParseOffsetAndLimit(query)
	if err != nil {
		return api.NewBatchQueryResponse(apimodel.Code_InvalidParameter)
	}

	filter := make(map[string]string)
	for key, value := range query {
		if _, ok := RoutingConfigFilterAttrs[key]; !ok {
			log.Errorf("[Server][RoutingConfig][Query] attribute(%s) is not allowed", key)
			return api.NewBatchQueryResponse(apimodel.Code_InvalidParameter)
		}
		filter[key] = value
	}
	// service -- > name This special treatment
	if service, ok := filter["service"]; ok {
		filter["name"] = service
		delete(filter, "service")
	}

	// Can be filtered according to name and namespace
	total, routings, err := s.storage.GetRoutingConfigs(filter, offset, limit)
	if err != nil {
		log.Error(err.Error(), utils.ZapRequestID(rid), utils.ZapPlatformID(pid))
		return api.NewBatchQueryResponse(commonstore.StoreCode2APICode(err))
	}

	resp := api.NewBatchQueryResponse(apimodel.Code_ExecuteSuccess)
	resp.Amount = utils.NewUInt32Value(total)
	resp.Size = utils.NewUInt32Value(uint32(len(routings)))
	resp.Routings = make([]*apitraffic.Routing, 0, len(routings))
	for _, entry := range routings {
		routing, err := routingConfig2API(entry.Config, entry.ServiceName, entry.NamespaceName)
		if err != nil {
			log.Error(err.Error(), utils.ZapRequestID(rid), utils.ZapPlatformID(pid))
			return api.NewBatchQueryResponse(apimodel.Code_ParseRoutingException)
		}
		resp.Routings = append(resp.Routings, routing)
	}

	return resp
}

// routingConfigCommonCheck Public examination of routing configuration operation
func (s *Server) routingConfigCommonCheck(
	ctx context.Context, req *apitraffic.Routing) (*model.Service, *apiservice.Response) {
	if resp := checkRoutingConfig(req); resp != nil {
		return nil, resp
	}

	rid := utils.ParseRequestID(ctx)
	pid := utils.ParsePlatformID(ctx)
	serviceName := req.GetService().GetValue()
	namespaceName := req.GetNamespace().GetValue()

	service, err := s.storage.GetService(serviceName, namespaceName)
	if err != nil {
		log.Error(err.Error(), utils.ZapRequestID(rid), utils.ZapPlatformID(pid))
		return nil, api.NewRoutingResponse(commonstore.StoreCode2APICode(err), req)
	}
	if service == nil {
		return nil, api.NewRoutingResponse(apimodel.Code_NotFoundService, req)
	}

	return service, nil
}

// checkRoutingConfig Check the validity of the basic parameter of the routing configuration
func checkRoutingConfig(req *apitraffic.Routing) *apiservice.Response {
	if req == nil {
		return api.NewRoutingResponse(apimodel.Code_EmptyRequest, req)
	}
	if err := utils.CheckResourceName(req.GetService()); err != nil {
		return api.NewRoutingResponse(apimodel.Code_InvalidServiceName, req)
	}

	if err := utils.CheckResourceName(req.GetNamespace()); err != nil {
		return api.NewRoutingResponse(apimodel.Code_InvalidNamespaceName, req)
	}

	if err := utils.CheckDbStrFieldLen(req.GetService(), MaxDbServiceNameLength); err != nil {
		return api.NewRoutingResponse(apimodel.Code_InvalidServiceName, req)
	}
	if err := utils.CheckDbStrFieldLen(req.GetNamespace(), MaxDbServiceNamespaceLength); err != nil {
		return api.NewRoutingResponse(apimodel.Code_InvalidNamespaceName, req)
	}
	if err := utils.CheckDbStrFieldLen(req.GetServiceToken(), MaxDbServiceToken); err != nil {
		return api.NewRoutingResponse(apimodel.Code_InvalidServiceToken, req)
	}

	return nil
}

// parseServiceRoutingToken Get token from RoutingConfig request parameters
func parseServiceRoutingToken(ctx context.Context, req *apitraffic.Routing) string {
	if reqToken := req.GetServiceToken().GetValue(); reqToken != "" {
		return reqToken
	}

	return utils.ParseToken(ctx)
}

// api2RoutingConfig Convert the API parameter to internal data structure
func api2RoutingConfig(serviceID string, req *apitraffic.Routing) (*model.RoutingConfig, error) {
	inBounds, outBounds, err := marshalRoutingConfig(req.GetInbounds(), req.GetOutbounds())
	if err != nil {
		return nil, err
	}

	out := &model.RoutingConfig{
		ID:        serviceID,
		InBounds:  string(inBounds),
		OutBounds: string(outBounds),
		Revision:  utils.NewUUID(),
	}

	return out, nil
}

// routingConfig2API Convert the internal data structure to API parameter to pass out
func routingConfig2API(req *model.RoutingConfig, service string, namespace string) (*apitraffic.Routing, error) {
	if req == nil {
		return nil, nil
	}

	out := &apitraffic.Routing{
		Service:   utils.NewStringValue(service),
		Namespace: utils.NewStringValue(namespace),
		Revision:  utils.NewStringValue(req.Revision),
		Ctime:     utils.NewStringValue(commontime.Time2String(req.CreateTime)),
		Mtime:     utils.NewStringValue(commontime.Time2String(req.ModifyTime)),
	}

	if req.InBounds != "" {
		var inBounds []*apitraffic.Route
		if err := json.Unmarshal([]byte(req.InBounds), &inBounds); err != nil {
			return nil, err
		}
		out.Inbounds = inBounds
	}
	if req.OutBounds != "" {
		var outBounds []*apitraffic.Route
		if err := json.Unmarshal([]byte(req.OutBounds), &outBounds); err != nil {
			return nil, err
		}
		out.Outbounds = outBounds
	}

	return out, nil
}

// marshalRoutingConfig Formulate Inbounds and OUTBOUNDS
func marshalRoutingConfig(in []*apitraffic.Route, out []*apitraffic.Route) ([]byte, []byte, error) {
	inBounds, err := json.Marshal(in)
	if err != nil {
		return nil, nil, err
	}

	outBounds, err := json.Marshal(out)
	if err != nil {
		return nil, nil, err
	}

	return inBounds, outBounds, nil
}

// checkBatchRoutingConfig Check batch request
func checkBatchRoutingConfig(req []*apitraffic.Routing) *apiservice.BatchWriteResponse {
	if len(req) == 0 {
		return api.NewBatchWriteResponse(apimodel.Code_EmptyRequest)
	}

	if len(req) > MaxBatchSize {
		return api.NewBatchWriteResponse(apimodel.Code_BatchSizeOverLimit)
	}

	return nil
}

// routingRecordEntry Construction of RoutingConfig's record Entry
func routingRecordEntry(ctx context.Context, req *apitraffic.Routing, svc *model.Service, md *model.RoutingConfig,
	opt model.OperationType) *model.RecordEntry {

	marshaler := jsonpb.Marshaler{}
	detail, _ := marshaler.MarshalToString(req)

	entry := &model.RecordEntry{
		ResourceType:  model.RRouting,
		ResourceName:  fmt.Sprintf("%s(%s)", svc.Name, svc.ID),
		Namespace:     svc.Namespace,
		OperationType: opt,
		Operator:      utils.ParseOperator(ctx),
		Detail:        detail,
		HappenTime:    time.Now(),
	}

	return entry
}

// routingV2RecordEntry Construction of RoutingConfig's record Entry
func routingV2RecordEntry(ctx context.Context, req *apitraffic.RouteRule, md *model.RouterConfig,
	opt model.OperationType) *model.RecordEntry {

	marshaler := jsonpb.Marshaler{}
	detail, _ := marshaler.MarshalToString(req)

	entry := &model.RecordEntry{
		ResourceType:  model.RRouting,
		ResourceName:  fmt.Sprintf("%s(%s)", md.Name, md.ID),
		Namespace:     req.GetNamespace(),
		OperationType: opt,
		Operator:      utils.ParseOperator(ctx),
		Detail:        detail,
		HappenTime:    time.Now(),
	}
	return entry
}

// wrapperRoutingStoreResponse Packing routing storage layer error
func wrapperRoutingStoreResponse(routing *apitraffic.Routing, err error) *apiservice.Response {
	if err == nil {
		return nil
	}
	resp := api.NewResponseWithMsg(commonstore.StoreCode2APICode(err), err.Error())
	resp.Routing = routing
	return resp
}
