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

package healthcheck

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"go.uber.org/zap"

	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/srand"
	"github.com/polarismesh/polaris/common/timewheel"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/plugin"
)

const (
	expireTtlCount = 3
)

// CheckScheduler schedule and run check actions
type CheckScheduler struct {
	rwMutex            *sync.RWMutex
	scheduledInstances map[string]*itemValue

	timeWheel           *timewheel.TimeWheel
	minCheckIntervalSec int64
	maxCheckIntervalSec int64

	adoptInstancesChan chan AdoptEvent
	ctx                context.Context
}

// AdoptEvent is the event for adopt
type AdoptEvent struct {
	InstanceId string
	Add        bool
	Checker    plugin.HealthChecker
}

//go:generate stringer -type=ItemType
type ItemType int

const (
	itemTypeInstance ItemType = iota
	itemTypeClient
)

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[itemTypeInstance-0]
	_ = x[itemTypeClient-1]
}

const _ItemType_name = "itemTypeInstanceitemTypeClient"

var _ItemType_index = [...]uint8{0, 16, 30}

func (i ItemType) String() string {
	if i < 0 || i >= ItemType(len(_ItemType_index)-1) {
		return "ItemType(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _ItemType_name[_ItemType_index[i]:_ItemType_index[i+1]]
}

type itemValue struct {
	mutex               *sync.Mutex
	id                  string
	host                string
	port                uint32
	scheduled           uint32
	lastSetEventTimeSec int64
	ttlDurationSec      uint32
	expireDurationSec   uint32
	checker             plugin.HealthChecker
	ItemType            ItemType
}

func (i *itemValue) eventExpired() (int64, bool) {
	curTimeSec := time.Now().Unix()
	return curTimeSec, curTimeSec-i.lastSetEventTimeSec >= int64(i.expireDurationSec)
}

func newCheckScheduler(ctx context.Context, slotNum int,
	minCheckInterval time.Duration, maxCheckInterval time.Duration) *CheckScheduler {

	scheduler := &CheckScheduler{
		rwMutex:             &sync.RWMutex{},
		scheduledInstances:  make(map[string]*itemValue),
		timeWheel:           timewheel.New(time.Second, slotNum, "health-interval-check"),
		minCheckIntervalSec: int64(minCheckInterval.Seconds()),
		maxCheckIntervalSec: int64(maxCheckInterval.Seconds()),
		adoptInstancesChan:  make(chan AdoptEvent, 1024),
		ctx:                 ctx,
	}

	go scheduler.doCheck(ctx)
	go scheduler.doAdopt(ctx)
	return scheduler
}

func (c *CheckScheduler) doCheck(ctx context.Context) {
	c.timeWheel.Start()
	log.Infof("[Health Check][Check]timeWheel has been started")

	<-ctx.Done()
	c.timeWheel.Stop()
	log.Infof("[Health Check][Check]timeWheel has been stopped")
}

const (
	batchAdoptInterval = 30 * time.Millisecond
	batchAdoptCount    = 30
)

func (c *CheckScheduler) doAdopt(ctx context.Context) {
	instancesToAdd := make(map[string]bool)
	instancesToRemove := make(map[string]bool)
	var checker plugin.HealthChecker
	ticker := time.NewTicker(batchAdoptInterval)
	defer func() {
		ticker.Stop()
	}()
	for {
		select {
		case event := <-c.adoptInstancesChan:
			instanceId := event.InstanceId
			if event.Add {
				instancesToAdd[instanceId] = true
				delete(instancesToRemove, instanceId)
			} else {
				instancesToRemove[instanceId] = true
				delete(instancesToAdd, instanceId)
			}
			checker = event.Checker
			if len(instancesToAdd) == batchAdoptCount {
				instancesToAdd = c.processAdoptEvents(instancesToAdd, true, checker)
			}
			if len(instancesToRemove) == batchAdoptCount {
				instancesToRemove = c.processAdoptEvents(instancesToRemove, false, checker)
			}
		case <-ticker.C:
			if len(instancesToAdd) > 0 {
				instancesToAdd = c.processAdoptEvents(instancesToAdd, true, checker)
			}
			if len(instancesToRemove) > 0 {
				instancesToRemove = c.processAdoptEvents(instancesToRemove, false, checker)
			}
		case <-ctx.Done():
			log.Infof("[Health Check][Check]adopting routine has been stopped")
			return
		}
	}
}

func (c *CheckScheduler) processAdoptEvents(
	instances map[string]bool, add bool, checker plugin.HealthChecker) map[string]bool {
	instanceIds := make([]string, 0, len(instances))
	for id := range instances {
		instanceIds = append(instanceIds, id)
	}
	var err error
	if add {
		log.Infof("[Health Check][Check]add adopting instances, ids are %v", instanceIds)
		err = checker.AddToCheck(&plugin.AddCheckRequest{
			Instances: instanceIds,
			LocalHost: server.localHost,
		})
	} else {
		log.Infof("[Health Check][Check]remove adopting instances, ids are %v", instanceIds)
		err = checker.RemoveFromCheck(&plugin.AddCheckRequest{
			Instances: instanceIds,
			LocalHost: server.localHost,
		})
	}
	if err != nil {
		log.Errorf("[Health Check][Check]fail to do adopt event, instances %v, localhost %s, add %v",
			instanceIds, server.localHost, add)
		return instances
	}
	return make(map[string]bool)
}

func (c *CheckScheduler) addAdopting(instanceId string, checker plugin.HealthChecker) {
	select {
	case c.adoptInstancesChan <- AdoptEvent{
		InstanceId: instanceId,
		Add:        true,
		Checker:    checker}:
	case <-c.ctx.Done():
		return
	}
}

func (c *CheckScheduler) removeAdopting(instanceId string, checker plugin.HealthChecker) {
	select {
	case c.adoptInstancesChan <- AdoptEvent{
		InstanceId: instanceId,
		Add:        false,
		Checker:    checker}:
	case <-c.ctx.Done():
		return
	}
}

func (c *CheckScheduler) putInstanceIfAbsent(instanceWithChecker *InstanceWithChecker) (bool, *itemValue) {
	c.rwMutex.Lock()
	defer c.rwMutex.Unlock()
	instance := instanceWithChecker.instance
	var instValue *itemValue
	var ok bool
	if instValue, ok = c.scheduledInstances[instance.ID()]; ok {
		return true, instValue
	}
	instValue = &itemValue{
		mutex:             &sync.Mutex{},
		host:              instance.Host(),
		port:              instance.Port(),
		id:                instance.ID(),
		expireDurationSec: getExpireDurationSec(instance.Proto),
		checker:           instanceWithChecker.checker,
		ttlDurationSec:    instance.HealthCheck().GetHeartbeat().GetTtl().GetValue(),
		ItemType:          itemTypeInstance,
	}
	c.scheduledInstances[instance.ID()] = instValue
	return false, instValue
}

const clientReportTtlSec uint32 = 120

func (c *CheckScheduler) putClientIfAbsent(clientWithChecker *ClientWithChecker) (bool, *itemValue) {
	c.rwMutex.Lock()
	defer c.rwMutex.Unlock()
	client := clientWithChecker.client
	var instValue *itemValue
	var ok bool
	clientId := client.Proto().GetId().GetValue()
	if instValue, ok = c.scheduledInstances[clientId]; ok {
		return true, instValue
	}
	instValue = &itemValue{
		mutex:             &sync.Mutex{},
		host:              client.Proto().GetHost().GetValue(),
		port:              0,
		id:                clientId,
		expireDurationSec: expireTtlCount * clientReportTtlSec,
		checker:           clientWithChecker.checker,
		ttlDurationSec:    clientReportTtlSec,
		ItemType:          itemTypeClient,
	}
	c.scheduledInstances[clientId] = instValue
	return false, instValue
}

func (c *CheckScheduler) getInstanceValue(instanceId string) (*itemValue, bool) {
	c.rwMutex.RLock()
	defer c.rwMutex.RUnlock()
	value, ok := c.scheduledInstances[instanceId]
	return value, ok
}

// AddInstance add instance to check
func (c *CheckScheduler) AddInstance(instanceWithChecker *InstanceWithChecker) {
	exists, instValue := c.putInstanceIfAbsent(instanceWithChecker)
	if exists {
		return
	}
	c.addAdopting(instValue.id, instValue.checker)
	instance := instanceWithChecker.instance
	log.Infof("[Health Check][Check]add check instance is %s, host is %s:%d",
		instance.ID(), instance.Host(), instance.Port())
	c.addUnHealthyCallback(instValue)
}

// AddInstance add instance to check
func (c *CheckScheduler) AddClient(clientWithChecker *ClientWithChecker) {
	exists, instValue := c.putClientIfAbsent(clientWithChecker)
	if exists {
		return
	}
	c.addAdopting(instValue.id, instValue.checker)
	client := clientWithChecker.client
	log.Infof("[Health Check][Check]add check instance is %s, host is %s:%d",
		client.Proto().GetId().GetValue(), client.Proto().GetHost(), 0)
	c.addUnHealthyCallback(instValue)
}

func getExpireDurationSec(instance *api.Instance) uint32 {
	ttlValue := instance.GetHealthCheck().GetHeartbeat().GetTtl().GetValue()
	return expireTtlCount * ttlValue
}

func getRandDelayMilli() uint32 {
	delayMilli := srand.Intn(1000)
	return uint32(delayMilli)
}

func (c *CheckScheduler) addHealthyCallback(instance *itemValue, lastHeartbeatTimeSec int64) {
	delaySec := instance.expireDurationSec
	var nextDelaySec int64
	if lastHeartbeatTimeSec > 0 {
		curTimeSec := currentTimeSec()
		timePassed := curTimeSec - lastHeartbeatTimeSec
		if timePassed > 0 {
			nextDelaySec = int64(delaySec) - timePassed
		}
	}
	if nextDelaySec > 0 && nextDelaySec < c.minCheckIntervalSec {
		nextDelaySec = c.minCheckIntervalSec
	}
	if nextDelaySec > 0 {
		delaySec = uint32(nextDelaySec)
	}
	host := instance.host
	port := instance.port
	instanceId := instance.id
	delayMilli := delaySec*1000 + getRandDelayMilli()
	log.Debugf("[Health Check][Check]add healthy callback, %s is %s:%d, id is %s, delay is %d(ms)",
		instance.ItemType.String(), host, port, instanceId, delayMilli)
	if instance.ItemType == itemTypeClient {
		c.timeWheel.AddTask(delayMilli, instanceId, c.checkCallbackClient)
	} else {
		c.timeWheel.AddTask(delayMilli, instanceId, c.checkCallbackInstance)
	}
}

func (c *CheckScheduler) addUnHealthyCallback(instance *itemValue) {
	delaySec := instance.expireDurationSec
	if c.maxCheckIntervalSec > 0 && int64(delaySec) > c.maxCheckIntervalSec {
		delaySec = uint32(c.maxCheckIntervalSec)
	}
	host := instance.host
	port := instance.port
	instanceId := instance.id
	delayMilli := delaySec*1000 + getRandDelayMilli()
	log.Debugf("[Health Check][Check]add unhealthy callback, %s is %s:%d, id is %s, delay is %d(ms)",
		instance.ItemType.String(), host, port, instanceId, delayMilli)
	if instance.ItemType == itemTypeClient {
		c.timeWheel.AddTask(delayMilli, instanceId, c.checkCallbackClient)
	} else {
		c.timeWheel.AddTask(delayMilli, instanceId, c.checkCallbackInstance)
	}
}

func (c *CheckScheduler) checkCallbackClient(value interface{}) {
	clientId := value.(string)
	instanceValue, ok := c.getInstanceValue(clientId)
	if !ok {
		log.Infof("[Health Check][Check]client %s has been removed from callback", clientId)
		return
	}
	instanceValue.mutex.Lock()
	defer instanceValue.mutex.Unlock()
	var checkResp *plugin.CheckResponse
	var err error
	defer func() {
		if checkResp != nil && checkResp.Regular && checkResp.Healthy {
			c.addHealthyCallback(instanceValue, checkResp.LastHeartbeatTimeSec)
		} else {
			c.addUnHealthyCallback(instanceValue)
		}
	}()
	cachedClient := server.cacheProvider.GetClient(clientId)
	if cachedClient == nil {
		log.Infof("[Health Check][Check]client %s has been deleted", instanceValue.id)
		return
	}
	request := &plugin.CheckRequest{
		QueryRequest: plugin.QueryRequest{
			InstanceId: toClientId(instanceValue.id),
			Host:       instanceValue.host,
			Port:       instanceValue.port,
			Healthy:    true,
		},
		CurTimeSec:        currentTimeSec,
		ExpireDurationSec: instanceValue.expireDurationSec,
	}
	checkResp, err = instanceValue.checker.Check(request)
	if err != nil {
		log.Errorf("[Health Check][Check]fail to check client %s, id is %s, err is %v",
			instanceValue.host, instanceValue.id, err)
		return
	}
	if !checkResp.StayUnchanged {
		if !checkResp.Healthy {
			log.Infof(
				"[Health Check][Check]client change from healthy to unhealthy, id is %s, address is %s",
				instanceValue.id, instanceValue.host)
			code := server.asyncDeleteClient(cachedClient.Proto())
			if code != api.ExecuteSuccess {
				log.Errorf("[Health Check][Check]fail to update client, id is %s, address is %s, code is %d",
					instanceValue.id, instanceValue.host, code)
			}
		}
	}
}

func (c *CheckScheduler) checkCallbackInstance(value interface{}) {
	instanceId := value.(string)
	instanceValue, ok := c.getInstanceValue(instanceId)
	if !ok {
		log.Infof("[Health Check][Check]instance %s has been removed from callback", instanceId)
		return
	}

	instanceValue.mutex.Lock()
	defer instanceValue.mutex.Unlock()

	var checkResp *plugin.CheckResponse
	var err error
	defer func() {
		if checkResp != nil && checkResp.Regular && checkResp.Healthy {
			c.addHealthyCallback(instanceValue, checkResp.LastHeartbeatTimeSec)
		} else {
			c.addUnHealthyCallback(instanceValue)
		}
	}()

	cachedInstance := server.cacheProvider.GetInstance(instanceId)
	if cachedInstance == nil {
		log.Infof("[Health Check][Check]instance %s has been deleted", instanceValue.id)
		return
	}
	request := &plugin.CheckRequest{
		QueryRequest: plugin.QueryRequest{
			InstanceId: instanceValue.id,
			Host:       instanceValue.host,
			Port:       instanceValue.port,
			Healthy:    cachedInstance.Healthy(),
		},
		CurTimeSec:        currentTimeSec,
		ExpireDurationSec: instanceValue.expireDurationSec,
	}
	checkResp, err = instanceValue.checker.Check(request)
	if err != nil {
		log.Errorf("[Health Check][Check]fail to check instance %s:%d, id is %s, err is %v",
			instanceValue.host, instanceValue.port, instanceValue.id, err)
		return
	}
	if !checkResp.StayUnchanged {
		code := setInsDbStatus(cachedInstance, checkResp.Healthy)
		if checkResp.Healthy {
			// from unhealthy to healthy
			log.Infof(
				"[Health Check][Check]instance change from unhealthy to healthy, id is %s, address is %s:%d",
				instanceValue.id, instanceValue.host, instanceValue.port)
		} else {
			// from healthy to unhealthy
			log.Infof(
				"[Health Check][Check]instance change from healthy to unhealthy, id is %s, address is %s:%d",
				instanceValue.id, instanceValue.host, instanceValue.port)
		}
		if code != api.ExecuteSuccess {
			log.Errorf(
				"[Health Check][Check]fail to update instance, id is %s, address is %s:%d, code is %d",
				instanceValue.id, instanceValue.host, instanceValue.port, code)
		}
	}
}

// DelInstance del instance from check
func (c *CheckScheduler) DelClient(clientWithChecker *ClientWithChecker) {
	client := clientWithChecker.client
	clientId := client.Proto().GetId().GetValue()
	exists := c.delIfPresent(clientId)
	log.Infof("[Health Check][Check]remove check instance is %s:%d, id is %s, exists is %v",
		client.Proto().GetHost().GetValue(), 0, clientId, exists)
	if exists {
		c.removeAdopting(clientId, clientWithChecker.checker)
	}
}

// DelInstance del instance from check
func (c *CheckScheduler) DelInstance(instanceWithChecker *InstanceWithChecker) {
	instance := instanceWithChecker.instance
	instanceId := instance.ID()
	exists := c.delIfPresent(instanceId)
	log.Infof("[Health Check][Check]remove check instance is %s:%d, id is %s, exists is %v",
		instance.Host(), instance.Port(), instanceId, exists)
	if exists {
		c.removeAdopting(instanceId, instanceWithChecker.checker)
	}
}

func (c *CheckScheduler) delIfPresent(instanceId string) bool {
	c.rwMutex.Lock()
	defer c.rwMutex.Unlock()
	_, ok := c.scheduledInstances[instanceId]
	delete(c.scheduledInstances, instanceId)
	return ok
}

// setInsDbStatus 修改实例状态, 需要打印操作记录
func setInsDbStatus(instance *model.Instance, healthStatus bool) uint32 {
	id := instance.ID()
	host := instance.Host()
	port := instance.Port()
	log.Infof("[Health Check][Check]addr:%s:%d id:%s set db status %v", host, port, id, healthStatus)

	var code uint32
	if server.bc.HeartbeatOpen() {
		code = server.asyncSetInsDbStatus(instance.Proto, healthStatus)
	} else {
		code = server.serialSetInsDbStatus(instance.Proto, healthStatus)
	}
	if code != api.ExecuteSuccess {
		return code
	}
	recordInstance := &model.Instance{
		Proto: &api.Instance{
			Host:     instance.Proto.GetId(),
			Port:     instance.Proto.GetPort(),
			Priority: instance.Proto.GetPriority(),
			Weight:   instance.Proto.GetWeight(),
			Healthy:  utils.NewBoolValue(healthStatus),
			Isolate:  instance.Proto.GetIsolate(),
		},
	}

	// 这里为了避免多次发送重复的事件，对实例原本的health 状态以及 healthStatus 状态进行对比，不一致才
	// 发布服务实例变更事件
	if instance.Healthy() != healthStatus {
		event := model.DiscoverEvent{
			Namespace: instance.Namespace(),
			Service:   instance.Service(),
			Host:      instance.Host(),
			Port:      int(instance.Port()),
		}

		// 实例状态变化进行 DiscoverEvent 输出
		if healthStatus {
			event.EType = model.EventInstanceTurnHealth
		} else {
			event.EType = model.EventInstanceTurnUnHealth
		}

		server.PublishDiscoverEvent(instance.ServiceID, event)
	}

	server.RecordHistory(instanceRecordEntry(recordInstance, model.OUpdate))
	return code
}

// asyncSetInsDbStatus 异步新建实例
// 底层函数会合并delete请求，增加并发创建的吞吐
// req 原始请求
// ins 包含了req数据与instanceID，serviceToken
func (s *Server) asyncSetInsDbStatus(ins *api.Instance, healthStatus bool) uint32 {
	future := s.bc.AsyncHeartbeat(ins, healthStatus)
	if err := future.Wait(); err != nil {
		log.Error(err.Error())
	}
	return future.Code()
}

// asyncDeleteClient 异步软删除客户端
// 底层函数会合并delete请求，增加并发创建的吞吐
// req 原始请求
// ins 包含了req数据与instanceID，serviceToken
func (s *Server) asyncDeleteClient(client *api.Client) uint32 {
	future := s.bc.AsyncDeregisterClient(client)
	if err := future.Wait(); err != nil {
		log.Error("[Health Check][Check] async delete client", zap.String("client-id", client.GetId().GetValue()),
			zap.Error(err))
	}
	return future.Code()
}

// serialSetInsDbStatus 同步串行创建实例
// req为原始的请求体
// ins包括了req的内容，并且填充了instanceID与serviceToken
func (s *Server) serialSetInsDbStatus(ins *api.Instance, healthStatus bool) uint32 {
	id := ins.GetId().GetValue()
	err := server.storage.SetInstanceHealthStatus(id, model.StatusBoolToInt(healthStatus), utils.NewUUID())
	if err != nil {
		log.Errorf("[Health Check][Check]id: %s set db status err:%s", id, err)
		return api.StoreLayerException
	}
	return api.ExecuteSuccess
}

// instanceRecordEntry generate instance record entry
func instanceRecordEntry(ins *model.Instance, opt model.OperationType) *model.RecordEntry {
	if ins == nil {
		return nil
	}
	entry := &model.RecordEntry{
		ResourceType:  model.RInstance,
		OperationType: opt,
		Namespace:     ins.Proto.GetNamespace().GetValue(),
		Service:       ins.Proto.GetService().GetValue(),
		Operator:      "Polaris",
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
