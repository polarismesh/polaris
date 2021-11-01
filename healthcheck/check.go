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
	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/common/srand"
	"github.com/polarismesh/polaris-server/common/timewheel"
	"github.com/polarismesh/polaris-server/common/utils"
	"github.com/polarismesh/polaris-server/plugin"
	"sync"
	"sync/atomic"
	"time"
)

const (
	NotHealthy = 0
	Healthy    = 1
)

const (
	expireTtlCount = 3
)

// CheckScheduler schedule and run check actions
type CheckScheduler struct {
	rwMutex            *sync.RWMutex
	scheduledInstances map[string]*instanceValue

	timeWheel *timewheel.TimeWheel
}

type instanceValue struct {
	mutex               *sync.Mutex
	id                  string
	host                string
	port                uint32
	scheduled           uint32
	lastSetEventTimeSec int64
	lastCheckTimeSec    int64
	ttlDurationSec      uint32
	expireDurationSec   int64
	checker             plugin.HealthChecker
}

func (i *instanceValue) eventExpired() (int64, bool) {
	curTimeSec := time.Now().Unix()
	return curTimeSec, curTimeSec-i.lastSetEventTimeSec >= i.expireDurationSec
}

func newCheckScheduler(ctx context.Context, slotNum int) *CheckScheduler {
	scheduler := &CheckScheduler{
		rwMutex:            &sync.RWMutex{},
		scheduledInstances: make(map[string]*instanceValue),
		timeWheel:          timewheel.New(time.Second, slotNum, "[Health Check]interval-check"),
	}
	go scheduler.startRoutines(ctx)
	return scheduler
}

func (c *CheckScheduler) startRoutines(ctx context.Context) {
	c.timeWheel.Start()
	log.Infof("[Health Check][Check]timeWheel has been started")
	for {
		select {
		case <-ctx.Done():
			c.timeWheel.Stop()
			log.Infof("[Health Check][Check]timeWheel has been stopped")
			return
		}
	}
}

func (c *CheckScheduler) putIfAbsent(instanceWithChecker *InstanceWithChecker) (bool, *instanceValue) {
	c.rwMutex.Lock()
	defer c.rwMutex.Unlock()
	instance := instanceWithChecker.instance
	var instValue *instanceValue
	var ok bool
	if instValue, ok = c.scheduledInstances[instance.GetId().GetValue()]; ok {
		return true, instValue
	}
	instValue = &instanceValue{
		mutex:             &sync.Mutex{},
		host:              instance.GetHost().GetValue(),
		port:              instance.GetPort().GetValue(),
		id:                instance.GetId().GetValue(),
		expireDurationSec: int64(getExpireDurationSec(instance)),
		checker:           instanceWithChecker.checker,
		ttlDurationSec:    instance.GetHealthCheck().GetHeartbeat().GetTtl().GetValue(),
	}
	c.scheduledInstances[instance.GetId().GetValue()] = instValue
	return false, instValue
}

func (c *CheckScheduler) checkExistsReadOnly(instance *instanceValue) bool {
	c.rwMutex.RLock()
	defer c.rwMutex.RUnlock()
	_, ok := c.scheduledInstances[instance.id]
	log.Debugf("[Health Check][Check]check exists ro for id is %s, host is %s:%d, result is %v",
		instance.id, instance.host, instance.port, ok)
	return ok
}

func (c *CheckScheduler) getInstanceValue(instanceId string) *instanceValue {
	c.rwMutex.RLock()
	defer c.rwMutex.RUnlock()
	return c.scheduledInstances[instanceId]
}

// AddInstance add instance to check
func (c *CheckScheduler) AddInstance(instanceWithChecker *InstanceWithChecker) {
	exists, instValue := c.putIfAbsent(instanceWithChecker)
	if exists {
		return
	}
	instance := instanceWithChecker.instance
	log.Debugf("[Health Check][Check]add for id is %s, host is %s:%d",
		instance.GetId().GetValue(), instance.GetHost().GetValue(), instance.GetPort().GetValue())
	c.addUnHealthyCallback(instValue)
}

func getExpireDurationSec(instance *api.Instance) uint32 {
	ttlValue := instance.GetHealthCheck().GetHeartbeat().GetTtl().GetValue()
	return expireTtlCount * ttlValue
}

func getRandDelayMilli() time.Duration {
	delayMilli := srand.Intn(1000)
	return time.Duration(delayMilli) * time.Millisecond
}

func (c *CheckScheduler) addHealthyCallback(instance *instanceValue, lastHeartbeatTimeSec int64) {
	delay := time.Duration(instance.expireDurationSec) * time.Second
	var nextDelay time.Duration
	if lastHeartbeatTimeSec > 0 {
		curTimeSec := currentTimeSec()
		timePassed := curTimeSec - lastHeartbeatTimeSec
		if timePassed > 0 {
			nextDelay = delay - time.Duration(timePassed)*time.Second - getRandDelayMilli()
		}
	}
	if nextDelay > 0 {
		delay = nextDelay
	}
	host := instance.host
	port := instance.port
	instanceId := instance.id
	log.Debugf("[Health Check][Check]add healthy callback, instance is %s:%d, id is %s, delay is %v",
		host, port, instanceId, delay)
	_ = c.timeWheel.AddTask(delay, instanceId, c.checkCallback)
}

