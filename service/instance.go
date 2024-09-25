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
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/golang/protobuf/ptypes/wrappers"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/wrapperspb"

	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/eventhub"
	"github.com/polarismesh/polaris/common/model"
	commonstore "github.com/polarismesh/polaris/common/store"
	commontime "github.com/polarismesh/polaris/common/time"
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
	// InsFilter2toreAttr 查询字段转为存储层的属性值，映射表
	InsFilter2toreAttr = map[string]string{
		"service": "name",
		"healthy": "health_status",
	}
	// NotInsFilterAttr 不属于 instance 表属性的字段
	NotInsFilterAttr = map[string]bool{
		"keys":   true,
		"values": true,
	}
)

// CreateInstances 批量创建服务实例
func (s *Server) CreateInstances(ctx context.Context, reqs []*apiservice.Instance) *apiservice.BatchWriteResponse {
	return batchOperateInstances(ctx, reqs, s.CreateInstance)
}

// CreateInstance create a single service instance
func (s *Server) CreateInstance(ctx context.Context, req *apiservice.Instance) *apiservice.Response {
	start := time.Now()

	// Prevent pollution api.Instance struct, copy and fill token
	ins := *req
	ins.ServiceToken = utils.NewStringValue(parseInstanceReqToken(ctx, req))
	data, resp := s.createInstance(ctx, req, &ins)
	if resp != nil {
		return resp
	}

	msg := fmt.Sprintf("create instance: id=%v, namespace=%v, service=%v, host=%v, port=%v",
		ins.GetId().GetValue(), req.GetNamespace().GetValue(), req.GetService().GetValue(),
		req.GetHost().GetValue(), req.GetPort().GetValue())
	log.Info(msg, utils.RequestID(ctx), zap.Duration("cost", time.Since(start)))
	svc := &model.Service{
		Name:      req.GetService().GetValue(),
		Namespace: req.GetNamespace().GetValue(),
	}
	instanceProto := data.Proto
	event := &model.InstanceEvent{
		Id:         req.GetId().GetValue(),
		Namespace:  svc.Namespace,
		Service:    svc.Name,
		Instance:   instanceProto,
		EType:      model.EventInstanceOnline,
		CreateTime: time.Time{},
	}
	event.InjectMetadata(ctx)
	s.sendDiscoverEvent(*event)
	s.RecordHistory(ctx, instanceRecordEntry(ctx, req, svc, data, model.OCreate))
	out := &apiservice.Instance{
		Id:        ins.GetId(),
		Service:   &wrappers.StringValue{Value: svc.Name},
		Namespace: &wrappers.StringValue{Value: svc.Namespace},
		VpcId:     instanceProto.GetVpcId(),
		Host:      instanceProto.GetHost(),
		Port:      instanceProto.GetPort(),
	}
	return api.NewInstanceResponse(apimodel.Code_ExecuteSuccess, out)
}

// createInstance store operate
func (s *Server) createInstance(ctx context.Context, req *apiservice.Instance, ins *apiservice.Instance) (
	*model.Instance, *apiservice.Response) {
	// create service if absent
	svcId, errResp := s.createWrapServiceIfAbsent(ctx, req)
	if errResp != nil {
		log.Errorf("[Instance] create service if absent fail : %+v, req : %+v", errResp.String(), req)
		return nil, errResp
	}
	if len(svcId) == 0 {
		log.Errorf("[Instance] create service if absent return service id is empty : %+v", req)
		return nil, api.NewResponseWithMsg(apimodel.Code_BadRequest, "service id is empty")
	}

	// fill instance location info
	s.packCmdb(ins)

	if namingServer.bc == nil || !namingServer.bc.CreateInstanceOpen() {
		return s.serialCreateInstance(ctx, svcId, req, ins) // 单个同步
	}
	return s.asyncCreateInstance(ctx, svcId, req, ins) // 批量异步
}

// asyncCreateInstance 异步新建实例
// 底层函数会合并create请求，增加并发创建的吞吐
// req 原始请求
// ins 包含了req数据与instanceID，serviceToken
func (s *Server) asyncCreateInstance(
	ctx context.Context, svcId string, req *apiservice.Instance, ins *apiservice.Instance) (
	*model.Instance, *apiservice.Response) {
	allowAsyncRegis, _ := ctx.Value(utils.ContextOpenAsyncRegis).(bool)
	future := s.bc.AsyncCreateInstance(svcId, ins, !allowAsyncRegis)

	if err := future.Wait(); err != nil {
		if future.Code() == apimodel.Code_ExistedResource {
			req.Id = utils.NewStringValue(ins.GetId().GetValue())
		}
		return nil, api.NewInstanceResponse(future.Code(), req)
	}

	return model.CreateInstanceModel(svcId, req), nil
}

