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
	"sync"
	"time"

	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/common/srand"
	"github.com/polarismesh/polaris-server/common/timewheel"
	"github.com/polarismesh/polaris-server/common/utils"
	"github.com/polarismesh/polaris-server/plugin"
)

const (
	expireTtlCount = 3
)

// CheckScheduler schedule and run check actions
type CheckScheduler struct {
	rwMutex            *sync.RWMutex
	scheduledInstances map[string]*instanceValue

	timeWheel           *timewheel.TimeWheel
	minCheckIntervalSec int64
	maxCheckIntervalSec int64

	adoptInstancesChan chan AdoptEvent
	ctx                context.Context
}

// AdoptEvent
type AdoptEvent struct {
	InstanceId string
	Add        bool
	Checker    plugin.HealthChecker
}

type instanceValue struct {
	mutex               *sync.Mutex
	id                  string
	host                string
	port                uint32
	scheduled           uint32
	lastSetEventTimeSec int64
	ttlDurationSec      uint32
	expireDurationSec   uint32
	checker             plugin.HealthChecker
}

func (i *instanceValue) eventExpired() (int64, bool) {
	curTimeSec := time.Now().Unix()
	return curTimeSec, curTimeSec-i.lastSetEventTimeSec >= int64(i.expireDurationSec)
}

func newCheckScheduler(ctx context.Context, slotNum int,
	minCheckInterval time.Duration, maxCheckInterval time.Duration) *CheckScheduler {
	scheduler := &CheckScheduler{
		rwMutex:             &sync.RWMutex{},
		scheduledInstances:  make(map[string]*instanceValue),
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

	for range ctx.Done() {
		c.timeWheel.Stop()
		log.Infof("[Health Check][Check]timeWheel has been stopped")
		return
	}
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

func (c *CheckScheduler) putIfAbsent(instanceWithChecker *InstanceWithChecker) (bool, *instanceValue) {
	c.rwMutex.Lock()
	defer c.rwMutex.Unlock()
	instance := instanceWithChecker.instance
	var instValue *instanceValue
	var ok bool
	if instValue, ok = c.scheduledInstances[instance.ID()]; ok {
		return true, instValue
	}
	instValue = &instanceValue{
		mutex:             &sync.Mutex{},
		host:              instance.Host(),
		port:              instance.Port(),
		id:                instance.ID(),
		expireDurationSec: getExpireDurationSec(instance.Proto),
		checker:           instanceWithChecker.checker,
		ttlDurationSec:    instance.HealthCheck().GetHeartbeat().GetTtl().GetValue(),
	}
	c.scheduledInstances[instance.ID()] = instValue
	return false, instValue
}

func (c *CheckScheduler) getInstanceValue(instanceId string) (*instanceValue, bool) {
	c.rwMutex.RLock()
	defer c.rwMutex.RUnlock()
	value, ok := c.scheduledInstances[instanceId]
	return value, ok
}

// AddInstance add instance to check
func (c *CheckScheduler) AddInstance(instanceWithChecker *InstanceWithChecker) {
	exists, instValue := c.putIfAbsent(instanceWithChecker)
	if exists {
		return
	}
	c.addAdopting(instValue.id, instValue.checker)
	instance := instanceWithChecker.instance
	log.Infof("[Health Check][Check]add check instance is %s, host is %s:%d",
		instance.ID(), instance.Host(), instance.Port())
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

func (c *CheckScheduler) addHealthyCallback(instance *instanceValue, lastHeartbeatTimeSec int64) {
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
	log.Debugf("[Health Check][Check]add healthy callback, instance is %s:%d, id is %s, delay is %d(ms)",
		host, port, instanceId, delayMilli)
	c.timeWheel.AddTask(delayMilli, instanceId, c.checkCallback)
}

func (c *CheckScheduler) addUnHealthyCallback(instance *instanceValue) {
	delaySec := instance.expireDurationSec
	if c.maxCheckIntervalSec > 0 && int64(delaySec) > c.maxCheckIntervalSec {
		delaySec = uint32(c.maxCheckIntervalSec)
	}
	host := instance.host
	port := instance.port
	instanceId := instance.id
	delayMilli := delaySec*1000 + getRandDelayMilli()
	log.Debugf("[Health Check][Check]add unhealthy callback, instance is %s:%d, id is %s, delay is %d(ms)",
		host, port, instanceId, delayMilli)
	c.timeWheel.AddTask(delayMilli, instanceId, c.checkCallback)
}

func (c *CheckScheduler) checkCallback(value interface{}) {
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
		var code uint32
		if checkResp.Healthy {
			//from unhealthy to healthy
			log.Infof(
				"[Health Check][Check]instance change from unhealthy to healthy, id is %s, address is %s:%d",
				instanceValue.id, instanceValue.host, instanceValue.port)
			code = setInsDbStatus(cachedInstance, checkResp.Healthy)
		} else {
			//from healthy to unhealthy
			log.Infof(
				"[Health Check][Check]instance change from healthy to unhealthy, id is %s, address is %s:%d",
				instanceValue.id, instanceValue.host, instanceValue.port)
			code = setInsDbStatus(cachedInstance, checkResp.Healthy)
		}
		if code != api.ExecuteSuccess {
			log.Errorf("[Health Check][Check]fail to update instance, id is %s, address is %s:%d, code is %d",
				instanceValue.id, instanceValue.host, instanceValue.port, code)
		}
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
// 底层函数会合并create请求，增加并发创建的吞吐
// req 原始请求
// ins 包含了req数据与instanceID，serviceToken
func (s *Server) asyncSetInsDbStatus(ins *api.Instance, healthStatus bool) uint32 {
	future := s.bc.AsyncHeartbeat(ins, healthStatus)
	if err := future.Wait(); err != nil {
		log.Error(err.Error())
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
