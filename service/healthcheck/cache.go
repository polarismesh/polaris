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
	"runtime"

	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"

	"github.com/polarismesh/polaris/common/eventhub"
	"github.com/polarismesh/polaris/common/hash"
	commonhash "github.com/polarismesh/polaris/common/hash"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/plugin"
)

var DefaultShardSize uint32

func init() {
	DefaultShardSize = uint32(runtime.GOMAXPROCS(0) * 16)
	// Different machines can adjust this parameter of 16.In more cases, 16 is suitable ,
	// can test it in shardmap_test.go
}

// CacheProvider provider health check objects for service cache
type CacheProvider struct {
	svr                  *Server
	selfService          string
	selfServiceInstances *utils.SegmentMap[string, ItemWithChecker]
	healthCheckInstances *utils.SegmentMap[string, ItemWithChecker]
	healthCheckClients   *utils.SegmentMap[string, ItemWithChecker]
}

// CacheEvent provides the event for cache changes
type CacheEvent struct {
	healthCheckInstancesChanged bool
	selfServiceInstancesChanged bool
	healthCheckClientChanged    bool
}

func newCacheProvider(selfService string, svr *Server) *CacheProvider {
	return &CacheProvider{
		svr:                  svr,
		selfService:          selfService,
		selfServiceInstances: utils.NewSegmentMap[string, ItemWithChecker](1, hash.Fnv32),
		healthCheckInstances: utils.NewSegmentMap[string, ItemWithChecker](int(DefaultShardSize), hash.Fnv32),
		healthCheckClients:   utils.NewSegmentMap[string, ItemWithChecker](int(DefaultShardSize), hash.Fnv32),
	}
}

func (c *CacheProvider) isSelfServiceInstance(instance *apiservice.Instance) bool {
	metadata := instance.GetMetadata()
	if svcName, ok := metadata[model.MetaKeyPolarisService]; ok {
		return svcName == c.selfService
	}
	return false
}

func (c *CacheProvider) handleInstanceCacheEvent(ctx context.Context, args interface{}) error {
	event, ok := args.(*eventhub.CacheInstanceEvent)
	if !ok {
		return nil
	}
	switch event.EventType {
	case eventhub.EventCreated:
		c.OnCreated(event.Instance)
	case eventhub.EventUpdated:
		c.OnUpdated(event.Instance)
	case eventhub.EventDeleted:
		c.OnDeleted(event.Instance)
	}
	return nil
}

func (c *CacheProvider) handleClientCacheEvent(ctx context.Context, args interface{}) error {
	event, ok := args.(*eventhub.CacheClientEvent)
	if !ok {
		return nil
	}
	switch event.EventType {
	case eventhub.EventCreated:
		c.OnCreated(event.Client)
	case eventhub.EventUpdated:
		c.OnUpdated(event.Client)
	case eventhub.EventDeleted:
		c.OnDeleted(event.Client)
	}
	return nil
}

// OnCreated callback when cache value created
func (c *CacheProvider) OnCreated(value interface{}) {
	switch actual := value.(type) {
	case *model.Instance:
		instProto := actual.Proto
		if c.isSelfServiceInstance(instProto) {
			storeServiceInstance(newInstanceWithChecker(actual, nil), c.selfServiceInstances)
			c.sendEvent(CacheEvent{selfServiceInstancesChanged: true})
			return
		}
		hcEnable, checker := c.isHealthCheckEnable(instProto)
		if !hcEnable {
			return
		}
		storeServiceInstance(newInstanceWithChecker(actual, checker), c.healthCheckInstances)
		c.sendEvent(CacheEvent{healthCheckInstancesChanged: true})
	case *model.Client:
		checker, ok := c.getHealthChecker(apiservice.HealthCheck_HEARTBEAT)
		if !ok {
			return
		}
		storeClient(newClientWithChecker(actual, checker), c.healthCheckClients)
		c.sendEvent(CacheEvent{healthCheckClientChanged: true})
	}
}

func (c *CacheProvider) getHealthChecker(hcType apiservice.HealthCheck_HealthCheckType) (plugin.HealthChecker, bool) {
	checker, ok := c.svr.checkers[int32(hcType)]
	return checker, ok
}

func (c *CacheProvider) isHealthCheckEnable(instance *apiservice.Instance) (bool, plugin.HealthChecker) {
	if !instance.GetEnableHealthCheck().GetValue() || instance.GetHealthCheck() == nil {
		return false, nil
	}
	checker, ok := c.getHealthChecker(instance.GetHealthCheck().GetType())
	if !ok {
		return false, nil
	}
	return true, checker
}

