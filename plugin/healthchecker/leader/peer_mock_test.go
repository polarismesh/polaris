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
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	"github.com/polarismesh/specification/source/go/api/v1/service_manage"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/polarismesh/polaris/common/eventhub"
	commonhash "github.com/polarismesh/polaris/common/hash"
	commontime "github.com/polarismesh/polaris/common/time"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/plugin"
	"github.com/polarismesh/polaris/store/mock"
)

type MockPeerImpl struct {
	OnServe   func(ctx context.Context, p *MockPeerImpl, listenIP string, listenPort uint32) error
	OnClose   func(mp *MockPeerImpl) error
	OnHost    func() string
	OnIsAlive func() bool
	OnStorage func() BeatRecordCache
}

// Initialize
func (mp *MockPeerImpl) Initialize(conf Config) {
	mp.OnServe = func(ctx context.Context, p *MockPeerImpl, listenIP string, listenPort uint32) error {
		return nil
	}
	mp.OnHost = func() string {
		return utils.LocalHost
	}
	mp.OnIsAlive = func() bool {
		return true
	}
	mp.OnStorage = func() BeatRecordCache {
		return newLocalBeatRecordCache(1, commonhash.Fnv32)
	}
	mp.OnClose = func(mp *MockPeerImpl) error {
		return nil
	}
}

// Serve
func (mp *MockPeerImpl) Serve(ctx context.Context, checker *LeaderHealthChecker, listenIP string, listenPort uint32) error {
	if mp.OnServe != nil {
		return mp.OnServe(ctx, mp, listenIP, listenPort)
	}
	return nil
}

func (mp *MockPeerImpl) IsAlive() bool {
	return mp.OnIsAlive()
}

// Get
func (mp *MockPeerImpl) Storage() BeatRecordCache {
	return mp.OnStorage()
}

// Close
func (mp *MockPeerImpl) Close() error {
	if mp.OnClose == nil {
		return nil
	}
	return mp.OnClose(mp)
}

// Host
func (mp *MockPeerImpl) Host() string {
	if mp.OnHost == nil {
		return ""
	}
	return mp.OnHost()
}

type MockStreamBeatClient struct {
	peer *LocalPeer
}

