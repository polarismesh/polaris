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

package core

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"

	nacosmodel "github.com/polarismesh/polaris/apiserver/nacosserver/model"
	cachetypes "github.com/polarismesh/polaris/cache/api"
	"github.com/polarismesh/polaris/common/eventhub"
	"github.com/polarismesh/polaris/common/model"
	commontime "github.com/polarismesh/polaris/common/time"
)

type (
	// FilterContext nacos 实例列表过滤上下文
	FilterContext struct {
		Service      *nacosmodel.ServiceMetadata
		Clusters     []string
		EnableOnly   bool
		HealthyOnly  bool
		SubscriberIP string
	}

	// InstanceFilter 实例过滤器
	InstanceFilter func(ctx *FilterContext, svcInfo *nacosmodel.ServiceInfo,
		ins []*nacosmodel.Instance, healthyCount int32) *nacosmodel.ServiceInfo
)

func NewNacosDataStorage(cacheMgr cachetypes.CacheManager) *NacosDataStorage {
	ctx, cancel := context.WithCancel(context.Background())
	notifier, notifierFinish := context.WithCancel(context.Background())
	store := &NacosDataStorage{
		cacheMgr:       cacheMgr,
		ctx:            ctx,
		cancel:         cancel,
		notifier:       notifier,
		notifierFinish: notifierFinish,
		namespaces:     map[string]map[string]*ServiceData{},
		revisions:      map[string]string{},
	}
	return store
}

// NacosDataStorage .
type NacosDataStorage struct {
	cacheMgr cachetypes.CacheManager
	ctx      context.Context
	cancel   context.CancelFunc

	triggeried     int32
	singleflight   singleflight.Group
	notifier       context.Context
	notifierFinish context.CancelFunc

	lock sync.RWMutex
	// namespace -> group+service -> *ServiceData
	namespaces map[string]map[string]*ServiceData
	revisions  map[string]string
}

func (n *NacosDataStorage) Cache() cachetypes.CacheManager {
	return n.cacheMgr
}

// ListInstances list nacos instances by filter
func (n *NacosDataStorage) ListInstances(filterCtx *FilterContext, filter InstanceFilter) *nacosmodel.ServiceInfo {
	// 必须等到第一次 syncData 动作任务完成
	if atomic.CompareAndSwapInt32(&n.triggeried, 0, 1) {
		go n.RunSync(n.ctx)
	}
	<-n.notifier.Done()

	n.lock.RLock()
	defer n.lock.RUnlock()

	svc := filterCtx.Service
	filterCtx.Service.Namespace = nacosmodel.ToNacosNamespace(svc.Namespace)
	clusters := filterCtx.Clusters

	services, ok := n.namespaces[svc.Namespace]
	if !ok {
		return nacosmodel.NewEmptyServiceInfo(svc.Name, svc.Group)
	}
	svcInfo, ok := services[svc.ServiceKey.String()]
	if !ok {
		return nacosmodel.NewEmptyServiceInfo(svc.Name, svc.Group)
	}

	clusterSet := make(map[string]struct{})
	for i := range clusters {
		if clusters[i] == "" {
			continue
		}
		clusterSet[clusters[i]] = struct{}{}
	}
	hasClusterSet := len(clusterSet) != 0

	ret := make([]*nacosmodel.Instance, 0, 32)

	svcInfo.lock.RLock()
	defer svcInfo.lock.RUnlock()

	resultInfo := &nacosmodel.ServiceInfo{
		Namespace:                svc.Namespace,
		CacheMillis:              1000,
		Name:                     svc.Name,
		GroupName:                svc.Group,
		Clusters:                 strings.Join(clusters, ","),
		Checksum:                 svcInfo.reversion,
		LastRefTime:              commontime.CurrentMillisecond(),
		ReachProtectionThreshold: false,
	}

	healthCount := int32(0)
	for i := range svcInfo.instances {
		ins := svcInfo.instances[i]
		if filterCtx.EnableOnly && !ins.Enabled {
			continue
		}
		if hasClusterSet {
			if _, ok := clusterSet[ins.ClusterName]; !ok {
				continue
			}
		}
		if ins.Healthy {
			healthCount++
		}
		ret = append(ret, ins)
	}

	resultInfo.Hosts = ret
	if filter == nil {
		return resultInfo
	}
	return filter(filterCtx, resultInfo, ret, healthCount)
}

// RunSync .
func (n *NacosDataStorage) RunSync(ctx context.Context) {
	n.realSync()
	ticker := time.NewTicker(time.Second)
	defer func() {
		ticker.Stop()
	}()
	for {
		select {
		case <-ctx.Done():
			nacoslog.Info("[NACOS-V2][Cache] stop data sync task")
			return
		case <-ticker.C:
			n.realSync()
		}
	}
}

func (n *NacosDataStorage) realSync() {
	defer func() {
		if err := recover(); err != nil {
			nacoslog.Error("[NACOS-V2][Cache] run sync occur panic", zap.Any("error", err))
		}
		n.notifierFinish()
	}()
	_, _, _ = n.singleflight.Do("NacosDataStorage", func() (interface{}, error) {
		n.syncTask()
		return nil, nil
	})
}

