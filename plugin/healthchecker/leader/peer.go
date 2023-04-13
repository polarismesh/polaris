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
	"io"
	"net"
	"sync"

	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"
	"google.golang.org/grpc"

	"github.com/polarismesh/polaris/common/batchjob"
	commonhash "github.com/polarismesh/polaris/common/hash"
)

type Peer interface {
	// Host
	Host() string
	// Initialize
	Initialize(conf Config) error
	// Serve
	Serve(ctx context.Context, listenIP string, listenPort uint32) error
	// Get
	Get(key string) (*ReadBeatRecord, error)
	// Put
	Put(record WriteBeatRecord) error
	// Del
	Del(key string) error
	// Close
	Close() error
}

func newLocalPeer(host string, port uint32) Peer {
	return &LocalPeer{
		host: host,
		port: port,
	}
}

func newRemotePeer(host string, port uint32) Peer {
	return &RemotePeer{
		host: host,
		port: port,
	}
}

// LocalPeer Heartbeat data storage node
type LocalPeer struct {
	once sync.Once
	// Host peer host
	host string
	// Port peer listen port to provider grpc service
	port uint32
	// GrpcSvr checker_peer_service server instance
	GrpcSvr *grpc.Server
	// Cache data storage
	Cache BeatRecordCache
	// cancel .
	cancel context.CancelFunc
}

func (p *LocalPeer) Initialize(conf Config) error {
	p.Cache = newLocalBeatRecordCache(conf.SoltNum, commonhash.Fnv32)
	return nil
}

func (p *LocalPeer) Serve(ctx context.Context, listenIP string, listenPort uint32) error {
	var err error
	ctx, cancel := context.WithCancel(context.Background())
	p.cancel = cancel
	ln, err := net.Listen("tcp", fmt.Sprintf("%v:%v", listenIP, listenPort))
	if err != nil {
		return err
	}
	p.GrpcSvr = grpc.NewServer()
	RegisterCheckerPeerServiceServer(p.GrpcSvr, &LocalPeerServiceHandler{p: p})
	go func() {
		if err := p.GrpcSvr.Serve(ln); err != nil {
			log.Error("[HealthCheck][Leader] local peer server serve", zap.String("host", p.Host()),
				zap.Uint32("port", p.port), zap.Error(err))
		}
	}()
	log.Info("[HealthCheck][Leader] local peer serve", zap.String("host", p.Host()), zap.Uint32("port", p.port))
	return err
}

func (p *LocalPeer) Host() string {
	return p.host
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
	log.Info("[HealthCheck][Leader] peer close", zap.String("host", p.Host()), zap.Uint32("port", p.port))
	if p.GrpcSvr != nil {
		p.GrpcSvr.Stop()
	}
	if p.cancel != nil {
		p.cancel()
	}
	return nil
}

type LocalPeerServiceHandler struct {
	p *LocalPeer
}

func (p *LocalPeerServiceHandler) GetRecords(_ context.Context, req *GetRecordsRequest) (*GetRecordsResponse, error) {
	if log.DebugEnabled() {
		log.Debug("[HealthCheck][Leader] receive get record request", zap.Any("req", req))
	}
	keys := req.Keys
	records := make([]*HeartbeatRecord, 0, len(keys))
	items := p.p.Cache.Get(keys...)
	for i := range keys {
		key := keys[i]
		item := items[key]
		record := &HeartbeatRecord{
			Key:   key,
			Value: item.Record.String(),
			Exist: item.Exist,
		}
		records = append(records, record)
	}

	return &GetRecordsResponse{
		Records: records,
	}, nil
}

func (p *LocalPeerServiceHandler) PutRecords(svr CheckerPeerService_PutRecordsServer) error {
	for {
		req, err := svr.Recv()
		if err != nil {
			log.Error("[HealthCheck][Leader] receive put record request", zap.Error(err))
			if io.EOF == err {
				return nil
			}
			return err
		}
		if log.DebugEnabled() {
			log.Debug("[HealthCheck][Leader] receive put record request", zap.Any("req", req))
		}
		writeItems := make([]WriteBeatRecord, 0, len(req.Records))
		for i := range req.Records {
			record := req.Records[i]
			val, ok := ParseRecordValue(record.Value)
			if !ok {
				continue
			}
			writeItems = append(writeItems, WriteBeatRecord{
				Record: *val,
				Key:    record.Key,
			})
		}
		p.p.Cache.Put(writeItems...)
		if err := svr.Send(&PutRecordsResponse{}); err != nil {
			return err
		}
	}
}

func (p *LocalPeerServiceHandler) DelRecords(svr CheckerPeerService_DelRecordsServer) error {
	for {
		req, err := svr.Recv()
		if err != nil {
			if io.EOF == err {
				return nil
			}
			return err
		}
		if log.DebugEnabled() {
			log.Debug("[HealthCheck][Leader] receive del record request", zap.Any("req", req))
		}

		for i := range req.Keys {
			key := req.Keys[i]
			p.p.Cache.Del(key)
		}
		if err := svr.Send(&DelRecordsResponse{}); err != nil {
			return err
		}
	}
}

// LocalPeer Heartbeat data storage node
type RemotePeer struct {
	once sync.Once
	// Host peer host
	host string
	// Port peer listen port to provider grpc service
	port uint32
	// Conn grpc connection
	Conn *grpc.ClientConn
	// Client checker_peer_service client instance
	Client CheckerPeerServiceClient
	// Putter put beat records client
	Putter CheckerPeerService_PutRecordsClient
	// Delter delete beat records client
	Delter CheckerPeerService_DelRecordsClient
	// putBatchCtrl 批任务执行器
	putBatchCtrl *batchjob.BatchController
	// getBatchCtrl 批任务执行器
	getBatchCtrl *batchjob.BatchController
	// Cache data storage
	Cache BeatRecordCache
	// single
	single singleflight.Group
	// cancel .
	cancel context.CancelFunc
}