func (mc *MockStreamBeatClient) Send(req *service_manage.HeartbeatsRequest) error {
	for i := range req.Heartbeats {
		item := req.Heartbeats[i]
		err := mc.peer.Cache.Put(WriteBeatRecord{
			Key: item.InstanceId,
			Record: RecordValue{
				Server:     mc.peer.Host(),
				CurTimeSec: commontime.CurrentMillisecond(),
				Count:      0,
			},
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (mc *MockStreamBeatClient) Recv() (*service_manage.HeartbeatsResponse, error) {
	return &service_manage.HeartbeatsResponse{}, nil
}

func (mc *MockStreamBeatClient) Header() (metadata.MD, error) {
	return metadata.MD{}, nil
}

func (mc *MockStreamBeatClient) Trailer() metadata.MD {
	return metadata.MD{}
}

func (mc *MockStreamBeatClient) CloseSend() error {
	return nil
}

func (mc *MockStreamBeatClient) Context() context.Context {
	return context.Background()
}

func (mc *MockStreamBeatClient) SendMsg(m any) error {
	return nil
}

func (mc *MockStreamBeatClient) RecvMsg(m any) error {
	return nil
}

type MockPolarisHeartbeatClient struct {
	peer *LocalPeer
}

// 被调方批量上报心跳
func (mc *MockPolarisHeartbeatClient) BatchHeartbeat(ctx context.Context,
	opts ...grpc.CallOption) (service_manage.PolarisHeartbeatGRPC_BatchHeartbeatClient, error) {
	return &MockStreamBeatClient{
		peer: mc.peer,
	}, nil
}

// 批量获取心跳记录
func (mc *MockPolarisHeartbeatClient) BatchGetHeartbeat(ctx context.Context,
	in *service_manage.GetHeartbeatsRequest, opts ...grpc.CallOption) (*service_manage.GetHeartbeatsResponse, error) {
	rsp := &service_manage.GetHeartbeatsResponse{
		Records: []*service_manage.HeartbeatRecord{},
	}
	for i := range in.InstanceIds {
		key := in.InstanceIds[i]
		ret, err := mc.peer.Storage().Get(key)
		if err != nil {
			return nil, err
		}
		val, ok := ret[key]
		rsp.Records = append(rsp.Records, &service_manage.HeartbeatRecord{
			InstanceId: key,
			LastHeartbeatSec: func() int64 {
				if ok {
					return val.Record.CurTimeSec
				}
				return 0
			}(),
			Exist: func() bool {
				if ok {
					return val.Exist
				}
				return false
			}(),
		})
	}

	return rsp, nil
}

// 批量删除心跳记录
func (mc *MockPolarisHeartbeatClient) BatchDelHeartbeat(ctx context.Context,
	in *service_manage.DelHeartbeatsRequest, opts ...grpc.CallOption) (*service_manage.DelHeartbeatsResponse, error) {
	for i := range in.InstanceIds {
		key := in.InstanceIds[i]
		err := mc.peer.Storage().Del(key)
		if err != nil {
			return nil, err
		}
	}
	return &service_manage.DelHeartbeatsResponse{}, nil
}

func newMockPolarisGRPCSever(t *testing.T, port uint32) (*MockPolarisGRPCServer, error) {
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = ln.Close()
	})
	ctrl := gomock.NewController(t)
	eventhub.InitEventHub()
	t.Cleanup(func() {
		ctrl.Finish()
	})
	mockStore := mock.NewMockStore(ctrl)
	mockStore.EXPECT().StartLeaderElection(gomock.Any()).Return(nil)
	checker := &LeaderHealthChecker{
		self: NewLocalPeerFunc(),
		s:    mockStore,
		conf: &Config{
			SoltNum: 1,
		},
	}
	err = checker.Initialize(&plugin.ConfigEntry{
		Option: map[string]interface{}{},
	})
	assert.NoError(t, err)
	lp := NewLocalPeerFunc().(*LocalPeer)
	lp.Initialize(Config{
		SoltNum: 1,
	})

	err = lp.Serve(context.Background(), checker, "127.0.0.1", port)
	assert.NoError(t, err)
	svr := &MockPolarisGRPCServer{
		peer: lp,
	}
	return svr, nil
}

// PolarisGRPCServer is the server API for PolarisGRPC service.
type MockPolarisGRPCServer struct {
	peer *LocalPeer
}

func (ms *MockPolarisGRPCServer) mockRemotePeerConnect(p *RemotePeer) error {
	mockClient := &MockPolarisHeartbeatClient{
		peer: ms.peer,
	}
	sender, err := mockClient.BatchHeartbeat(context.Background())
	if err != nil {
		return err
	}
	p.conns = map[int]*grpc.ClientConn{
		0: &grpc.ClientConn{},
	}
	p.puters = map[int]*beatSender{
		0: {
			lock:   &sync.RWMutex{},
			sender: sender,
			peer:   p,
			cancel: p.cancel,
		},
	}
	return nil
}

// BatchHeartbeat 批量上报心跳
func (ms *MockPolarisGRPCServer) BatchHeartbeat(svr service_manage.PolarisHeartbeatGRPC_BatchHeartbeatServer) error {
	for {
		req, err := svr.Recv()
		if err != nil {
			if io.EOF == err {
				return nil
			}
			return err
		}

		heartbeats := req.GetHeartbeats()
		for i := range heartbeats {
			ms.peer.Storage().Put(WriteBeatRecord{
				Record: RecordValue{
					CurTimeSec: time.Now().Unix(),
				},
				Key: heartbeats[i].GetInstanceId(),
			})
		}

		if err = svr.Send(&service_manage.HeartbeatsResponse{}); err != nil {
			return err
		}
	}
}

// 批量获取心跳记录
func (ms *MockPolarisGRPCServer) BatchGetHeartbeat(_ context.Context,
	req *service_manage.GetHeartbeatsRequest) (*service_manage.GetHeartbeatsResponse, error) {
	keys := req.GetInstanceIds()
	records := make([]*service_manage.HeartbeatRecord, 0, len(keys))
	for i := range keys {
		ret, err := ms.peer.Storage().Get(keys[i])
		if err != nil {
			return nil, err
		}
		record := &service_manage.HeartbeatRecord{
			InstanceId:       keys[i],
			LastHeartbeatSec: ret[keys[i]].Record.CurTimeSec,
			Exist:            ret[keys[i]].Exist,
		}
		records = append(records, record)
	}
	return &service_manage.GetHeartbeatsResponse{
		Records: records,
	}, nil
}

// 批量删除心跳记录
func (ms *MockPolarisGRPCServer) BatchDelHeartbeat(_ context.Context,
	req *service_manage.DelHeartbeatsRequest) (*service_manage.DelHeartbeatsResponse, error) {
	keys := req.GetInstanceIds()
	for i := range keys {
		if err := ms.peer.Storage().Del(keys[i]); err != nil {
			return nil, err
		}
	}
	return &service_manage.DelHeartbeatsResponse{
		Code: uint32(apimodel.Code_ExecuteSuccess),
	}, nil
}
