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

package batch

import (
	"context"
	"errors"
	"fmt"
	"time"

	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/common/utils"
	"github.com/polarismesh/polaris-server/naming/auth"
	"github.com/polarismesh/polaris-server/plugin"
	"github.com/polarismesh/polaris-server/store"
)

// InstanceCtrl 批量操作实例的类
type InstanceCtrl struct {
	config          *CtrlConfig
	storage         store.Store
	authority       auth.Authority
	auth            plugin.Auth
	storeThreadCh   []chan []*InstanceFuture      // store协程，负责写操作
	instanceHandler func([]*InstanceFuture) error // store协程里面调用的instance处理函数，可以是注册和反注册
	idleStoreThread chan int                      // 空闲的store协程，记录每一个空闲id
	waitDuration    time.Duration
	queue           chan *InstanceFuture // 请求接受协程
	label           string
	hbOpen          bool // 是否开启了心跳上报功能
}

// 注册实例批量操作对象
func NewBatchRegisterCtrl(storage store.Store, authority auth.Authority, auth plugin.Auth,
	config *CtrlConfig) (*InstanceCtrl, error) {
	register, err := newBatchInstanceCtrl(storage, authority, auth, config)
	if err != nil {
		return nil, err
	}
	if register == nil {
		return nil, nil
	}

	log.Infof("[Batch] open batch register")
	register.label = "register"
	register.instanceHandler = register.registerHandler
	return register, nil
}

// 实例反注册的操作对象
func NewBatchDeregisterCtrl(storage store.Store, authority auth.Authority, auth plugin.Auth, config *CtrlConfig) (
	*InstanceCtrl, error) {
	deregister, err := newBatchInstanceCtrl(storage, authority, auth, config)
	if err != nil {
		return nil, err
	}
	if deregister == nil {
		return nil, nil
	}

	log.Infof("[Batch] open batch deregister")
	deregister.label = "deregister"
	deregister.instanceHandler = deregister.deregisterHandler

	return deregister, nil
}

// 开始启动批量操作实例的相关协程
func (ctrl *InstanceCtrl) Start(ctx context.Context) {
	log.Infof("[Batch] Start batch instance, config: %+v", ctrl.config)

	// 初始化并且启动多个store协程，并发对数据库写
	for i := 0; i < ctrl.config.Concurrency; i++ {
		ctrl.storeThreadCh = append(ctrl.storeThreadCh, make(chan []*InstanceFuture))
	}
	for i := 0; i < ctrl.config.Concurrency; i++ {
		go ctrl.storeWorker(ctx, i)
	}

	// 进入主循环
	ctrl.mainLoop(ctx)
}

// 创建批量控制instance的对象
func newBatchInstanceCtrl(storage store.Store, authority auth.Authority, auth plugin.Auth,
	config *CtrlConfig) (*InstanceCtrl, error) {
	if config == nil || !config.Open {
		return nil, nil
	}

	duration, err := time.ParseDuration(config.WaitTime)
	if err != nil {
		log.Errorf("[Batch] parse waitTime(%s) err: %s", config.WaitTime, err.Error())
		return nil, err
	}
	if duration == 0 {
		log.Errorf("[Batch] config waitTime is invalid")
		return nil, errors.New("config waitTime is invalid")
	}

	instance := &InstanceCtrl{
		config:          config,
		storage:         storage,
		authority:       authority,
		auth:            auth,
		storeThreadCh:   make([]chan []*InstanceFuture, 0, config.Concurrency),
		idleStoreThread: make(chan int, config.Concurrency),
		queue:           make(chan *InstanceFuture, config.QueueSize),
		waitDuration:    duration,
	}
	return instance, nil
}

