/*
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

package naming

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"sync"
	"time"

	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/common/redispool"
	"github.com/polarismesh/polaris-server/common/timewheel"
	"github.com/polarismesh/polaris-server/common/utils"
	"go.uber.org/zap"
)

/**
 * HealthCheckConfig 健康检查配置
 */
type HealthCheckConfig struct {
	Open          bool   `yaml:"open"`
	KvConnNum     int    `yaml:"kvConnNum"`
	KvServiceName string `yaml:"kvServiceName"`
	KvNamespace   string `yaml:"kvNamespace"`
	KvPasswd      string `yaml:"kvPasswd"`
	SlotNum       int    `yaml:"slotNum"`
	LocalHost     string `yaml:"localHost"`
	MaxIdle       int    `yaml:"maxIdle"`
	IdleTimeout   int    `yaml:"idleTimeout"`
}

/**
 * HbInfo 记录实例心跳信息
 */
type HbInfo struct {
	id       string
	addr     string
	beatTime int64
	ttl      uint32
}

/**
 * HeartBeatMgr 心跳管理器结构体
 * 包括时间轮、ckv连接池、存储实例心跳信息的map
 */
type HeartBeatMgr struct {
	ctx   context.Context
	mu    sync.Mutex
	hbMap map[string]*HbInfo
	ckvTw *timewheel.TimeWheel
	dbTw  *timewheel.TimeWheel
	// ckvPool   *ckv.Pool
	redisPool *redispool.Pool
}

/**
 * TimeWheelTask 时间轮任务结构体
 */
type TimeWheelTask struct {
	lastBeatTime int64
	hbInfo       *HbInfo
}

const (
	NotHealthy    = 0
	Healthy       = 1
	RedisNoKeyErr = "redigo: nil returned"
)

var (
	healthCheckConf *HealthCheckConfig
	hbMgr           *HeartBeatMgr

	/**
	* @brief 时间轮回调函数1：实例上报心跳1 ttl后执行，发现这
	* 段时间内该实例没有再次上报心跳则改写ckv中的实例状态为不健康
	 */
	ckvCallback timewheel.Callback = func(data interface{}) {
		task := data.(*TimeWheelTask)
		lastBeatTime := task.lastBeatTime
		// 说明本机收到了心跳，实例状态正常，不做任何处理
		if lastBeatTime < task.hbInfo.beatTime {
			return
		}

		// 更新上报心跳时间
		now := time.Now().Unix()
		task.lastBeatTime = now

		// 从ckv获取实例状态，看其他server这段时间有没有收到心跳
		respCh := make(chan *redispool.Resp)
		hbMgr.redisPool.Get(task.hbInfo.id, respCh)
		resp := <-respCh
		if !resp.Local {
			if resp.Err != nil {
				// 获取ckv失败，不能退出，直接进入set ckv unhealthy & dbCallback流程
				log.Errorf("[health check] addr:%s id:%s 1ttl get redis err:%s",
					task.hbInfo.addr, task.hbInfo.id, resp.Err)
			} else {
				// ckv中value格式 健康状态(1健康 0不健康):心跳时间戳:写者ip
				// 如: 1:timestamp:10.60.31.22
				// 基于解析性能考虑，没有使用json
				res := strings.Split(resp.Value, ":")
				kvBeatTime, err := strconv.ParseInt(res[1], 10, 64)
				if err != nil {
					log.Errorf("[health check] addr:%s id:%s redis beat time parse err:%s",
						task.hbInfo.addr, task.hbInfo.id, err)
				}

				// ckv中的实例状态为已经被改为不健康
				// 或ckv心跳时间 > 本地上次心跳时间，说明其他server收到了心跳，不做任何处理
				if res[0] == "0" || kvBeatTime > lastBeatTime {
					return
				}
			}

			// 将ckv中状态改为不健康
			log.Infof("[health check] addr:%s id:%s 1ttl overtime, set redis not healthy", task.hbInfo.addr, task.hbInfo.id)
			hbMgr.redisPool.Set(task.hbInfo.id, NotHealthy, now, respCh)
			resp = <-respCh
			if resp.Err != nil {
				log.Errorf("[health check] addr:%s id:%s set redis err:%s", task.hbInfo.addr, task.hbInfo.id, resp.Err)
			}
		}
		// 添加时间轮任务：再过2ttl后仍未收到心跳，改写db实例状态为不健康
		_ = hbMgr.dbTw.AddTask(time.Duration(2*task.hbInfo.ttl-1)*time.Second, task, dbCallback)
	}

	/**
	* @brief 时间轮回调函数2：实例上报心跳1 ttl后若
	* 未上报心跳，则再过2ttl后执行此函数，负责判定实例死亡、改写db状态
	 */
	dbCallback timewheel.Callback = func(data interface{}) {
		task := data.(*TimeWheelTask)
		// 说明本机收到了心跳，实例状态正常，不做任何处理
		if task.lastBeatTime < task.hbInfo.beatTime {
			return
		}

		// 从ckv获取下key状态，看其他server有没有收到心跳
		respCh := make(chan *redispool.Resp)
		hbMgr.redisPool.Get(task.hbInfo.id, respCh)
		resp := <-respCh
		if !resp.Local {
			if resp.Err != nil {
				log.Errorf("[healthCheck] dbCallback get addr(%s) id(%s) from redis err: %s",
					task.hbInfo.addr, task.hbInfo.id, resp.Err)
			} else {
				res := strings.Split(resp.Value, ":")
				if res[0] == "1" {
					log.Infof("[health check] addr: %s id: %s redis status is healthy, ignore set db unhealthy",
						task.hbInfo.addr, task.hbInfo.id)
					return
				}
			}

			// 删除kv
			log.Infof("[health check] del redis id:%s", task.hbInfo.id)
			hbMgr.redisPool.Del(task.hbInfo.id, respCh)
			resp = <-respCh
			if resp.Err != nil {
				log.Errorf("[health check] addr:%s id:%s del redis err:%s", task.hbInfo.addr, task.hbInfo.id, resp.Err)
			}
		}

		insCache := server.caches.Instance().GetInstance(task.hbInfo.id)
		if insCache == nil {
			log.Errorf(`[health check] addr:%s id:%s ready to set db status 
			not health, but not found instance`, task.hbInfo.addr, task.hbInfo.id)
			return
		}

		// 修改db状态为不健康
		// 如果用户关闭了健康检查功能，则不做任何处理
		if insCache.EnableHealthCheck() && insCache.Healthy() != false {
			setInsDbStatus(task.hbInfo.id, task.hbInfo.addr, NotHealthy)
		}

		// 从本机map中删除
		hbMgr.mu.Lock()
		delete(hbMgr.hbMap, task.hbInfo.id)
		hbMgr.mu.Unlock()
	}
)