// OnUpdated callback when cache value updated
func (c *CacheProvider) OnUpdated(value interface{}) {
	switch actual := value.(type) {
	case *model.Instance:
		instProto := actual.Proto
		if c.isSelfServiceInstance(instProto) {
			if compareAndStoreServiceInstance(newInstanceWithChecker(actual, nil), c.selfServiceInstances) {
				c.sendEvent(CacheEvent{selfServiceInstancesChanged: true})
			}
			return
		}
		// check exists
		instanceId := actual.ID()
		value, exists := c.healthCheckInstances.Get(instanceId)
		hcEnable, checker := c.isHealthCheckEnable(instProto)
		if !hcEnable {
			if !exists {
				// instance is unhealthy, not exist, just return.
				return
			}
			log.Infof("[Health Check][Cache]delete health check disabled instance is %s:%d, id is %s",
				actual.Host(), actual.Port(), instanceId)
			// instance is unhealthy, but exist, delete it.
			ok := c.healthCheckInstances.Del(instanceId)
			if ok {
				c.sendEvent(CacheEvent{healthCheckInstancesChanged: true})
			}
			return
		}
		var noChanged bool
		if exists {
			// instance is healthy, exists, consistent healthCheckInstance.Revision(), no need to change。
			healthCheckInstance := value.GetInstance()
			noChanged = healthCheckInstance.Revision() == actual.Revision()
		}
		if !noChanged {
			log.Infof("[Health Check][Cache]update service instance is %s:%d, id is %s",
				actual.Host(), actual.Port(), instanceId)
			//   In the concurrent scenario, when the healthCheckInstance.Revision() of the same health instance is the same,
			//   if it arrives here at the same time, it will be saved multiple times
			c.healthCheckInstances.Put(instanceId, newInstanceWithChecker(actual, checker))
			c.sendEvent(CacheEvent{healthCheckInstancesChanged: true})
		}
	case *model.Client:
		checker, ok := c.getHealthChecker(apiservice.HealthCheck_HEARTBEAT)
		if !ok {
			return
		}
		if compareAndStoreClient(newClientWithChecker(actual, checker), c.healthCheckClients) {
			c.sendEvent(CacheEvent{healthCheckClientChanged: true})
		}
	}
}

// OnDeleted callback when cache value deleted
func (c *CacheProvider) OnDeleted(value interface{}) {
	switch actual := value.(type) {
	case *model.Instance:
		instProto := actual.Proto
		if c.isSelfServiceInstance(instProto) {
			deleteServiceInstance(instProto, c.selfServiceInstances)
			c.sendEvent(CacheEvent{selfServiceInstancesChanged: true})
			return
		}
		if !instProto.GetEnableHealthCheck().GetValue() || instProto.GetHealthCheck() == nil {
			return
		}
		deleteServiceInstance(instProto, c.healthCheckInstances)
		c.sendEvent(CacheEvent{healthCheckInstancesChanged: true})
	case *model.Client:
		deleteClient(actual.Proto(), c.healthCheckClients)
		c.sendEvent(CacheEvent{healthCheckClientChanged: true})
	}
}

// RangeHealthCheckInstances range loop values
func (c *CacheProvider) RangeHealthCheckInstances(check func(itemChecker ItemWithChecker, ins *model.Instance)) {
	c.healthCheckInstances.Range(func(instanceId string, value ItemWithChecker) {
		check(value, value.GetInstance())
	})
}

// RangeHealthCheckClients range loop values
func (c *CacheProvider) RangeHealthCheckClients(check func(itemChecker ItemWithChecker, client *model.Client)) {
	c.healthCheckClients.Range(func(instanceId string, value ItemWithChecker) {
		check(value, value.GetClient())
	})
}

// RangeSelfServiceInstances range loop selfServiceInstances
func (c *CacheProvider) RangeSelfServiceInstances(check func(instance *apiservice.Instance)) {
	c.selfServiceInstances.Range(func(instanceId string, value ItemWithChecker) {
		check(value.GetInstance().Proto)
	})
}

// GetSelfServiceInstance get self service instance by id
func (c *CacheProvider) GetSelfServiceInstance(instanceId string) *model.Instance {
	value, ok := c.selfServiceInstances.Get(instanceId)
	if !ok {
		return nil
	}

	ins := value.GetInstance()
	if ins == nil {
		return nil
	}
	return ins
}

// GetInstance get instance by id
func (c *CacheProvider) GetInstance(instanceId string) *model.Instance {
	value, ok := c.healthCheckInstances.Get(instanceId)
	if !ok {
		return nil
	}

	ins := value.GetInstance()
	if ins == nil {
		return nil
	}
	return ins
}

// GetInstance get instance by id
func (c *CacheProvider) GetClient(clientId string) *model.Client {
	value, ok := c.healthCheckClients.Get(clientId)
	if !ok {
		return nil
	}

	client := value.GetClient()
	if client == nil {
		return nil
	}
	return client
}

func (c *CacheProvider) sendEvent(event CacheEvent) {
	c.svr.dispatcher.UpdateStatusByEvent(event)
}

func compareAndStoreServiceInstance(instanceWithChecker *InstanceWithChecker,
	values *utils.SegmentMap[string, ItemWithChecker]) bool {
	instanceId := instanceWithChecker.instance.ID()
	value, isNew := values.PutIfAbsent(instanceId, instanceWithChecker)
	if isNew {
		log.Infof("[Health Check][Cache]create service instance is %s:%d, id is %s",
			instanceWithChecker.instance.Host(), instanceWithChecker.instance.Port(),
			instanceId)
		return true
	}
	instanceValue := value.(*InstanceWithChecker)
	lastInstance := instanceValue.instance
	if lastInstance.Revision() == instanceWithChecker.instance.Revision() {
		return false
	}
	log.Infof("[Health Check][Cache]update service instance is %s:%d, id is %s",
		instanceWithChecker.instance.Host(), instanceWithChecker.instance.Port(), instanceId)
	// In the concurrent scenario, when the key and version are the same,
	// if they arrive here at the same time, they will be saved multiple times.
	values.Put(instanceId, instanceWithChecker)
	return true
}

