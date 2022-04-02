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
	"runtime"

	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/plugin"
)

var DefaultShardSize uint32

func init() {
	DefaultShardSize = uint32(runtime.GOMAXPROCS(0) * 16)
	// Different machines can adjust this parameter of 16.In more cases, 16 is suitable ,
	// can test it in shardmap_test.go
}

// CacheProvider provider health check objects for service cache
type CacheProvider struct {
	healthCheckInstances *shardMap
	selfServiceInstances *shardMap
	selfService          string
}

// CacheEvent provides the event for cache changes
type CacheEvent struct {
	healthCheckInstancesChanged bool
	selfServiceInstancesChanged bool
}

func newCacheProvider(selfService string) *CacheProvider {
	return &CacheProvider{
		healthCheckInstances: NewShardMap(DefaultShardSize),
		selfServiceInstances: NewShardMap(1),
		selfService:          selfService,
	}
}

func (c *CacheProvider) isSelfServiceInstance(instance *api.Instance) bool {
	metadata := instance.GetMetadata()
	if svcName, ok := metadata[model.MetaKeyPolarisService]; ok {
		return svcName == c.selfService
	}
	return false
}

func (c *CacheProvider) sendEvent(event CacheEvent) {
	server.dispatcher.UpdateStatusByEvent(event)
}

func compareAndStoreServiceInstance(instanceWithChecker *InstanceWithChecker, values *shardMap) bool {
	instanceId := instanceWithChecker.instance.ID()
	value, isNew := values.PutIfAbsent(instanceId, instanceWithChecker)
	if value == nil {
		return false
	}
	if isNew {
		log.Infof("[Health Check][Cache]create service instance is %s:%d, id is %s",
			instanceWithChecker.instance.Host(), instanceWithChecker.instance.Port(),
			instanceId)
		return true
	}
	lastInstance := value.instance
	if lastInstance.Revision() == instanceWithChecker.instance.Revision() {
		return false
	}
	log.Infof("[Health Check][Cache]update service instance is %s:%d, id is %s",
		instanceWithChecker.instance.Host(), instanceWithChecker.instance.Port(), instanceId)
	// In the concurrent scenario, when the key and version are the same,
	// if they arrive here at the same time, they will be saved multiple times.
	values.Store(instanceId, instanceWithChecker)
	return true
}

func storeServiceInstance(instanceWithChecker *InstanceWithChecker, values *shardMap) bool {
	log.Infof("[Health Check][Cache]create service instance is %s:%d, id is %s",
		instanceWithChecker.instance.Host(), instanceWithChecker.instance.Port(),
		instanceWithChecker.instance.ID())
	instanceId := instanceWithChecker.instance.ID()
	values.Store(instanceId, instanceWithChecker)
	return true
}

func deleteServiceInstance(instance *api.Instance, values *shardMap) bool {
	instanceId := instance.GetId().GetValue()
	ok := values.DeleteIfExist(instanceId)
	if ok {
		log.Infof("[Health Check][Cache]delete service instance is %s:%d, id is %s",
			instance.GetHost().GetValue(), instance.GetPort().GetValue(), instanceId)
	}
	return true
}

// InstanceWithChecker instance and checker combine
type InstanceWithChecker struct {
	instance  *model.Instance
	checker   plugin.HealthChecker
	hashValue uint
}

func newInstanceWithChecker(instance *model.Instance, checker plugin.HealthChecker) *InstanceWithChecker {
	return &InstanceWithChecker{
		instance:  instance,
		checker:   checker,
		hashValue: hashString(instance.ID()),
	}
}

// OnCreated callback when cache value created
func (c *CacheProvider) OnCreated(value interface{}) {
	if instance, ok := value.(*model.Instance); ok {
		instProto := instance.Proto
		if c.isSelfServiceInstance(instProto) {
			storeServiceInstance(newInstanceWithChecker(instance, nil), c.selfServiceInstances)
			c.sendEvent(CacheEvent{selfServiceInstancesChanged: true})
			//return
		}
		hcEnable, checker := isHealthCheckEnable(instProto)
		if !hcEnable {
			return
		}
		storeServiceInstance(newInstanceWithChecker(instance, checker), c.healthCheckInstances)
		c.sendEvent(CacheEvent{healthCheckInstancesChanged: true})
	}
}