// 同步串行创建实例
// req为原始的请求体
// ins包括了req的内容，并且填充了instanceID与serviceToken
func (s *Server) serialCreateInstance(
	ctx context.Context, svcId string, req *apiservice.Instance, ins *apiservice.Instance) (
	*model.Instance, *apiservice.Response) {

	instance, err := s.storage.GetInstance(ins.GetId().GetValue())
	if err != nil {
		log.Error("[Instance] get instance from store",
			utils.RequestID(ctx), zap.Error(err))
		return nil, api.NewInstanceResponse(commonstore.StoreCode2APICode(err), req)
	}
	// 如果存在，则替换实例的属性数据，但是需要保留用户设置的隔离状态，以免出现关键状态丢失
	if instance != nil && ins.Isolate == nil {
		ins.Isolate = instance.Proto.Isolate
	}
	// 直接同步创建服务实例
	data := model.CreateInstanceModel(svcId, ins)
	if err := s.storage.AddInstance(data); err != nil {
		log.Error(err.Error(), utils.RequestID(ctx))
		return nil, wrapperInstanceStoreResponse(req, err)
	}

	return data, nil
}

// DeleteInstances 批量删除服务实例
func (s *Server) DeleteInstances(ctx context.Context, req []*apiservice.Instance) *apiservice.BatchWriteResponse {
	return batchOperateInstances(ctx, req, s.DeleteInstance)
}

// DeleteInstance 删除单个服务实例
func (s *Server) DeleteInstance(ctx context.Context, req *apiservice.Instance) *apiservice.Response {
	ins := *req // 防止污染外部的req
	ins.ServiceToken = utils.NewStringValue(parseInstanceReqToken(ctx, req))
	return s.deleteInstance(ctx, req, &ins)
}

// 删除实例的store操作
// req 原始请求
// ins 填充了instanceID与serviceToken
func (s *Server) deleteInstance(
	ctx context.Context, req *apiservice.Instance, ins *apiservice.Instance) *apiservice.Response {
	if s.bc == nil || !s.bc.DeleteInstanceOpen() {
		return s.serialDeleteInstance(ctx, req, ins)
	}

	return s.asyncDeleteInstance(ctx, req, ins)
}

// 串行删除实例
// 返回实例所属的服务和resp
func (s *Server) serialDeleteInstance(
	ctx context.Context, req *apiservice.Instance, ins *apiservice.Instance) *apiservice.Response {
	start := time.Now()
	// 检查服务实例是否存在
	instance, err := s.storage.GetInstance(ins.GetId().GetValue())
	if err != nil {
		log.Error(err.Error(), utils.RequestID(ctx))
		return api.NewInstanceResponse(commonstore.StoreCode2APICode(err), req)
	}
	if instance == nil {
		// 实例不存在，则返回成功
		return api.NewInstanceResponse(apimodel.Code_ExecuteSuccess, req)
	}
	// 鉴权
	service, resp := s.instanceAuth(ctx, req, instance.ServiceID)
	if resp != nil {
		return resp
	}

	// 存储层操作
	if err := s.storage.DeleteInstance(instance.ID()); err != nil {
		log.Error(err.Error(), utils.RequestID(ctx))
		return wrapperInstanceStoreResponse(req, err)
	}

	msg := fmt.Sprintf("delete instance: id=%v, namespace=%v, service=%v, host=%v, port=%v",
		instance.ID(), service.Namespace, service.Name, instance.Host(), instance.Port())
	log.Info(msg, utils.RequestID(ctx), zap.Duration("cost", time.Since(start)))
	s.RecordHistory(ctx, instanceRecordEntry(ctx, req, service, instance, model.ODelete))
	event := &model.InstanceEvent{
		Id:         instance.ID(),
		Namespace:  service.Namespace,
		Service:    service.Name,
		Instance:   instance.Proto,
		EType:      model.EventInstanceOffline,
		CreateTime: time.Time{},
	}
	event.InjectMetadata(ctx)
	s.sendDiscoverEvent(*event)

	return api.NewInstanceResponse(apimodel.Code_ExecuteSuccess, req)
}

// 异步删除实例
// 返回实例所属的服务和resp
func (s *Server) asyncDeleteInstance(
	ctx context.Context, req *apiservice.Instance, ins *apiservice.Instance) *apiservice.Response {
	start := time.Now()
	allowAsyncRegis, _ := ctx.Value(utils.ContextOpenAsyncRegis).(bool)
	future := s.bc.AsyncDeleteInstance(ins, !allowAsyncRegis)
	if err := future.Wait(); err != nil {
		// 如果发现不存在资源，意味着实例已经被删除，直接返回成功
		if future.Code() == apimodel.Code_NotFoundResource {
			return api.NewInstanceResponse(apimodel.Code_ExecuteSuccess, req)
		}
		log.Error(err.Error(), utils.RequestID(ctx))
		return api.NewInstanceResponse(future.Code(), req)
	}
	instance := future.Instance()

	// 打印本地日志与操作记录
	msg := fmt.Sprintf("delete instance: id=%v, namespace=%v, service=%v, host=%v, port=%v",
		instance.ID(), instance.Namespace(), instance.Service(), instance.Host(), instance.Port())
	log.Info(msg, utils.RequestID(ctx), zap.Duration("cost", time.Since(start)))
	service := &model.Service{Name: instance.Service(), Namespace: instance.Namespace()}
	s.RecordHistory(ctx, instanceRecordEntry(ctx, req, service, instance, model.ODelete))
	event := &model.InstanceEvent{
		Id:         instance.ID(),
		Namespace:  service.Namespace,
		Service:    service.Name,
		Instance:   instance.Proto,
		EType:      model.EventInstanceOffline,
		CreateTime: time.Time{},
	}
	event.InjectMetadata(ctx)
	s.sendDiscoverEvent(*event)

	return api.NewInstanceResponse(apimodel.Code_ExecuteSuccess, req)
}

