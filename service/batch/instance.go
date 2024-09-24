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
	"strconv"
	"time"

	"github.com/golang/protobuf/ptypes/wrappers"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	"go.uber.org/zap"

	"github.com/polarismesh/polaris/cache"
	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/model"
	commonstore "github.com/polarismesh/polaris/common/store"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/store"
)

var (
	ErrorNotFoundService          = errors.New("not found service")
	ErrorSameRegIsInstanceRequest = errors.New("there is the same instance request")
	ErrorRegIsInstanceTimeout     = errors.New("polaris-sever regis instance busy")
)

const (
	defaultWaitTime = 32 * time.Millisecond
	defaultTaskLife = 30 * time.Second
)

// InstanceCtrl 批量操作实例的类
type InstanceCtrl struct {
	config   *CtrlConfig
	storage  store.Store
	cacheMgn *cache.CacheManager

	// store协程，负责写操作
	storeThreadCh []chan []*InstanceFuture

	// store协程里面调用的instance处理函数，可以是注册和反注册
	instanceHandler func([]*InstanceFuture) error

	// 空闲的store协程，记录每一个空闲id
	idleStoreThread chan int
	waitDuration    time.Duration

	// 任务的有效时间
	taskLife time.Duration

	// 请求接受协程
	queue chan *InstanceFuture
	label string

	// 是否开启了心跳上报功能
	hbOpen bool
}

// NewBatchRegisterCtrl 注册实例批量操作对象
func NewBatchRegisterCtrl(storage store.Store, cacheMgn *cache.CacheManager,
	config *CtrlConfig) (*InstanceCtrl, error) {

	register, err := newBatchInstanceCtrl("register", storage, cacheMgn, config)
	if err != nil {
		return nil, err
	}
	if register == nil {
		return nil, nil
	}

	log.Info("[Batch] open batch register")
	register.label = "register"
	register.instanceHandler = register.registerHandler
	return register, nil
}

// NewBatchDeregisterCtrl 实例反注册的操作对象
func NewBatchDeregisterCtrl(storage store.Store, cacheMgn *cache.CacheManager, config *CtrlConfig) (
	*InstanceCtrl, error) {
	deregister, err := newBatchInstanceCtrl("deregister", storage, cacheMgn, config)
	if err != nil {
		return nil, err
	}
	if deregister == nil {
		return nil, nil
	}

	log.Info("[Batch] open batch deregister")
	deregister.label = "deregister"
	deregister.instanceHandler = deregister.deregisterHandler

	return deregister, nil
}

// NewBatchHeartbeatCtrl 实例心跳的操作对象
func NewBatchHeartbeatCtrl(storage store.Store, cacheMgn *cache.CacheManager, config *CtrlConfig) (
	*InstanceCtrl, error) {
	heartbeat, err := newBatchInstanceCtrl("heartbeat", storage, cacheMgn, config)
	if err != nil {
		return nil, err
	}
	if heartbeat == nil {
		return nil, nil
	}

	log.Info("[Batch] open batch heartbeat")
	heartbeat.label = "heartbeat"
	heartbeat.instanceHandler = heartbeat.heartbeatHandler

	return heartbeat, nil
}

// Start 开始启动批量操作实例的相关协程
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

// newBatchInstanceCtrl 创建批量控制instance的对象
func newBatchInstanceCtrl(label string, storage store.Store, cacheMgn *cache.CacheManager,
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
		log.Infof("[Batch] waitTime(%s) is 0, use default %v", config.WaitTime, defaultWaitTime)
		duration = defaultWaitTime
	}

	taskLife := defaultTaskLife
	if config.TaskLife != "" {
		taskLife, err := time.ParseDuration(config.TaskLife)
		if err != nil {
			log.Errorf("[Batch] parse taskLife(%s) err: %s", config.TaskLife, err.Error())
			return nil, err
		}
		if taskLife == 0 {
			log.Infof("[Batch] taskLife(%s) is 0, use default %v", config.TaskLife, defaultTaskLife)
			taskLife = defaultTaskLife
		}
	} else {
		// mean not allow drop expire task
		taskLife = time.Duration(0)
	}
	log.Info("[Batch] drop expire task", zap.String("type", label), zap.Bool("switch-open", taskLife == 0))

	instance := &InstanceCtrl{
		config:          config,
		storage:         storage,
		cacheMgn:        cacheMgn,
		storeThreadCh:   make([]chan []*InstanceFuture, 0, config.Concurrency),
		idleStoreThread: make(chan int, config.Concurrency),
		queue:           make(chan *InstanceFuture, config.QueueSize),
		waitDuration:    duration,
		taskLife:        taskLife,
	}
	return instance, nil
}

