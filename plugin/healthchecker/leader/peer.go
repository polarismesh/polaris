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
	ErrorLeaderNotAlive         = errors.New("leader not alive")
	ErrorConnectionNotAvailable = errors.New("connection not available")
)

type (
	// 仅支持测试场景塞入即可
	ConnectFuncContextKey struct{}
	ConnectPeerFunc       func(*RemotePeer) error

	PingFuncContextKey struct{}
	PingFunc           func() error
)

var (
	NewLocalPeerFunc  = newLocalPeer
	NewRemotePeerFunc = newRemotePeer

	ConnectPeer          = doConnect
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
	//
	cmutex *sync.RWMutex
	// Conn grpc connection
	conns map[int]*grpc.ClientConn
	// Puters 批量心跳发送, 由于一个 stream 对于 server 是一个 goroutine，为了加快 follower 发往 leader 的效率
	// 这里采用多个 Putter Client 创建多个 Stream
	puters map[int]*beatSender
	// Cache data storage
	Cache BeatRecordCache
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
	subCtx, cancel := context.WithCancel(ctx)
	p.cancel = cancel
	p.host = listenIP
	p.port = listenPort
	p.cmutex = &sync.RWMutex{}
	if err := execConnectPeer(ctx, p); err != nil {
		return err
	}
	p.Cache = newRemoteBeatRecordCache(p.GetFunc, p.PutFunc, p.DelFunc, p.Ping)
	go p.checkLeaderAlive(subCtx)
	return nil
}

func execConnectPeer(ctx context.Context, p *RemotePeer) error {
	val := ctx.Value(ConnectFuncContextKey{})
	connectFunc, ok := val.(ConnectPeerFunc)
	if !ok {
		connectFunc = ConnectPeer
	}
	return connectFunc(p)
}

func (p *RemotePeer) Host() string {
	return p.host
}

func (p *RemotePeer) IsAlive() bool {
	return atomic.LoadInt32(&p.leaderAlive) == 1
}

