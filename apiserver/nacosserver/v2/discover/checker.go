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

package discover

import (
	"context"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/polarismesh/specification/source/go/api/v1/service_manage"
	"go.uber.org/zap"

	nacosmodel "github.com/polarismesh/polaris/apiserver/nacosserver/model"
	"github.com/polarismesh/polaris/apiserver/nacosserver/v2/remote"
	cachetypes "github.com/polarismesh/polaris/cache/api"
	"github.com/polarismesh/polaris/common/eventhub"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/plugin"
	"github.com/polarismesh/polaris/service"
	"github.com/polarismesh/polaris/service/healthcheck"
	"github.com/polarismesh/polaris/store"
)

type Checker struct {
	discoverSvr service.DiscoverServer
	healthSvr   *healthcheck.Server

	cacheMgr  cachetypes.CacheManager
	connMgr   *remote.ConnectionManager
	clientMgr *ConnectionClientManager

	lock sync.RWMutex
	// selfInstances 北极星服务端节点信息数据
	selfInstances map[string]*service_manage.Instance
	// nacos v2 客户端节点数据
	instances map[string]*service_manage.Instance

	leader int32

	syncCtx   context.Context
	cancel    context.CancelFunc
	watchCtxs []*eventhub.SubscribtionContext
}

const (
	eventhubSubscriberName = "nacos-v2-checker"
)

// newChecker 创建 nacos 长连接和实例信息绑定关系的健康检查，如果长连接不存在，则该连接上绑定的实例信息将失效
func newChecker(
	discoverSvr service.DiscoverServer,
	healthSvr *healthcheck.Server,
	connMgr *remote.ConnectionManager,
	clientMgr *ConnectionClientManager) (*Checker, error) {

	ctx, cancel := context.WithCancel(context.Background())
	syncCtx, cancel := context.WithCancel(context.Background())

	checker := &Checker{
		discoverSvr:   discoverSvr,
		healthSvr:     healthSvr,
		cacheMgr:      discoverSvr.Cache(),
		connMgr:       connMgr,
		clientMgr:     clientMgr,
		selfInstances: make(map[string]*service_manage.Instance),
		instances:     make(map[string]*service_manage.Instance),
		syncCtx:       syncCtx,
		cancel:        cancel,
		watchCtxs:     make([]*eventhub.SubscribtionContext, 0, 2),
	}

	checker.syncCacheData(cancel)
	subCtx, err := eventhub.Subscribe(eventhub.CacheInstanceEventTopic, checker)
	if err != nil {
		return nil, err
	}
	checker.watchCtxs = append(checker.watchCtxs, subCtx)
	// 注册 leader 变化事件
	subCtx, err = eventhub.Subscribe(eventhub.LeaderChangeEventTopic, &CheckerLeaderSubscriber{checker: checker})
	if err != nil {
		return nil, err
	}
	checker.watchCtxs = append(checker.watchCtxs, subCtx)
	// 最后启动实例健康检查任务
	go checker.runCheck(ctx)
	return checker, nil
}

func (c *Checker) syncCacheData(cancel context.CancelFunc) {
	defer cancel()

	handle := func(key string, val *model.Instance) {
		c.OnUpsert(val)
	}

	_ = c.cacheMgr.Instance().IteratorInstances(func(key string, value *model.Instance) (bool, error) {
		handle(key, value)
		return true, nil
	})
}

func (c *Checker) isLeader() bool {
	return atomic.LoadInt32(&c.leader) == 1
}

func (c *Checker) PreProcess(ctx context.Context, value any) any {
	return value
}

func (c *Checker) OnEvent(ctx context.Context, value any) error {
	// 需要等待前面的 sync 任务完成，才可以开始处理增量的 event 事件
	<-c.syncCtx.Done()

	event, ok := value.(*eventhub.CacheInstanceEvent)
	if !ok {
		return nil
	}
	switch event.EventType {
	case eventhub.EventCreated, eventhub.EventUpdated:
		c.OnUpsert(event.Instance)
	case eventhub.EventDeleted:
		c.OnDeleted(event.Instance)
	}
	return nil
}

// OnUpsert callback when cache value upsert
func (c *Checker) OnUpsert(value interface{}) {
	ins, _ := value.(*model.Instance)
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.isSelfServiceInstance(ins.Proto) {
		c.selfInstances[ins.ID()] = ins.Proto
		return
	}

	if _, ok := ins.Proto.GetMetadata()[nacosmodel.InternalNacosClientConnectionID]; ok {
		c.instances[ins.ID()] = ins.Proto
	}
}

// OnDeleted callback when cache value deleted
func (c *Checker) OnDeleted(value interface{}) {
	ins, _ := value.(*model.Instance)
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.isSelfServiceInstance(ins.Proto) {
		c.selfInstances[ins.ID()] = ins.Proto
		return
	}

	if _, ok := ins.Proto.GetMetadata()[nacosmodel.InternalNacosClientConnectionID]; ok {
		delete(c.instances, ins.ID())
	}
}

func (c *Checker) runCheck(ctx context.Context) {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.realCheck()
		}
	}
}