func isHealthCheckEnable(instance *api.Instance) (bool, plugin.HealthChecker) {
	if !instance.GetEnableHealthCheck().GetValue() || instance.GetHealthCheck() == nil {
		return false, nil
	}
	checker, ok := server.checkers[int32(instance.GetHealthCheck().GetType())]
	if !ok {
		return false, nil
	}
	return true, checker
}

// OnUpdated callback when cache value updated
func (c *CacheProvider) OnUpdated(value interface{}) {
	if instance, ok := value.(*model.Instance); ok {
		instProto := instance.Proto
		if c.isSelfServiceInstance(instProto) {
			if compareAndStoreServiceInstance(newInstanceWithChecker(instance, nil), c.selfServiceInstances) {
				c.sendEvent(CacheEvent{selfServiceInstancesChanged: true})
			}
			//return
		}
		// check exists
		instanceId := instance.ID()
		healthCheckInstanceValue, exists := c.healthCheckInstances.Load(instanceId)
		hcEnable, checker := isHealthCheckEnable(instProto)
		if !hcEnable {
			if !exists {
				// instance is unhealthy, not exist, just return.
				return
			}
			log.Infof("[Health Check][Cache]delete health check disabled instance is %s:%d, id is %s",
				instance.Host(), instance.Port(), instanceId)
			// instance is unhealthy, but exist, delete it.
			ok := c.healthCheckInstances.DeleteIfExist(instanceId)
			if ok {
				c.sendEvent(CacheEvent{healthCheckInstancesChanged: true})
			}
			return
		}
		var noChanged bool
		if exists {
			// instance is healthy, exists, consistent healthCheckInstance.Revision(), no need to change。
			healthCheckInstance := healthCheckInstanceValue.instance
			noChanged = healthCheckInstance.Revision() == instance.Revision()
		}
		if !noChanged {
			log.Infof("[Health Check][Cache]update service instance is %s:%d, id is %s",
				instance.Host(), instance.Port(), instanceId)
			//   In the concurrent scenario, when the healthCheckInstance.Revision() of the same health instance is the same,
			//   if it arrives here at the same time, it will be saved multiple times
			c.healthCheckInstances.Store(instanceId, newInstanceWithChecker(instance, checker))
			c.sendEvent(CacheEvent{healthCheckInstancesChanged: true})
		}
	}
}

// OnDeleted callback when cache value deleted
func (c *CacheProvider) OnDeleted(value interface{}) {
	if instance, ok := value.(*model.Instance); ok {
		instProto := instance.Proto
		if c.isSelfServiceInstance(instProto) {
			deleteServiceInstance(instProto, c.selfServiceInstances)
			c.sendEvent(CacheEvent{selfServiceInstancesChanged: true})
			//return
		}
		if !instProto.GetEnableHealthCheck().GetValue() || instProto.GetHealthCheck() == nil {
			return
		}
		deleteServiceInstance(instProto, c.healthCheckInstances)
		c.sendEvent(CacheEvent{healthCheckInstancesChanged: true})
	}
}

// OnBatchCreated callback when cache value created
func (c *CacheProvider) OnBatchCreated(value interface{}) {

}

// OnBatchUpdated callback when cache value updated
func (c *CacheProvider) OnBatchUpdated(value interface{}) {

}

// OnBatchDeleted callback when cache value deleted
func (c *CacheProvider) OnBatchDeleted(value interface{}) {

}

// RangeHealthCheckInstances range loop healthCheckInstances
func (c *CacheProvider) RangeHealthCheckInstances(check func(instance *InstanceWithChecker)) {

	c.healthCheckInstances.Range(func(instanceId string, healthCheckInstance *InstanceWithChecker) {
		check(healthCheckInstance)
	})
}

// RangeSelfServiceInstances range loop selfServiceInstances
func (c *CacheProvider) RangeSelfServiceInstances(check func(instance *api.Instance)) {

	c.selfServiceInstances.Range(func(instanceId string, selfServiceInstance *InstanceWithChecker) {
		check(selfServiceInstance.instance.Proto)
	})
}

// GetInstance get instance by id
func (c *CacheProvider) GetInstance(instanceId string) *model.Instance {
	value, ok := c.healthCheckInstances.Load(instanceId)
	if !ok {
		return nil
	}
	return value.instance
}