/**
* SetHealthCheckConfig 设置健康检查配置
 */
func SetHealthCheckConfig(conf *HealthCheckConfig) {
	healthCheckConf = conf
}

/**
 * NewHeartBeatMgr 初始化心跳管理器
 */
func NewHeartBeatMgr(ctx context.Context) (*HeartBeatMgr, error) {
	kvService := server.caches.Service().
		GetServiceByName(healthCheckConf.KvServiceName, healthCheckConf.KvNamespace)
	var kvInstances []*model.Instance

	if kvService != nil {
		kvInstances = server.caches.Instance().GetInstancesByServiceID(kvService.ID)
	}
	// if len(kvInstances) == 0 {
	//	return nil, fmt.Errorf("no available ckv instance, serviceId:%s", kvService.ID)
	// }

	redisPool, err := redispool.NewPool(healthCheckConf.KvConnNum, healthCheckConf.KvPasswd,
		healthCheckConf.LocalHost, kvInstances, healthCheckConf.MaxIdle, healthCheckConf.IdleTimeout)
	if err != nil {
		return nil, err
	}

	mgr := &HeartBeatMgr{
		ctx:       ctx,
		hbMap:     make(map[string]*HbInfo),
		ckvTw:     timewheel.New(time.Second, healthCheckConf.SlotNum, "ckv task timewheel"),
		dbTw:      timewheel.New(time.Second, healthCheckConf.SlotNum, "db task timewheel"),
		redisPool: redisPool,
	}
	if kvService != nil {
		go mgr.watchCkvService(kvService.ID)
	}
	return mgr, nil
}

/**
 * Start 启动心跳管理器，启动健康检查功能
 */
func (hb *HeartBeatMgr) Start() {
	hb.redisPool.Start()
	hb.ckvTw.Start()
	hb.dbTw.Start()
}