// DeleteInstancesByHost 根据host批量删除服务实例
func (s *Server) DeleteInstancesByHost(
	ctx context.Context, req []*apiservice.Instance) *apiservice.BatchWriteResponse {
	return batchOperateInstances(ctx, req, s.DeleteInstanceByHost)
}

// DeleteInstanceByHost 根据host删除服务实例
func (s *Server) DeleteInstanceByHost(ctx context.Context, req *apiservice.Instance) *apiservice.Response {
	// 获取实例
	instances, service, err := s.getInstancesMainByService(ctx, req)
	if err != nil {
		return err
	}

	if instances == nil {
		return api.NewInstanceResponse(apimodel.Code_ExecuteSuccess, req)
	}

	ids := make([]interface{}, 0, len(instances))
	for _, instance := range instances {
		ids = append(ids, instance.ID())
	}

	if err := s.storage.BatchDeleteInstances(ids); err != nil {
		log.Error(err.Error(), utils.RequestID(ctx))
		return wrapperInstanceStoreResponse(req, err)
	}

	for _, instance := range instances {
		msg := fmt.Sprintf("delete instance: id=%v, namespace=%v, service=%v, host=%v, port=%v",
			instance.ID(), service.Namespace, service.Name, instance.Host(), instance.Port())
		log.Info(msg, utils.RequestID(ctx))
		s.RecordHistory(ctx, instanceRecordEntry(ctx, req, service, instance, model.ODelete))
		s.sendDiscoverEvent(model.InstanceEvent{
			Id:         instance.ID(),
			Namespace:  service.Namespace,
			Service:    service.Name,
			Instance:   instance.Proto,
			EType:      model.EventInstanceOffline,
			CreateTime: time.Time{},
		})
	}
	return api.NewInstanceResponse(apimodel.Code_ExecuteSuccess, req)
}

// UpdateInstances 批量修改服务实例
func (s *Server) UpdateInstances(ctx context.Context, req []*apiservice.Instance) *apiservice.BatchWriteResponse {
	return batchOperateInstances(ctx, req, s.UpdateInstance)
}

// UpdateInstance 修改单个服务实例
func (s *Server) UpdateInstance(ctx context.Context, req *apiservice.Instance) *apiservice.Response {
	service, instance, preErr := s.execInstancePreStep(ctx, req)
	if preErr != nil {
		return preErr
	}
	// 修改
	log.Info(fmt.Sprintf("old instance: %+v", instance), utils.RequestID(ctx))

	var eventTypes map[model.InstanceEventType]bool
	var needUpdate bool
	// 存储层操作
	if needUpdate, eventTypes = s.updateInstanceAttribute(req, instance); !needUpdate {
		log.Info("update instance no data change, no need update",
			utils.RequestID(ctx), zap.String("instance", req.String()))
		return api.NewInstanceResponse(apimodel.Code_NoNeedUpdate, req)
	}
	if err := s.storage.UpdateInstance(instance); err != nil {
		log.Error(err.Error(), utils.RequestID(ctx))
		return wrapperInstanceStoreResponse(req, err)
	}

	msg := fmt.Sprintf("update instance: id=%v, namespace=%v, service=%v, host=%v, port=%v, healthy = %v",
		instance.ID(), service.Namespace, service.Name, instance.Host(),
		instance.Port(), instance.Healthy())
	log.Info(msg, utils.RequestID(ctx))
	s.RecordHistory(ctx, instanceRecordEntry(ctx, req, service, instance, model.OUpdate))

	for eventType := range eventTypes {
		event := &model.InstanceEvent{
			Id:         instance.ID(),
			Namespace:  service.Namespace,
			Service:    service.Name,
			Instance:   instance.Proto,
			EType:      eventType,
			CreateTime: time.Time{},
		}
		event.InjectMetadata(ctx)
		s.sendDiscoverEvent(*event)
	}

	for i := range s.instanceChains {
		s.instanceChains[i].AfterUpdate(ctx, instance)
	}

	return api.NewInstanceResponse(apimodel.Code_ExecuteSuccess, req)
}

// UpdateInstancesIsolate 批量修改服务实例隔离状态
// @note 必填参数为service+namespace+host
func (s *Server) UpdateInstancesIsolate(
	ctx context.Context, req []*apiservice.Instance) *apiservice.BatchWriteResponse {
	return batchOperateInstances(ctx, req, s.UpdateInstanceIsolate)
}

