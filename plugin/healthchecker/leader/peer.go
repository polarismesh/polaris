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
	"io"
	"net"
	"sync"

	"go.uber.org/zap"
	"google.golang.org/grpc"

	"github.com/polarismesh/polaris/common/batchjob"
	commonhash "github.com/polarismesh/polaris/common/hash"
)

// Peer Heartbeat data storage node
type Peer struct {
	once sync.Once
	// Leader current peer is Leader
	Leader bool
	// ID peer id
	ID string
	// Host peer host
	Host string
	// Port peer listen port to provider grpc service
	Port uint32
	// GrpcSvr checker_peer_service server instance
	GrpcSvr *grpc.Server
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
	// cancel .
	cancel context.CancelFunc
}

func (p *Peer) Serve(soltNum int32, listenIP string, batchConf batchjob.CtrlConfig) error {
	var err error
	p.once.Do(func() {
		ctx, cancel := context.WithCancel(context.Background())
		p.cancel = cancel
		if p.Leader {
			err = p.initLocal(ctx, soltNum, listenIP)
			return
		}
		if err = p.initRemote(ctx, batchConf); err != nil {
			return
		}
	})
	return err
}

func (p *Peer) initLocal(ctx context.Context, soltNum int32, listenIP string) error {
	log.Info("local peer serve", zap.String("host", p.Host), zap.Uint32("port", p.Port))
	p.Cache = newLocalBeatRecordCache(soltNum, commonhash.Fnv32)
	ln, err := net.Listen("tcp", fmt.Sprintf("%v:%v", listenIP, p.Port))
	if err != nil {
		return err
	}
	p.GrpcSvr = grpc.NewServer()
	RegisterCheckerPeerServiceServer(p.GrpcSvr, &PeerServiceHandler{p: p})
	go func() {
		if err := p.GrpcSvr.Serve(ln); err != nil {
			log.Error("local peer server serve", zap.String("host", p.Host),
				zap.Uint32("port", p.Port), zap.Error(err))
		}
	}()
	return err
}

func (p *Peer) initRemote(ctx context.Context, batchConf batchjob.CtrlConfig) error {
	log.Info("remote peer client init", zap.String("host", p.Host), zap.Uint32("port", p.Port))
	handler := &PeerBatchHandler{p: p}
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
	conn, err := grpc.DialContext(ctx, fmt.Sprintf("%s:%d", p.Host, p.Port), grpc.WithBlock(), grpc.WithInsecure())
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
			select {
			case <-ctx.Done():
			default:
				_, _ = putter.Recv()
			}
		}
	}()
	go func() {
		for {
			select {
			case <-ctx.Done():
			default:
				_, _ = delter.Recv()
			}
		}
	}()
	p.Cache = newRemoteBeatRecordCache(
		func(req *GetRecordsRequest) *GetRecordsResponse {
			log.Debug("send get record request", zap.String("host", p.Host),
				zap.Uint32("port", p.Port), zap.Any("req", req))
			resp, err := p.Client.GetRecords(context.Background(), req)
			if err != nil {
				log.Error("send get record request", zap.String("host", p.Host),
					zap.Uint32("port", p.Port), zap.Any("req", req), zap.Error(err))
				return nil
			}
			return resp
		}, func(req *PutRecordsRequest) {
			log.Debug("send put record request", zap.String("host", p.Host),
				zap.Uint32("port", p.Port), zap.Any("req", req))
			if err := p.Putter.Send(req); err != nil {
				log.Error("send put record request", zap.String("host", p.Host),
					zap.Uint32("port", p.Port), zap.Any("req", req), zap.Error(err))
			}
		}, func(req *DelRecordsRequest) {
			log.Debug("send del record request", zap.String("host", p.Host),
				zap.Uint32("port", p.Port), zap.Any("req", req))
			if err := p.Delter.Send(req); err != nil {
				log.Error("send del record request", zap.String("host", p.Host),
					zap.Uint32("port", p.Port), zap.Any("req", req), zap.Error(err))
			}
		})
	return nil
}

// Get get records
func (p *Peer) Get(key string) (map[string]*ReadBeatRecord, error) {
	if p.Leader {
		return p.Cache.Get(key), nil
	}
	future := p.getBatchCtrl.Submit(key)
	resp, err := future.Done()
	if err != nil {
		return nil, err
	}
	ret := resp.(map[string]*ReadBeatRecord)
	return ret, nil
}

// Put put records
func (p *Peer) Put(record WriteBeatRecord) error {
	if p.Leader {
		p.Cache.Put(record)
		return nil
	}
	future := p.putBatchCtrl.Submit(record)
	_, err := future.Done()
	return err
}

// Del del records
func (p *Peer) Del(key string) error {
	p.Cache.Del(key)
	return nil
}

// Close close peer life
func (p *Peer) Close() error {
	log.Info("peer close", zap.String("host", p.Host), zap.Uint32("port", p.Port))
	if p.Conn != nil {
		if err := p.Conn.Close(); err != nil {
			log.Error("remote peer client close", zap.String("host", p.Host),
				zap.Uint32("port", p.Port), zap.Error(err))
		}
	}
	if p.GrpcSvr != nil {
		p.GrpcSvr.Stop()
	}
	if p.cancel != nil {
		p.cancel()
	}
	return nil
}

type PeerBatchHandler struct {
	p *Peer
}

func (p *PeerBatchHandler) handleSendGetRecords(tasks []batchjob.Future) {
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
	for i := range ret {
		fs := futures[i]
		for _, f := range fs {
			f.Reply(map[string]*ReadBeatRecord{
				i: ret[i],
			}, nil)
		}
	}
}

func (p *PeerBatchHandler) handleSendPutRecords(tasks []batchjob.Future) {
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

type PeerServiceHandler struct {
	p *Peer
}

func (p *PeerServiceHandler) GetRecords(_ context.Context, req *GetRecordsRequest) (*GetRecordsResponse, error) {
	log.Debug("receive get record request", zap.String("host", p.p.Host),
		zap.Uint32("port", p.p.Port), zap.Any("req", req))
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

func (p *PeerServiceHandler) PutRecords(svr CheckerPeerService_PutRecordsServer) error {
	for {
		req, err := svr.Recv()
		if err != nil {
			if io.EOF == err {
				return nil
			}
			return err
		}
		log.Debug("receive put record request", zap.String("host", p.p.Host),
			zap.Uint32("port", p.p.Port), zap.Any("req", req))

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

func (p *PeerServiceHandler) DelRecords(svr CheckerPeerService_DelRecordsServer) error {
	for {
		req, err := svr.Recv()
		if err != nil {
			if io.EOF == err {
				return nil
			}
			return err
		}
		log.Debug("receive del record request", zap.String("host", p.p.Host),
			zap.Uint32("port", p.p.Port), zap.Any("req", req))

		for i := range req.Keys {
			key := req.Keys[i]
			p.p.Cache.Del(key)
		}
	}
}
