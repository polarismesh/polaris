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
	"sync"
	"sync/atomic"
	"time"

	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"

	commonhash "github.com/polarismesh/polaris/common/hash"
	"github.com/polarismesh/polaris/common/model"
)

const (
	// eventInterval, trigger after instance change event
	eventInterval = 5 * time.Second
	// ensureInterval, trigger when timeout
	ensureInterval = 61 * time.Second
)

// Dispatcher dispatch all instances using consistent hash ring
type Dispatcher struct {
	svr *Server

	healthCheckInstancesChanged uint32
	healthCheckClientsChanged   uint32
	selfServiceInstancesChanged uint32
	managedInstances            map[string]*InstanceWithChecker
	managedClients              map[string]*ClientWithChecker

	selfServiceBuckets map[commonhash.Bucket]bool
	continuum          *commonhash.Continuum
	mutex              *sync.Mutex

	noAvailableServers bool
}

func newDispatcher(ctx context.Context, svr *Server) *Dispatcher {
	dispatcher := &Dispatcher{
		svr:   svr,
		mutex: &sync.Mutex{},
	}
	return dispatcher
}

// UpdateStatusByEvent 更新变更状态
func (d *Dispatcher) UpdateStatusByEvent(event CacheEvent) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	if event.selfServiceInstancesChanged {
		atomic.StoreUint32(&d.selfServiceInstancesChanged, 1)
	}
	if event.healthCheckInstancesChanged {
		atomic.StoreUint32(&d.healthCheckInstancesChanged, 1)
	}
	if event.healthCheckClientChanged {
		atomic.StoreUint32(&d.healthCheckClientsChanged, 1)
	}
}

// startDispatchingJob start job to dispatch instances
func (d *Dispatcher) startDispatchingJob(ctx context.Context) {
	go func() {
		eventTicker := time.NewTicker(eventInterval)
		defer eventTicker.Stop()
		ensureTicker := time.NewTicker(ensureInterval)
		defer ensureTicker.Stop()

		for {
			select {
			case <-eventTicker.C:
				d.processEvent()
			case <-ensureTicker.C:
				d.processEnsure()
			case <-ctx.Done():
				return
			}
		}
	}()
}

const weight = 100

func compareBuckets(src map[commonhash.Bucket]bool, dst map[commonhash.Bucket]bool) bool {
	if len(src) != len(dst) {
		return false
	}
	if len(src) == 0 {
		return false
	}
	for bucket := range dst {
		if _, ok := src[bucket]; !ok {
			return false
		}
	}
	return true
}

func (d *Dispatcher) reloadSelfContinuum() bool {
	nextBuckets := make(map[commonhash.Bucket]bool)
	d.svr.cacheProvider.RangeSelfServiceInstances(func(instance *apiservice.Instance) {
		if instance.GetIsolate().GetValue() || !instance.GetHealthy().GetValue() {
			return
		}
		nextBuckets[commonhash.Bucket{
			Host:   instance.GetHost().GetValue(),
			Weight: weight,
		}] = true
	})
	if len(nextBuckets) == 0 {
		d.noAvailableServers = true
	}
	originBucket := d.selfServiceBuckets
	log.Debugf("[Health Check][Dispatcher]reload continuum by %v, origin is %v", nextBuckets, originBucket)
	if compareBuckets(originBucket, nextBuckets) {
		return false
	}
	if d.noAvailableServers && len(nextBuckets) > 0 {
		// no available buckets, we need to suspend all the checkers
		for _, checker := range d.svr.checkers {
			checker.Suspend()
		}
		d.noAvailableServers = false
	}
	d.selfServiceBuckets = nextBuckets
	d.continuum = commonhash.New(d.selfServiceBuckets)
	return true
}

func (d *Dispatcher) reloadManagedClients() {
	nextClients := make(map[string]*ClientWithChecker)

	if d.continuum != nil {
		d.svr.cacheProvider.RangeHealthCheckClients(func(itemChecker ItemWithChecker, client *model.Client) {
			clientId := client.Proto().GetId().GetValue()
			host := d.continuum.Hash(itemChecker.GetHashValue())
			if host == d.svr.localHost {
				nextClients[clientId] = itemChecker.(*ClientWithChecker)
			}
		})
	}
	log.Infof("[Health Check][Dispatcher]count %d clients has been dispatched to %s, total is %d",
		len(nextClients), d.svr.localHost, d.svr.cacheProvider.healthCheckClients.Count())
	originClients := d.managedClients
	d.managedClients = nextClients
	if len(nextClients) > 0 {
		for id, client := range nextClients {
			if len(originClients) == 0 {
				d.svr.checkScheduler.AddClient(client)
				continue
			}
			if _, ok := originClients[id]; !ok {
				d.svr.checkScheduler.AddClient(client)
			}
		}
	}
	if len(originClients) > 0 {
		for id, client := range originClients {
			if len(nextClients) == 0 {
				d.svr.checkScheduler.DelClient(client)
				continue
			}
			if _, ok := nextClients[id]; !ok {
				d.svr.checkScheduler.DelClient(client)
			}
		}
	}
}

func (d *Dispatcher) reloadManagedInstances() {
	nextInstances := make(map[string]*InstanceWithChecker)
	if d.continuum != nil {
		d.svr.cacheProvider.RangeHealthCheckInstances(func(itemChecker ItemWithChecker, instance *model.Instance) {
			instanceId := instance.ID()
			host := d.continuum.Hash(itemChecker.GetHashValue())
			if host == d.svr.localHost {
				nextInstances[instanceId] = itemChecker.(*InstanceWithChecker)
			}
		})
	}
	log.Infof("[Health Check][Dispatcher]count %d instances has been dispatched to %s, total is %d",
		len(nextInstances), d.svr.localHost, d.svr.cacheProvider.healthCheckInstances.Count())
	originInstances := d.managedInstances
	d.managedInstances = nextInstances
	if len(nextInstances) > 0 {
		for _, instance := range nextInstances {
			d.svr.checkScheduler.UpsertInstance(instance)
		}
	}
	if len(originInstances) > 0 {
		for id, instance := range originInstances {
			if len(nextInstances) == 0 {
				d.svr.checkScheduler.DelInstance(instance)
				continue
			}
			if _, ok := nextInstances[id]; !ok {
				d.svr.checkScheduler.DelInstance(instance)
			}
		}
	}
}

func (d *Dispatcher) processEvent() {
	var selfContinuumReloaded bool
	if atomic.CompareAndSwapUint32(&d.selfServiceInstancesChanged, 1, 0) {
		selfContinuumReloaded = d.reloadSelfContinuum()
	}
	if selfContinuumReloaded || atomic.CompareAndSwapUint32(&d.healthCheckInstancesChanged, 1, 0) {
		d.reloadManagedInstances()
	}
	if selfContinuumReloaded || atomic.CompareAndSwapUint32(&d.healthCheckClientsChanged, 1, 0) {
		d.reloadManagedClients()
	}
}

func (d *Dispatcher) processEnsure() {
	d.reloadSelfContinuum()
	d.reloadManagedInstances()
	d.reloadManagedClients()
}