// UpdateInstanceIsolate 修改服务实例隔离状态
// @note 必填参数为service+namespace+ip
func (s *Server) UpdateInstanceIsolate(ctx context.Context, req *apiservice.Instance) *apiservice.Response {
	if req.GetIsolate() == nil {
		return api.NewInstanceResponse(apimodel.Code_InvalidInstanceIsolate, req)
	}

	// 获取实例
	instances, service, err := s.getInstancesMainByService(ctx, req)
	if err != nil {
		return err
	}
	if instances == nil {
		return api.NewInstanceResponse(apimodel.Code_NotFoundInstance, req)
	}

	// 判断是否需要更新
	needUpdate := false
	for _, instance := range instances {
		if req.Isolate != nil && instance.Isolate() != req.GetIsolate().GetValue() {
			needUpdate = true
			break
		}
	}
	if !needUpdate {
		return api.NewInstanceResponse(apimodel.Code_NoNeedUpdate, req)
	}

	isolate := 0
	if req.GetIsolate().GetValue() {
		isolate = 1
	}

	ids := make([]interface{}, 0, len(instances))
	for _, instance := range instances {
		// 方便后续打印操作记录
		instance.Proto.Isolate = req.GetIsolate()
		ids = append(ids, instance.ID())
	}

	if err := s.storage.BatchSetInstanceIsolate(ids, isolate, utils.NewUUID()); err != nil {
		log.Error(err.Error(), utils.RequestID(ctx))
		return wrapperInstanceStoreResponse(req, err)
	}

	for _, instance := range instances {
		msg := fmt.Sprintf("update instance: id=%v, namespace=%v, service=%v, host=%v, port=%v, isolate=%v",
			instance.ID(), service.Namespace, service.Name, instance.Host(), instance.Port(), instance.Isolate())
		log.Info(msg, utils.RequestID(ctx))
		s.RecordHistory(ctx, instanceRecordEntry(ctx, req, service, instance, model.OUpdateIsolate))

		// 比对下更新前后的 isolate 状态
		if req.Isolate != nil && instance.Isolate() != req.Isolate.GetValue() {
			eventType := model.EventInstanceCloseIsolate
			if req.Isolate.GetValue() {
				eventType = model.EventInstanceOpenIsolate
			}
			s.sendDiscoverEvent(model.InstanceEvent{
				Id:         instance.ID(),
				Namespace:  req.Namespace.GetValue(),
				Service:    req.Service.GetValue(),
				Instance:   instance.Proto,
				EType:      eventType,
				CreateTime: time.Time{},
			})
		}
		instance.Proto.Isolate = utils.NewBoolValue(req.GetIsolate().GetValue())
	}
	for i := range s.instanceChains {
		s.instanceChains[i].AfterUpdate(ctx, instances...)
	}

	return api.NewInstanceResponse(apimodel.Code_ExecuteSuccess, req)
}

/**
 * @brief 根据服务和host获取服务实例
 */
func (s *Server) getInstancesMainByService(ctx context.Context, req *apiservice.Instance) (
	[]*model.Instance, *model.Service, *apiservice.Response) {
	// 检查服务
	// 这里获取的是源服务的token。如果是别名,service=nil
	service, err := s.storage.GetSourceServiceToken(req.GetService().GetValue(), req.GetNamespace().GetValue())
	if err != nil {
		log.Error(err.Error(), utils.RequestID(ctx))
		return nil, nil, api.NewInstanceResponse(commonstore.StoreCode2APICode(err), req)
	}
	if service == nil {
		return nil, nil, api.NewInstanceResponse(apimodel.Code_NotFoundService, req)
	}

	// 获取服务实例
	instances, err := s.storage.GetInstancesMainByService(service.ID, req.GetHost().GetValue())
	if err != nil {
		log.Error(err.Error(), utils.RequestID(ctx))
		return nil, nil, api.NewInstanceResponse(commonstore.StoreCode2APICode(err), req)
	}
	return instances, service, nil
}

/**
 * @brief 修改服务属性
 */
func (s *Server) updateInstanceAttribute(
	req *apiservice.Instance, instance *model.Instance) (bool, map[model.InstanceEventType]bool) {
	// #lizard forgives
	instance.MallocProto()
	needUpdate := false
	insProto := instance.Proto
	var updateEvents = make(map[model.InstanceEventType]bool)
	if ok := utils.IsNotEqualMap(req.GetMetadata(), instance.Metadata()); ok {
		insProto.Metadata = req.GetMetadata()
		needUpdate = true
		updateEvents[model.EventInstanceUpdate] = true
	}

	if ok := instanceLocationNeedUpdate(req.GetLocation(), instance.Proto.GetLocation()); ok {
		insProto.Location = req.Location
		needUpdate = true
		updateEvents[model.EventInstanceUpdate] = true
	}

	if req.GetProtocol() != nil && req.GetProtocol().GetValue() != instance.Protocol() {
		insProto.Protocol = req.GetProtocol()
		needUpdate = true
		updateEvents[model.EventInstanceUpdate] = true
	}

	if req.GetVersion() != nil && req.GetVersion().GetValue() != instance.Version() {
		insProto.Version = req.GetVersion()
		needUpdate = true
		updateEvents[model.EventInstanceUpdate] = true
	}

	if req.GetPriority() != nil && req.GetPriority().GetValue() != instance.Priority() {
		insProto.Priority = req.GetPriority()
		needUpdate = true
		updateEvents[model.EventInstanceUpdate] = true
	}

	if req.GetWeight() != nil && req.GetWeight().GetValue() != instance.Weight() {
		insProto.Weight = req.GetWeight()
		needUpdate = true
		updateEvents[model.EventInstanceUpdate] = true
	}

	if req.GetHealthy() != nil && req.GetHealthy().GetValue() != instance.Healthy() {
		insProto.Healthy = req.GetHealthy()
		needUpdate = true
		if req.Healthy.GetValue() {
			updateEvents[model.EventInstanceTurnHealth] = true
		} else {
			updateEvents[model.EventInstanceTurnUnHealth] = true
		}
	}

	if req.GetIsolate() != nil && req.GetIsolate().GetValue() != instance.Isolate() {
		insProto.Isolate = req.GetIsolate()
		needUpdate = true
		if req.Isolate.GetValue() {
			updateEvents[model.EventInstanceOpenIsolate] = true
		} else {
			updateEvents[model.EventInstanceCloseIsolate] = true
		}
	}

	if req.GetLogicSet() != nil && req.GetLogicSet().GetValue() != instance.LogicSet() {
		insProto.LogicSet = req.GetLogicSet()
		needUpdate = true
		updateEvents[model.EventInstanceUpdate] = true
	}

	if ok := updateHealthCheck(req, instance); ok {
		needUpdate = true
		updateEvents[model.EventInstanceUpdate] = true
	}

	// 每次更改，都要生成一个新的uuid
	if needUpdate {
		insProto.Revision = utils.NewStringValue(utils.NewUUID())
	}

	return needUpdate, updateEvents
}

