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

	"github.com/polarismesh/polaris-server/cache"
	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/store"
)

// Controller 批量控制器
type Controller struct {
	register         *InstanceCtrl
	deregister       *InstanceCtrl
	heartbeat        *InstanceCtrl
	clientRegister   *ClientCtrl
	clientDeregister *ClientCtrl
}

// NewBatchCtrlWithConfig 根据配置文件创建一个批量控制器
func NewBatchCtrlWithConfig(storage store.Store, cacheMgn *cache.CacheManager, config *Config) (*Controller, error) {
	if config == nil {
		return nil, nil
	}

	var err error
	var register *InstanceCtrl
	register, err = NewBatchRegisterCtrl(storage, cacheMgn, config.Register)
	if err != nil {
		log.Errorf("[Batch] new batch register instance ctrl err: %s", err.Error())
		return nil, err
	}

	var deregister *InstanceCtrl
	deregister, err = NewBatchDeregisterCtrl(storage, cacheMgn, config.Deregister)
	if err != nil {
		log.Errorf("[Batch] new batch deregister instance ctrl err: %s", err.Error())
		return nil, err
	}

	var heartbeat *InstanceCtrl
	heartbeat, err = NewBatchHeartbeatCtrl(storage, cacheMgn, config.Heartbeat)
	if err != nil {
		log.Errorf("[Batch] new batch heartbeat instance ctrl err: %s", err.Error())
		return nil, err
	}

	var clientRegister *ClientCtrl
	clientRegister, err = NewBatchRegisterClientCtrl(storage, config.ClientRegister)
	if err != nil {
		log.Errorf("[Batch] new batch client register ctrl err: %s", err.Error())
		return nil, err
	}

	var clientDeregister *ClientCtrl
	clientDeregister, err = NewBatchDeregisterClientCtrl(storage, config.ClientDeregister)
	if err != nil {
		log.Errorf("[Batch] new batch client deregister ctrl err: %s", err.Error())
		return nil, err
	}

	bc := &Controller{
		register:         register,
		deregister:       deregister,
		heartbeat:        heartbeat,
		clientRegister:   clientRegister,
		clientDeregister: clientDeregister,
	}
	return bc, nil
}

// Start 开启批量控制器
// 启动多个协程，接受外部create/delete请求
func (bc *Controller) Start(ctx context.Context) {
	if bc.CreateInstanceOpen() {
		bc.register.Start(ctx)
	}
	if bc.DeleteInstanceOpen() {
		bc.deregister.Start(ctx)
	}
	if bc.HeartbeatOpen() {
		bc.heartbeat.Start(ctx)
	}
	if bc.ClientRegisterOpen() {
		bc.clientRegister.Start(ctx)
	}
	if bc.ClientDeregisterOpen() {
		bc.clientDeregister.Start(ctx)
	}
}

// CreateInstanceOpen 创建是否开启
func (bc *Controller) CreateInstanceOpen() bool {
	return bc.register != nil
}

// DeleteInstanceOpen 删除实例是否开启
func (bc *Controller) DeleteInstanceOpen() bool {
	return bc.deregister != nil
}

// HeartbeatOpen 心跳是否开启
func (bc *Controller) HeartbeatOpen() bool {
	return bc.heartbeat != nil
}

// ClientRegisterOpen 添加客户端是否开启
func (bc *Controller) ClientRegisterOpen() bool {
	return bc.clientRegister != nil
}

// ClientDeregisterOpen 删除客户端是否开启
func (bc *Controller) ClientDeregisterOpen() bool {
	return bc.clientDeregister != nil
}

// AsyncCreateInstance 异步创建实例，返回一个future，根据future获取创建结果
func (bc *Controller) AsyncCreateInstance(svcId string, instance *api.Instance, needWait bool) *InstanceFuture {
	future := &InstanceFuture{
		serviceId: svcId,
		needWait:  needWait,
		request:   instance,
	}

	if needWait {
		future.result = make(chan error, 1)
	}

	// 发送到注册请求队列
	bc.register.queue <- future
	return future
}

// AsyncDeleteInstance 异步合并反注册
func (bc *Controller) AsyncDeleteInstance(instance *api.Instance) *InstanceFuture {
	future := &InstanceFuture{
		request:  instance,
		result:   make(chan error, 1),
		needWait: true,
	}

	bc.deregister.queue <- future
	return future
}

// AsyncHeartbeat 异步心跳
func (bc *Controller) AsyncHeartbeat(instance *api.Instance, healthy bool) *InstanceFuture {
	future := &InstanceFuture{
		request:  instance,
		result:   make(chan error, 1),
		healthy:  healthy,
		needWait: true,
	}

	bc.heartbeat.queue <- future
	return future
}

// AsyncRegisterClient 异步合并反注册
func (bc *Controller) AsyncRegisterClient(client *api.Client) *ClientFuture {
	future := &ClientFuture{
		request: client,
		result:  make(chan error, 1),
	}

	bc.clientRegister.queue <- future
	return future
}

// AsyncDeregisterClient 异步合并反注册
func (bc *Controller) AsyncDeregisterClient(client *api.Client) *ClientFuture {
	future := &ClientFuture{
		request: client,
		result:  make(chan error, 1),
	}

	bc.clientDeregister.queue <- future
	return future
}