func storeServiceInstance(instanceWithChecker *InstanceWithChecker,
	values *utils.SegmentMap[string, ItemWithChecker]) bool {
	log.Infof("[Health Check][Cache]create service instance is %s:%d, id is %s",
		instanceWithChecker.instance.Host(), instanceWithChecker.instance.Port(),
		instanceWithChecker.instance.ID())
	instanceId := instanceWithChecker.instance.ID()
	values.Put(instanceId, instanceWithChecker)
	return true
}

func deleteServiceInstance(instance *apiservice.Instance, values *utils.SegmentMap[string, ItemWithChecker]) bool {
	instanceId := instance.GetId().GetValue()
	ok := values.Del(instanceId)
	if ok {
		log.Infof("[Health Check][Cache]delete service instance is %s:%d, id is %s",
			instance.GetHost().GetValue(), instance.GetPort().GetValue(), instanceId)
	}
	return true
}

func compareAndStoreClient(clientWithChecker *ClientWithChecker,
	values *utils.SegmentMap[string, ItemWithChecker]) bool {
	clientId := clientWithChecker.client.Proto().GetId().GetValue()
	_, isNew := values.PutIfAbsent(clientId, clientWithChecker)
	if isNew {
		log.Infof("[Health Check][Cache]create client is %s, id is %s",
			clientWithChecker.client.Proto().GetHost().GetValue(), clientId)
		return true
	}
	return false
}

func storeClient(clientWithChecker *ClientWithChecker,
	values *utils.SegmentMap[string, ItemWithChecker]) bool {

	log.Infof("[Health Check][Cache]create client is %s, id is %s",
		clientWithChecker.client.Proto().GetHost().GetValue(), clientWithChecker.client.Proto().GetId().GetValue())
	clientId := clientWithChecker.client.Proto().GetId().GetValue()
	values.Put(clientId, clientWithChecker)
	return true
}

func deleteClient(client *apiservice.Client, values *utils.SegmentMap[string, ItemWithChecker]) bool {
	clientId := client.GetId().GetValue()
	ok := values.Del(clientId)
	if ok {
		log.Infof("[Health Check][Cache]delete client is %s, id is %s",
			client.GetHost().GetValue(), clientId)
	}
	return true
}

// ItemWithChecker item and checker combine
// GetInstance 与 GetClient 互斥
type ItemWithChecker interface {
	// GetInstance 获取服务实例
	GetInstance() *model.Instance
	// GetClient 获取上报客户端信息
	GetClient() *model.Client
	// GetChecker 获取对应的 checker 对象
	GetChecker() plugin.HealthChecker
	// GetHashValue 获取 hashvalue 信息
	GetHashValue() uint
}

// InstanceWithChecker instance and checker combine
type InstanceWithChecker struct {
	instance  *model.Instance
	checker   plugin.HealthChecker
	hashValue uint
}

// GetInstance 获取服务实例
func (ic *InstanceWithChecker) GetInstance() *model.Instance {
	return ic.instance
}

// GetClient 获取上报客户端信息
func (ic *InstanceWithChecker) GetClient() *model.Client {
	return nil
}

// GetChecker 获取对应的 checker 对象
func (ic *InstanceWithChecker) GetChecker() plugin.HealthChecker {
	return ic.checker
}

// GetHashValue 获取 hashvalue 信息
func (ic *InstanceWithChecker) GetHashValue() uint {
	return ic.hashValue
}

func newInstanceWithChecker(instance *model.Instance, checker plugin.HealthChecker) *InstanceWithChecker {
	return &InstanceWithChecker{
		instance:  instance,
		checker:   checker,
		hashValue: commonhash.HashString(instance.ID()),
	}
}

// ClientWithChecker instance and checker combine
type ClientWithChecker struct {
	client    *model.Client
	checker   plugin.HealthChecker
	hashValue uint
}

// GetInstance 获取服务实例
func (ic *ClientWithChecker) GetInstance() *model.Instance {
	return nil
}

// GetClient 获取上报客户端信息
func (ic *ClientWithChecker) GetClient() *model.Client {
	return ic.client
}

// GetChecker 获取对应的 checker 对象
func (ic *ClientWithChecker) GetChecker() plugin.HealthChecker {
	return ic.checker
}

// GetHashValue 获取 hashvalue 信息
func (ic *ClientWithChecker) GetHashValue() uint {
	return ic.hashValue
}

func newClientWithChecker(client *model.Client, checker plugin.HealthChecker) *ClientWithChecker {
	return &ClientWithChecker{
		client:    client,
		checker:   checker,
		hashValue: commonhash.HashString(client.Proto().GetId().GetValue()),
	}
}