func instanceLocationNeedUpdate(req *apimodel.Location, old *apimodel.Location) bool {
	if req.GetRegion().GetValue() != old.GetRegion().GetValue() {
		return true
	}
	if req.GetZone().GetValue() != old.GetZone().GetValue() {
		return true
	}
	if req.GetCampus().GetValue() != old.GetCampus().GetValue() {
		return true
	}

	return false
}

// 健康检查的更新
func updateHealthCheck(req *apiservice.Instance, instance *model.Instance) bool {
	needUpdate := false
	insProto := instance.Proto
	// health Check，healthCheck不能为空，且没有把enable_health_check置为false
	if req.GetHealthCheck().GetHeartbeat() != nil &&
		(req.GetEnableHealthCheck() == nil || req.GetEnableHealthCheck().GetValue()) {
		// 如果数据库中实例原有是不打开健康检查，
		// 那么一旦打开，status需置为false，等待一次心跳成功才能变成true
		if !instance.EnableHealthCheck() {
			// 需要重置healthy，则认为有变更
			insProto.Healthy = utils.NewBoolValue(false)
			insProto.EnableHealthCheck = utils.NewBoolValue(true)
			needUpdate = true
		}

		ttl := req.GetHealthCheck().GetHeartbeat().GetTtl().GetValue()
		if ttl == 0 || ttl > 60 {
			ttl = DefaultTLL
		}
		if ttl != instance.HealthCheck().GetHeartbeat().GetTtl().GetValue() {
			// ttl有变更
			needUpdate = true
		}
		if apiservice.HealthCheck_HEARTBEAT != instance.HealthCheck().GetType() {
			// health check type有变更
			needUpdate = true
		}
		insProto.HealthCheck = req.GetHealthCheck()
		insProto.HealthCheck.Type = apiservice.HealthCheck_HEARTBEAT
		if insProto.HealthCheck.Heartbeat.Ttl == nil {
			insProto.HealthCheck.Heartbeat.Ttl = utils.NewUInt32Value(0)
		}
		insProto.HealthCheck.Heartbeat.Ttl.Value = ttl
	}

	// update的时候，修改了enableHealthCheck的值
	if req.GetEnableHealthCheck() != nil && !req.GetEnableHealthCheck().GetValue() {
		if req.GetEnableHealthCheck().GetValue() != instance.EnableHealthCheck() {
			needUpdate = true
		}
		if insProto.GetHealthCheck() != nil {
			needUpdate = true
		}

		insProto.EnableHealthCheck = utils.NewBoolValue(false)
		insProto.HealthCheck = nil
	}

	return needUpdate
}

// GetInstances 查询服务实例
func (s *Server) GetInstances(ctx context.Context, query map[string]string) *apiservice.BatchQueryResponse {
	showLastHeartbeat := query["show_last_heartbeat"] == "true"
	delete(query, "show_last_heartbeat")
	showServiceRevision := query["show_service_revision"] == "true"
	delete(query, "show_service_revision")
	// 对数据先进行提前处理一下
	filters, metaFilter, batchErr := preGetInstances(query)
	if batchErr != nil {
		return batchErr
	}
	// 分页数据
	offset, limit, _ := utils.ParseOffsetAndLimit(filters)

	total, instances, err := s.Cache().Instance().QueryInstances(filters, metaFilter, offset, limit)
	if err != nil {
		log.Errorf("[Server][Instances][Query] instances store err: %s", err.Error())
		return api.NewBatchQueryResponse(commonstore.StoreCode2APICode(err))
	}

	out := api.NewBatchQueryResponse(apimodel.Code_ExecuteSuccess)
	out.Amount = utils.NewUInt32Value(total)
	out.Size = utils.NewUInt32Value(uint32(len(instances)))

	svcInfos := make(map[string]*model.Service, 4)
	apiInstances := make([]*apiservice.Instance, 0, len(instances))
	for _, instance := range instances {
		svc, _ := s.loadServiceByID(instance.ServiceID)
		if svc == nil {
			continue
		}
		protoIns := copyOSSInstance(instance.Proto)
		protoIns.Service = wrapperspb.String(svc.Name)
		protoIns.Namespace = wrapperspb.String(svc.Namespace)
		protoIns.ServiceToken = wrapperspb.String(svc.Token)
		s.packCmdb(protoIns)
		apiInstances = append(apiInstances, protoIns)
	}
	if showLastHeartbeat {
		s.fillLastHeartbeatTime(apiInstances)
	}
	if showServiceRevision {
		// 额外显示每个服务的 revision 版本列表信息数据
		out.Services = make([]*apiservice.Service, 0, len(svcInfos))
		for i := range svcInfos {
			svc := svcInfos[i].ToSpec()
			revision := s.caches.Service().GetRevisionWorker().GetServiceInstanceRevision(svc.GetId().GetValue())
			svc.Revision = wrapperspb.String(revision)
			out.Services = append(out.Services, svc)
		}
	}
	out.Instances = apiInstances
	return out
}

