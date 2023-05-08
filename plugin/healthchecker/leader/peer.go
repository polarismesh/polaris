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
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"

	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	"github.com/polarismesh/polaris/common/batchjob"
	commonhash "github.com/polarismesh/polaris/common/hash"
	"github.com/polarismesh/polaris/common/utils"
)

var (
	NewLocalPeerFunc  = newLocalPeer
	NewRemotePeerFunc = newRemotePeer
)

func newLocalPeer() Peer {
	return &LocalPeer{}
}

func newRemotePeer() Peer {
	return &RemotePeer{}
}

// Peer peer
type Peer interface {
	// Initialize .
	Initialize(conf Config)
	// Serve .
	Serve(ctx context.Context, listenIP string, listenPort uint32) error
	// Get .
	Get(key string) (*ReadBeatRecord, error)
	// Put .
	Put(record WriteBeatRecord) error
	// Del .
	Del(key string) error
	// Close .
	Close() error
	// Host .
	Host() string
	// Storage
	Storage() BeatRecordCache
}

// LocalPeer Heartbeat data storage node
type LocalPeer struct {
	once sync.Once
	// Cache data storage
	Cache BeatRecordCache
	// cancel .
	cancel context.CancelFunc
}

func (p *LocalPeer) Initialize(conf Config) {
	p.Cache = newLocalBeatRecordCache(conf.SoltNum, commonhash.Fnv32)
}

func (p *LocalPeer) Serve(ctx context.Context, listenIP string, listenPort uint32) error {
	log.Info("[HealthCheck][Leader] local peer serve")
	return nil
}

// Get get records
func (p *LocalPeer) Host() string {
	return utils.LocalHost
}

// Get get records
func (p *LocalPeer) Get(key string) (*ReadBeatRecord, error) {
	ret := p.Cache.Get(key)
	return ret[key], nil
}

// Put put records
func (p *LocalPeer) Put(record WriteBeatRecord) error {
	p.Cache.Put(record)
	return nil
}

// Del del records
func (p *LocalPeer) Del(key string) error {
	p.Cache.Del(key)
	return nil
}

// Close close peer life
func (p *LocalPeer) Close() error {
	log.Info("[HealthCheck][Leader] local peer close")
	if p.cancel != nil {
		p.cancel()
	}
	return nil
}

func (p *LocalPeer) Storage() BeatRecordCache {
	return p.Cache
}

// LocalPeer Heartbeat data storage node
type RemotePeer struct {
	// Host peer host
	host string
	// Port peer listen port to provider grpc service
	port uint32
	// Conn grpc connection
	Conns []*grpc.ClientConn
	// Client checker_peer_service client instance
	Client apiservice.PolarisGRPCClient
	// putBatchCtrl 批任务执行器
	putBatchCtrl *batchjob.BatchController
	// getBatchCtrl 批任务执行器
	getBatchCtrl *batchjob.BatchController
	// Puters 批量心跳发送, 由于一个 stream 对于 server 是一个 goroutine，为了加快 follower 发往 leader 的效率
	// 这里采用多个 Putter Client 创建多个 Stream
	Puters []apiservice.PolarisGRPC_BatchHeartbeatClient
	// Cache data storage
	Cache BeatRecordCache
	// cancel .
	cancel context.CancelFunc
	// conf .
	conf Config
}

func (p *RemotePeer) Initialize(conf Config) {
	p.conf = conf
}

func (p *RemotePeer) Serve(_ context.Context, listenIP string, listenPort uint32) error {
	ctx, cancel := context.WithCancel(context.Background())
	p.cancel = cancel
	p.host = listenIP
	p.port = listenPort
	p.Conns = make([]*grpc.ClientConn, 0, streamNum)
	p.Puters = make([]apiservice.PolarisGRPC_BatchHeartbeatClient, 0, streamNum)
	for i := 0; i < streamNum; i++ {
		conn, err := grpc.DialContext(ctx, fmt.Sprintf("%s:%d", listenIP, listenPort),
			grpc.WithBlock(),
			grpc.WithInsecure(),
			grpc.WithTimeout(5*time.Second),
		)
		if err != nil {
			_ = p.Close()
			return err
		}
		p.Conns = append(p.Conns, conn)
	}
	p.Client = apiservice.NewPolarisGRPCClient(p.Conns[0])
	for i := 0; i < streamNum; i++ {
		puter, err := p.Client.BatchHeartbeat(ctx)
		if err != nil {
			_ = p.Close()
			return err
		}
		p.Puters = append(p.Puters, puter)
	}
	batchConf := p.conf.Batch
	p.getBatchCtrl = batchjob.NewBatchController(ctx, batchjob.CtrlConfig{
		Label:         "RecordGetter",
		QueueSize:     batchConf.QueueSize,
		WaitTime:      batchConf.WaitTime,
		MaxBatchCount: batchConf.MaxBatchCount,
		Concurrency:   batchConf.Concurrency,
		Handler:       p.handleSendGetRecords,
	})
	p.putBatchCtrl = batchjob.NewBatchController(ctx, batchjob.CtrlConfig{
		Label:         "RecordPutter",
		QueueSize:     batchConf.QueueSize,
		WaitTime:      batchConf.WaitTime,
		MaxBatchCount: batchConf.MaxBatchCount,
		Concurrency:   batchConf.Concurrency,
		Handler:       p.handleSendPutRecords,
	})
	p.Cache = newRemoteBeatRecordCache(p.GetFunc, p.PutFunc, p.DelFunc)
	return nil
}

