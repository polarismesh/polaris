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
	// DefaultListenPort default p2p checker listen port
	DefaultListenPort = 8100
	// DefaultSoltNum default soltNum of LocalBeatRecordCache
	DefaultSoltNum = 64
)

// LeaderHealthChecker 对等节点心跳健康检查
// 1. LeaderHealthChecker 获取当前 polaris.checker 服务下的所有节点
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
	// listenPort peer agreement listen gRPC port info
	listenPort int64
	// soltNum BeatRecordCache of segmentMap soltNum
	soltNum int32
	// peers peer directory
	leader *Peer
	s      store.Store
	cancel context.CancelFunc
}

// Name
func (c *LeaderHealthChecker) Name() string {
	return PluginName
}

// Initialize
func (c *LeaderHealthChecker) Initialize(configEntry *plugin.ConfigEntry) error {
	listenPort, _ := configEntry.Option["listenPort"].(int)
	if listenPort == 0 {
		listenPort = DefaultListenPort
	}
	c.listenPort = int64(listenPort)
	soltNum, _ := configEntry.Option["soltNum"].(int)
	if soltNum == 0 {
		soltNum = DefaultSoltNum
	}
	c.soltNum = int32(soltNum)
	storage, err := store.GetStore()
	if err != nil {
		return err
	}
	c.s = storage
	if err := c.s.StartLeaderElection(PluginName); err != nil {
		return err
	}
	eventhub.Subscribe(eventhub.LeaderChangeEventTopic, PluginName, c)
	return nil
}

// PreProcess do preprocess logic for event
func (c *LeaderHealthChecker) PreProcess(ctx context.Context, value any) any {
	return value
}

// OnEvent event trigger
func (c *LeaderHealthChecker) OnEvent(ctx context.Context, i interface{}) error {
	e := i.(store.LeaderChangeEvent)
	if e.Key != PluginName {
		return nil
	}

	if e.Leader {
		c.becomeLeader()
	} else {
		c.becomeFollower()
	}
	return nil
}

func (c *LeaderHealthChecker) becomeLeader() {
	if c.leader != nil {
		// 关闭原来的 leader 节点信息
		_ = c.leader.Close()
	}
	localLeader := &Peer{
		ID:     fmt.Sprintf("%s:%d", utils.LocalHost, c.listenPort),
		Host:   utils.LocalHost,
		Port:   uint32(c.listenPort),
		Leader: true,
	}
	if err := localLeader.Serve(c.soltNum); err != nil {
		log.Error("leader run serve", zap.Error(err))
		if err = c.s.ReleaseLeaderElection(PluginName); err != nil {
			log.Error("leader release self election", zap.Error(err))
		}
		return
	}
	c.leader = localLeader
}

func (c *LeaderHealthChecker) becomeFollower() {
	if c.leader != nil {
		// 关闭原来的 leader 节点信息
		_ = c.leader.Close()
	}
	elections, err := c.s.ListLeaderElections()
	if err != nil {
		log.Error("follower list elections", zap.Error(err))
		return
	}
	for i := range elections {
		election := elections[i]
		if election.ElectKey == PluginName {
			remoteLeader := &Peer{
				ID:     fmt.Sprintf("%s:%d", election.Host, c.listenPort),
				Host:   election.Host,
				Port:   uint32(c.listenPort),
				Leader: false,
			}
			if err := remoteLeader.Serve(c.soltNum); err != nil {
				log.Error("follower run serve", zap.Error(err))
				break
			}
			c.leader = remoteLeader
			break
		}
	}
}

// Destroy
func (c *LeaderHealthChecker) Destroy() error {
	eventhub.Unsubscribe(eventhub.LeaderChangeEventTopic, PluginName)
	return nil
}

// SetCheckerPeers
func (c *LeaderHealthChecker) SetCheckerPeers(checkerPeers []plugin.CheckerPeer) {

}

// Type for health check plugin, only one same type plugin is allowed
func (c *LeaderHealthChecker) Type() plugin.HealthCheckType {
	return plugin.HealthCheckerHeartbeat
}

// Report process heartbeat info report
func (c *LeaderHealthChecker) Report(request *plugin.ReportRequest) error {
	if !c.isInitialize() {
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
	future := responsible.putBatchCtrl.Submit(record)
	_, err := future.Done()
	if err != nil {
		return err
	}
	log.Debugf("[HealthCheck][P2P] add hb record, instanceId %s, record %+v", request.InstanceId, record)
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
	log.Debugf("[HealthCheck][P2P] check hb record, cur is %d, last is %d", curTimeSec, lastHeartbeatTime)
	if c.skipCheck(request.InstanceId, int64(request.ExpireDurationSec)) {
		checkResp.StayUnchanged = true
		return checkResp, nil
	}
	if curTimeSec > lastHeartbeatTime {
		if curTimeSec-lastHeartbeatTime >= int64(request.ExpireDurationSec) {
			// 心跳超时
			checkResp.Healthy = false
			if request.Healthy {
				log.Infof("[Health Check][P2P] health check expired, "+
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
		log.Infof("[Health Check][P2P] health check resumed, "+
			"last hb timestamp is %d, curTimeSec is %d, expireDurationSec is %d instanceId %s",
			lastHeartbeatTime, curTimeSec, request.ExpireDurationSec, request.InstanceId)
	} else {
		checkResp.StayUnchanged = true
	}

	return checkResp, nil
}

// Query queries the heartbeat time
func (c *LeaderHealthChecker) Query(request *plugin.QueryRequest) (*plugin.QueryResponse, error) {
	if !c.isInitialize() {
		return &plugin.QueryResponse{
			LastHeartbeatSec: 0,
		}, nil
	}
	responsible := c.finLeaderPeer()

	key := request.InstanceId
	future := responsible.getBatchCtrl.Submit(key)
	resp, err := future.Done()
	if err != nil {
		return nil, err
	}
	ret := resp.(map[string]*ReadBeatRecord)
	record, ok := ret[key]
	if !ok {
		return &plugin.QueryResponse{
			LastHeartbeatSec: 0,
		}, nil
	}
	log.Debugf("[HealthCheck][P2P] query hb record, instanceId %s, record %+v", request.InstanceId, record)
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
	responsible := c.finLeaderPeer()
	responsible.Cache.Del(key)
	return nil
}

// Suspend checker for an entire expired interval
func (c *LeaderHealthChecker) Suspend() {
	curTimeMilli := commontime.CurrentMillisecond() / 1000
	log.Infof("[Health Check][P2P] suspend checker, start time %d", curTimeMilli)
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
		return true
	}

	suspendTimeSec := c.SuspendTimeSec()
	localCurTimeSec := commontime.CurrentMillisecond() / 1000
	if suspendTimeSec > 0 && localCurTimeSec >= suspendTimeSec &&
		localCurTimeSec-suspendTimeSec < expireDurationSec {
		log.Infof("[Health Check][P2P]health check peers suspended, "+
			"suspendTimeSec is %d, localCurTimeSec is %d, expireDurationSec is %d, id %s",
			suspendTimeSec, localCurTimeSec, expireDurationSec, key)
		return true
	}

	// 当 T1 时刻出现 Leader 节点切换，到 T2 时刻 Leader 节点切换成，在这期间，可能会出现
	leaderChangeTimeSec := c.LeaderChangeTimeSec()
	if leaderChangeTimeSec > 0 && localCurTimeSec >= leaderChangeTimeSec &&
		localCurTimeSec-leaderChangeTimeSec < expireDurationSec {
		log.Infof("[Health Check][P2P]health check peers on refresh, "+
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
	return atomic.LoadInt32(&c.initialize) == 1
}
