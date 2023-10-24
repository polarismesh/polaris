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
	"time"

	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"

	"github.com/polarismesh/polaris/common/model"
	commonstore "github.com/polarismesh/polaris/common/store"
	"github.com/polarismesh/polaris/store"
)

// InstanceCtrl 批量操作实例的类
type ClientCtrl struct {
	config          *CtrlConfig
	storage         store.Store
	storeThreadCh   []chan []*ClientFuture      // store协程，负责写操作
	clientHandler   func([]*ClientFuture) error // store协程里面调用的instance处理函数，可以是注册和反注册
	idleStoreThread chan int                    // 空闲的store协程，记录每一个空闲id
	waitDuration    time.Duration
	queue           chan *ClientFuture // 请求接受协程
	label           string
}

// NewBatchRegisterClientCtrl 注册客户端批量操作对象
func NewBatchRegisterClientCtrl(storage store.Store, config *CtrlConfig) (*ClientCtrl, error) {
	register, err := newBatchClientCtrl(storage, config)
	if err != nil {
		return nil, err
	}
	if register == nil {
		return nil, nil
	}

	log.Infof("[Batch] open batch register client")
	register.label = "register"
	register.clientHandler = register.registerHandler
	return register, nil
}

// NewBatchDeregisterClientCtrl 注册客户端批量操作对象
func NewBatchDeregisterClientCtrl(storage store.Store, config *CtrlConfig) (*ClientCtrl, error) {
	deregister, err := newBatchClientCtrl(storage, config)
	if err != nil {
		return nil, err
	}
	if deregister == nil {
		return nil, nil
	}

	log.Infof("[Batch] open batch deregister client")
	deregister.label = "deregister"
	deregister.clientHandler = deregister.deregisterHandler
	return deregister, nil
}

// Start 开始启动批量操作实例的相关协程
func (ctrl *ClientCtrl) Start(ctx context.Context) {
	log.Infof("[Batch][Client] Start batch instance, config: %+v", ctrl.config)

	// 初始化并且启动多个store协程，并发对数据库写
	for i := 0; i < ctrl.config.Concurrency; i++ {
		ctrl.storeThreadCh = append(ctrl.storeThreadCh, make(chan []*ClientFuture))
	}
	for i := 0; i < ctrl.config.Concurrency; i++ {
		go ctrl.storeWorker(ctx, i)
	}

	// 进入主循环
	ctrl.mainLoop(ctx)
}

// newBatchInstanceCtrl 创建批量控制instance的对象
func newBatchClientCtrl(storage store.Store, config *CtrlConfig) (*ClientCtrl, error) {
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

	instance := &ClientCtrl{
		config:          config,
		storage:         storage,
		storeThreadCh:   make([]chan []*ClientFuture, 0, config.Concurrency),
		idleStoreThread: make(chan int, config.Concurrency),
		queue:           make(chan *ClientFuture, config.QueueSize),
		waitDuration:    duration,
	}
	return instance, nil
}

// mainLoop 注册主协程
// 从注册队列中获取注册请求，当达到b.config.MaxBatchCount，
// 或当到了一个超时时间b.waitDuration，则发起一个写请求
// 写请求发送到store协程，规则：从空闲的管道idleStoreThread中挑选一个
func (ctrl *ClientCtrl) mainLoop(ctx context.Context) {
	futures := make([]*ClientFuture, 0, ctrl.config.MaxBatchCount)
	idx := 0
	triggerConsume := func(data []*ClientFuture) {
		if idx == 0 {
			return
		}
		// 选择一个idle的store协程写数据 TODO 这里需要统计一下
		idleIdx := <-ctrl.idleStoreThread
		ctrl.storeThreadCh[idleIdx] <- data
		futures = make([]*ClientFuture, 0, ctrl.config.MaxBatchCount)
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
				log.Debugf("[Batch] %s main loop exited", ctrl.label)
				return
			}
		}
	}()
}

// storeWorker store写协程的主循环
// 从chan中获取数据，直接写数据库
// 每次写完，设置协程为空闲
func (ctrl *ClientCtrl) storeWorker(ctx context.Context, index int) {
	log.Debugf("[Batch][Client] %s worker(%d) running in main loop", ctrl.label, index)
	// store协程启动，先把自己注册到idle中
	ctrl.idleStoreThread <- index
	// 主循环
	for {
		select {
		case futures := <-ctrl.storeThreadCh[index]:
			if err := ctrl.clientHandler(futures); err != nil {
				// 所有的错误都在instanceHandler函数里面进行答复和处理，这里只需记录一条日志
				log.Errorf("[Batch][Client] %s clients err: %s", ctrl.label, err.Error())
			}
			ctrl.idleStoreThread <- index
		case <-ctx.Done():
			// idle is not ready
			log.Infof("[Batch][Client] %s worker(%d) exited", ctrl.label, index)
			return
		}
	}
}

// registerHandler 外部应该把鉴权完成
// 判断实例是否存在，也可以提前判断，减少batch复杂度
// 提前通过token判断，再进入batch操作
// batch操作，只是写操作
func (ctrl *ClientCtrl) registerHandler(futures []*ClientFuture) error {
	if len(futures) == 0 {
		return nil
	}

	log.Infof("[Batch] Start batch creating clients count: %d", len(futures))

	// 调用batch接口，创建实例
	clients := make([]*model.Client, 0, len(futures))
	for _, entry := range futures {
		clients = append(clients, model.NewClient(entry.request))
	}
	if err := ctrl.storage.BatchAddClients(clients); err != nil {
		SendClientReply(futures, commonstore.StoreCode2APICode(err), err)
		return err
	}

	SendClientReply(futures, apimodel.Code_ExecuteSuccess, nil)
	return nil
}

// deregisterHandler 外部应该把鉴权完成
// 判断实例是否存在，也可以提前判断，减少batch复杂度
// 提前通过token判断，再进入batch操作
// batch操作，只是写操作
func (ctrl *ClientCtrl) deregisterHandler(futures []*ClientFuture) error {
	if len(futures) == 0 {
		return nil
	}

	log.Infof("[Batch] Start batch deleting clients count: %d", len(futures))

	// 调用batch接口，创建实例
	clients := make([]string, 0, len(futures))
	for _, entry := range futures {
		id := entry.request.GetId().GetValue()
		clients = append(clients, id)
	}
	if err := ctrl.storage.BatchDeleteClients(clients); err != nil {
		SendClientReply(futures, commonstore.StoreCode2APICode(err), err)
		return err
	}

	SendClientReply(futures, apimodel.Code_ExecuteSuccess, nil)
	return nil
}