/**
 * @brief 监控ckv实例有没有变化
 */
func (hb *HeartBeatMgr) watchCkvService(id string) {
	kvInstances := server.caches.Instance().GetInstancesByServiceID(id)
	lastRevision, err := server.GetServiceInstanceRevision(id, kvInstances)
	if err != nil {
		log.Errorf("[health check] get redis revision err:%s", err)
	}

	for range time.NewTicker(10 * time.Second).C {
		kvInstances = server.caches.Instance().GetInstancesByServiceID(id)
		if len(kvInstances) == 0 {
			// need alert
			log.Errorf("[health check] get redis ins nil")
			continue
		}
		newRevision, err := server.GetServiceInstanceRevision(id, kvInstances)
		if err != nil {
			log.Errorf("[health check] get redis revision err:%s", err)
			continue
		}
		if lastRevision != newRevision {
			err := hb.redisPool.Update(kvInstances)
			if err != nil {
				// need alert
				log.Errorf("[health check] update redis pool err:%s", err)
				continue
			}
			lastRevision = newRevision
		}
	}
}

/**
* @brief 心跳处理函数
 */
func (hb *HeartBeatMgr) healthCheck(ctx context.Context, instance *api.Instance) *api.Response {
	id, errRsp := checkHeartbeatInstance(instance)
	if errRsp != nil {
		return errRsp
	}
	instance.Id = utils.NewStringValue(id)
	insCache := server.caches.Instance().GetInstance(id)
	if insCache == nil {
		return api.NewInstanceResponse(api.NotFoundResource, instance)
	}

	service := server.caches.Service().GetServiceByID(insCache.ServiceID)
	if service == nil {
		return api.NewInstanceResponse(api.NotFoundResource, instance)
	}
	// 鉴权
	token := instance.GetServiceToken().GetValue()
	if !server.authority.VerifyToken(token) {
		return api.NewInstanceResponse(api.InvalidServiceToken, instance)
	}
	ok := server.authority.VerifyInstance(service.Token, token)
	if !ok {
		return api.NewInstanceResponse(api.Unauthorized, instance)
	}

	// 如果实例未开启健康检查，返回
	if !insCache.EnableHealthCheck() || insCache.HealthCheck() == nil {
		return api.NewInstanceResponse(api.HeartbeatOnDisabledIns, instance)
	}

	// 记录收到心跳的instance日志，方便定位实例是否上报心跳
	log.Info("receive heartbeat", ZapRequestID(ParseRequestID(ctx)), zap.String("id", id),
		zap.String("service", service.Namespace+":"+service.Name),
		zap.String("host", insCache.Host()), zap.Uint32("port", insCache.Port()))
	addr := insCache.Host() + ":" + strconv.Itoa(int(insCache.Port()))
	ttl := insCache.HealthCheck().GetHeartbeat().GetTtl().GetValue()
	now := time.Now().Unix()
	var hbInfo *HbInfo
	respCh := make(chan *redispool.Resp)

	hb.mu.Lock()
	hbInfo, ok = hb.hbMap[id]
	if !ok {
		hbInfo = &HbInfo{id, addr, now, ttl}
		hb.hbMap[id] = hbInfo
		hb.mu.Unlock()

		// hbMap中没有找到该实例，说明这是实例近期第一次上报心跳，set ckv中实例状态为健康
		log.Infof("[health check] addr:%s id:%s ttl:%d heartbeat first time, set redis", addr, id, ttl)
		hbMgr.redisPool.Set(id, Healthy, now, respCh)
		resp := <-respCh
		if resp.Err != nil {
			log.Errorf("[health check] addr:%s id:%s set redis err:%s", addr, id, resp.Err)
			return api.NewInstanceResponse(api.HeartbeatException, instance)
		}
	} else {
		hb.mu.Unlock()
		lastBeatTime := hbInfo.beatTime
		if now == lastBeatTime {
			log.Debugf("[health check] addr:%s id:%s ins heartbeat exceed 1 time/s", addr, id)
			return api.NewInstanceResponse(api.HeartbeatExceedLimit, instance)
		}

		// 修改实例心跳上报时间
		hbInfo.beatTime = now
		hbInfo.ttl = ttl

		// 本机超过1 ttl + 1s未收到心跳，set一次ckv状态
		if now-lastBeatTime >= int64(ttl+1) {
			log.Infof("[health check] addr:%s, id:%s receive heart beat after ttl + 1s, set redis healthy", addr, id)
			hbMgr.redisPool.Set(id, Healthy, now, respCh)
			resp := <-respCh

			if resp.Err != nil {
				log.Errorf("[health check] addr:%s id:%s set redis err:%s", addr, id, resp.Err)
				return api.NewInstanceResponse(api.HeartbeatException, instance)
			}
		}
	}

	// db中实例状态若为不健康，设为健康
	if insCache.Healthy() != true {
		setInsDbStatus(id, addr, Healthy)
	}

	// 将超时检查任务放入时间轮
	task := &TimeWheelTask{now, hbInfo}
	_ = hb.ckvTw.AddTask(time.Duration(ttl+1)*time.Second, task, ckvCallback)

	return api.NewInstanceResponse(api.ExecuteSuccess, instance)
}

