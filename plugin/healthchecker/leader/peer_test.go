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
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	"github.com/polarismesh/specification/source/go/api/v1/service_manage"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"

	"github.com/polarismesh/polaris/common/eventhub"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/plugin"
	"github.com/polarismesh/polaris/store/mock"
)

type MockPeerImpl struct {
	OnServe func(ctx context.Context, p *MockPeerImpl, listenIP string, listenPort uint32) error
	OnGet   func(key string) (*ReadBeatRecord, error)
	OnPut   func(record WriteBeatRecord) error
	OnDel   func(key string) error
	OnClose func(mp *MockPeerImpl) error
	OnHost  func() string
}

// Initialize
func (mp *MockPeerImpl) Initialize(conf Config) {}

// Serve
func (mp *MockPeerImpl) Serve(ctx context.Context, listenIP string, listenPort uint32) error {
	if mp.OnServe != nil {
		return mp.OnServe(ctx, mp, listenIP, listenPort)
	}
	return nil
}

// Get
func (mp *MockPeerImpl) Get(key string) (*ReadBeatRecord, error) {
	if mp.OnGet == nil {
		return &ReadBeatRecord{}, nil
	}
	return mp.OnGet(key)
}

// Put
func (mp *MockPeerImpl) Put(record WriteBeatRecord) error {
	if mp.OnPut == nil {
		return nil
	}
	return mp.OnPut(record)
}

// Del
func (mp *MockPeerImpl) Del(key string) error {
	if mp.OnDel == nil {
		return nil
	}
	return mp.OnDel(key)
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

func TestLocalPeer(t *testing.T) {
	t.SkipNow()
	localPeer := newLocalPeer()
	assert.NotNil(t, localPeer)
	ctrl := gomock.NewController(t)
	eventhub.InitEventHub()
	mockStore := mock.NewMockStore(ctrl)
	mockStore.EXPECT().StartLeaderElection(gomock.Any()).Return(nil).AnyTimes()
	checker := &LeaderHealthChecker{
		self: NewLocalPeerFunc(),
		s:    mockStore,
		conf: &Config{
			SoltNum: 1,
		},
	}
	err := checker.Initialize(&plugin.ConfigEntry{
		Option: map[string]interface{}{},
	})
	assert.NoError(t, err)

	t.Cleanup(func() {
		_ = checker.Destroy()
		eventhub.InitEventHub()
		ctrl.Finish()
	})

	localPeer.Initialize(Config{
		SoltNum: 1,
	})

	err = localPeer.Serve(context.Background(), checker, "127.0.0.1", 21111)
	assert.NoError(t, err)

	mockKey := utils.NewUUID()
	mockVal := time.Now().Unix()

	ret, err := localPeer.Get(mockKey)
	assert.NoError(t, err)
	assert.NotNil(t, ret)
	assert.False(t, ret.Exist)

	err = localPeer.Put(WriteBeatRecord{
		Record: RecordValue{
			CurTimeSec: mockVal,
			Count:      0,
		},
		Key: mockKey,
	})
	assert.NoError(t, err)

	ret, err = localPeer.Get(mockKey)
	assert.NoError(t, err)
	assert.NotNil(t, ret)
	assert.True(t, ret.Exist)
	assert.Equal(t, mockVal, ret.Record.CurTimeSec)

	err = localPeer.Del(mockKey)
	assert.NoError(t, err)

	ret, err = localPeer.Get(mockKey)
	assert.NoError(t, err)
	assert.NotNil(t, ret)
	assert.False(t, ret.Exist)

	err = localPeer.Close()
	assert.NoError(t, err)
}

func TestRemotePeer(t *testing.T) {
	t.SkipNow()
	// close old event hub
	eventhub.InitEventHub()
	ctrl := gomock.NewController(t)
	mockStore := mock.NewMockStore(ctrl)
	mockStore.EXPECT().StartLeaderElection(gomock.Any()).Return(nil)
	checker := &LeaderHealthChecker{
		self: NewLocalPeerFunc(),
		s:    mockStore,
		conf: &Config{
			SoltNum: 1,
		},
	}

	err := checker.Initialize(&plugin.ConfigEntry{
		Option: map[string]interface{}{},
	})
	assert.NoError(t, err)
	t.Cleanup(func() {
		_ = checker.Destroy()
		eventhub.InitEventHub()
		ctrl.Finish()
	})

	mockPort := uint32(21111)
	_, err = newMockPolarisGRPCSever(t, mockPort)
	assert.NoError(t, err)
	remotePeer := NewRemotePeerFunc()
	assert.NotNil(t, remotePeer)
	remotePeer.Initialize(Config{
		SoltNum: 0,
	})

	err = remotePeer.Serve(context.Background(), checker, "127.0.0.1", mockPort)
	assert.NoError(t, err)

	mockKey := utils.NewUUID()
	mockVal := time.Now().Unix()

	ret, err := remotePeer.Get(mockKey)
	assert.NoError(t, err)
	assert.NotNil(t, ret)
	assert.False(t, ret.Exist)

	err = remotePeer.Put(WriteBeatRecord{
		Record: RecordValue{
			CurTimeSec: mockVal,
			Count:      0,
		},
		Key: mockKey,
	})
	assert.NoError(t, err)

	ret, err = remotePeer.Get(mockKey)
	assert.NoError(t, err)
	assert.NotNil(t, ret)
	assert.True(t, ret.Exist)
	assert.True(t, mockVal <= ret.Record.CurTimeSec)

	err = remotePeer.Del(mockKey)
	assert.NoError(t, err)

	ret, err = remotePeer.Get(mockKey)
	assert.NoError(t, err)
	assert.NotNil(t, ret)
	assert.False(t, ret.Exist)

	err = remotePeer.Close()
	assert.NoError(t, err)
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

	server := grpc.NewServer()
	service_manage.RegisterPolarisHeartbeatGRPCServer(server, svr)

	t.Cleanup(func() {
		server.Stop()
	})

	go func(t *testing.T) {
		if err := server.Serve(ln); err != nil {
			t.Error(err)
		}
	}(t)

	return svr, nil
}

// PolarisGRPCServer is the server API for PolarisGRPC service.
type MockPolarisGRPCServer struct {
	peer *LocalPeer
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
			ms.peer.Put(WriteBeatRecord{
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
		ret, err := ms.peer.Get(keys[i])
		if err != nil {
			return nil, err
		}
		record := &service_manage.HeartbeatRecord{
			InstanceId:       keys[i],
			LastHeartbeatSec: ret.Record.CurTimeSec,
			Exist:            ret.Exist,
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
		if err := ms.peer.Del(keys[i]); err != nil {
			return nil, err
		}
	}
	return &service_manage.DelHeartbeatsResponse{
		Code: uint32(apimodel.Code_ExecuteSuccess),
	}, nil
}