// 注册主协程
// 从注册队列中获取注册请求，当达到b.config.MaxBatchCount，
// 或当到了一个超时时间b.waitDuration，则发起一个写请求
// 写请求发送到store协程，规则：从空闲的管道idleStoreThread中挑选一个
func (ctrl *InstanceCtrl) mainLoop(ctx context.Context) {
	futures := make([]*InstanceFuture, 0, ctrl.config.MaxBatchCount)
	idx := 0
	triggerConsume := func(data []*InstanceFuture) {
		if idx == 0 {
			return
		}
		// 选择一个idle的store协程写数据 TODO 这里需要统计一下
		idleIdx := <-ctrl.idleStoreThread
		ctrl.storeThreadCh[idleIdx] <- data
		futures = make([]*InstanceFuture, 0, ctrl.config.MaxBatchCount)
		idx = 0
	}
	// 启动接受注册请求的协程
	go func() {
		ticker := time.NewTicker(ctrl.waitDuration)
		defer ticker.Stop()
		for {
			select {
			case future := <-ctrl.queue:
				futures = append(futures, future)
				idx++
				if idx == ctrl.config.MaxBatchCount {
					triggerConsume(futures[0:idx])
				}
			case <-ticker.C:
				triggerConsume(futures[0:idx])
			case <-ctx.Done():
				log.Infof("[Batch] %s main loop exited", ctrl.label)
				return
			}
		}
	}()
}

// store写协程的主循环
// 从chan中获取数据，直接写数据库
// 每次写完，设置协程为空闲
func (ctrl *InstanceCtrl) storeWorker(ctx context.Context, index int) {
	log.Infof("[Batch] %s worker(%d) running in main loop", ctrl.label, index)
	// store协程启动，先把自己注册到idle中
	ctrl.idleStoreThread <- index
	// 主循环
	for {
		select {
		case futures := <-ctrl.storeThreadCh[index]:
			if err := ctrl.instanceHandler(futures); err != nil {
				// 所有的错误都在instanceHandler函数里面进行答复和处理，这里只需记录一条日志
				log.Errorf("[Batch] %s instances err: %s", ctrl.label, err.Error())
			}
			ctrl.idleStoreThread <- index
		case <-ctx.Done():
			// idle is not ready
			log.Infof("[Batch] %s worker(%d) exited", ctrl.label, index)
			return
		}
	}
}

// 外部应该把鉴权完成
// 判断实例是否存在，也可以提前判断，减少batch复杂度
// 提前通过token判断，再进入batch操作
// batch操作，只是写操作
func (ctrl *InstanceCtrl) registerHandler(futures []*InstanceFuture) error {
	if len(futures) == 0 {
		return nil
	}

	log.Infof("[Batch] Start batch creating instances count: %d", len(futures))
	remains := make(map[string]*InstanceFuture, len(futures))
	for _, entry := range futures {
		if _, ok := remains[entry.request.GetId().GetValue()]; ok {
			entry.Reply(api.SameInstanceRequest, errors.New("there is the same instance request"))
			continue
		}

		remains[entry.request.GetId().GetValue()] = entry
	}

	// 统一判断实例是否存在
	remains, err := ctrl.batchCheckInstancesExisted(remains)
	if err != nil {
		log.Errorf("[Batch] batch check instances existed err: %s", err.Error())
	}
	// 这里可能全部都重复了
	if len(remains) == 0 {
		log.Infof("[Batch] all instances is existed, return create instances process")
		return nil
	}

	// 统一鉴权
	remains, serviceIDs, _ := ctrl.batchVerifyInstances(remains)
	if len(remains) == 0 {
		log.Infof("[Batch] all instances verify failed, no remain any instances")
		return nil
	}

	// 构造model数据
	for id, entry := range remains {
		serviceID, ok := serviceIDs[entry.request.GetId().GetValue()]
		if !ok || serviceID == "" {
			log.Errorf("[Batch] not found instance(%s) service, ignore it", entry.request.GetId().GetValue())
			delete(remains, id)
			entry.Reply(api.NotFoundResource, errors.New("not found service"))
			continue
		}
		entry.SetInstance(utils.CreateInstanceModel(serviceID, entry.request))
	}

	// 调用batch接口，创建实例
	instances := make([]*model.Instance, 0, len(remains))
	for _, entry := range remains {
		instances = append(instances, entry.instance)
	}
	if err := ctrl.storage.BatchAddInstances(instances); err != nil {
		SendReply(remains, StoreCode2APICode(err), err)
		return err
	}

	SendReply(remains, api.ExecuteSuccess, nil)
	return nil
}

