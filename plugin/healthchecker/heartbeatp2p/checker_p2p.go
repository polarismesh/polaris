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

package heartbeatp2p

import (
	"fmt"
	"sync"
	"sync/atomic"

	commonhash "github.com/polarismesh/polaris/common/hash"
	commontime "github.com/polarismesh/polaris/common/time"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/plugin"
	"go.uber.org/zap"
)

func init() {
	d := &PeerToPeerHealthChecker{}
	plugin.RegisterPlugin(d.Name(), d)
}

// 把操作记录记录到日志文件中
const (
	// PluginName plugin name
	PluginName = "heartbeatP2P"
	// Servers key to manage hb servers
	Servers = "servers"
	// CountSep separator to divide server and count
	Split = "|"
	// DefaultListenPort default p2p checker listen port
	DefaultListenPort = 7000
	// DefaultSoltNum default soltNum of LocalBeatRecordCache
	DefaultSoltNum = 64
)

// PeerToPeerHealthChecker 对等节点心跳健康检查
// 1. PeerToPeerHealthChecker 获取当前 polaris.checker 服务下的所有节点
// 2. peer 之间建立 gRPC 长连接
// 3. PeerToPeerHealthChecker 在处理 Report/Query/Check/Delete 先判断自己处理的心跳节点是否应该由自己负责
//   - 责任节点
//     a. 心跳数据的读写直接写本地 map 内存
//   - 非责任节点
//     a. 心跳写请求通过 gRPC 长连接直接发给对应责任节点
//     b. 心跳读请求通过 gRPC 长连接直接发给对应责任节点，责任节点返回心跳时间戳信息
type PeerToPeerHealthChecker struct {
	// refreshPeerTimeSec last peer list start refresh occur timestamp
	refreshPeerTimeSec int64
	// endRefreshPeerTimeSec last peer list end refresh occur timestamp
	endRefreshPeerTimeSec int64
	// suspendTimeSec healthcheck last suspend timestamp
	suspendTimeSec int64
	// listenPort peer agreement listen gRPC port info
	listenPort int64
	// soltNum BeatRecordCache of segmentMap soltNum
	soltNum int32
	// hash calculate key of responsible peer
	hash *commonhash.Continuum
	// lock
	lock sync.RWMutex
	// peers peer directory
	peers map[string]*Peer
}

// Name
func (c *PeerToPeerHealthChecker) Name() string {
	return PluginName
}

// Initialize
func (c *PeerToPeerHealthChecker) Initialize(configEntry *plugin.ConfigEntry) error {
	listenPort, _ := configEntry.Option["listenPort"].(int64)
	if listenPort == 0 {
		listenPort = DefaultListenPort
	}
	c.listenPort = listenPort
	soltNum, _ := configEntry.Option["soltNum"].(int64)
	if soltNum == 0 {
		soltNum = DefaultSoltNum
	}
	c.soltNum = int32(soltNum)
	c.peers = make(map[string]*Peer)
	return nil
}

// Destroy
func (c *PeerToPeerHealthChecker) Destroy() error {
	c.lock.Lock()
	defer c.lock.Unlock()
	for _, peer := range c.peers {
		_ = peer.Close()
	}
	c.hash = nil
	c.peers = map[string]*Peer{}
	return nil
}

// SetCheckerPeers
func (c *PeerToPeerHealthChecker) SetCheckerPeers(checkerPeers []plugin.CheckerPeer) {
	c.lock.Lock()
	defer c.lock.Unlock()

	log.Info("[HealthCheck][P2P] receive checker peers change", zap.Any("peers", checkerPeers))
	atomic.StoreInt64(&c.refreshPeerTimeSec, commontime.CurrentMillisecond())
	c.refreshPeers(checkerPeers)
	c.servePeers()
	c.calculateContinuum()
	atomic.StoreInt64(&c.endRefreshPeerTimeSec, commontime.CurrentMillisecond())
	log.Info("[HealthCheck][P2P] end checker peers change", zap.Any("peers", c.peers))
}

