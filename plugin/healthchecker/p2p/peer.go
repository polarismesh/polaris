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
	"context"
	"fmt"
	"io"
	"net"
	"sync"

	"go.uber.org/zap"
	"google.golang.org/grpc"

	commonhash "github.com/polarismesh/polaris/common/hash"
)

// Peer Heartbeat data storage node
type Peer struct {
	once sync.Once
	// Local current peer is local
	Local bool
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
	// Cache data storage
	Cache BeatRecordCache
	// cancel
	cancel context.CancelFunc
}

func (p *Peer) Serve(soltNum int32) error {
	var err error
	p.once.Do(func() {
		ctx, cancel := context.WithCancel(context.Background())
		p.cancel = cancel
		if p.Local {
			log.Info("[HealthCheck][P2P] local peer serve", zap.String("host", p.Host), zap.Uint32("port", p.Port))
			p.Cache = newLocalBeatRecordCache(soltNum, commonhash.Fnv32)
			err = p.initLocal(ctx)
			return
		}
		log.Info("[HealthCheck][P2P] remote peer client init", zap.String("host", p.Host), zap.Uint32("port", p.Port))
		if err = p.initRemote(ctx); err != nil {
			return
		}
	})
	return err
}

func (p *Peer) initLocal(ctx context.Context) error {
	ln, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%v", p.Port))
	if err != nil {
		return err
	}
	p.GrpcSvr = grpc.NewServer()
	RegisterCheckerPeerServiceServer(p.GrpcSvr, p)
	go func() {
		log.Info("[HealthCheck][P2P] local peer serve CheckerPeerService", zap.Any("address", ln.Addr()))
		if err := p.GrpcSvr.Serve(ln); err != nil {
			log.Error("[HealthCheck][P2P] local peer server serve", zap.String("host", p.Host),
				zap.Uint32("port", p.Port), zap.Error(err))
		}
	}()
	return err
}

func (p *Peer) initRemote(ctx context.Context) error {
	opts := []grpc.DialOption{
		grpc.WithBlock(),
		grpc.WithInsecure(),
	}
	conn, err := grpc.DialContext(context.Background(), fmt.Sprintf("%s:%d", p.Host, p.Port), opts...)
	if err != nil {
		return err
	}
	p.Conn = conn
	p.Client = NewCheckerPeerServiceClient(p.Conn)
	putter, err := p.Client.PutRecords(context.Background())
	if err != nil {
		return err
	}
	p.Putter = putter
	delter, err := p.Client.DelRecords(context.Background())
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
			log.Debug("[HealthCheck][P2P] send get record request", zap.String("host", p.Host),
				zap.Uint32("port", p.Port), zap.Any("req", req))
			resp, err := p.Client.GetRecords(context.Background(), req)
			if err != nil {
				log.Error("[HealthCheck][P2P] send get record request", zap.String("host", p.Host),
					zap.Uint32("port", p.Port), zap.Any("req", req), zap.Error(err))
				return nil
			}
			return resp
		}, func(req *PutRecordsRequest) {
			log.Debug("[HealthCheck][P2P] send put record request", zap.String("host", p.Host),
				zap.Uint32("port", p.Port), zap.Any("req", req))
			if err := p.Putter.Send(req); err != nil {
				log.Error("[HealthCheck][P2P] send put record request", zap.String("host", p.Host),
					zap.Uint32("port", p.Port), zap.Any("req", req), zap.Error(err))
			}
		}, func(req *DelRecordsRequest) {
			log.Debug("[HealthCheck][P2P] send del record request", zap.String("host", p.Host),
				zap.Uint32("port", p.Port), zap.Any("req", req))
			if err := p.Delter.Send(req); err != nil {
				log.Error("[HealthCheck][P2P] send del record request", zap.String("host", p.Host),
					zap.Uint32("port", p.Port), zap.Any("req", req), zap.Error(err))
			}
		})
	return nil
}

func (p *Peer) GetRecords(_ context.Context, req *GetRecordsRequest) (*GetRecordsResponse, error) {
	log.Debug("[HealthCheck][P2P] receive get record request", zap.String("host", p.Host),
		zap.Uint32("port", p.Port), zap.Any("req", req))
	keys := req.Keys
	records := make([]*HeartbeatRecord, 0, len(keys))
	items := p.Cache.Get(keys...)
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

func (p *Peer) PutRecords(svr CheckerPeerService_PutRecordsServer) error {
	for {
		req, err := svr.Recv()
		if err != nil {
			if io.EOF == err {
				return nil
			}
			return err
		}
		log.Debug("[HealthCheck][P2P] receive put record request", zap.String("host", p.Host),
			zap.Uint32("port", p.Port), zap.Any("req", req))

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
		p.Cache.Put(writeItems...)
		if err := svr.Send(&PutRecordsResponse{}); err != nil {
			return err
		}
	}
}

func (p *Peer) DelRecords(svr CheckerPeerService_DelRecordsServer) error {
	for {
		req, err := svr.Recv()
		if err != nil {
			if io.EOF == err {
				return nil
			}
			return err
		}
		log.Debug("[HealthCheck][P2P] receive del record request", zap.String("host", p.Host),
			zap.Uint32("port", p.Port), zap.Any("req", req))

		for i := range req.Keys {
			key := req.Keys[i]
			p.Cache.Del(key)
		}
	}
}

// Close close peer life
func (p *Peer) Close() error {
	log.Info("[HealthCheck][P2P] peer close", zap.String("host", p.Host), zap.Uint32("port", p.Port))
	if p.Conn != nil {
		if err := p.Conn.Close(); err != nil {
			log.Error("[HealthCheck][P2P] remote peer client close", zap.String("host", p.Host),
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
