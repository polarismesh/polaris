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
	"time"

	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/wrapperspb"

	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/common/utils"
)

var (
	// InstanceFilterAttributes 查询实例支持的过滤字段
	InstanceFilterAttributes = map[string]bool{
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
func (s *Server) CreateInstances(ctx context.Context, reqs []*api.Instance) *api.BatchWriteResponse {
	if checkError := checkBatchInstance(reqs); checkError != nil {
		return checkError
	}

	return batchOperateInstances(ctx, reqs, s.CreateInstance)
}

// CreateInstance 创建单个服务实例
// 注意：创建实例需要对服务进行加锁保护服务不被删除
func (s *Server) CreateInstance(ctx context.Context, req *api.Instance) *api.Response {
	rid := ParseRequestID(ctx)
	pid := ParsePlatformID(ctx)
	start := time.Now()
	// 参数检查
	instanceID, checkError := checkCreateInstance(req)
	if checkError != nil {
		return checkError
	}
	// 限制instance频繁注册
	if ok := s.allowInstanceAccess(instanceID); !ok {
		log.Error("create instance not allowed to access: exceed ratelimit",
			ZapRequestID(rid), ZapPlatformID(pid), ZapInstanceID(instanceID))
		return api.NewInstanceResponse(api.InstanceTooManyRequests, req)
	}

	// 防止污染req，拷贝一份出来，并且填充一下token ID
	ins := *req
	ins.ServiceToken = utils.NewStringValue(parseInstanceReqToken(ctx, req))
	ins.Id = utils.NewStringValue(instanceID)
	data, resp := s.createInstance(ctx, req, &ins)
	if resp != nil {
		return resp
	}

	// 处理create成功的消息
	msg := fmt.Sprintf("create instance: id=%v, namespace=%v, service=%v, host=%v, port=%v",
		ins.GetId().GetValue(), req.GetNamespace().GetValue(), req.GetService().GetValue(),
		req.GetHost().GetValue(), req.GetPort().GetValue())
	log.Info(msg, ZapRequestID(rid), ZapPlatformID(pid), zap.Duration("cost", time.Since(start)))
	service := &model.Service{
		Name:      req.GetService().GetValue(),
		Namespace: req.GetNamespace().GetValue(),
	}
	s.RecordHistory(instanceRecordEntry(ctx, service, data, model.OCreate))
	out := &api.Instance{
		Id:        ins.GetId(),
		Service:   req.GetService(),
		Namespace: req.GetNamespace(),
		VpcId:     req.GetVpcId(),
		Host:      req.GetHost(),
		Port:      req.GetPort(),
	}
	return api.NewInstanceResponse(api.ExecuteSuccess, out)
}

// createInstance store operate
func (s *Server) createInstance(ctx context.Context, req *api.Instance, ins *api.Instance) (
	*model.Instance, *api.Response) {

	// create service if absent
	code, svcId, err := s.createServiceIfAbsent(ctx, req)

	if err != nil {
		return nil, api.NewInstanceResponse(code, req)
	}

	if namingServer.bc == nil || !namingServer.bc.CreateInstanceOpen() {
		return s.serialCreateInstance(ctx, svcId, req, ins) // 单个同步
	}
	return s.asyncCreateInstance(ctx, svcId, req, ins) // 批量异步
}

// asyncCreateInstance 异步新建实例
// 底层函数会合并create请求，增加并发创建的吞吐
// req 原始请求
// ins 包含了req数据与instanceID，serviceToken
func (s *Server) asyncCreateInstance(ctx context.Context, svcId string, req *api.Instance, ins *api.Instance) (
	*model.Instance, *api.Response) {

	allowAsyncRegis, _ := ctx.Value(utils.ContextOpenAsyncRegis).(bool)
	future := s.bc.AsyncCreateInstance(svcId, ins, !allowAsyncRegis)

	if err := future.Wait(); err != nil {
		if future.Code() == api.ExistedResource {
			req.Id = utils.NewStringValue(ins.GetId().GetValue())
		}
		return nil, api.NewInstanceResponse(future.Code(), req)
	}

	return utils.CreateInstanceModel(svcId, req), nil
}

// 同步串行创建实例
// req为原始的请求体
// ins包括了req的内容，并且填充了instanceID与serviceToken
func (s *Server) serialCreateInstance(ctx context.Context, svcId string, req *api.Instance, ins *api.Instance) (
	*model.Instance, *api.Response) {
	rid := ParseRequestID(ctx)
	pid := ParsePlatformID(ctx)

	instance, err := s.storage.GetInstance(ins.GetId().GetValue())
	if err != nil {
		log.Error("[Instance] get instance from store", ZapRequestID(rid), ZapPlatformID(pid), zap.Error(err))
		return nil, api.NewInstanceResponse(api.StoreLayerException, req)
	}
	// 如果存在，则替换实例的属性数据，但是需要保留用户设置的隔离状态，以免出现关键状态丢失
	if instance != nil && ins.Isolate == nil {
		ins.Isolate = instance.Proto.Isolate
	}
	// 直接同步创建服务实例
	data := utils.CreateInstanceModel(svcId, ins)
	if err := s.storage.AddInstance(data); err != nil {
		log.Error(err.Error(), ZapRequestID(rid), ZapPlatformID(pid))
		return nil, wrapperInstanceStoreResponse(req, err)
	}

	return data, nil
}

// DeleteInstances 批量删除服务实例
func (s *Server) DeleteInstances(ctx context.Context, req []*api.Instance) *api.BatchWriteResponse {
	if checkError := checkBatchInstance(req); checkError != nil {
		return checkError
	}

	return batchOperateInstances(ctx, req, s.DeleteInstance)
}

// DeleteInstance 删除单个服务实例
func (s *Server) DeleteInstance(ctx context.Context, req *api.Instance) *api.Response {
	rid := ParseRequestID(ctx)
	pid := ParsePlatformID(ctx)

	// 参数检查
	instanceID, checkError := checkReviseInstance(req)
	if checkError != nil {
		return checkError
	}
	// 限制instance频繁反注册
	if ok := s.allowInstanceAccess(instanceID); !ok {
		log.Error("delete instance is not allow access", ZapRequestID(rid), ZapPlatformID(pid))
		return api.NewInstanceResponse(api.InstanceTooManyRequests, req)
	}

	ins := *req // 防止污染外部的req
	ins.Id = utils.NewStringValue(instanceID)
	ins.ServiceToken = utils.NewStringValue(parseInstanceReqToken(ctx, req))
	return s.deleteInstance(ctx, req, &ins)
}

// 删除实例的store操作
// req 原始请求
// ins 填充了instanceID与serviceToken
func (s *Server) deleteInstance(ctx context.Context, req *api.Instance, ins *api.Instance) *api.Response {
	if s.bc == nil || !s.bc.DeleteInstanceOpen() {
		return s.serialDeleteInstance(ctx, req, ins)
	}

	return s.asyncDeleteInstance(ctx, req, ins)
}

// 串行删除实例
// 返回实例所属的服务和resp
func (s *Server) serialDeleteInstance(ctx context.Context, req *api.Instance, ins *api.Instance) *api.Response {
	start := time.Now()
	// 检查服务实例是否存在
	rid := ParseRequestID(ctx)
	pid := ParsePlatformID(ctx)
	instance, err := s.storage.GetInstance(ins.GetId().GetValue())
	if err != nil {
		log.Error(err.Error(), ZapRequestID(rid))
		return api.NewInstanceResponse(api.StoreLayerException, req)
	}
	if instance == nil {
		// 实例不存在，则返回成功
		return api.NewInstanceResponse(api.ExecuteSuccess, req)
	}
	// 鉴权
	service, resp := s.instanceAuth(ctx, req, instance.ServiceID)
	if resp != nil {
		return resp
	}

	// 存储层操作
	if err := s.storage.DeleteInstance(instance.ID()); err != nil {
		log.Error(err.Error(), ZapRequestID(rid), ZapPlatformID(pid))
		return wrapperInstanceStoreResponse(req, err)
	}

	msg := fmt.Sprintf("delete instance: id=%v, namespace=%v, service=%v, host=%v, port=%v",
		instance.ID(), service.Namespace, service.Name, instance.Host(), instance.Port())
	log.Info(msg, ZapRequestID(rid), ZapPlatformID(pid), zap.Duration("cost", time.Since(start)))
	s.RecordHistory(instanceRecordEntry(ctx, service, instance, model.ODelete))

	s.sendDiscoverEvent(model.EventInstanceOffline, service.Namespace,
		service.Name,
		instance.Host(),
		int(instance.Port()))

	return api.NewInstanceResponse(api.ExecuteSuccess, req)
}

// 异步删除实例
// 返回实例所属的服务和resp
func (s *Server) asyncDeleteInstance(ctx context.Context, req *api.Instance, ins *api.Instance) *api.Response {
	start := time.Now()
	rid := ParseRequestID(ctx)
	pid := ParsePlatformID(ctx)

	future := s.bc.AsyncDeleteInstance(ins)
	if err := future.Wait(); err != nil {
		// 如果发现不存在资源，意味着实例已经被删除，直接返回成功
		if future.Code() == api.NotFoundResource {
			return api.NewInstanceResponse(api.ExecuteSuccess, req)
		}
		log.Error(err.Error(), ZapRequestID(rid), ZapPlatformID(pid))
		return api.NewInstanceResponse(future.Code(), req)
	}
	instance := future.Instance()

	// 打印本地日志与操作记录
	msg := fmt.Sprintf("delete instance: id=%v, namespace=%v, service=%v, host=%v, port=%v",
		instance.ID(), instance.Namespace(), instance.Service(), instance.Host(), instance.Port())
	log.Info(msg, ZapRequestID(rid), ZapPlatformID(pid), zap.Duration("cost", time.Since(start)))
	service := &model.Service{Name: instance.Service(), Namespace: instance.Namespace()}
	s.RecordHistory(instanceRecordEntry(ctx, service, instance, model.ODelete))

	s.sendDiscoverEvent(model.EventInstanceOffline, service.Namespace, service.Name, instance.Host(),
		int(instance.Port()))

	return api.NewInstanceResponse(api.ExecuteSuccess, req)
}

// DeleteInstancesByHost 根据host批量删除服务实例
func (s *Server) DeleteInstancesByHost(ctx context.Context, req []*api.Instance) *api.BatchWriteResponse {
	if checkError := checkBatchInstance(req); checkError != nil {
		return checkError
	}

	return batchOperateInstances(ctx, req, s.DeleteInstanceByHost)
}

// DeleteInstanceByHost 根据host删除服务实例
func (s *Server) DeleteInstanceByHost(ctx context.Context, req *api.Instance) *api.Response {
	requestID := ParseRequestID(ctx)
	platformID := ParsePlatformID(ctx)

	// 参数校验
	if err := checkInstanceByHost(req); err != nil {
		return err
	}

	// 获取实例
	instances, service, err := s.getInstancesMainByService(ctx, req)
	if err != nil {
		return err
	}

	if instances == nil {
		return api.NewInstanceResponse(api.ExecuteSuccess, req)
	}

	ids := make([]interface{}, 0, len(instances))
	for _, instance := range instances {
		ids = append(ids, instance.ID())
	}

	if err := s.storage.BatchDeleteInstances(ids); err != nil {
		log.Error(err.Error(), ZapRequestID(requestID), ZapPlatformID(platformID))
		return wrapperInstanceStoreResponse(req, err)
	}

	for _, instance := range instances {
		msg := fmt.Sprintf("delete instance: id=%v, namespace=%v, service=%v, host=%v, port=%v",
			instance.ID(), service.Namespace, service.Name, instance.Host(), instance.Port())
		log.Info(msg, ZapRequestID(requestID), ZapPlatformID(platformID))
		s.RecordHistory(instanceRecordEntry(ctx, service, instance, model.ODelete))

		s.sendDiscoverEvent(model.EventInstanceOffline, instance.Namespace(),
			instance.Service(),
			instance.Host(),
			int(instance.Port()))

	}

	return api.NewInstanceResponse(api.ExecuteSuccess, req)
}

// UpdateInstances 批量修改服务实例
func (s *Server) UpdateInstances(ctx context.Context, req []*api.Instance) *api.BatchWriteResponse {
	if checkError := checkBatchInstance(req); checkError != nil {
		return checkError
	}

	return batchOperateInstances(ctx, req, s.UpdateInstance)
}

// UpdateInstance 修改单个服务实例
func (s *Server) UpdateInstance(ctx context.Context, req *api.Instance) *api.Response {
	service, instance, preErr := s.execInstancePreStep(ctx, req)
	if preErr != nil {
		return preErr
	}
	if err := checkMetadata(req.GetMetadata()); err != nil {
		return api.NewInstanceResponse(api.InvalidMetadata, req)
	}

	// 修改
	requestID := ParseRequestID(ctx)
	platformID := ParsePlatformID(ctx)
	log.Info(fmt.Sprintf("old instance: %+v", instance), ZapRequestID(requestID), ZapPlatformID(platformID))

	// 比对下更新前后的 isolate 状态
	eventTypes := diffInstanceEvent(req, instance)

	// 存储层操作
	if needUpdate := s.updateInstanceAttribute(req, instance); !needUpdate {
		log.Info("update instance no data change, no need update",
			ZapRequestID(requestID), ZapPlatformID(platformID), zap.String("instance", req.String()))
		return api.NewInstanceResponse(api.NoNeedUpdate, req)
	}
	if err := s.storage.UpdateInstance(instance); err != nil {
		log.Error(err.Error(), ZapRequestID(requestID), ZapPlatformID(platformID))
		return wrapperInstanceStoreResponse(req, err)
	}

	msg := fmt.Sprintf("update instance: id=%v, namespace=%v, service=%v, host=%v, port=%v, healthy = %v",
		instance.ID(), service.Namespace, service.Name, instance.Host(),
		instance.Port(), instance.Healthy())
	log.Info(msg, ZapRequestID(requestID), ZapPlatformID(platformID))
	s.RecordHistory(instanceRecordEntry(ctx, service, instance, model.OUpdate))

	for i := range eventTypes {
		s.sendDiscoverEvent(eventTypes[i], service.Namespace, service.Name, instance.Host(), int(instance.Port()))
	}

	return api.NewInstanceResponse(api.ExecuteSuccess, req)
}

// UpdateInstancesIsolate 批量修改服务实例隔离状态
// @note 必填参数为service+namespace+host
func (s *Server) UpdateInstancesIsolate(ctx context.Context, req []*api.Instance) *api.BatchWriteResponse {
	if checkError := checkBatchInstance(req); checkError != nil {
		return checkError
	}

	return batchOperateInstances(ctx, req, s.UpdateInstanceIsolate)
}

// UpdateInstanceIsolate 修改服务实例隔离状态
// @note 必填参数为service+namespace+ip
func (s *Server) UpdateInstanceIsolate(ctx context.Context, req *api.Instance) *api.Response {
	requestID := ParseRequestID(ctx)
	platformID := ParsePlatformID(ctx)

	// 参数校验
	if err := checkInstanceByHost(req); err != nil {
		return err
	}
	if req.GetIsolate() == nil {
		return api.NewInstanceResponse(api.InvalidInstanceIsolate, req)
	}

	// 获取实例
	instances, service, err := s.getInstancesMainByService(ctx, req)
	if err != nil {
		return err
	}
	if instances == nil {
		return api.NewInstanceResponse(api.NotFoundInstance, req)
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
		return api.NewInstanceResponse(api.NoNeedUpdate, req)
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
		log.Error(err.Error(), ZapRequestID(requestID), ZapPlatformID(platformID))
		return wrapperInstanceStoreResponse(req, err)
	}

	for _, instance := range instances {
		msg := fmt.Sprintf("update instance: id=%v, namespace=%v, service=%v, host=%v, port=%v, isolate=%v",
			instance.ID(), service.Namespace, service.Name, instance.Host(), instance.Port(), instance.Isolate())
		log.Info(msg, ZapRequestID(requestID), ZapPlatformID(platformID))
		s.RecordHistory(instanceRecordEntry(ctx, service, instance, model.OUpdateIsolate))

		// 比对下更新前后的 isolate 状态
		if req.Isolate != nil && instance.Isolate() != req.Isolate.GetValue() {
			eventType := model.EventInstanceCloseIsolate
			if req.Isolate.GetValue() {
				eventType = model.EventInstanceOpenIsolate
			}
			s.sendDiscoverEvent(eventType, req.Namespace.GetValue(), req.Service.GetValue(), req.Host.GetValue(),
				int(req.Port.GetValue()))
		}
	}

	return api.NewInstanceResponse(api.ExecuteSuccess, req)
}

/**
 * @brief 根据ip隔离和删除服务实例的参数检查
 */
func checkInstanceByHost(req *api.Instance) *api.Response {
	if req == nil {
		return api.NewInstanceResponse(api.EmptyRequest, req)
	}
	if err := checkResourceName(req.GetService()); err != nil {
		return api.NewInstanceResponse(api.InvalidServiceName, req)
	}
	if err := checkResourceName(req.GetNamespace()); err != nil {
		return api.NewInstanceResponse(api.InvalidNamespaceName, req)
	}
	if err := checkInstanceHost(req.GetHost()); err != nil {
		return api.NewInstanceResponse(api.InvalidInstanceHost, req)
	}
	return nil
}

/**
 * @brief 根据服务和host获取服务实例
 */
func (s *Server) getInstancesMainByService(ctx context.Context, req *api.Instance) (
	[]*model.Instance, *model.Service, *api.Response) {
	requestID := ParseRequestID(ctx)
	platformID := ParsePlatformID(ctx)

	// 检查服务
	// 这里获取的是源服务的token。如果是别名,service=nil
	service, err := s.storage.GetSourceServiceToken(req.GetService().GetValue(), req.GetNamespace().GetValue())
	if err != nil {
		log.Error(err.Error(), ZapRequestID(requestID), ZapPlatformID(platformID))
		return nil, nil, api.NewInstanceResponse(api.StoreLayerException, req)
	}
	if service == nil {
		return nil, nil, api.NewInstanceResponse(api.NotFoundService, req)
	}

	// 获取服务实例
	instances, err := s.storage.GetInstancesMainByService(service.ID, req.GetHost().GetValue())
	if err != nil {
		log.Error(err.Error(), ZapRequestID(requestID), ZapPlatformID(platformID))
		return nil, nil, api.NewInstanceResponse(api.StoreLayerException, req)
	}
	return instances, service, nil
}

/**
 * @brief 修改服务属性
 */
func (s *Server) updateInstanceAttribute(req *api.Instance, instance *model.Instance) bool {
	// #lizard forgives
	instance.MallocProto()
	needUpdate := false
	insProto := instance.Proto
	if ok := instanceMetaNeedUpdate(req.GetMetadata(), instance.Metadata()); ok {
		insProto.Metadata = req.GetMetadata()
		needUpdate = true
	}
	if !needUpdate {
		// 不需要更新metadata，则置空
		insProto.Metadata = nil
	}

	if req.GetProtocol() != nil && req.GetProtocol().GetValue() != instance.Protocol() {
		insProto.Protocol = req.GetProtocol()
		needUpdate = true
	}

	if req.GetVersion() != nil && req.GetVersion().GetValue() != instance.Version() {
		insProto.Version = req.GetVersion()
		needUpdate = true
	}

	if req.GetPriority() != nil && req.GetPriority().GetValue() != instance.Priority() {
		insProto.Priority = req.GetPriority()
		needUpdate = true
	}

	if req.GetWeight() != nil && req.GetWeight().GetValue() != instance.Weight() {
		insProto.Weight = req.GetWeight()
		needUpdate = true
	}

	if req.GetHealthy() != nil && req.GetHealthy().GetValue() != instance.Healthy() {
		insProto.Healthy = req.GetHealthy()
		needUpdate = true
	}

	if req.GetIsolate() != nil && req.GetIsolate().GetValue() != instance.Isolate() {
		insProto.Isolate = req.GetIsolate()
		needUpdate = true
	}

	if req.GetLogicSet() != nil && req.GetLogicSet().GetValue() != instance.LogicSet() {
		insProto.LogicSet = req.GetLogicSet()
		needUpdate = true
	}

	if ok := updateHealthCheck(req, instance); ok {
		needUpdate = true
	}

	// 每次更改，都要生成一个新的uuid
	if needUpdate {
		insProto.Revision = utils.NewStringValue(utils.NewUUID())
	}

	return needUpdate
}

// 健康检查的更新
func updateHealthCheck(req *api.Instance, instance *model.Instance) bool {
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
		if api.HealthCheck_HEARTBEAT != instance.HealthCheck().GetType() {
			// health check type有变更
			needUpdate = true
		}
		insProto.HealthCheck = req.GetHealthCheck()
		insProto.HealthCheck.Type = api.HealthCheck_HEARTBEAT
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
func (s *Server) GetInstances(ctx context.Context, query map[string]string) *api.BatchQueryResponse {
	// 对数据先进行提前处理一下
	filters, metaFilter, batchErr := preGetInstances(query)
	if batchErr != nil {
		return batchErr
	}
	// 分页数据
	offset, limit, err := ParseOffsetAndLimit(filters)
	if err != nil {
		return api.NewBatchQueryResponse(api.InvalidParameter)
	}

	total, instances, err := s.storage.GetExpandInstances(filters, metaFilter, offset, limit)
	if err != nil {
		log.Errorf("[Server][Instances][Query] instances store err: %s", err.Error())
		return api.NewBatchQueryResponse(api.StoreLayerException)
	}

	out := api.NewBatchQueryResponse(api.ExecuteSuccess)
	out.Amount = utils.NewUInt32Value(total)
	out.Size = utils.NewUInt32Value(uint32(len(instances)))

	apiInstances := make([]*api.Instance, 0, len(instances))
	for _, instance := range instances {
		// 数据来源于数据库，不需要拷贝一份，直接填充后返回
		s.packCmdb(instance.Proto)
		apiInstances = append(apiInstances, instance.Proto)
	}
	out.Instances = apiInstances

	return out
}

// GetInstancesCount 查询总的服务实例，不带过滤条件的
func (s *Server) GetInstancesCount(ctx context.Context) *api.BatchQueryResponse {
	count, err := s.storage.GetInstancesCount()
	if err != nil {
		log.Errorf("[Server][Instance][Count] storage get err: %s", err.Error())
		return api.NewBatchQueryResponse(api.StoreLayerException)
	}

	out := api.NewBatchQueryResponse(api.ExecuteSuccess)
	out.Amount = utils.NewUInt32Value(count)
	out.Instances = make([]*api.Instance, 0)
	return out
}

// CleanInstance 清理无效的实例(flag == 1)
func (s *Server) CleanInstance(ctx context.Context, req *api.Instance) *api.Response {
	// 无效数据，不需要鉴权，直接删除
	getInstanceID := func() (string, *api.Response) {
		if req.GetId() != nil {
			if req.GetId().GetValue() == "" {
				return "", api.NewInstanceResponse(api.InvalidInstanceID, req)
			}
			return req.GetId().GetValue(), nil
		}
		return utils.CheckInstanceTetrad(req)
	}

	instanceID, resp := getInstanceID()
	if resp != nil {
		return resp
	}
	if err := s.storage.CleanInstance(instanceID); err != nil {
		log.Error("Clean instance", zap.String("err", err.Error()), ZapRequestID(ParseRequestID(ctx)))
		return api.NewInstanceResponse(api.StoreLayerException, req)
	}

	log.Info("Clean instance", ZapRequestID(ParseRequestID(ctx)), zap.String("instanceID", instanceID))
	return api.NewInstanceResponse(api.ExecuteSuccess, req)
}

// update/delete instance前置条件
func (s *Server) execInstancePreStep(ctx context.Context, req *api.Instance) (
	*model.Service, *model.Instance, *api.Response) {
	rid := ParseRequestID(ctx)

	// 参数检查
	instanceID, checkError := checkReviseInstance(req)
	if checkError != nil {
		return nil, nil, checkError
	}

	// 检查服务实例是否存在
	instance, err := s.storage.GetInstance(instanceID)
	if err != nil {
		log.Error(err.Error(), ZapRequestID(rid))
		return nil, nil, api.NewInstanceResponse(api.StoreLayerException, req)
	}
	if instance == nil {
		return nil, nil, api.NewInstanceResponse(api.NotFoundInstance, req)
	}

	service, resp := s.instanceAuth(ctx, req, instance.ServiceID)
	if resp != nil {
		return nil, nil, resp
	}

	return service, instance, nil
}

// 实例鉴权
func (s *Server) instanceAuth(ctx context.Context, req *api.Instance, serviceID string) (
	*model.Service, *api.Response) {
	service, err := s.storage.GetServiceByID(serviceID)
	if err != nil {
		log.Error(err.Error(), ZapRequestID(ParseRequestID(ctx)))
		return nil, api.NewInstanceResponse(api.StoreLayerException, req)
	}
	if service == nil {
		return nil, api.NewInstanceResponse(api.NotFoundResource, req)
	}

	return service, nil
}

// 获取api.instance
func (s *Server) getInstance(service *api.Service, instance *api.Instance) *api.Instance {
	out := &api.Instance{
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
		// Ctime:             instance.GetCtime(),
		Mtime:    instance.GetMtime(),
		Revision: instance.GetRevision(),
	}

	s.packCmdb(out)
	return out
}

// 获取cmdb
func (s *Server) packCmdb(instance *api.Instance) {
	if instance == nil || instance.GetLocation() != nil {
		return
	}
	if s.cmdb == nil {
		return
	}

	location, err := s.cmdb.GetLocation(instance.GetHost().GetValue())
	if err == nil && location != nil {
		instance.Location = location.Proto
	}

}

func (s *Server) sendDiscoverEvent(eventType model.DiscoverEventType, namespace, service, host string, port int) {
	// 发布隔离状态变化事件
	event := model.DiscoverEvent{
		Namespace: namespace,
		Service:   service,
		Host:      host,
		Port:      port,
		EType:     eventType,
	}

	s.PublishDiscoverEvent(event)
}

// createServiceIfAbsent 如果服务不存在，则进行创建，并返回服务的ID信息
func (s *Server) createServiceIfAbsent(ctx context.Context, instance *api.Instance) (uint32, string, error) {

	svc, err := s.loadService(instance)
	if err != nil {
		return api.ExecuteException, "", err
	}
	if svc != nil {
		return api.ExecuteSuccess, svc.ID, nil
	}

	simpleService := &api.Service{
		Name:      utils.NewStringValue(instance.GetService().GetValue()),
		Namespace: utils.NewStringValue(instance.GetNamespace().GetValue()),
		Owners: func() *wrapperspb.StringValue {
			owner := utils.ParseOwnerID(ctx)
			if owner == "" {
				return utils.NewStringValue("Polaris")
			}
			return utils.NewStringValue(owner)
		}(),
		Metadata: map[string]string{
			MetadataInternalAutoCreated: "true",
		},
	}

	key := fmt.Sprintf("%s:%s", simpleService.Namespace, simpleService.Name)

	ret, _, _ := s.createServiceSingle.Do(key, func() (interface{}, error) {
		resp := s.CreateService(ctx, simpleService)
		return resp, nil
	})

	resp := ret.(*api.Response)

	retCode := resp.GetCode().GetValue()

	if retCode != api.ExecuteSuccess && retCode != api.ExistedResource {
		return retCode, "", errors.New(resp.GetInfo().GetValue())
	}

	svcId := resp.Service.Id.GetValue()

	return retCode, svcId, nil
}

func (s *Server) loadService(instance *api.Instance) (*model.Service, error) {

	svc := s.caches.Service().GetServiceByName(instance.GetService().GetValue(), instance.GetNamespace().GetValue())

	if svc != nil {
		if svc.IsAlias() {
			return nil, errors.New("service is alias")
		}
		return svc, nil
	}

	// 再走数据库查询一遍
	svc, err := s.storage.GetService(instance.GetService().GetValue(), instance.GetNamespace().GetValue())
	if err != nil {
		return nil, err
	}

	if svc != nil && svc.IsAlias() {
		return nil, errors.New("service is alias")
	}

	return svc, nil
}

/*
 * @brief 检查批量请求
 */
func checkBatchInstance(req []*api.Instance) *api.BatchWriteResponse {
	if len(req) == 0 {
		return api.NewBatchWriteResponse(api.EmptyRequest)
	}

	if len(req) > MaxBatchSize {
		return api.NewBatchWriteResponse(api.BatchSizeOverLimit)
	}

	return nil
}

/*
 * @brief 检查创建服务实例请求参数
 */
func checkCreateInstance(req *api.Instance) (string, *api.Response) {
	if req == nil {
		return "", api.NewInstanceResponse(api.EmptyRequest, req)
	}

	if err := checkMetadata(req.GetMetadata()); err != nil {
		return "", api.NewInstanceResponse(api.InvalidMetadata, req)
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
func checkReviseInstance(req *api.Instance) (string, *api.Response) {
	if req == nil {
		return "", api.NewInstanceResponse(api.EmptyRequest, req)
	}

	if req.GetId() != nil {
		if req.GetId().GetValue() == "" {
			return "", api.NewInstanceResponse(api.InvalidInstanceID, req)
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

/*
 * @brief 检查心跳实例请求参数
 * 检查是否存在token，以及 id或者四元组
 * 注意：心跳上报只允许从client上报，因此token只会存在req中
 */
func checkHeartbeatInstance(req *api.Instance) (string, *api.Response) {
	if req == nil {
		return "", api.NewInstanceResponse(api.EmptyRequest, req)
	}
	if req.GetId() != nil {
		if req.GetId().GetValue() == "" {
			return "", api.NewInstanceResponse(api.InvalidInstanceID, req)
		}
		return req.GetId().GetValue(), nil
	}
	return utils.CheckInstanceTetrad(req)
}

// 获取instance请求的token信息
func parseInstanceReqToken(ctx context.Context, req *api.Instance) string {
	if reqToken := req.GetServiceToken().GetValue(); reqToken != "" {
		return reqToken
	}

	return ParseToken(ctx)
}

// 实例查询前置处理
func preGetInstances(query map[string]string) (map[string]string, map[string]string, *api.BatchQueryResponse) {
	// 不允许全量查询服务实例
	if len(query) == 0 {
		return nil, nil, api.NewBatchQueryResponse(api.EmptyQueryParameter)
	}
	_, serviceIsAvail := query["service"]
	_, namespaceIsAvail := query["namespace"]
	_, hostIsAvail := query["host"]
	// 要么（service，namespace）存在，要么host存在，不然视为参数不完整
	if !((serviceIsAvail && namespaceIsAvail) || hostIsAvail) {
		return nil, nil, api.NewBatchQueryResponse(api.InvalidQueryInsParameter)
	}

	var metaFilter map[string]string
	metaKey, metaKeyAvail := query["keys"]
	metaValue, metaValueAvail := query["values"]
	if metaKeyAvail != metaValueAvail {
		return nil, nil, api.NewBatchQueryResponseWithMsg(
			api.InvalidQueryInsParameter, "instance metadata key and value must be both provided")
	}
	if metaKeyAvail {
		metaFilter = map[string]string{metaKey: metaValue}
	}

	// 以healthy为准
	_, lhs := query["health_status"]
	_, rhs := query["healthy"]
	if lhs && rhs {
		delete(query, "health_status")
	}

	bool2Str := func(key string) {
		val, ok := query[key]
		if !ok {
			return
		}
		if val == "true" {
			query[key] = "1"
		} else if val == "false" {
			query[key] = "0"
		}
	}

	// 处理一下两个bool值的字段
	bool2Str("health_status")
	bool2Str("healthy")
	bool2Str("isolate")

	filters := make(map[string]string)
	for key, value := range query {
		if _, ok := InstanceFilterAttributes[key]; !ok {
			log.Errorf("[Server][Instance][Query] attribute(%s) is not allowed", key)
			return nil, metaFilter, api.NewBatchQueryResponseWithMsg(api.InvalidParameter, key+" is not allowed")
		}

		if value == "" {
			log.Errorf("[Server][Instance][Query] attribute(%s: %s) is not allowed empty", key, value)
			return nil, metaFilter,
				api.NewBatchQueryResponseWithMsg(api.InvalidParameter, "the value for "+key+" is empty")
		}
		if attr, ok := InsFilter2toreAttr[key]; ok {
			key = attr
		}
		if !NotInsFilterAttr[key] {
			filters[key] = value
		}
	}

	return filters, metaFilter, nil
}

// instance metadata need update
func instanceMetaNeedUpdate(req map[string]string, old map[string]string) bool {
	if req == nil {
		return false
	}

	if len(req) != len(old) {
		return true
	}

	needUpdate := false
	for key, value := range req {
		oldValue, ok := old[key]
		if !ok {
			needUpdate = true
			break
		}
		if value != oldValue {
			needUpdate = true
			break
		}
	}
	if needUpdate {
		return needUpdate
	}

	for key, value := range old {
		newValue, ok := req[key]
		if !ok {
			needUpdate = true
			break
		}
		if value != newValue {
			needUpdate = true
			break
		}
	}

	return needUpdate
}

// 批量操作实例
func batchOperateInstances(ctx context.Context, reqs []*api.Instance,
	handler func(ctx context.Context, req *api.Instance) *api.Response) *api.BatchWriteResponse {
	responses := api.NewBatchWriteResponse(api.ExecuteSuccess)

	chs := make([]chan *api.Response, 0, len(reqs))
	for i, instance := range reqs {
		chs = append(chs, make(chan *api.Response))
		go func(index int, ins *api.Instance) {
			chs[index] <- handler(ctx, ins)
		}(i, instance)
	}

	for _, ch := range chs {
		resp := <-ch
		responses.Collect(resp)
	}

	return api.FormatBatchWriteResponse(responses)
}

// wrapper instance store response
func wrapperInstanceStoreResponse(instance *api.Instance, err error) *api.Response {
	resp := storeError2Response(err)
	if resp == nil {
		return nil
	}

	resp.Instance = instance
	return resp
}

// 生成instance的记录entry
func instanceRecordEntry(ctx context.Context, service *model.Service, ins *model.Instance,
	opt model.OperationType) *model.RecordEntry {
	if service == nil || ins == nil {
		return nil
	}
	entry := &model.RecordEntry{
		ResourceType:  model.RInstance,
		OperationType: opt,
		Namespace:     service.Namespace,
		Service:       service.Name,
		Operator:      ParseOperator(ctx),
		CreateTime:    time.Now(),
	}
	if opt == model.OCreate || opt == model.OUpdate {
		entry.Context = fmt.Sprintf("host:%s,port:%d,weight:%d,healthy:%v,isolate:%v,priority:%d,meta:%+v",
			ins.Host(), ins.Port(), ins.Weight(), ins.Healthy(), ins.Isolate(),
			ins.Priority(), ins.Metadata())
	} else if opt == model.OUpdateIsolate {
		entry.Context = fmt.Sprintf("host:%s,port=%d,isolate:%v", ins.Host(), ins.Port(), ins.Isolate())
	} else {
		entry.Context = fmt.Sprintf("host:%s,port:%d", ins.Host(), ins.Port())
	}
	return entry
}

// CheckDbInstanceFieldLen 检查DB中service表对应的入参字段合法性
func CheckDbInstanceFieldLen(req *api.Instance) (*api.Response, bool) {
	if err := CheckDbStrFieldLen(req.GetService(), MaxDbServiceNameLength); err != nil {
		return api.NewInstanceResponse(api.InvalidServiceName, req), true
	}
	if err := CheckDbStrFieldLen(req.GetNamespace(), MaxDbServiceNamespaceLength); err != nil {
		return api.NewInstanceResponse(api.InvalidNamespaceName, req), true
	}
	if err := CheckDbStrFieldLen(req.GetHost(), MaxDbInsHostLength); err != nil {
		return api.NewInstanceResponse(api.InvalidInstanceHost, req), true
	}
	if err := CheckDbStrFieldLen(req.GetProtocol(), MaxDbInsProtocolLength); err != nil {
		return api.NewInstanceResponse(api.InvalidInstanceProtocol, req), true
	}
	if err := CheckDbStrFieldLen(req.GetVersion(), MaxDbInsVersionLength); err != nil {
		return api.NewInstanceResponse(api.InvalidInstanceVersion, req), true
	}
	if err := CheckDbStrFieldLen(req.GetLogicSet(), MaxDbInsLogicSetLength); err != nil {
		return api.NewInstanceResponse(api.InvalidInstanceLogicSet, req), true
	}
	if err := CheckDbMetaDataFieldLen(req.GetMetadata()); err != nil {
		return api.NewInstanceResponse(api.InvalidMetadata, req), true
	}
	if req.GetPort().GetValue() > 65535 {
		return api.NewInstanceResponse(api.InvalidInstancePort, req), true
	}

	if req.GetWeight().GetValue() > 65535 {
		return api.NewInstanceResponse(api.InvalidParameter, req), true
	}
	return nil, false
}

func diffInstanceEvent(req *api.Instance, save *model.Instance) []model.DiscoverEventType {
	eventTypes := make([]model.DiscoverEventType, 0)

	if req.Isolate != nil && save.Isolate() != req.Isolate.GetValue() {
		if req.Isolate.GetValue() {
			eventTypes = append(eventTypes, model.EventInstanceOpenIsolate)
		} else {
			eventTypes = append(eventTypes, model.EventInstanceCloseIsolate)
		}
	}

	if req.Healthy != nil && save.Healthy() != req.Healthy.GetValue() {
		if req.Healthy.GetValue() {
			eventTypes = append(eventTypes, model.EventInstanceTurnHealth)
		} else {
			eventTypes = append(eventTypes, model.EventInstanceTurnUnHealth)
		}
	}

	return eventTypes
}