func (c *PeerToPeerHealthChecker) refreshPeers(checkerPeers []plugin.CheckerPeer) {
	tmp := map[string]plugin.CheckerPeer{}
	for i := range checkerPeers {
		peer := checkerPeers[i]
		tmp[peer.ID] = peer
	}
	for i := range c.peers {
		if _, ok := tmp[i]; !ok {
			_ = c.peers[i].Close()
		}
	}
	for i := range checkerPeers {
		checkerPeer := checkerPeers[i]
		if _, ok := c.peers[checkerPeer.ID]; ok {
			continue
		}
		port := checkerPeer.Port
		if port == 0 {
			port = uint32(c.listenPort)
		}

		c.peers[checkerPeer.ID] = &Peer{
			ID:    checkerPeer.ID,
			Host:  checkerPeer.Host,
			Port:  port,
			Local: checkerPeer.Host == utils.LocalHost,
		}
	}
}

func (c *PeerToPeerHealthChecker) servePeers() {
	serveFunc := func(peers map[string]*Peer, filter func(*Peer) bool) {
		for i := range c.peers {
			peer := c.peers[i]
			if !filter(peer) {
				continue
			}
			if err := peer.Serve(c.soltNum); err != nil {
				log.Error("[HealthCheck][P2P] peer serve fail", zap.String("host", peer.Host),
					zap.Uint32("port", peer.Port), zap.Bool("local", peer.Local), zap.Error(err))
			}
		}
	}

	// 启动 local peer
	serveFunc(c.peers, func(p *Peer) bool {
		return p.Local
	})

	// 启动 remote peer
	serveFunc(c.peers, func(p *Peer) bool {
		return !p.Local
	})
}

func (c *PeerToPeerHealthChecker) calculateContinuum() {
	// 重新计算 hash
	bucket := map[commonhash.Bucket]bool{}
	for i := range c.peers {
		peer := c.peers[i]
		bucket[commonhash.Bucket{
			Host:   peer.ID,
			Weight: 100,
		}] = true
	}
	c.hash = commonhash.New(bucket)
}

// Type for health check plugin, only one same type plugin is allowed
func (c *PeerToPeerHealthChecker) Type() plugin.HealthCheckType {
	return plugin.HealthCheckerHeartbeat
}

// Report process heartbeat info report
func (c *PeerToPeerHealthChecker) Report(request *plugin.ReportRequest) error {
	key := request.InstanceId
	responsible, ok := c.findResponsiblePeer(key)
	if !ok {
		return fmt.Errorf("write key:%s not found responsible peer", key)
	}

	record := WriteBeatRecord{
		Record: RecordValue{
			Server:     responsible.Host,
			CurTimeSec: request.CurTimeSec,
			Count:      request.Count,
		},
		Key: key,
	}
	responsible.Cache.Put(record)
	log.Debugf("[HealthCheck][P2P] add hb record, instanceId %s, record %+v", request.InstanceId, record)
	return nil
}