// 根据元数据的 Metadata ConnID 进行判断当前的长连接是否存在，如果对应长连接不存在，则反注册改实例信息数据。
// BUT: 一个实例 T1 时刻对应长连接为 Conn-1，T2 时刻对应的长连接为 Conn-2，但是在 T1 ～ T2 之间的某个时刻检测发现长连接不存在
// 此时发起一个反注册请求，该请求在 T3 时刻发起，是否会影响 T2 时刻注册上来的实例？
func (c *Checker) realCheck() {
	svr := c.discoverSvr.(*service.Server)

	defer func() {
		if err := recover(); err != nil {
			var buf [4086]byte
			n := runtime.Stack(buf[:], false)
			nacoslog.Errorf("panic recovered: %v, STACK: %s", err, buf[0:n])
		}
	}()

	// 减少锁的耗时
	c.lock.RLock()
	copyMap := make(map[string]*service_manage.Instance, len(c.instances))
	for k, v := range c.instances {
		copyMap[k] = v
	}
	c.lock.RUnlock()

	turnUnhealth := map[string]struct{}{}
	turnHealth := map[string]struct{}{}

	for instanceID, instance := range copyMap {
		connID := instance.Metadata[nacosmodel.InternalNacosClientConnectionID]
		// 如果不是 ConnID 的负责 server
		if !strings.HasSuffix(connID, utils.LocalHost) {
			if !c.isLeader() {
				continue
			}
			// 看下 ConnID 对应的负责 Server 是否健康
			found := false
			for i := range c.selfInstances {
				selfIns := c.selfInstances[i]
				if strings.HasSuffix(connID, selfIns.GetHost().GetValue()) && selfIns.GetHealthy().GetValue() {
					found = true
					break
				}
			}
			if !found {
				// connID 的负责 server 不存在，直接变为不健康
				turnUnhealth[instanceID] = struct{}{}
			}
			continue
		}
		_, exist := c.clientMgr.getClient(connID)
		isHealth := instance.GetHealthy().GetValue()
		if !exist && isHealth {
			// 如果实例对应的连接ID不存在，设置为不健康
			turnUnhealth[instanceID] = struct{}{}
			plugin.GetDiscoverEvent().PublishEvent(model.InstanceEvent{
				Id:        instanceID,
				Namespace: instance.GetNamespace().GetValue(),
				Service:   instance.GetService().GetValue(),
				Instance:  instance,
				EType:     model.EventInstanceTurnUnHealth,
			})
			continue
		}
		if !isHealth && exist {
			turnHealth[instanceID] = struct{}{}
			plugin.GetDiscoverEvent().PublishEvent(model.InstanceEvent{
				Id:        instanceID,
				SvcId:     instance.GetService().GetValue(),
				Namespace: instance.GetNamespace().GetValue(),
				Service:   instance.GetService().GetValue(),
				Instance:  instance,
				EType:     model.EventInstanceTurnHealth,
			})
		}
	}

	ids := make([]interface{}, 0, len(turnUnhealth))
	if len(turnUnhealth) > 0 {
		for id := range turnUnhealth {
			ids = append(ids, id)
		}
		nacoslog.Info("[NACOS-V2][Checker] batch set instance health_status to unhealthy",
			zap.Any("instance-ids", ids))
		if err := svr.Store().
			BatchSetInstanceHealthStatus(ids, model.StatusBoolToInt(false), utils.NewUUID()); err != nil {
			nacoslog.Error("[NACOS-V2][Checker] batch set instance health_status to unhealthy",
				zap.Any("instance-ids", ids), zap.Error(err))
		}
	}

	ids = make([]interface{}, 0, len(turnUnhealth))
	if len(turnHealth) > 0 {
		for id := range turnHealth {
			ids = append(ids, id)
		}
		nacoslog.Info("[NACOS-V2][Checker] batch set instance health_status to healty",
			zap.Any("instance-ids", ids))
		if err := svr.Store().
			BatchSetInstanceHealthStatus(ids, model.StatusBoolToInt(true), utils.NewUUID()); err != nil {
			nacoslog.Error("[NACOS-V2][Checker] batch set instance health_status to healty",
				zap.Any("instance-ids", ids), zap.Error(err))
		}
	}
}

// CheckerLeaderSubscriber
type CheckerLeaderSubscriber struct {
	checker *Checker
}

// PreProcess do preprocess logic for event
func (c *CheckerLeaderSubscriber) PreProcess(ctx context.Context, value any) any {
	return value
}

// OnEvent event trigger
func (c *CheckerLeaderSubscriber) OnEvent(ctx context.Context, i interface{}) error {
	electionEvent, ok := i.(store.LeaderChangeEvent)
	if !ok {
		return nil
	}
	if electionEvent.Key != store.ElectionKeySelfServiceChecker {
		return nil
	}
	if electionEvent.Leader {
		atomic.StoreInt32(&c.checker.leader, 1)
	} else {
		atomic.StoreInt32(&c.checker.leader, 0)
	}
	return nil
}

func (c *Checker) isSelfServiceInstance(instance *service_manage.Instance) bool {
	metadata := instance.GetMetadata()
	if svcName, ok := metadata[model.MetaKeyPolarisService]; ok {
		return svcName == c.healthSvr.SelfService()
	}
	return false
}