// 获取上一次的心跳时间
func (hb *HeartBeatMgr) acquireLastHeartbeat(instance *api.Instance) error {
	id := instance.GetId().GetValue()
	if instance.Metadata == nil {
		instance.Metadata = make(map[string]string)
	}

	// 先获取本地记录的时间，这里可能为空的
	// （该实例不是上报到这台server，可以根据ckv信息获取其上报的server）
	hb.mu.Lock()
	info, ok := hb.hbMap[id]
	hb.mu.Unlock()
	if ok {
		instance.Metadata["last-heartbeat-time"] = time2String(time.Unix(info.beatTime, 0))
		instance.Metadata["system-time"] = time2String(time.Now())
	}

	// 获取ckv记录的时间
	respCh := make(chan *redispool.Resp)
	hbMgr.redisPool.Get(id, respCh)
	resp := <-respCh
	if resp.Err != nil {
		if resp.Err.Error() == RedisNoKeyErr {
			return nil
		}

		log.Errorf("[health check] get id(%s) from redis err: %s", id, resp.Err.Error())
		return resp.Err
	}
	if resp.Local {
		return nil
	}

	res := strings.Split(resp.Value, ":")
	if len(res) != 3 {
		log.Errorf("[health check] id(%s) redis record invalid(%s)", id, resp.Value)
		return errors.New("invalid ckv record")
	}
	tm, err := strconv.ParseInt(res[1], 10, 64)
	if err != nil {
		log.Errorf("[health check] id(%s) redis record heartbeat time(%s) is invalid", id, res[1])
		return err
	}

	// ckv记录的心跳时间与心跳server，
	// 根据这个心跳server可以获取到实例上报到哪台心跳server
	instance.Metadata["ckv-record-healthy"] = res[0]
	instance.Metadata["ckv-record-heartbeat-time"] = time2String(time.Unix(tm, 0))
	instance.Metadata["ckv-record-heartbeat-server"] = res[2]
	return nil
}

/**
* @brief 修改实例状态
		需要打印操作记录
		server 是当前package的全局变量
*/
func setInsDbStatus(id, addr string, status int) {
	log.Infof("[health check] addr:%s id:%s set db status %d", addr, id, status)
	err := server.storage.SetInstanceHealthStatus(id, status, NewUUID())
	if err != nil {
		log.Errorf("[health check] id: %s set db status err:%s", id, err)
		return
	}

	instance := server.caches.Instance().GetInstance(id)
	if instance == nil {
		log.Errorf("[HealthCheck] not found instance(%s)", id)
		return
	}
	service := server.caches.Service().GetServiceByID(instance.ServiceID)
	if service == nil {
		log.Errorf("[HealthCheck] not found serviceID(%s) for instance(%s)",
			instance.ServiceID, id)
		return
	}

	healthStatus := true
	if status == 0 {
		healthStatus = false
	}
	recordInstance := &model.Instance{
		Proto: &api.Instance{
			Host:     instance.Proto.GetHost(),
			Port:     instance.Proto.GetPort(),
			Priority: instance.Proto.GetPriority(),
			Weight:   instance.Proto.GetWeight(),
			Healthy:  utils.NewBoolValue(healthStatus),
			Isolate:  instance.Proto.GetIsolate(),
		},
	}

	server.RecordHistory(instanceRecordEntry(nil, service, recordInstance, model.OUpdate))
}
