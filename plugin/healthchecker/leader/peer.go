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
	"math/rand"
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
	ErrorLeaderNotAlive = errors.New("leader not alive")
)

type (
	// 仅支持测试场景塞入即可
	ConnectFuncContextKey struct{}
	ConnectPeerFunc       func(*RemotePeer) error
)

var (
	NewLocalPeerFunc  = newLocalPeer
	NewRemotePeerFunc = newRemotePeer
	ConnectPeer       = doConnect

	CreateBeatClientFunc = createBeatClient
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
	cache BeatRecordCache
	// cancel .
	cancel context.CancelFunc
	// conf .
	conf Config
	// closed .
	closed int32
	// leaderAlive .
	leaderAlive int32
}

func (p *RemotePeer) Initialize(conf Config) {
	p.conf = conf
}

func (p *RemotePeer) Serve(ctx context.Context, checker *LeaderHealthChecker,
	listenIP string, listenPort uint32) error {
	ctx, cancel := context.WithCancel(ctx)
	p.cancel = cancel
	p.host = listenIP
	p.port = listenPort

	connectPeer := ConnectPeer
	val := ctx.Value(ConnectFuncContextKey{})
	if val != nil {
		// 正常情况下只是为了测试场景使用
		connectPeer = val.(ConnectPeerFunc)
	}
	if err := connectPeer(p); err != nil {
		return err
	}
	p.cache = newRemoteBeatRecordCache(p.GetFunc, p.PutFunc, p.DelFunc, p.Ping)
	// 启动前先设置 Leader 为 alive 状态
	atomic.StoreInt32(&p.leaderAlive, 1)
	go p.checkLeaderAlive(ctx)
	return nil
}

func (p *RemotePeer) Host() string {
	return p.host
}

func (p *RemotePeer) IsAlive() bool {
	return atomic.LoadInt32(&p.leaderAlive) == 1
}

func (p *RemotePeer) Ping() error {
	client := p.choseOneClient()
	_, err := client.BatchGetHeartbeat(context.Background(), &apiservice.GetHeartbeatsRequest{},
		grpc.Header(&metadata.MD{
			sendResource: []string{utils.LocalHost},
		}))
	return err
}

func (p *RemotePeer) GetFunc(req *apiservice.GetHeartbeatsRequest) (*apiservice.GetHeartbeatsResponse, error) {
	if !p.IsAlive() {
		return nil, ErrorLeaderNotAlive
	}
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
		return nil, err
	}
	return resp, nil
}

func (p *RemotePeer) PutFunc(req *apiservice.HeartbeatsRequest) error {
	if !p.IsAlive() {
		return ErrorLeaderNotAlive
	}
	start := time.Now()
	code := "0"
	defer func() {
		observer := beatRecordCost.With(map[string]string{
			labelAction: "PUT",
			labelCode:   code,
		})
		observer.Observe(float64(time.Since(start).Milliseconds()))
	}()
	if err := p.choseOneSender().Send(req); err != nil {
		code = "-1"
		plog.Error("[HealthCheck][Leader] send put record request", zap.String("host", p.Host()),
			zap.Uint32("port", p.port), zap.Error(err))
		return err
	}
	return nil
}

func (p *RemotePeer) DelFunc(req *apiservice.DelHeartbeatsRequest) error {
	if !p.IsAlive() {
		return ErrorLeaderNotAlive
	}
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
		return err
	}
	return nil
}

func (p *RemotePeer) Storage() BeatRecordCache {
	return p.cache
}

// Close close peer life
func (p *RemotePeer) Close() error {
	if !atomic.CompareAndSwapInt32(&p.closed, 0, 1) {
		return nil
	}
	p.doClose()
	return nil
}

func (p *RemotePeer) choseOneClient() apiservice.PolarisHeartbeatGRPCClient {
	index := rand.Intn(len(p.conns))
	return CreateBeatClientFunc(p.conns[index])
}

func (p *RemotePeer) choseOneSender() *beatSender {
	index := rand.Intn(len(p.puters))
	return p.puters[index]
}

func (p *RemotePeer) checkLeaderAlive(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	for {
		select {
		case <-ctx.Done():
			ticker.Stop()
		case <-ticker.C:
			var errCount int
			for i := 0; i < maxCheckCount; i++ {
				if err := p.Ping(); err != nil {
					plog.Error("check leader is alive fail", zap.String("host", p.Host()),
						zap.Uint32("port", p.port), zap.Error(err))
					errCount++
				}
			}
			if errCount >= errCountThreshold {
				log.Warn("[Health Check][Leader] leader peer not alive, set leader is dead", zap.String("host", p.Host()),
					zap.Uint32("port", p.port))
				atomic.StoreInt32(&p.leaderAlive, 0)
			} else {
				atomic.StoreInt32(&p.leaderAlive, 1)
			}
		}
	}
}

func (p *RemotePeer) doClose() {
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
}

func createBeatClient(conn *grpc.ClientConn) apiservice.PolarisHeartbeatGRPCClient {
	return apiservice.NewPolarisHeartbeatGRPCClient(conn)
}

func doConnect(p *RemotePeer) error {
	p.conns = make([]*grpc.ClientConn, 0, streamNum)
	p.puters = make([]*beatSender, 0, streamNum)
	for i := 0; i < streamNum; i++ {
		conn, err := grpc.DialContext(context.Background(), fmt.Sprintf("%s:%d", p.Host(), p.port),
			grpc.WithBlock(),
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithTimeout(5*time.Second),
		)
		if err != nil {
			p.doClose()
			return err
		}
		p.conns = append(p.conns, conn)
	}
	for i := 0; i < streamNum; i++ {
		client := apiservice.NewPolarisHeartbeatGRPCClient(p.conns[i])
		puter, err := client.BatchHeartbeat(context.Background(), grpc.Header(&metadata.MD{
			sendResource: []string{utils.LocalHost},
		}))
		if err != nil {
			p.doClose()
			return err
		}
		sender := &beatSender{
			peer:   p,
			lock:   &sync.RWMutex{},
			sender: puter,
		}
		p.puters = append(p.puters, sender)
	}
	return nil
}

func newBeatSender(p *RemotePeer, client apiservice.PolarisHeartbeatGRPC_BatchHeartbeatClient) *beatSender {
	ctx, cancel := context.WithCancel(context.Background())
	sender := &beatSender{
		peer:   p,
		lock:   &sync.RWMutex{},
		sender: client,
		cancel: cancel,
	}
	go sender.doRecv(ctx)
	return sender
}

type beatSender struct {
	peer   *RemotePeer
	lock   *sync.RWMutex
	sender apiservice.PolarisHeartbeatGRPC_BatchHeartbeatClient
	cancel context.CancelFunc
}

func (s *beatSender) Send(req *apiservice.HeartbeatsRequest) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.sender.Send(req)
}

func (s *beatSender) doRecv(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			plog.Info("[HealthCheck][Leader] cancel receive put record result", zap.String("host", s.peer.Host()),
				zap.Uint32("port", s.peer.port))
			return
		default:
			if _, err := s.sender.Recv(); err != nil {
				if err != io.EOF {
					plog.Error("[HealthCheck][Leader] receive put record result", zap.String("host", s.peer.Host()),
						zap.Uint32("port", s.peer.port), zap.Error(err))
				}
			}
		}
	}
}

func (s *beatSender) close() error {
	if s.cancel != nil {
		s.cancel()
	}
	return s.sender.CloseSend()
}
