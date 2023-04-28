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

package leader

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"

	"go.uber.org/zap"

	"github.com/polarismesh/polaris/common/eventhub"
	commontime "github.com/polarismesh/polaris/common/time"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/plugin"
	"github.com/polarismesh/polaris/store"
)

func init() {
	d := &LeaderHealthChecker{}
	plugin.RegisterPlugin(d.Name(), d)
}

const (
	// PluginName plugin name
	PluginName = "heartbeatLeader"
	// Servers key to manage hb servers
	Servers = "servers"
	// CountSep separator to divide server and count
	Split = "|"
	// optionSoltNum option key of soltNum
	optionSoltNum = "soltNum"
	// electionKey use election key
	electionKey = store.ElectionKeySelfServiceChecker
	// subscriberName eventhub subscriber name
	subscriberName = PluginName
	// uninitializeSignal .
	uninitializeSignal = int32(0)
	// initializedSignal .
	initializedSignal = int32(1)
)

var (
	// DefaultSoltNum default soltNum of LocalBeatRecordCache
	DefaultSoltNum = int32(runtime.GOMAXPROCS(0) * 16)
)

// LeaderHealthChecker Leader~Follower 节点心跳健康检查
// 1. 监听 LeaderChangeEvent 事件，
// 2. LeaderHealthChecker 启动时先根据 store 层的 LeaderElection 选举能力选出一个 Leader
// 3. Leader 和 Follower 之间建立 gRPC 长连接
// 4. LeaderHealthChecker 在处理 Report/Query/Check/Delete 先判断自己是否为 Leader
//   - Leader 节点
//     a. 心跳数据的读写直接写本地 map 内存
//   - 非 Leader 节点
//     a. 心跳写请求通过 gRPC 长连接直接发给 Leader 节点
//     b. 心跳读请求通过 gRPC 长连接直接发给 Leader 节点，Leader 节点返回心跳时间戳信息
type LeaderHealthChecker struct {
	initialize int32
	// leaderChangeTimeSec last peer list start refresh occur timestamp
	leaderChangeTimeSec int64
	// suspendTimeSec healthcheck last suspend timestamp
	suspendTimeSec int64
	// conf leaderChecker config
	conf *Config
	// lock keeps safe to change leader info
	lock sync.RWMutex
	// leader leader signal
	leader int32
	// remote remote peer info
	remote Peer
	// self self peer info
	self Peer
	// s store.Store
	s store.Store
}

// Name .
func (c *LeaderHealthChecker) Name() string {
	return PluginName
}

// Initialize .
func (c *LeaderHealthChecker) Initialize(entry *plugin.ConfigEntry) error {
	conf, err := Unmarshal(entry.Option)
	if err != nil {
		return err
	}
	c.conf = conf
	c.self = NewLocalPeerFunc()
	c.self.Initialize(*conf)
	if err := c.self.Serve(context.Background(), "", 0); err != nil {
		return err
	}
	if err := eventhub.Subscribe(eventhub.LeaderChangeEventTopic, subscriberName, c); err != nil {
		return err
	}
	storage, err := store.GetStore()
	if err != nil {
		return err
	}
	c.s = storage
	if err := c.s.StartLeaderElection(electionKey); err != nil {
		return err
	}
	runIfDebugEnable(c)
	return nil
}

// PreProcess do preprocess logic for event
func (c *LeaderHealthChecker) PreProcess(ctx context.Context, value any) any {
	return value
}

// OnEvent event trigger
func (c *LeaderHealthChecker) OnEvent(ctx context.Context, i interface{}) error {
	e := i.(store.LeaderChangeEvent)
	if e.Key != electionKey {
		return nil
	}

	c.lock.Lock()
	defer c.lock.Unlock()
	atomic.StoreInt32(&c.initialize, uninitializeSignal)
	if e.Leader {
		c.becomeLeader()
	} else {
		c.becomeFollower()
	}
	c.refreshLeaderChangeTimeSec()
	return nil
}

func (c *LeaderHealthChecker) becomeLeader() {
	if c.remote != nil {
		if log.DebugEnabled() {
			log.Debug("[HealthCheck][Leader] become leader and close old leader",
				zap.String("leader", c.remote.Host()))
		}
		// 关闭原来自己跟随的 leader 节点信息
		oldLeader := c.remote
		c.remote = nil
		_ = oldLeader.Close()
	}
	// leader 指向自己
	atomic.StoreInt32(&c.leader, 1)
	atomic.StoreInt32(&c.initialize, initializedSignal)
	log.Info("[HealthCheck][Leader] self become leader")
}