func (n *NacosDataStorage) syncTask() {
	// 定期将服务数据转为 Nacos 的服务数据缓存
	nsList := n.cacheMgr.Namespace().GetNamespaceList()
	svcInfos := make([]*nacosmodel.ServiceMetadata, 0, 8)

	// 计算需要 refresh 的服务信息列表
	for _, ns := range nsList {
		_, svcs := n.cacheMgr.Service().ListServices(ns.Name)
		for _, svc := range svcs {
			revision := n.cacheMgr.Service().GetRevisionWorker().GetServiceInstanceRevision(svc.ID)
			oldRevision, ok := n.revisions[svc.ID]
			if !ok || revision != oldRevision {
				nacoslog.Info("[NACOS-V2][Cache] service reversion update",
					zap.String("namespace", svc.Namespace), zap.String("service", svc.Name),
					zap.String("old-reversion", oldRevision), zap.String("reversion", revision))
				svcData := n.loadNacosService(revision, svc)
				svcInfos = append(svcInfos, svcData.specService)
				instances := n.cacheMgr.Instance().GetInstances(svc.ID)
				svcData.loadInstances(instances)
			}
			n.revisions[svc.ID] = revision
		}
	}

	if len(svcInfos) == 0 {
		return
	}
	// 发布服务信息变更事件
	_ = eventhub.Publish(nacosmodel.NacosServicesChangeEventTopic, &nacosmodel.NacosServicesChangeEvent{
		Services: svcInfos,
	})
}

func (n *NacosDataStorage) loadNacosService(reversion string, svc *model.Service) *ServiceData {
	n.lock.Lock()
	defer n.lock.Unlock()

	nacosNs := nacosmodel.ToNacosNamespace(svc.Namespace)

	if _, ok := n.namespaces[nacosNs]; !ok {
		n.namespaces[nacosNs] = map[string]*ServiceData{}
	}
	services := n.namespaces[nacosNs]

	key := nacosmodel.ServiceKey{
		Namespace: nacosNs,
		Group:     nacosmodel.GetGroupName(svc.Name),
		Name:      nacosmodel.GetServiceName(svc.Name),
	}
	if val, ok := services[key.String()]; ok {
		val.lock.Lock()
		val.reversion = reversion
		val.lock.Unlock()
		return val
	}

	ret := &ServiceData{
		specService: &nacosmodel.ServiceMetadata{
			ServiceKey:          key,
			ServiceID:           svc.ID,
			ProtectionThreshold: 0.0,
			ExtendData:          svc.Meta,
		},
		name:      key.Name,
		group:     key.Group,
		reversion: reversion,
		instances: map[string]*nacosmodel.Instance{},
	}
	if val, ok := ret.specService.ExtendData[nacosmodel.InternalNacosServiceProtectThreshold]; ok {
		if threshold, _ := strconv.ParseFloat(val, 64); threshold != 0 {
			ret.specService.ProtectionThreshold = threshold
		}
	}

	n.namespaces[nacosNs][key.String()] = ret
	return ret
}

// ServiceData nacos 的服务数据模型
type ServiceData struct {
	reachProtectionThreshold bool
	specService              *nacosmodel.ServiceMetadata
	name                     string
	group                    string
	lock                     sync.RWMutex
	reversion                string
	instances                map[string]*nacosmodel.Instance
}

func (s *ServiceData) loadInstances(svcIns *model.ServiceInstances) {
	if svcIns == nil {
		return
	}
	var (
		finalInstances = map[string]*nacosmodel.Instance{}
	)

	instances := svcIns.GetInstances(false)
	healthCount := 0
	for i := range instances {
		ins := &nacosmodel.Instance{}
		ins.FromSpecInstance(instances[i])
		finalInstances[ins.Id] = ins
		if ins.Healthy {
			healthCount++
		}
	}

	s.lock.Lock()
	defer s.lock.Unlock()
	s.instances = finalInstances
}

func NoopSelectInstances(ctx *FilterContext, result *nacosmodel.ServiceInfo,
	instances []*nacosmodel.Instance, healthCount int32) *nacosmodel.ServiceInfo {
	return result
}

func SelectInstancesWithHealthyProtection(ctx *FilterContext, result *nacosmodel.ServiceInfo,
	instances []*nacosmodel.Instance, healthCount int32) *nacosmodel.ServiceInfo {
	protectThreshold := ctx.Service.ProtectionThreshold
	if len(instances) > 0 && float64(healthCount)/float64(len(instances)) >= protectThreshold {
		ret := instances
		if ctx.HealthyOnly {
			healthyIns := make([]*nacosmodel.Instance, 0, len(instances))
			for i := range instances {
				if instances[i].Healthy {
					healthyIns = append(healthyIns, instances[i])
				}
			}
			ret = healthyIns
		}
		result.Hosts = ret
		return result
	}

	ret := make([]*nacosmodel.Instance, 0, len(instances))

	for i := range instances {
		if !instances[i].Healthy {
			copyIns := instances[i].DeepClone()
			copyIns.Healthy = true
			ret = append(ret, copyIns)
		} else {
			ret = append(ret, instances[i])
		}
	}

	result.ReachProtectionThreshold = true
	result.Hosts = ret
	return result
}

func ToNacosService(cacheMgr cachetypes.CacheManager, namespace, service, group string) *nacosmodel.ServiceMetadata {
	ret := &nacosmodel.ServiceMetadata{
		ServiceKey: nacosmodel.ServiceKey{
			Namespace: namespace,
			Group:     group,
			Name:      service,
		},
		ProtectionThreshold: 0.0,
	}

	polarisSvcName := nacosmodel.BuildServiceName(service, group)
	polarisSvc := cacheMgr.Service().GetServiceByName(polarisSvcName, namespace)
	if polarisSvc == nil {
		return ret
	}

	if val, ok := polarisSvc.Meta[nacosmodel.InternalNacosServiceProtectThreshold]; ok {
		if threshold, _ := strconv.ParseFloat(val, 64); threshold != 0 {
			ret.ProtectionThreshold = threshold
		}
	}
	return ret
}