// Check process the instance check
// 大部分情况下，Check 的检查都是在本节点进行处理，只有出现 Refresh 节点时才可能存在将 CheckRequest 请求转发相应的对等节点
func (c *PeerToPeerHealthChecker) Check(request *plugin.CheckRequest) (*plugin.CheckResponse, error) {
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
func (c *PeerToPeerHealthChecker) Query(request *plugin.QueryRequest) (*plugin.QueryResponse, error) {
	key := request.InstanceId
	responsible, ok := c.findResponsiblePeer(key)
	if !ok {
		return nil, fmt.Errorf("query key:%s not found responsible peer", key)
	}

	ret := responsible.Cache.Get(key)
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
// NOTE: not support in PeerToPeerHealthChecker
func (c *PeerToPeerHealthChecker) AddToCheck(request *plugin.AddCheckRequest) error {
	return nil
}

// RemoveFromCheck removes the instances from check procedure
// NOTE: not support in PeerToPeerHealthChecker
func (c *PeerToPeerHealthChecker) RemoveFromCheck(request *plugin.AddCheckRequest) error {
	return nil
}

// Delete delete record by key
func (c *PeerToPeerHealthChecker) Delete(key string) error {
	responsible, ok := c.findResponsiblePeer(key)
	if !ok {
		return fmt.Errorf("delete key:%s not found responsible peer", key)
	}
	responsible.Cache.Del(key)
	return nil
}

// Suspend checker for an entire expired interval
func (c *PeerToPeerHealthChecker) Suspend() {
	curTimeMilli := commontime.CurrentMillisecond() / 1000
	log.Infof("[Health Check][P2P] suspend checker, start time %d", curTimeMilli)
	atomic.StoreInt64(&c.suspendTimeSec, curTimeMilli)
}

// SuspendTimeSec get suspend time in seconds
func (c *PeerToPeerHealthChecker) SuspendTimeSec() int64 {
	return atomic.LoadInt64(&c.suspendTimeSec)
}

func (c *PeerToPeerHealthChecker) findResponsiblePeer(key string) (*Peer, bool) {
	index := c.hash.Hash(commonhash.HashString(key))
	c.lock.RLock()
	defer c.lock.RUnlock()
	responsible, ok := c.peers[index]
	return responsible, ok
}

func (c *PeerToPeerHealthChecker) skipCheck(key string, expireDurationSec int64) bool {
	suspendTimeSec := c.SuspendTimeSec()
	localCurTimeSec := commontime.CurrentMillisecond() / 1000
	if suspendTimeSec > 0 && localCurTimeSec >= suspendTimeSec &&
		localCurTimeSec-suspendTimeSec < expireDurationSec {
		log.Infof("[Health Check][P2P]health check peers suspended, "+
			"suspendTimeSec is %d, localCurTimeSec is %d, expireDurationSec is %d, id %s",
			suspendTimeSec, localCurTimeSec, expireDurationSec, key)
		return true
	}

	// 当 peers 列表出现刷新时，key 的存在性有一下几种情况
	// case 1: key hash 之后 responsible peer 不变
	// 			这种情况下，不会出现心跳数据找不到的情况，假设 T1 时刻开始出现 peer 列表变化，到 T2 时刻变化结束
	// 			那么在 T1 时刻之前，key 的 responsible peer 为 P1，T1～T2 期间，各个节点的最终 peers 列表可能不一致，
	// 			但是只会存在两种情况的 peers 列表，即 T1 时刻以及 T2 时刻，而这两个时刻 key 的 responsible 均为 P1.
	// 			因此 Report、Query、Check、Del 请求均可以正常路由到 P1 节点
	// case 2: key hash 之后 responsible peer 变
	// 			这种情况下，会出现心跳数据找不到的情况，假设 T1 时刻开始出现 peer 列表变化，到 T2 时刻变化结束
	// 			那么在 T1 时刻之前，key 的 responsible peer 为 P1，T2 时刻 key 的 responsible peer 为 P2
	// 			则 T2 时刻开始，针对每一个实例来说，最多有一个 TTL 的周期查询不到心跳数据，当 peers 列表变更完之后，
	// 			在 1TTL 之后实例心跳概率存在，2TTL 之后实例心跳肯定存在
	refreshPeerTimeSec := c.getRefreshPeerTimeSec()
	endRefreshPeerTimeSec := c.getEndRefreshPeerTimeSec()
	if endRefreshPeerTimeSec > 0 && localCurTimeSec >= refreshPeerTimeSec &&
		localCurTimeSec-endRefreshPeerTimeSec < expireDurationSec {
		log.Infof("[Health Check][P2P]health check peers on refresh, "+
			"refreshPeerTimeSec is %d, localCurTimeSec is %d, expireDurationSec is %d, id %s",
			suspendTimeSec, localCurTimeSec, expireDurationSec, key)
		return true
	}
	return false
}

func (c *PeerToPeerHealthChecker) getEndRefreshPeerTimeSec() int64 {
	return atomic.LoadInt64(&c.endRefreshPeerTimeSec)
}

func (c *PeerToPeerHealthChecker) getRefreshPeerTimeSec() int64 {
	return atomic.LoadInt64(&c.refreshPeerTimeSec)
}