func (c *CheckScheduler) addUnHealthyCallback(instance *instanceValue) {
	delay := time.Duration(instance.expireDurationSec-int64(instance.ttlDurationSec)) * time.Second
	delay = delay - getRandDelayMilli()
	host := instance.host
	port := instance.port
	instanceId := instance.id
	log.Debugf("[Health Check][Check]add first/unhealthy callback, instance is %s:%d, id is %s, delay is %v",
		host, port, instanceId, delay)
	_ = c.timeWheel.AddTask(delay, instanceId, c.checkCallback)
}

func (c *CheckScheduler) addInstCallback(value *instanceValue) {
	log.Debugf("[Health Check][Check]add instant callback, instance is %s:%d, id is %s",
		value.host, value.port, value.id)
	_ = c.timeWheel.AddTask(1*time.Second, value, c.instCheckCallback)
}

func (c *CheckScheduler) instCheckCallback(value interface{}) {
	instValue := value.(*instanceValue)
	c.checkCallback(instValue.id)
	atomic.StoreUint32(&instValue.scheduled, 0)
}

func (c *CheckScheduler) checkCallback(value interface{}) {
	instanceId := value.(string)
	instanceValue := c.getInstanceValue(instanceId)
	if nil == instanceValue {
		log.Infof("[Health Check][Check]instance %s has been deleted", instanceId)
		return
	}
	instanceValue.mutex.Lock()
	defer instanceValue.mutex.Unlock()
	curTimeSec := time.Now().Unix()

	if instanceValue.lastCheckTimeSec == curTimeSec {
		return
	}
	instanceValue.lastCheckTimeSec = curTimeSec

	log.Debugf("[Health Check][Check]start to check instance %s:%d, id is %s",
		instanceValue.host, instanceValue.port, instanceValue.id)
	cachedInstance := server.cacheProvider.GetInstance(instanceValue.id)
	request := &plugin.CheckRequest{
		QueryRequest: plugin.QueryRequest{
			InstanceId: instanceValue.id,
			Host:       instanceValue.host,
			Port:       instanceValue.port,
			Healthy:    cachedInstance.GetHealthy().GetValue(),
		},
		CurTimeSec:        currentTimeSec(),
		ExpireDurationSec: uint32(instanceValue.expireDurationSec),
	}
	checkResp, err := instanceValue.checker.Check(request)
	if nil != err {
		log.Errorf("[Health Check][Check]fail to check instance %s:%d, id is %s, err is %v",
			instanceValue.host, instanceValue.port, instanceValue.id, err)
		if c.checkExistsReadOnly(instanceValue) {
			c.addUnHealthyCallback(instanceValue)
		}
		return
	}
	if checkResp.Healthy && !cachedInstance.GetHealthy().GetValue() {
		//from unhealthy to healthy
		log.Infof("[Health Check][Check]instance change from unhealthy to healthy, id is %s, address is %s:%d",
			instanceValue.id, instanceValue.host, instanceValue.port)
		err = setInsDbStatus(cachedInstance, Healthy)
	}
	if !checkResp.Healthy && cachedInstance.GetHealthy().GetValue() && !checkResp.OnRecover {
		//from healthy to unhealthy
		log.Infof("[Health Check][Check]instance change from healthy to unhealthy, id is %s, address is %s:%d",
			instanceValue.id, instanceValue.host, instanceValue.port)
		err = setInsDbStatus(cachedInstance, NotHealthy)
	}
	if nil != err {
		log.Errorf("[Health Check][Check]fail to update instance, id is %s, address is %s:%d, err is %v",
			instanceValue.id, instanceValue.host, instanceValue.port, err)
		if c.checkExistsReadOnly(instanceValue) {
			c.addHealthyCallback(instanceValue, checkResp.LastHeartbeatTimeSec)
		}
		return
	}
	if c.checkExistsReadOnly(instanceValue) &&
		(checkResp.Healthy || (checkResp.OnRecover && cachedInstance.GetHealthy().GetValue())) {
		c.addHealthyCallback(instanceValue, checkResp.LastHeartbeatTimeSec)
	} else {
		c.addUnHealthyCallback(instanceValue)
	}
}

// DelInstance del instance from check
func (c *CheckScheduler) DelInstance(instanceWithChecker *InstanceWithChecker) {
	c.rwMutex.Lock()
	defer c.rwMutex.Unlock()
	instance := instanceWithChecker.instance
	log.Infof("[Health Check][Check]remove check instance is %s:%d, id is %s",
		instance.GetHost().GetValue(), instance.GetPort().GetValue(), instance.GetId().GetValue())
	delete(c.scheduledInstances, instance.GetId().GetValue())
}

// setInsDbStatus 修改实例状态, 需要打印操作记录
func setInsDbStatus(instance *api.Instance, status int) error {
	id := instance.GetId().GetValue()
	host := instance.GetHost().GetValue()
	port := instance.GetPort().GetValue()
	log.Infof("[Health Check][Check]addr:%s:%d id:%s set db status %d", host, port, id, status)
	err := server.storage.SetInstanceHealthStatus(id, status, utils.NewUUID())
	if err != nil {
		log.Errorf("[Health Check][Check]id: %s set db status err:%s", id, err)
		return err
	}
	healthStatus := true
	if status == 0 {
		healthStatus = false
	}
	recordInstance := &model.Instance{
		Proto: &api.Instance{
			Host:     instance.GetHost(),
			Port:     instance.GetPort(),
			Priority: instance.GetPriority(),
			Weight:   instance.GetWeight(),
			Healthy:  utils.NewBoolValue(healthStatus),
			Isolate:  instance.GetIsolate(),
		},
	}

	server.RecordHistory(instanceRecordEntry(recordInstance, model.OUpdate))
	return nil
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