func (s *Server) fillLastHeartbeatTime(instances []*apiservice.Instance) {
	checker, ok := s.healthServer.Checkers()[int32(apiservice.HealthCheck_HEARTBEAT)]
	if !ok {
		return
	}
	req := &plugin.BatchQueryRequest{Requests: make([]*plugin.QueryRequest, 0, len(instances))}
	for i := range instances {
		item := instances[i]
		req.Requests = append(req.Requests, &plugin.QueryRequest{
			InstanceId: item.GetId().GetValue(),
		})
	}
	rsp, err := checker.BatchQuery(context.Background(), req)
	if err != nil {
		return
	}
	for i := range rsp.Responses {
		item := instances[i]
		copyMetadata := make(map[string]string, len(item.GetMetadata()))
		for k, v := range item.GetMetadata() {
			copyMetadata[k] = v
		}
		if queryRsp := rsp.Responses[i]; queryRsp.Exists {
			copyMetadata["last-heartbeat-timestamp"] = strconv.Itoa(int(queryRsp.LastHeartbeatSec))
			copyMetadata["last-heartbeat-time"] = commontime.Time2String(time.Unix(queryRsp.LastHeartbeatSec, 0))
		}
		item.Metadata = copyMetadata
	}
}

var (
	ignoreReturnOSSInstanceMetadata = map[string]struct{}{
		"version":  {},
		"protocol": {},
		"region":   {},
		"zone":     {},
		"campus":   {},
	}
)

func copyOSSInstance(instance *apiservice.Instance) *apiservice.Instance {
	copyIns := &apiservice.Instance{
		Id:                instance.Id,
		Service:           instance.Service,
		Namespace:         instance.Namespace,
		VpcId:             instance.VpcId,
		Host:              instance.Host,
		Port:              instance.Port,
		Protocol:          instance.Protocol,
		Version:           instance.Version,
		Priority:          instance.Priority,
		Weight:            instance.Weight,
		EnableHealthCheck: instance.EnableHealthCheck,
		HealthCheck:       instance.HealthCheck,
		Healthy:           instance.Healthy,
		Isolate:           instance.Isolate,
		Location:          instance.Location,
		LogicSet:          instance.LogicSet,
		Ctime:             instance.Ctime,
		Mtime:             instance.Mtime,
		Revision:          instance.Revision,
		ServiceToken:      instance.ServiceToken,
	}

	copym := map[string]string{}
	for k, v := range instance.Metadata {
		if _, ok := ignoreReturnOSSInstanceMetadata[k]; ok {
			continue
		}
		copym[k] = v
	}

	copyIns.Metadata = copym
	return copyIns
}

// GetInstanceLabels 获取实例标签列表
func (s *Server) GetInstanceLabels(ctx context.Context, query map[string]string) *apiservice.Response {
	var (
		serviceId string
		namespace = DefaultNamespace
	)

	if val, ok := query["namespace"]; ok {
		namespace = val
	}

	if service, ok := query["service"]; ok {
		svc := s.Cache().Service().GetServiceByName(service, namespace)
		if svc != nil {
			serviceId = svc.ID
		}
	}

	if id, ok := query["service_id"]; ok {
		serviceId = id
	}

	if serviceId == "" {
		resp := api.NewResponse(apimodel.Code_ExecuteSuccess)
		resp.InstanceLabels = &apiservice.InstanceLabels{}
		return resp
	}

	ret := s.Cache().Instance().GetInstanceLabels(serviceId)
	resp := api.NewResponse(apimodel.Code_ExecuteSuccess)
	resp.InstanceLabels = ret
	return resp
}

// GetInstancesCount 查询总的服务实例，不带过滤条件的
func (s *Server) GetInstancesCount(ctx context.Context) *apiservice.BatchQueryResponse {
	count, err := s.storage.GetInstancesCount()
	if err != nil {
		log.Errorf("[Server][Instance][Count] storage get err: %s", err.Error())
		return api.NewBatchQueryResponse(commonstore.StoreCode2APICode(err))
	}

	out := api.NewBatchQueryResponse(apimodel.Code_ExecuteSuccess)
	out.Amount = utils.NewUInt32Value(count)
	out.Instances = make([]*apiservice.Instance, 0)
	return out
}