func (p *RemotePeer) Initialize(conf Config) error {
	ctx, cancel := context.WithCancel(context.Background())
	p.cancel = cancel
	batchConf := conf.Batch
	handler := &RemotePeerBatchHandler{p: p}
	p.getBatchCtrl = batchjob.NewBatchController(ctx, batchjob.CtrlConfig{
		Label:         "RecordGetter",
		QueueSize:     batchConf.QueueSize,
		WaitTime:      batchConf.WaitTime,
		MaxBatchCount: batchConf.MaxBatchCount,
		Concurrency:   batchConf.Concurrency,
		Handler:       handler.handleSendGetRecords,
	})
	p.putBatchCtrl = batchjob.NewBatchController(ctx, batchjob.CtrlConfig{
		Label:         "RecordSaver",
		QueueSize:     batchConf.QueueSize,
		WaitTime:      batchConf.WaitTime,
		MaxBatchCount: batchConf.MaxBatchCount,
		Concurrency:   batchConf.Concurrency,
		Handler:       handler.handleSendPutRecords,
	})
	p.Cache = newRemoteBeatRecordCache(
		func(req *GetRecordsRequest) *GetRecordsResponse {
			if log.DebugEnabled() {
				log.Debug("[HealthCheck][Leader] send get record request", zap.String("host", p.Host()),
					zap.Uint32("port", p.port), zap.Any("req", req))
			}
			resp, err := p.Client.GetRecords(context.Background(), req)
			if err != nil {
				log.Error("[HealthCheck][Leader] send get record request", zap.String("host", p.Host()),
					zap.Uint32("port", p.port), zap.Any("req", req), zap.Error(err))
				return &GetRecordsResponse{}
			}
			return resp
		}, func(req *PutRecordsRequest) {
			if log.DebugEnabled() {
				log.Debug("[HealthCheck][Leader] send put record request", zap.String("host", p.Host()),
					zap.Uint32("port", p.port), zap.Any("req", req))
			}
			if err := p.Putter.Send(req); err != nil {
				log.Error("[HealthCheck][Leader] send put record request", zap.String("host", p.Host()),
					zap.Uint32("port", p.port), zap.Any("req", req), zap.Error(err))
			}
		}, func(req *DelRecordsRequest) {
			if log.DebugEnabled() {
				log.Debug("[HealthCheck][Leader] send del record request", zap.String("host", p.Host()),
					zap.Uint32("port", p.port), zap.Any("req", req))
			}
			if err := p.Delter.Send(req); err != nil {
				log.Error("send del record request", zap.String("host", p.Host()),
					zap.Uint32("port", p.port), zap.Any("req", req), zap.Error(err))
			}
		})
	return nil
}

func (p *RemotePeer) Serve(ctx context.Context, listenIP string, listenPort uint32) error {
	conn, err := grpc.DialContext(ctx, fmt.Sprintf("%s:%d", p.Host(), p.port), grpc.WithBlock(), grpc.WithInsecure())
	if err != nil {
		return err
	}
	p.Conn = conn
	p.Client = NewCheckerPeerServiceClient(p.Conn)
	putter, err := p.Client.PutRecords(ctx)
	if err != nil {
		return err
	}
	p.Putter = putter
	delter, err := p.Client.DelRecords(ctx)
	if err != nil {
		return err
	}
	p.Delter = delter

	go func() {
		for {
			if _, err := putter.Recv(); err != nil {
				log.Error("receive put record response", zap.Error(err))
				return
			}
		}
	}()
	go func() {
		for {
			if _, err := delter.Recv(); err != nil {
				log.Error("receive del record response", zap.Error(err))
				return
			}
		}
	}()
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

// Close close peer life
func (p *RemotePeer) Close() error {
	if p.cancel != nil {
		p.cancel()
	}
	if p.Conn != nil {
		p.Conn.Close()
	}
	if p.getBatchCtrl != nil {
		p.getBatchCtrl.Stop()
	}
	if p.putBatchCtrl != nil {
		p.putBatchCtrl.Stop()
	}
	return nil
}

type RemotePeerBatchHandler struct {
	p *RemotePeer
}

var (
	ErrorBadGetRecordRequest = errors.New("bad get record request")
)

func (p *RemotePeerBatchHandler) handleSendGetRecords(tasks []batchjob.Future) {
	keys := make([]string, 0, len(tasks))
	futures := make(map[string][]batchjob.Future)
	for i := range tasks {
		taskInfo := tasks[i].TaskInfo()
		key := taskInfo.(string)
		keys = append(keys, key)
		if _, ok := futures[key]; !ok {
			futures[key] = make([]batchjob.Future, 0, 4)
		}
		futures[key] = append(futures[key], tasks[i])
		keys = append(keys, key)
	}

	ret := p.p.Cache.Get(keys...)
	for key := range ret {
		fs := futures[key]
		for _, f := range fs {
			f.Reply(map[string]*ReadBeatRecord{
				key: ret[key],
			}, nil)
		}
		delete(futures, key)
	}
	for i := range futures {
		for _, f := range futures[i] {
			f.Reply(nil, ErrorBadGetRecordRequest)
		}
	}
}

func (p *RemotePeerBatchHandler) handleSendPutRecords(tasks []batchjob.Future) {
	records := make([]WriteBeatRecord, 0, len(tasks))
	for i := range tasks {
		taskInfo := tasks[i].TaskInfo()
		req := taskInfo.(WriteBeatRecord)
		records = append(records, req)
	}

	p.p.Cache.Put(records...)
	for i := range tasks {
		tasks[i].Reply(struct{}{}, nil)
	}
}