func (c *LeaderHealthChecker) becomeFollower() {
	elections, err := c.s.ListLeaderElections()
	if err != nil {
		log.Error("[HealthCheck][Leader] follower list elections", zap.Error(err))
		return
	}
	for i := range elections {
		election := elections[i]
		if election.ElectKey != electionKey {
			continue
		}
		if election.Host == "" || election.Host == utils.LocalHost {
			atomic.StoreInt32(&c.leader, 0)
			atomic.StoreInt32(&c.initialize, initializedSignal)
			return
		}
		log.Info("[HealthCheck][Leader] self become follower")
		if c.remote != nil {
			// leader 未发生变化
			if election.Host == c.remote.Host() {
				atomic.StoreInt32(&c.initialize, initializedSignal)
				return
			}
			// leader 出现变更切换
			if election.Host != c.remote.Host() {
				log.Info("[HealthCheck][Leader] become follower and leader change",
					zap.String("leader", election.Host))
				// 关闭原来的 leader 节点信息
				oldLeader := c.remote
				c.remote = nil
				_ = oldLeader.Close()
			}
		}
		remoteLeader := NewRemotePeerFunc()
		remoteLeader.Initialize(*c.conf)
		if err := remoteLeader.Serve(context.Background(), election.Host, uint32(utils.LocalPort)); err != nil {
			log.Error("[HealthCheck][Leader] follower run serve", zap.Error(err))
			return
		}
		c.remote = remoteLeader
		atomic.StoreInt32(&c.leader, 0)
		atomic.StoreInt32(&c.initialize, initializedSignal)
		return
	}
}

// Destroy .
func (c *LeaderHealthChecker) Destroy() error {
	eventhub.Unsubscribe(eventhub.LeaderChangeEventTopic, subscriberName)
	return nil
}

// Type for health check plugin, only one same type plugin is allowed
func (c *LeaderHealthChecker) Type() plugin.HealthCheckType {
	return plugin.HealthCheckerHeartbeat
}

// Report process heartbeat info report
func (c *LeaderHealthChecker) Report(request *plugin.ReportRequest) error {
	c.lock.RLock()
	defer c.lock.RUnlock()
	if !c.isInitialize() {
		log.Warn("[Health Check][Leader] leader checker uninitialize, ignore report")
		return nil
	}
	responsible := c.findLeaderPeer()
	record := WriteBeatRecord{
		Record: RecordValue{
			Server:     responsible.Host(),
			CurTimeSec: request.CurTimeSec,
			Count:      request.Count,
		},
		Key: request.InstanceId,
	}
	if err := responsible.Put(record); err != nil {
		return err
	}
	if log.DebugEnabled() {
		log.Debugf("[HealthCheck][Leader] add hb record, instanceId %s, record %+v", request.InstanceId, record)
	}
	return nil
}

// Check process the instance check
func (c *LeaderHealthChecker) Check(request *plugin.CheckRequest) (*plugin.CheckResponse, error) {
	queryResp, err := c.Query(&request.QueryRequest)
	if err != nil {
		return nil, err
	}
	lastHeartbeatTime := queryResp.LastHeartbeatSec
	checkResp := &plugin.CheckResponse{
		LastHeartbeatTimeSec: lastHeartbeatTime,
	}
	curTimeSec := request.CurTimeSec()
	if c.skipCheck(request.InstanceId, int64(request.ExpireDurationSec)) {
		checkResp.StayUnchanged = true
		return checkResp, nil
	}
	if log.DebugEnabled() {
		log.Debug("[HealthCheck][Leader] check hb record", zap.String("id", request.InstanceId),
			zap.Int64("curTimeSec", curTimeSec), zap.Int64("lastHeartbeatTime", lastHeartbeatTime))
	}
	if curTimeSec > lastHeartbeatTime {
		if curTimeSec-lastHeartbeatTime >= int64(request.ExpireDurationSec) {
			// 心跳超时
			checkResp.Healthy = false
			if request.Healthy {
				log.Infof("[Health Check][Leader] health check expired, "+
					"last hb timestamp is %d, curTimeSec is %d, expireDurationSec is %d, instanceId %s",
					lastHeartbeatTime, curTimeSec, request.ExpireDurationSec, request.InstanceId)
			} else {
				checkResp.StayUnchanged = true
			}
			return checkResp, nil
		}
	}
	checkResp.Healthy = true
	if !request.Healthy {
		log.Infof("[Health Check][Leader] health check resumed, "+
			"last hb timestamp is %d, curTimeSec is %d, expireDurationSec is %d instanceId %s",
			lastHeartbeatTime, curTimeSec, request.ExpireDurationSec, request.InstanceId)
	} else {
		checkResp.StayUnchanged = true
	}

	return checkResp, nil
}