// update/delete instance前置条件
func (s *Server) execInstancePreStep(ctx context.Context, req *apiservice.Instance) (
	*model.Service, *model.Instance, *apiservice.Response) {
	// 检查服务实例是否存在
	instance, err := s.storage.GetInstance(req.GetId().GetValue())
	if err != nil {
		log.Error("[Instance] get instance from store", utils.RequestID(ctx), utils.ZapInstanceID(req.GetId().GetValue()),
			zap.Error(err))
		return nil, nil, api.NewInstanceResponse(commonstore.StoreCode2APICode(err), req)
	}
	if instance == nil {
		return nil, nil, api.NewInstanceResponse(apimodel.Code_NotFoundInstance, req)
	}

	service, resp := s.instanceAuth(ctx, req, instance.ServiceID)
	if resp != nil {
		return nil, nil, resp
	}

	return service, instance, nil
}

// 实例鉴权
func (s *Server) instanceAuth(ctx context.Context, req *apiservice.Instance, serviceID string) (
	*model.Service, *apiservice.Response) {
	service, err := s.storage.GetServiceByID(serviceID)
	if err != nil {
		log.Error(err.Error(), utils.ZapRequestID(utils.ParseRequestID(ctx)))
		return nil, api.NewInstanceResponse(commonstore.StoreCode2APICode(err), req)
	}
	if service == nil {
		return nil, api.NewInstanceResponse(apimodel.Code_NotFoundResource, req)
	}

	return service, nil
}

// 获取api.instance
func (s *Server) getInstance(service *apiservice.Service, instance *apiservice.Instance) *apiservice.Instance {
	out := &apiservice.Instance{
		Id:                instance.GetId(),
		Service:           service.GetName(),
		Namespace:         service.GetNamespace(),
		VpcId:             instance.GetVpcId(),
		Host:              instance.GetHost(),
		Port:              instance.GetPort(),
		Protocol:          instance.GetProtocol(),
		Version:           instance.GetVersion(),
		Priority:          instance.GetPriority(),
		Weight:            instance.GetWeight(),
		EnableHealthCheck: instance.GetEnableHealthCheck(),
		HealthCheck:       instance.GetHealthCheck(),
		Healthy:           instance.GetHealthy(),
		Isolate:           instance.GetIsolate(),
		Location:          instance.GetLocation(),
		Metadata:          instance.GetMetadata(),
		LogicSet:          instance.GetLogicSet(),
		Ctime:             instance.GetCtime(),
		Mtime:             instance.GetMtime(),
		Revision:          instance.GetRevision(),
	}

	s.packCmdb(out)
	return out
}

// 获取cmdb
func (s *Server) packCmdb(instance *apiservice.Instance) {
	if s.cmdb == nil {
		return
	}
	if instance == nil || !isEmptyLocation(instance.GetLocation()) {
		return
	}

	location, err := s.cmdb.GetLocation(instance.GetHost().GetValue())
	if err != nil {
		log.Error("[Instance] pack cmdb info fail",
			zap.String("namespace", instance.GetNamespace().GetValue()),
			zap.String("service", instance.GetService().GetValue()),
			zap.String("host", instance.GetHost().GetValue()),
			zap.Uint32("port", instance.GetPort().GetValue()))
		return
	}
	if location != nil {
		instance.Location = location.Proto
	}
}

func isEmptyLocation(loc *apimodel.Location) bool {
	return loc == nil || (loc.GetRegion().GetValue() == "" &&
		loc.GetZone().GetValue() == "" &&
		loc.GetCampus().GetValue() == "")
}

func (s *Server) sendDiscoverEvent(event model.InstanceEvent) {
	if event.Instance != nil {
		// In order not to cause `panic` in cause multi-corporate data op, do deep copy
		// event.Instance = proto.Clone(event.Instance).(*apiservice.Instance)
	}
	_ = eventhub.Publish(eventhub.InstanceEventTopic, event)
}

type wrapSvcName interface {
	// GetService 获取服务名
	GetService() *wrappers.StringValue
	// GetNamespace 获取命名空间
	GetNamespace() *wrappers.StringValue
}

type rawSvcName interface {
	// GetService 获取服务名
	GetService() string
	// GetNamespace 获取命名空间
	GetNamespace() string
}

// createWrapServiceIfAbsent 如果服务不存在，则进行创建，并返回服务的ID信息
func (s *Server) createWrapServiceIfAbsent(ctx context.Context, instance wrapSvcName) (string, *apiservice.Response) {
	return s.createServiceIfAbsent(ctx, instance.GetNamespace().GetValue(), instance.GetService().GetValue())
}

