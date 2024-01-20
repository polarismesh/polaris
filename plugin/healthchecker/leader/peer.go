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
	"net"
	"sync"
	"sync/atomic"
	"time"

	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

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
	Serve(ctx context.Context, checker *LeaderHealthChecker, listenIP string, listenPort uint32) error
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
	// Storage .
	Storage() BeatRecordCache
	// IsAlive .
	IsAlive() bool
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

func (p *LocalPeer) Serve(ctx context.Context, checker *LeaderHealthChecker,
	listenIP string, listenPort uint32) error {
	log.Info("[HealthCheck][Leader] local peer serve")
	return nil
}

func (p *LocalPeer) IsAlive() bool {
	return true
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
	conns []*grpc.ClientConn
	// Puters 批量心跳发送, 由于一个 stream 对于 server 是一个 goroutine，为了加快 follower 发往 leader 的效率
	// 这里采用多个 Putter Client 创建多个 Stream
	puters []*beatSender
	// Cache data storage
	Cache BeatRecordCache
	// cancel .
	cancel context.CancelFunc
	// conf .
	conf Config
	// closed .
	closed int32
}

func (p *RemotePeer) Initialize(conf Config) {
	p.conf = conf
}

func (p *RemotePeer) isClose() bool {
	return atomic.LoadInt32(&p.closed) == 1
}

func (p *RemotePeer) Serve(_ context.Context, checker *LeaderHealthChecker,
	listenIP string, listenPort uint32) error {
	ctx, cancel := context.WithCancel(context.Background())
	p.cancel = cancel
	p.host = listenIP
	p.port = listenPort
	p.conns = make([]*grpc.ClientConn, 0, streamNum)
	p.puters = make([]*beatSender, 0, streamNum)
	for i := 0; i < streamNum; i++ {
		conn, err := grpc.DialContext(ctx, fmt.Sprintf("%s:%d", listenIP, listenPort),
			grpc.WithBlock(),
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		if err != nil {
			_ = p.Close()
			return err
		}
		p.conns = append(p.conns, conn)
	}
	for i := 0; i < streamNum; i++ {
		client := apiservice.NewPolarisHeartbeatGRPCClient(p.conns[i])
		puter, err := client.BatchHeartbeat(ctx, grpc.Header(&metadata.MD{
			sendResource: []string{utils.LocalHost},
		}))
		if err != nil {
			_ = p.Close()
			return err
		}
		p.puters = append(p.puters, newBeatSender(ctx, p, puter))
	}
	p.Cache = newRemoteBeatRecordCache(p.GetFunc, p.PutFunc, p.DelFunc)
	return nil
}

func (p *RemotePeer) Host() string {
	return p.host
}

func (p *RemotePeer) IsAlive() bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%v", p.Host(), p.port), time.Second)
	defer func() {
		if conn != nil {
			_ = conn.Close()
		}
	}()

	if err != nil {
		return false
	}
	return true
}

// Get get records
func (p *RemotePeer) Get(key string) (*ReadBeatRecord, error) {
	ret := p.Cache.Get(key)
	return ret[key], nil
}

// Put put records
func (p *RemotePeer) Put(record WriteBeatRecord) error {
	p.Cache.Put(record)
	return nil
}

// Del del records
func (p *RemotePeer) Del(key string) error {
	p.Cache.Del(key)
	return nil
}

func (p *RemotePeer) GetFunc(req *apiservice.GetHeartbeatsRequest) *apiservice.GetHeartbeatsResponse {
	start := time.Now()
	code := "0"
	defer func() {
		observer := beatRecordCost.With(map[string]string{
			labelAction: "GET",
			labelCode:   code,
		})
		observer.Observe(float64(time.Since(start).Milliseconds()))
	}()
	client := p.choseOneClient()
	resp, err := client.BatchGetHeartbeat(context.Background(), req, grpc.Header(&metadata.MD{
		sendResource: []string{utils.LocalHost},
	}))
	if err != nil {
		code = "-1"
		plog.Error("[HealthCheck][Leader] send get record request", zap.String("host", p.Host()),
			zap.Uint32("port", p.port), zap.Error(err))
		return &apiservice.GetHeartbeatsResponse{}
	}
	return resp
}

func (p *RemotePeer) PutFunc(req *apiservice.HeartbeatsRequest) {
	start := time.Now()
	code := "0"
	defer func() {
		observer := beatRecordCost.With(map[string]string{
			labelAction: "PUT",
			labelCode:   code,
		})
		observer.Observe(float64(time.Since(start).Milliseconds()))
	}()
	index := rand.Intn(len(p.puters))
	if err := p.puters[index].Send(req); err != nil {
		code = "-1"
		plog.Error("[HealthCheck][Leader] send put record request", zap.String("host", p.Host()),
			zap.Uint32("port", p.port), zap.Error(err))
	}
}

func (p *RemotePeer) DelFunc(req *apiservice.DelHeartbeatsRequest) {
	start := time.Now()
	code := "0"
	defer func() {
		observer := beatRecordCost.With(map[string]string{
			labelAction: "DEL",
			labelCode:   code,
		})
		observer.Observe(float64(time.Since(start).Milliseconds()))
	}()
	client := p.choseOneClient()
	if _, err := client.BatchDelHeartbeat(context.Background(), req, grpc.Header(&metadata.MD{
		sendResource: []string{utils.LocalHost},
	})); err != nil {
		code = "-1"
		plog.Error("send del record request", zap.String("host", p.Host()),
			zap.Uint32("port", p.port), zap.Error(err))
	}
}

func (p *RemotePeer) choseOneClient() apiservice.PolarisHeartbeatGRPCClient {
	index := rand.Intn(len(p.conns))
	return apiservice.NewPolarisHeartbeatGRPCClient(p.conns[index])
}

func (p *RemotePeer) Storage() BeatRecordCache {
	return p.Cache
}

// Close close peer life
func (p *RemotePeer) Close() error {
	if !atomic.CompareAndSwapInt32(&p.closed, 0, 1) {
		return nil
	}
	if p.cancel != nil {
		p.cancel()
	}
	if len(p.puters) != 0 {
		for i := range p.puters {
			_ = p.puters[i].close()
		}
	}
	if len(p.conns) != 0 {
		for i := range p.conns {
			_ = p.conns[i].Close()
		}
	}
	return nil
}

var (
	ErrorRecordNotFound = errors.New("beat record not found")
	ErrorPeerClosed     = errors.New("peer alrady closed")
)

type beatSender struct {
	lock   sync.RWMutex
	sender apiservice.PolarisHeartbeatGRPC_BatchHeartbeatClient
}

func newBeatSender(ctx context.Context, p *RemotePeer, sender apiservice.PolarisHeartbeatGRPC_BatchHeartbeatClient) *beatSender {
	go func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				plog.Info("[HealthCheck][Leader] cancel receive put record result", zap.String("host", p.Host()),
					zap.Uint32("port", p.port))
				return
			default:
				if _, err := sender.Recv(); err != nil {
					plog.Error("[HealthCheck][Leader] receive put record result", zap.String("host", p.Host()),
						zap.Uint32("port", p.port), zap.Error(err))
				}
			}
		}
	}(ctx)

	return &beatSender{
		sender: sender,
	}
}

func (s *beatSender) Send(req *apiservice.HeartbeatsRequest) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.sender.Send(req)
}

func (s *beatSender) close() error {
	return s.sender.CloseSend()
}
