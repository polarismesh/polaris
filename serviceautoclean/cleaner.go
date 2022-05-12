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

package serviceautoclean

import (
	"context"
	"fmt"
	"time"

	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/model"
	. "github.com/polarismesh/polaris-server/common/utils"
	"github.com/polarismesh/polaris-server/service"
)

const weight = 100

type ServiceAutoCleaner struct {
	config             *Config
	localHost          string
	namingServer       *service.Server
	selfServiceBuckets map[Bucket]bool
	continuum          *Continuum
	checkedServices    map[string]*ServiceCheckUnit
	ignoredNamespaces  map[string]bool
}

type ServiceCheckUnit struct {
	service    *model.Service
	checkCount int
}

func Start(ctx context.Context, config *Config, cacheOpen bool, namingServer *service.Server) error {
	if !config.Open {
		return nil
	}

	if !cacheOpen {
		return fmt.Errorf("[ServiceAutoClean]cache not open")
	}

	ignoredSet := make(map[string]bool)
	for _, v := range config.IgnoredNamespaces {
		ignoredSet[v] = true
	}

	cleaner := &ServiceAutoCleaner{
		config:            config,
		localHost:         config.LocalHost,
		namingServer:      namingServer,
		ignoredNamespaces: ignoredSet,
	}

	cleaner.start(ctx)
	return nil
}

func (c *ServiceAutoCleaner) start(ctx context.Context) {

	log.Infof("[ServiceAutoClean] Start service auto clean, config: %+v", c.config)
	go func() {
		eventTicker := time.NewTicker(c.config.CheckInterval)
		defer eventTicker.Stop()

		for {
			select {
			case <-eventTicker.C:
				c.reloadSelfContinuum()
				c.reloadManagedServices()
				c.check()
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (c *ServiceAutoCleaner) reloadSelfContinuum() bool {

	selfService := c.namingServer.Cache().Service().GetServiceByName(c.config.Service, c.config.Namespace)
	if selfService == nil {
		return false
	}
	instances := c.namingServer.Cache().Instance().GetInstancesByServiceID(selfService.ID)
	if instances == nil {
		return false
	}

	nextBuckets := make(map[Bucket]bool)
	for _, instance := range instances {
		if instance.Isolate() || !instance.Healthy() {
			continue
		}
		nextBuckets[Bucket{
			Host:   instance.Host(),
			Weight: weight,
		}] = true

	}

	originBucket := c.selfServiceBuckets
	log.Debugf("[ServiceAutoClean]reload continuum by %v, origin is %v", nextBuckets, originBucket)
	if CompareBuckets(originBucket, nextBuckets) {
		return false
	}
	c.selfServiceBuckets = nextBuckets
	c.continuum = New(c.selfServiceBuckets)
	return true
}

func (c *ServiceAutoCleaner) reloadManagedServices() {
	if c.continuum == nil {
		return
	}

	totalCount := 0
	nextService := make(map[string]*ServiceCheckUnit)
	serviceIterProc := func(key string, value *model.Service) (bool, error) {
		if _, ok := c.ignoredNamespaces[value.Namespace]; ok {
			return true, nil
		}

		totalCount++
		serviceId := value.ID
		host := c.continuum.Hash(HashString(serviceId))
		if host != c.localHost {
			return true, nil
		}

		preCount := 0
		if unit, ok := c.checkedServices[serviceId]; ok {
			preCount = unit.checkCount
		}

		nextService[serviceId] = &ServiceCheckUnit{
			service:    value,
			checkCount: preCount,
		}
		return true, nil
	}
	c.namingServer.Cache().Service().IteratorServices(serviceIterProc)

	log.Infof("[ServiceAutoClean]count %d servicess has been dispatched to %s, total is %d",
		len(nextService), c.localHost, totalCount)

	c.checkedServices = nextService
}

func (c *ServiceAutoCleaner) deleteService(svc *model.Service) {
	log.Infof("[ServiceAutoClean]delete service %s:%s", svc.Namespace, svc.Name)
	req := &api.Service{
		Name:      NewStringValue(svc.Name),
		Namespace: NewStringValue(svc.Namespace),
	}
	ctx := context.Background()
	resp := c.namingServer.DeleteService(ctx, req)
	if resp.GetCode().GetValue() != api.ExecuteSuccess {
		log.Warnf("[ServiceAutoClean]delete service %s:%s failed, error info: %s",
			svc.Namespace, svc.Name, resp.GetInfo().GetValue())
	}
}

func (c *ServiceAutoCleaner) check() {
	if c.checkedServices == nil {
		return
	}

	now := time.Now()
	for _, unit := range c.checkedServices {
		instanceCount := c.namingServer.Cache().Instance().GetInstancesCountByServiceID(unit.service.ID)

		if instanceCount.TotalInstanceCount == 0 &&
			now.After(unit.service.ModifyTime.Add(c.config.ExpireTime)) {
			if unit.checkCount >= c.config.CheckCountBeforeClean {
				c.deleteService(unit.service)
				unit.checkCount = 0
			} else {
				unit.checkCount++
			}
		} else {
			unit.checkCount = 0
		}
	}
}