func (p *RemotePeer) Host() string {
	return p.host
}

// Get get records
func (p *RemotePeer) Get(key string) (*ReadBeatRecord, error) {
	future := p.getBatchCtrl.Submit(key)
	resp, err := future.Done()
	if err != nil {
		return nil, err
	}
	ret := resp.(map[string]*ReadBeatRecord)
	return ret[key], nil
}

// Put put records
func (p *RemotePeer) Put(record WriteBeatRecord) error {
	future := p.putBatchCtrl.Submit(record)
	_, err := future.Done()
	return err
}

// Del del records
func (p *RemotePeer) Del(key string) error {
	p.Cache.Del(key)
	return nil
}

func (p *RemotePeer) GetFunc(req *apiservice.GetHeartbeatsRequest) *apiservice.GetHeartbeatsResponse {
	resp, err := p.Client.BatchGetHeartbeat(context.Background(), req)
	if err != nil {
		plog.Error("[HealthCheck][Leader] send get record request", zap.String("host", p.Host()),
			zap.Uint32("port", p.port), zap.Error(err))
		return &apiservice.GetHeartbeatsResponse{}
	}
	return resp
}

func (p *RemotePeer) PutFunc(req *apiservice.HeartbeatsRequest) {
	index := rand.Intn(len(p.Puters))
	if err := p.Puters[index].Send(req); err != nil {
		plog.Error("[HealthCheck][Leader] send put record request", zap.String("host", p.Host()),
			zap.Uint32("port", p.port), zap.Error(err))
	}
}

func (p *RemotePeer) DelFunc(req *apiservice.DelHeartbeatsRequest) {
	if _, err := p.Client.BatchDelHeartbeat(context.Background(), req); err != nil {
		plog.Error("send del record request", zap.String("host", p.Host()),
			zap.Uint32("port", p.port), zap.Error(err))
	}
}

func (p *RemotePeer) Storage() BeatRecordCache {
	return p.Cache
}

// Close close peer life
func (p *RemotePeer) Close() error {
	if p.cancel != nil {
		p.cancel()
	}
	if len(p.Puters) != 0 {
		for i := range p.Puters {
			_ = p.Puters[i].CloseSend()
		}
	}
	if len(p.Conns) != 0 {
		for i := range p.Conns {
			_ = p.Conns[i].Close()
		}
	}
	if p.getBatchCtrl != nil {
		p.getBatchCtrl.Stop()
	}
	if p.putBatchCtrl != nil {
		p.putBatchCtrl.Stop()
	}
	return nil
}

var (
	ErrorRecordNotFound = errors.New("beat record not found")
)

func (p *RemotePeer) handleSendGetRecords(tasks []batchjob.Future) {
	keys := make([]string, 0, len(tasks))
	futures := make(map[string][]batchjob.Future)
	for i := range tasks {
		taskInfo := tasks[i].Param()
		key := taskInfo.(string)
		keys = append(keys, key)
		if _, ok := futures[key]; !ok {
			futures[key] = make([]batchjob.Future, 0, 4)
		}
		futures[key] = append(futures[key], tasks[i])
		keys = append(keys, key)
	}

	ret := p.Cache.Get(keys...)
	for key := range ret {
		fs := futures[key]
		for _, f := range fs {
			f.Reply(map[string]*ReadBeatRecord{
				key: ret[key],
			}, nil)
		}
	}
	for i := range futures {
		for _, f := range futures[i] {
			f.Reply(nil, ErrorRecordNotFound)
		}
	}
	for i := range futures {
		for _, f := range futures[i] {
			f.Reply(nil, ErrorRecordNotFound)
		}
	}
}

func (p *RemotePeer) handleSendPutRecords(tasks []batchjob.Future) {
	records := make([]WriteBeatRecord, 0, len(tasks))
	for i := range tasks {
		taskInfo := tasks[i].Param()
		req := taskInfo.(WriteBeatRecord)
		records = append(records, req)
	}

	p.Cache.Put(records...)
	for i := range tasks {
		tasks[i].Reply(struct{}{}, nil)
	}
}