// 反注册处理函数
// 步骤：
// - 从数据库中批量读取实例ID对应的实例简要信息：
//   包括：ID，host，port，serviceName，serviceNamespace，serviceToken
// - 对instance做存在与token的双重校验，较少与数据库的交互
//   - 对于不存在的token，返回notFoundResource
//   - 对于token校验失败的，返回校验失败
// - 调用批量接口删除实例
func (ctrl *InstanceCtrl) deregisterHandler(futures []*InstanceFuture) error {
	if len(futures) == 0 {
		return nil
	}

	log.Infof("[Batch] Start batch deregister instances count: %d", len(futures))
	remains := make(map[string]*InstanceFuture, len(futures))
	ids := make(map[string]bool, len(futures))
	for _, entry := range futures {
		if _, ok := remains[entry.request.GetId().GetValue()]; ok {
			entry.Reply(api.SameInstanceRequest, errors.New("there is the same instance request"))
			continue
		}

		remains[entry.request.GetId().GetValue()] = entry
		ids[entry.request.GetId().GetValue()] = false
	}

	// 统一鉴权与判断是否存在
	instances, err := ctrl.storage.GetInstancesBrief(ids)
	if err != nil {
		log.Errorf("[Batch] get instances service token err: %s", err.Error())
		SendReply(remains, api.StoreLayerException, err)
		return err
	}
	for _, future := range futures {
		instance, ok := instances[future.request.GetId().GetValue()]
		if !ok {
			// 不存在，意味着不需要删除了
			future.Reply(api.NotFoundResource, fmt.Errorf("%s", api.Code2Info(api.NotFoundResource)))
			delete(remains, future.request.GetId().GetValue())
			continue
		}

		future.SetInstance(instance) // 这里保存instance的目的：方便上层使用model数据
		if ok, code := ctrl.verifyInstanceAuth(future.platformID, future.platformToken, instance.ServiceToken(),
			instance.ServicePlatformID, future.request); !ok {
			future.Reply(code, fmt.Errorf("instances: %s %s", future.request.GetId().GetValue(),
				api.Code2Info(code)))
			delete(remains, future.request.GetId().GetValue())
			continue
		}
	}

	if len(remains) == 0 {
		log.Infof("[Batch] deregister all instances verify failed or instances is not existed, no remain any instances")
		return nil
	}

	// 调用storage batch接口，删除实例
	args := make([]interface{}, 0, len(remains))
	for _, entry := range remains {
		args = append(args, entry.request.GetId().GetValue())
	}
	if err := ctrl.storage.BatchDeleteInstances(args); err != nil {
		log.Errorf("[Batch] batch delete instances err: %s", err.Error())
		SendReply(remains, api.StoreLayerException, err)
		return err
	}

	SendReply(remains, api.ExecuteSuccess, nil)
	return nil
}

// 批量检查实例是否存在
func (ctrl *InstanceCtrl) batchCheckInstancesExisted(futures map[string]*InstanceFuture) (
	map[string]*InstanceFuture, error) {

	if len(futures) == 0 {
		return nil, nil
	}

	// 初始化所有的id都是不存在的
	ids := make(map[string]bool, len(futures))
	for _, entry := range futures {
		ids[entry.request.GetId().GetValue()] = false
	}
	if _, err := ctrl.storage.CheckInstancesExisted(ids); err != nil {
		log.Errorf("[Batch] check instances existed storage err: %s", err.Error())
		SendReply(futures, api.StoreLayerException, err)
		return nil, err
	}

	for id, existed := range ids {
		if !existed {
			continue
		}

		entry, ok := futures[id]
		if !ok {
			// 返回了没有查询的id，告警
			log.Warnf("[Batch] check instances existed get not track id : %s", id)
			continue
		}

		entry.Reply(api.ExistedResource, fmt.Errorf("instance(%s) is existed", entry.request.GetId().GetValue()))
		delete(futures, id)
	}

	return futures, nil
}