func (s *Server) createServiceIfAbsent(
	ctx context.Context, namespace string, svcName string) (string, *apiservice.Response) {
	svc, errResp := s.loadService(namespace, svcName)
	if errResp != nil {
		return "", errResp
	}
	if svc != nil {
		return svc.ID, nil
	}
	// if auto_create_service config is false, return service not found
	if !s.allowAutoCreate() {
		return "", api.NewResponse(apimodel.Code_NotFoundService)
	}
	simpleService := &apiservice.Service{
		Name:      utils.NewStringValue(svcName),
		Namespace: utils.NewStringValue(namespace),
		Owners: func() *wrapperspb.StringValue {
			owner := utils.ParseOwnerID(ctx)
			if owner == "" {
				return utils.NewStringValue("Polaris")
			}
			return utils.NewStringValue(owner)
		}(),
	}
	key := fmt.Sprintf("%s:%s", simpleService.Namespace, simpleService.Name)
	ret, err, _ := s.createServiceSingle.Do(key, func() (interface{}, error) {
		resp := s.CreateService(ctx, simpleService)
		return resp, nil
	})
	if err != nil {
		return "", api.NewResponseWithMsg(apimodel.Code_ExecuteException, err.Error())
	}
	resp := ret.(*apiservice.Response)
	retCode := apimodel.Code(resp.GetCode().GetValue())
	if retCode != apimodel.Code_ExecuteSuccess && retCode != apimodel.Code_ExistedResource {
		return "", resp
	}
	svcId := resp.GetService().GetId().GetValue()
	return svcId, nil
}

func (s *Server) loadService(namespace string, svcName string) (*model.Service, *apiservice.Response) {
	svc := s.caches.Service().GetServiceByName(svcName, namespace)
	if svc != nil {
		if svc.IsAlias() {
			return nil, api.NewResponseWithMsg(apimodel.Code_BadRequest, "service is alias")
		}
		return svc, nil
	}
	// 再走数据库查询一遍
	svc, err := s.storage.GetService(svcName, namespace)
	if err != nil {
		return nil, api.NewResponseWithMsg(commonstore.StoreCode2APICode(err), err.Error())
	}
	if svc != nil && svc.IsAlias() {
		return nil, api.NewResponseWithMsg(apimodel.Code_BadRequest, "service is alias")
	}
	return svc, nil
}

func (s *Server) loadServiceByID(svcID string) (*model.Service, error) {
	svc := s.caches.Service().GetServiceByID(svcID)
	if svc != nil {
		if svc.IsAlias() {
			return nil, errors.New("service is alias")
		}
		return svc, nil
	}

	// 再走数据库查询一遍
	svc, err := s.storage.GetServiceByID(svcID)
	if err != nil {
		return nil, err
	}

	if svc != nil && svc.IsAlias() {
		return nil, errors.New("service is alias")
	}

	return svc, nil
}

// 获取instance请求的token信息
func parseInstanceReqToken(ctx context.Context, req *apiservice.Instance) string {
	if reqToken := req.GetServiceToken().GetValue(); reqToken != "" {
		return reqToken
	}

	return utils.ParseToken(ctx)
}

// 实例查询前置处理
func preGetInstances(query map[string]string) (map[string]string, map[string]string, *apiservice.BatchQueryResponse) {
	var metaFilter map[string]string
	metaKey, metaKeyAvail := query["keys"]
	if metaKeyAvail {
		metaFilter = map[string]string{}
		keys := strings.Split(metaKey, ",")
		values := strings.Split(query["values"], ",")
		for i := range keys {
			metaFilter[keys[i]] = values[i]
		}
	}

	// 以healthy为准
	_, lhs := query["health_status"]
	_, rhs := query["healthy"]
	if lhs && rhs {
		delete(query, "health_status")
	}

	filters := make(map[string]string)
	for key, value := range query {
		if attr, ok := InsFilter2toreAttr[key]; ok {
			key = attr
		}
		if !NotInsFilterAttr[key] {
			filters[key] = value
		}
	}

	return filters, metaFilter, nil
}

// 批量操作实例
func batchOperateInstances(ctx context.Context, reqs []*apiservice.Instance,
	handler func(ctx context.Context, req *apiservice.Instance) *apiservice.Response) *apiservice.BatchWriteResponse {
	responses := api.NewBatchWriteResponse(apimodel.Code_ExecuteSuccess)

	chs := make([]chan *apiservice.Response, 0, len(reqs))
	for i, instance := range reqs {
		chs = append(chs, make(chan *apiservice.Response))
		go func(index int, ins *apiservice.Instance) {
			chs[index] <- handler(ctx, ins)
		}(i, instance)
	}

	for _, ch := range chs {
		resp := <-ch
		api.Collect(responses, resp)
	}

	return api.FormatBatchWriteResponse(responses)
}

// wrapper instance store response
func wrapperInstanceStoreResponse(instance *apiservice.Instance, err error) *apiservice.Response {
	if err == nil {
		return nil
	}
	resp := api.NewResponseWithMsg(commonstore.StoreCode2APICode(err), err.Error())
	resp.Instance = instance
	return resp
}

// 生成instance的记录entry
func instanceRecordEntry(ctx context.Context, req *apiservice.Instance, service *model.Service, ins *model.Instance,
	opt model.OperationType) *model.RecordEntry {
	if service == nil || ins == nil {
		return nil
	}
	marshaler := jsonpb.Marshaler{}
	datail, _ := marshaler.MarshalToString(req)
	entry := &model.RecordEntry{
		ResourceType:  model.RInstance,
		ResourceName:  fmt.Sprintf("%s(%s:%d)", service.Name, ins.Host(), ins.Port()),
		Namespace:     service.Namespace,
		OperationType: opt,
		Operator:      utils.ParseOperator(ctx),
		Detail:        datail,
		HappenTime:    time.Now(),
	}
	return entry
}

type InstanceChain interface {
	// AfterUpdate .
	AfterUpdate(ctx context.Context, instances ...*model.Instance)
}