// mainLoop 注册主协程
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

// storeWorker store写协程的主循环
// 从chan中获取数据，直接写数据库
// 每次写完，设置协程为空闲
func (ctrl *InstanceCtrl) storeWorker(ctx context.Context, index int) {
	log.Debugf("[Batch] %s worker(%d) running in main loop", ctrl.label, index)
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

// registerHandler 外部应该把鉴权完成
// 判断实例是否存在，也可以提前判断，减少batch复杂度
// 提前通过token判断，再进入batch操作
// batch操作，只是写操作
func (ctrl *InstanceCtrl) registerHandler(futures []*InstanceFuture) error {
	if len(futures) == 0 {
		log.Warn("[Batch] futures is empty")
		return nil
	}

	cur := time.Now()
	taskLife := ctrl.taskLife
	dropExpire := taskLife != 0

	log.Infof("[Batch] Start batch creating instances count: %d", len(futures))
	remains := make(map[string]*InstanceFuture, len(futures))
	for i := range futures {
		entry := futures[i]
		entry.isRegis = true

		if _, ok := remains[entry.request.GetId().GetValue()]; ok {
			entry.Reply(cur, apimodel.Code_SameInstanceRequest, ErrorSameRegIsInstanceRequest)
			continue
		}

		if dropExpire && entry.CanDrop() && entry.begin.Add(taskLife).Before(cur) {
			entry.Reply(cur, apimodel.Code_InstanceRegisTimeout, ErrorRegIsInstanceTimeout)
			continue
		}

		remains[entry.request.GetId().GetValue()] = entry
	}

	// 统一判断实例是否存在，存在则需要更新部分数据
	if err := ctrl.batchRestoreInstanceIsolate(remains); err != nil {
		log.Errorf("[Batch] batch check instances existed err: %s", err.Error())
	}

	// 判断入参数组是否为0
	if len(remains) == 0 {
		log.Info("[Batch] all instances is existed, return create instances process")
		return nil
	}
	// 构造model数据
	for _, entry := range remains {
		ins := model.CreateInstanceModel(entry.serviceId, entry.request)
		entry.SetInstance(ins)
	}
	// 调用batch接口，创建实例
	instances := make([]*model.Instance, 0, len(remains))
	for _, entry := range remains {
		instances = append(instances, entry.instance)
	}
	if err := ctrl.storage.BatchAddInstances(instances); err != nil {
		sendReply(remains, commonstore.StoreCode2APICode(err), err)
		return err
	}

	sendReply(remains, apimodel.Code_ExecuteSuccess, nil)
	return nil
}

// heartbeatHandler 心跳状态变更处理函数
func (ctrl *InstanceCtrl) heartbeatHandler(futures []*InstanceFuture) error {
	if len(futures) == 0 {
		return nil
	}
	log.Infof("[Batch] start batch heartbeat instances count: %d", len(futures))
	ids := make(map[string]bool, len(futures))
	statusToIds := map[bool]map[string]int64{
		true:  make(map[string]int64, len(futures)),
		false: make(map[string]int64, len(futures)),
	}
	for _, entry := range futures {
		// 多个记录，只有后面的一个生效
		id := entry.request.GetId().GetValue()
		if _, ok := ids[id]; ok {
			values := statusToIds[!entry.healthy]
			delete(values, id)
		}
		ids[id] = false
		statusToIds[entry.healthy][id] = entry.lastHeartbeatTimeSec
	}

	// 转为不健康的实例，需要添加 metadata
	appendMetaReqs := make([]*store.InstanceMetadataRequest, 0, len(statusToIds[false]))
	// 转为健康的实例，需要删除 metadata
	removeMetaReqs := make([]*store.InstanceMetadataRequest, 0, len(statusToIds[true]))
	revision := utils.NewUUID()
	for healthy, values := range statusToIds {
		if len(values) == 0 {
			continue
		}
		idValues := make([]interface{}, 0, len(values))
		for id := range values {
			if healthy {
				removeMetaReqs = append(removeMetaReqs, &store.InstanceMetadataRequest{
					InstanceID: id,
					Revision:   revision,
					Keys:       []string{model.MetadataInstanceLastHeartbeatTime},
				})
			} else {
				appendMetaReqs = append(appendMetaReqs, &store.InstanceMetadataRequest{
					InstanceID: id,
					Revision:   revision,
					Metadata: map[string]string{
						model.MetadataInstanceLastHeartbeatTime: strconv.FormatInt(values[id], 10),
					},
				})
			}
			idValues = append(idValues, id)
		}
		err := ctrl.storage.BatchSetInstanceHealthStatus(idValues, model.StatusBoolToInt(healthy), utils.NewUUID())
		if err != nil {
			log.Errorf("[Batch] batch healthy check instances err: %s", err.Error())
			sendReply(futures, commonstore.StoreCode2APICode(err), err)
			return err
		}
		if err := ctrl.storage.BatchAppendInstanceMetadata(appendMetaReqs); err != nil {
			log.Errorf("[Batch] batch healthy check instances append metadata err: %s", err.Error())
			sendReply(futures, commonstore.StoreCode2APICode(err), err)
			return err
		}
		if err := ctrl.storage.BatchRemoveInstanceMetadata(removeMetaReqs); err != nil {
			log.Errorf("[Batch] batch healthy check instances remove metadata err: %s", err.Error())
			sendReply(futures, commonstore.StoreCode2APICode(err), err)
			return err
		}
	}
	sendReply(futures, apimodel.Code_ExecuteSuccess, nil)
	return nil
}

// deregisterHandler 反注册处理函数
// 步骤：
//   - 从数据库中批量读取实例ID对应的实例简要信息：
//     包括：ID，host，port，serviceName，serviceNamespace，serviceToken
//   - 对instance做存在与token的双重校验，较少与数据库的交互
//   - 对于不存在的token，返回notFoundResource
//   - 对于token校验失败的，返回校验失败
//   - 调用批量接口删除实例
func (ctrl *InstanceCtrl) deregisterHandler(futures []*InstanceFuture) error {
	if len(futures) == 0 {
		return nil
	}

	cur := time.Now()
	log.Infof("[Batch] Start batch deregister instances count: %d", len(futures))
	remains := make(map[string]*InstanceFuture, len(futures))
	ids := make(map[string]bool, len(futures))
	for _, entry := range futures {
		if _, ok := remains[entry.request.GetId().GetValue()]; ok {
			entry.Reply(cur, apimodel.Code_SameInstanceRequest, ErrorSameRegIsInstanceRequest)
			continue
		}

		remains[entry.request.GetId().GetValue()] = entry
		ids[entry.request.GetId().GetValue()] = false
	}

	// 统一鉴权与判断是否存在
	instances, err := ctrl.storage.GetInstancesBrief(ids)
	if err != nil {
		log.Errorf("[Batch] get instances service token err: %s", err.Error())
		sendReply(remains, commonstore.StoreCode2APICode(err), err)
		return err
	}
	for _, future := range futures {
		instance, ok := instances[future.request.GetId().GetValue()]
		if !ok {
			// 不存在，意味着不需要删除了
			future.Reply(cur, apimodel.Code_NotFoundResource, fmt.Errorf("%s", api.Code2Info(api.NotFoundResource)))
			delete(remains, future.request.GetId().GetValue())
			continue
		}

		future.SetInstance(instance) // 这里保存instance的目的：方便上层使用model数据
	}

	if len(remains) == 0 {
		log.Infof("[Batch] deregister instances verify failed or instances is not existed, no remain any instances")
		return nil
	}

	// 调用storage batch接口，删除实例
	args := make([]interface{}, 0, len(remains))
	for _, entry := range remains {
		args = append(args, entry.request.GetId().GetValue())
	}
	if err := ctrl.storage.BatchDeleteInstances(args); err != nil {
		log.Errorf("[Batch] batch delete instances err: %s", err.Error())
		sendReply(remains, commonstore.StoreCode2APICode(err), err)
		return err
	}

	sendReply(remains, apimodel.Code_ExecuteSuccess, nil)
	return nil
}

// batchRestoreInstanceIsolate 批量恢复实例的隔离状态，以请求为准，请求如果不存在，就以数据库为准
func (ctrl *InstanceCtrl) batchRestoreInstanceIsolate(futures map[string]*InstanceFuture) error {
	if len(futures) == 0 {
		return nil
	}

	// 初始化所有的id都是不存在的
	ids := make(map[string]bool, len(futures))
	for _, entry := range futures {
		ids[entry.request.GetId().GetValue()] = false
	}
	var id2Isolate map[string]bool
	var err error
	if id2Isolate, err = ctrl.storage.BatchGetInstanceIsolate(ids); err != nil {
		log.Errorf("[Batch] check instances existed storage err: %s", err.Error())
		sendReply(futures, commonstore.StoreCode2APICode(err), err)
		return err
	}

	if len(id2Isolate) == 0 {
		return nil
	}

	if len(id2Isolate) > 0 {
		for id, isolate := range id2Isolate {
			if future, ok := futures[id]; ok && future.request.Isolate == nil {
				future.request.Isolate = &wrappers.BoolValue{Value: isolate}
			}
		}
	}
	return nil
}
