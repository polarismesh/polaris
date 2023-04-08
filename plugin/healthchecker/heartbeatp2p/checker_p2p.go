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
	"sync"
	"sync/atomic"
	"time"

	commonhash "github.com/polarismesh/polaris/common/hash"
	commontime "github.com/polarismesh/polaris/common/time"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/plugin"
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
)

// PeerToPeerHealthChecker
type PeerToPeerHealthChecker struct {
	listenPort         int64
	hash               *commonhash.Continuum
	lock               sync.RWMutex
	peers              map[string]*Peer
	refreshPeerTimeSec int64
	suspendTimeSec     int64
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
	c.peers = map[string]*Peer{}
	return nil
}

// Destroy
func (c *PeerToPeerHealthChecker) Destroy() error {
	c.lock.Lock()
	defer c.lock.Unlock()
	for _, peer := range c.peers {
		_ = peer.Close()
	}
	return nil
}

func (r *PeerToPeerHealthChecker) SetCheckerPeers(peers []plugin.CheckerPeer) {
	r.lock.Lock()
	defer r.lock.Unlock()
	atomic.StoreInt64(&r.refreshPeerTimeSec, commontime.CurrentMillisecond())

	var (
		tmp = map[string]plugin.CheckerPeer{}
	)
	for i := range peers {
		peer := peers[i]
		tmp[peer.ID] = peer
	}
	for i := range r.peers {
		if _, ok := tmp[i]; !ok {
			_ = r.peers[i].Close()
		}
	}
	for i := range peers {
		peer := peers[i]
		if _, ok := r.peers[peer.ID]; ok {
			continue
		}
		r.peers[peer.ID] = &Peer{
			ID:    peer.ID,
			Host:  peer.Host,
			Port:  uint32(r.listenPort),
			Local: peer.Host == utils.LocalHost,
		}
		_ = r.peers[peer.ID].Serve()
	}

	// 重新计算 hash
	bucket := map[commonhash.Bucket]bool{}
	for i := range r.peers {
		bucket[commonhash.Bucket{
			Host:   r.peers[i].Host,
			Weight: 100,
		}] = true
	}
	r.hash = commonhash.New(bucket)
	atomic.StoreInt64(&r.refreshPeerTimeSec, 0)
}

// Type for health check plugin, only one same type plugin is allowed
func (c *PeerToPeerHealthChecker) Type() plugin.HealthCheckType {
	return plugin.HealthCheckerHeartbeat
}

// Report process heartbeat info report
func (c *PeerToPeerHealthChecker) Report(request *plugin.ReportRequest) error {
	key := request.InstanceId
	index := c.hash.Hash(commonhash.HashString(key))
	c.lock.RLock()
	responsible := c.peers[index]
	c.lock.RUnlock()

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
	index := c.hash.Hash(commonhash.HashString(key))
	c.lock.RLock()
	responsible := c.peers[index]
	c.lock.RUnlock()

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
	}, nil
}

// AddToCheck add the instances to check procedure
func (c *PeerToPeerHealthChecker) AddToCheck(request *plugin.AddCheckRequest) error {
	return nil
}

// RemoveFromCheck removes the instances from check procedure
func (c *PeerToPeerHealthChecker) RemoveFromCheck(request *plugin.AddCheckRequest) error {
	return nil
}

// Delete delete the id
func (c *PeerToPeerHealthChecker) Delete(id string) error {
	index := c.hash.Hash(commonhash.HashString(id))
	c.lock.RLock()
	responsible := c.peers[index]
	c.lock.RUnlock()

	responsible.Cache.Del(id)
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

const maxCheckDuration = 500 * time.Second

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
	refreshPeerTimeSec := atomic.LoadInt64(&c.refreshPeerTimeSec)
	// redis恢复期，不做变更
	if refreshPeerTimeSec > 0 && localCurTimeSec >= refreshPeerTimeSec &&
		localCurTimeSec-refreshPeerTimeSec < expireDurationSec {
		log.Infof("[Health Check][P2P]health check peers on refresh, "+
			"refreshPeerTimeSec is %d, localCurTimeSec is %d, expireDurationSec is %d, id %s",
			suspendTimeSec, localCurTimeSec, expireDurationSec, key)
		return true
	}
	return false
}