// 对请求futures进行统一的鉴权
// 目的：遇到同名的服务，可以减少getService的次数
// 返回：过滤后的futures, 实例ID->ServiceID, error
func (ctrl *InstanceCtrl) batchVerifyInstances(futures map[string]*InstanceFuture) (
	map[string]*InstanceFuture, map[string]string, error) {

	if len(futures) == 0 {
		return nil, nil, nil
	}

	serviceIDs := make(map[string]string)       // 实例ID -> ServiceID
	services := make(map[string]*model.Service) // 保存Service的鉴权结果
	for id, entry := range futures {
		serviceStr := entry.request.GetService().GetValue() + entry.request.GetNamespace().GetValue()
		service, ok := services[serviceStr]
		if !ok {
			// 鉴权，这里拿的是源服务token，如果是别名，service=nil
			tmpService, err := ctrl.storage.GetSourceServiceToken(entry.request.GetService().GetValue(),
				entry.request.GetNamespace().GetValue())
			if err != nil {
				log.Errorf("[Controller] get source service(%s, %s) token err: %s",
					entry.request.GetService().GetValue(), entry.request.GetNamespace().GetValue(), err.Error())
				entry.Reply(api.StoreLayerException, err)
				delete(futures, id)
				continue
			}

			// 注册的实例对应的源服务不存在
			if tmpService == nil {
				log.Errorf("[Controller] get source service(%s, %s) token is empty, verify failed",
					entry.request.GetService().GetValue(), entry.request.GetNamespace().GetValue())
				entry.Reply(api.NotFoundResource, errors.New("not found service"))
				delete(futures, id)
				continue
			}
			// 保存查询到的最新服务信息，后续可能会使用到
			service = tmpService
			services[serviceStr] = service
		}

		if ok, code := ctrl.verifyInstanceAuth(entry.platformID, entry.platformToken,
			service.Token, service.PlatformID, entry.request); !ok {
			entry.Reply(code, fmt.Errorf("service: %s, namepace: %s, instance: %s %s",
				entry.request.GetService().GetValue(), entry.request.GetNamespace().GetValue(),
				entry.request.GetId().GetValue(), api.Code2Info(code)))
			delete(futures, id)
			continue
		}

		// 保存每个instance注册到的服务ID
		serviceIDs[entry.request.GetId().GetValue()] = service.ID
	}

	return futures, serviceIDs, nil
}

/**
 * @brief 实例鉴权
 */
func (ctrl *InstanceCtrl) verifyInstanceAuth(platformID, platformToken, expectServiceToken, sPlatformID string,
	req *api.Instance) (bool, uint32) {
	if ok := ctrl.verifyAuthByPlatform(platformID, platformToken, sPlatformID); !ok {
		// 检查token是否存在
		actualServiceToken := req.GetServiceToken().GetValue()
		if !ctrl.authority.VerifyToken(actualServiceToken) {
			return false, api.InvalidServiceToken
		}

		// 检查token是否ok
		if ok := ctrl.authority.VerifyInstance(expectServiceToken, actualServiceToken); !ok {
			return false, api.Unauthorized
		}
	}
	return true, 0
}

/**
 * @brief 使用平台ID鉴权
 */
func (ctrl *InstanceCtrl) verifyAuthByPlatform(platformID, platformToken, sPlatformID string) bool {
	if ctrl.auth == nil {
		return false
	}

	if sPlatformID == "" {
		return false
	}

	if ctrl.auth.Allow(platformID, platformToken) && platformID == sPlatformID {
		return true
	}
	return false
}
