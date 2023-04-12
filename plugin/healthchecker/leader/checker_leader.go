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
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/polarismesh/polaris/common/eventhub"
	commontime "github.com/polarismesh/polaris/common/time"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/plugin"
	"github.com/polarismesh/polaris/store"
	"go.uber.org/zap"
)

func init() {
	d := &LeaderHealthChecker{}
	plugin.RegisterPlugin(d.Name(), d)
}

// 把操作记录记录到日志文件中
const (
	// PluginName plugin name
	PluginName = "heartbeatLeader"
	// Servers key to manage hb servers
	Servers = "servers"
	// CountSep separator to divide server and count
	Split = "|"
	// DefaultListenPort default leader checker listen port
	DefaultListenPort = 8100
	// DefaultListenIP default leader checker listen ip
	DefaultListenIP = "0.0.0.0"
	// DefaultSoltNum default soltNum of LocalBeatRecordCache
	DefaultSoltNum = 64
	// optionListenPort option key of listenPort
	optionListenPort = "listenPort"
	// optionListenIP option key of listenIP
	optionListenIP = "listenIP"
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

// LeaderHealthChecker 对等节点心跳健康检查
// 1. LeaderHealthChecker 启动时先根据 store 层的 LeaderElection 选举能力选出一个 Leader
// 2. 监听 LeaderChangeEvent 事件，
// 2. Leader 和 Follower 之间建立 gRPC 长连接
// 3. LeaderHealthChecker 在处理 Report/Query/Check/Delete 先判断自己是否为 Leader
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
	// lock keep save to change leader info
	lock sync.RWMutex
	// peers peer directory
	leader *Peer
	// s store.Store
	s store.Store
	// cancel .
	cancel context.CancelFunc
}

// Name
func (c *LeaderHealthChecker) Name() string {
	return PluginName
}

// Initialize
func (c *LeaderHealthChecker) Initialize(entry *plugin.ConfigEntry) error {
	conf, err := Unmarshal(entry.Option)
	if err != nil {
		return err
	}
	c.conf = conf
	storage, err := store.GetStore()
	if err != nil {
		return err
	}
	c.s = storage
	if err := c.s.StartLeaderElection(electionKey); err != nil {
		return err
	}
	eventhub.Subscribe(eventhub.LeaderChangeEventTopic, subscriberName, c)
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
	return nil
}

func (c *LeaderHealthChecker) becomeLeader() {
	if c.leader != nil {
		log.Error("[HealthCheck][Leader] becomd leader, close old leader")
		// 关闭原来的 leader 节点信息
		oldLeader := c.leader
		c.leader = nil
		_ = oldLeader.Close()
	}
	localLeader := &Peer{
		ID:     fmt.Sprintf("%s:%d", utils.LocalHost, c.conf.ListenPort),
		Host:   utils.LocalHost,
		Port:   uint32(c.conf.ListenPort),
		Leader: true,
	}
	if err := localLeader.Serve(c.conf.SoltNum, c.conf.ListenIP, c.conf.Batch); err != nil {
		log.Error("[HealthCheck][Leader] leader run serve", zap.Error(err))
		if err = c.s.ReleaseLeaderElection(electionKey); err != nil {
			log.Error("[HealthCheck][Leader] leader release self election", zap.Error(err))
		}
		return
	}
	c.leader = localLeader
	atomic.StoreInt32(&c.initialize, initializedSignal)
}

func (c *LeaderHealthChecker) becomeFollower() {
	if c.leader != nil {
		log.Error("[HealthCheck][Leader] becomd follower, close old leader")
		// 关闭原来的 leader 节点信息
		oldLeader := c.leader
		c.leader = nil
		_ = oldLeader.Close()
	}
	elections, err := c.s.ListLeaderElections()
	if err != nil {
		log.Error("[HealthCheck][Leader] follower list elections", zap.Error(err))
		return
	}
	for i := range elections {
		election := elections[i]
		if election.ElectKey == electionKey {
			if election.Host == "" {
				return
			}
			remoteLeader := &Peer{
				ID:     fmt.Sprintf("%s:%d", election.Host, c.conf.ListenPort),
				Host:   election.Host,
				Port:   uint32(c.conf.ListenPort),
				Leader: false,
			}
			if err := remoteLeader.Serve(c.conf.SoltNum, c.conf.ListenIP, c.conf.Batch); err != nil {
				log.Error("[HealthCheck][Leader] follower run serve", zap.Error(err))
				break
			}
			c.leader = remoteLeader
			atomic.StoreInt32(&c.initialize, initializedSignal)
			break
		}
	}
}

// Destroy
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
	key := request.InstanceId
	responsible := c.finLeaderPeer()
	record := WriteBeatRecord{
		Record: RecordValue{
			Server:     responsible.Host,
			CurTimeSec: request.CurTimeSec,
			Count:      request.Count,
		},
		Key: key,
	}
	if err := responsible.Put(record); err != nil {
		return err
	}
	log.Debugf("[HealthCheck][Leader] add hb record, instanceId %s, record %+v", request.InstanceId, record)
	return nil
}

// Check process the instance check
// 大部分情况下，Check 的检查都是在本节点进行处理，只有出现 Refresh 节点时才可能存在将 CheckRequest 请求转发相应的对等节点
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
	log.Debugf("[HealthCheck][Leader] check hb record, cur is %d, last is %d", curTimeSec, lastHeartbeatTime)
	if c.skipCheck(request.InstanceId, int64(request.ExpireDurationSec)) {
		checkResp.StayUnchanged = true
		return checkResp, nil
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
	responsible := c.finLeaderPeer()

	key := request.InstanceId
	ret, err := responsible.Get(key)
	if err != nil {
		return nil, err
	}
	record, ok := ret[key]
	if !ok {
		return &plugin.QueryResponse{
			LastHeartbeatSec: 0,
		}, nil
	}
	log.Debugf("[HealthCheck][Leader] query hb record, instanceId %s, record %+v", request.InstanceId, record)
	return &plugin.QueryResponse{
		Server:           responsible.Host,
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
	responsible := c.finLeaderPeer()
	responsible.Del(key)
	return nil
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

func (c *LeaderHealthChecker) finLeaderPeer() *Peer {
	return c.leader
}

func (c *LeaderHealthChecker) skipCheck(key string, expireDurationSec int64) bool {
	// 如果没有初始化，则忽略检查
	if !c.isInitialize() {
		log.Infof("[Health Check][Leader] leader checker uninitialize, ignore check")
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
	// case 1: T1~T2 时刻不存在 Leader，这种情况利用
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

func (c *LeaderHealthChecker) LeaderChangeTimeSec() int64 {
	return atomic.LoadInt64(&c.leaderChangeTimeSec)
}

func (c *LeaderHealthChecker) isInitialize() bool {
	return atomic.LoadInt32(&c.initialize) == initializedSignal
}