func (p *RemotePeer) Ping() error {
	client, err := p.choseOneClient()
	if err != nil {
		return err
	}
	_, err = client.BatchGetHeartbeat(context.Background(), &apiservice.GetHeartbeatsRequest{},
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
	client, err := p.choseOneClient()
	if err != nil {
		return nil, err
	}
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
	client, err := p.choseOneSender()
	if err != nil {
		return err
	}
	if err := client.Send(req); err != nil {
		code = "-1"
		plog.Error("[HealthCheck][Leader] send put record request", zap.String("info", req.String()),
			zap.String("host", p.Host()), zap.Uint32("port", p.port), zap.Error(err))
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
	client, err := p.choseOneClient()
	if err != nil {
		return err
	}
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
	return p.Cache
}

// Close close peer life
func (p *RemotePeer) Close() error {
	if !atomic.CompareAndSwapInt32(&p.closed, 0, 1) {
		return nil
	}
	p.doClose()
	return nil
}

func (p *RemotePeer) choseOneClient() (apiservice.PolarisHeartbeatGRPCClient, error) {
	p.cmutex.RLock()
	defer p.cmutex.RUnlock()

	if len(p.conns) == 0 {
		return nil, ErrorConnectionNotAvailable
	}

	index := rand.Intn(len(p.conns))
	return CreateBeatClientFunc(p.conns[index])
}

func (p *RemotePeer) choseOneSender() (*beatSender, error) {
	p.cmutex.RLock()
	defer p.cmutex.RUnlock()

	if len(p.puters) == 0 {
		return nil, ErrorConnectionNotAvailable
	}

	index := rand.Intn(len(p.puters))
	return p.puters[index], nil
}

const (
	errCountThreshold = 2
	maxCheckCount     = 3
)

func (p *RemotePeer) checkLeaderAlive(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	for {
		select {
		case <-ctx.Done():
			ticker.Stop()
			plog.Info("check leader alive job stop", zap.String("host", p.Host()), zap.Uint32("port", p.port))
			return
		case <-ticker.C:
			var errCount int
			for i := 0; i < maxCheckCount; i++ {
				if err := execPing(ctx, p); err != nil {
					plog.Error("check leader is alive fail", zap.String("host", p.Host()),
						zap.Uint32("port", p.port), zap.Error(err))
					errCount++
				}
			}
			if errCount >= errCountThreshold {
				plog.Warn("[Health Check][Leader] leader peer not alive, set leader is dead", zap.String("host", p.Host()),
					zap.Uint32("port", p.port))
				atomic.StoreInt32(&p.leaderAlive, 0)
			} else {
				atomic.StoreInt32(&p.leaderAlive, 1)
			}
		}
	}
}

func (p *RemotePeer) doClose() {
	p.cmutex.Lock()
	defer p.cmutex.Unlock()

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

func createBeatClient(conn *grpc.ClientConn) (apiservice.PolarisHeartbeatGRPCClient, error) {
	return apiservice.NewPolarisHeartbeatGRPCClient(conn), nil
}

func (p *RemotePeer) reconnect(i int) {
	for {
		if ok := p.doReconnect(i); ok {
			plog.Info("[HealthCheck][Leader] reconnect all success", zap.String("host", p.Host()),
				zap.Uint32("port", p.port))
			return
		}
		time.Sleep(time.Second)
	}
}

func (p *RemotePeer) doReconnect(i int) bool {
	p.cmutex.Lock()
	defer p.cmutex.Unlock()

	// 先关闭老的连接
	if oldConn := p.conns[i]; oldConn != nil {
		_ = oldConn.Close()
	}
	if oldSender := p.puters[i]; oldSender != nil {
		_ = oldSender.close()
	}

	// 先删除有问题的 connection
	delete(p.conns, i)
	delete(p.puters, i)

	conn, err := grpc.DialContext(context.Background(), fmt.Sprintf("%s:%d", p.Host(), p.port),
		grpc.WithBlock(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithTimeout(5*time.Second),
	)
	if err != nil {
		plog.Error("[HealthCheck][Leader] reconnect connection", zap.String("host", p.Host()),
			zap.Uint32("port", p.port), zap.Error(err))
		return false
	}

	sender, err := newBeatSender(i, conn, p)
	if err != nil {
		plog.Error("[HealthCheck][Leader] reconnect grpc-client", zap.String("host", p.Host()),
			zap.Uint32("port", p.port), zap.Error(err))
		_ = conn.Close()
		return false
	}

	p.conns[i] = conn
	p.puters[i] = sender
	return true
}

func execPing(ctx context.Context, p *RemotePeer) error {
	val := ctx.Value(PingFuncContextKey{})
	pingFunc, ok := val.(PingFunc)
	if !ok {
		pingFunc = p.Ping
	}
	return pingFunc()
}

func doConnect(p *RemotePeer) error {
	p.conns = make(map[int]*grpc.ClientConn, streamNum)
	p.puters = make(map[int]*beatSender, streamNum)
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
		p.conns[i] = conn
	}
	for i := 0; i < streamNum; i++ {
		sender, err := newBeatSender(i, p.conns[i], p)
		if err != nil {
			p.doClose()
			return err
		}
		p.puters[i] = sender
	}
	return nil
}

func newBeatSender(index int, conn *grpc.ClientConn, p *RemotePeer) (*beatSender, error) {
	client := apiservice.NewPolarisHeartbeatGRPCClient(conn)
	puter, err := client.BatchHeartbeat(context.Background(), grpc.Header(&metadata.MD{
		sendResource: []string{utils.LocalHost},
	}))
	if err != nil {
		return nil, err
	}

	sender := &beatSender{
		index:  index,
		peer:   p,
		sender: puter,
		lock:   &sync.RWMutex{},
	}
	go sender.Recv()
	return sender, nil
}

type beatSender struct {
	index  int
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

// Recv .
func (s *beatSender) Recv() {
	for {
		if _, err := s.sender.Recv(); err != nil {
			plog.Error("[HealthCheck][Leader] receive put record result", zap.String("host", s.peer.Host()),
				zap.Uint32("port", s.peer.port), zap.Error(err))
			// 先关闭自己
			s.close()
			go s.peer.reconnect(s.index)
			return
		}
	}
}

func (s *beatSender) close() error {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.sender.CloseSend()
}