// Query queries the heartbeat time
func (c *LeaderHealthChecker) Query(request *plugin.QueryRequest) (*plugin.QueryResponse, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	if !c.isInitialize() {
		log.Infof("[Health Check][Leader] leader checker uninitialize, ignore query")
		return &plugin.QueryResponse{
			LastHeartbeatSec: 0,
		}, nil
	}
	responsible := c.findLeaderPeer()
	record, err := responsible.Get(request.InstanceId)
	if err != nil {
		return nil, err
	}
	if log.DebugEnabled() {
		log.Debugf("[HealthCheck][Leader] query hb record, instanceId %s, record %+v", request.InstanceId, record)
	}
	return &plugin.QueryResponse{
		Server:           responsible.Host(),
		LastHeartbeatSec: record.Record.CurTimeSec,
		Count:            record.Record.Count,
		Exists:           record.Exist,
	}, nil
}

// AddToCheck add the instances to check procedure
// NOTE: not support in LeaderHealthChecker
func (c *LeaderHealthChecker) AddToCheck(request *plugin.AddCheckRequest) error {
	return nil
}

// RemoveFromCheck removes the instances from check procedure
// NOTE: not support in LeaderHealthChecker
func (c *LeaderHealthChecker) RemoveFromCheck(request *plugin.AddCheckRequest) error {
	return nil
}

// Delete delete record by key
func (c *LeaderHealthChecker) Delete(key string) error {
	c.lock.RLock()
	defer c.lock.RUnlock()
	responsible := c.findLeaderPeer()
	return responsible.Del(key)
}

// Suspend checker for an entire expired interval
func (c *LeaderHealthChecker) Suspend() {
	curTimeMilli := commontime.CurrentMillisecond() / 1000
	log.Infof("[Health Check][Leader] suspend checker, start time %d", curTimeMilli)
	atomic.StoreInt64(&c.suspendTimeSec, curTimeMilli)
}

// SuspendTimeSec get suspend time in seconds
func (c *LeaderHealthChecker) SuspendTimeSec() int64 {
	return atomic.LoadInt64(&c.suspendTimeSec)
}

func (c *LeaderHealthChecker) findLeaderPeer() Peer {
	if c.isLeader() {
		return c.self
	}
	return c.remote
}

func (c *LeaderHealthChecker) skipCheck(key string, expireDurationSec int64) bool {
	// 如果没有初始化，则忽略检查
	if !c.isInitialize() {
		log.Infof("[Health Check][Leader] leader checker uninitialize, ensure skip check")
		return true
	}

	suspendTimeSec := c.SuspendTimeSec()
	localCurTimeSec := commontime.CurrentMillisecond() / 1000
	if suspendTimeSec > 0 && localCurTimeSec >= suspendTimeSec &&
		localCurTimeSec-suspendTimeSec < expireDurationSec {
		log.Infof("[Health Check][Leader] health check peers suspended, "+
			"suspendTimeSec is %d, localCurTimeSec is %d, expireDurationSec is %d, id %s",
			suspendTimeSec, localCurTimeSec, expireDurationSec, key)
		return true
	}

	// 当 T1 时刻出现 Leader 节点切换，到 T2 时刻 Leader 节点切换成，在这期间，可能会出现以下情况
	// case 1: T1~T2 时刻不存在 Leader
	// case 2: T1～T2 时刻存在多个 Leader
	leaderChangeTimeSec := c.LeaderChangeTimeSec()
	if leaderChangeTimeSec > 0 && localCurTimeSec >= leaderChangeTimeSec &&
		localCurTimeSec-leaderChangeTimeSec < expireDurationSec {
		log.Infof("[Health Check][Leader] health check peers on refresh, "+
			"refreshPeerTimeSec is %d, localCurTimeSec is %d, expireDurationSec is %d, id %s",
			suspendTimeSec, localCurTimeSec, expireDurationSec, key)
		return true
	}
	return false
}

func (c *LeaderHealthChecker) refreshLeaderChangeTimeSec() {
	atomic.StoreInt64(&c.leaderChangeTimeSec, commontime.CurrentMillisecond()/1000)
}

func (c *LeaderHealthChecker) LeaderChangeTimeSec() int64 {
	return atomic.LoadInt64(&c.leaderChangeTimeSec)
}

func (c *LeaderHealthChecker) isInitialize() bool {
	return atomic.LoadInt32(&c.initialize) == initializedSignal
}

func (c *LeaderHealthChecker) isLeader() bool {
	return atomic.LoadInt32(&c.leader) == 1
}

func (c *LeaderHealthChecker) DebugHandlers() []plugin.DebugHandler {
	return runIfDebugEnable(c)
}
