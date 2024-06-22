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
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/polarismesh/specification/source/go/api/v1/service_manage"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"

	"github.com/polarismesh/polaris/common/eventhub"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/plugin"
	"github.com/polarismesh/polaris/store/mock"
)

func TestLocalPeer(t *testing.T) {
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

	localPeer.Initialize(Config{
		SoltNum: 1,
	})

	err = localPeer.Serve(context.Background(), checker, "127.0.0.1", 21111)
	assert.NoError(t, err)

	mockKey := utils.NewUUID()
	mockVal := time.Now().Unix()

	t.Cleanup(func() {
		_ = checker.Destroy()
		eventhub.InitEventHub()
		ctrl.Finish()
		err = localPeer.Close()
		assert.NoError(t, err)
	})

	t.Run("获取不存在的key", func(t *testing.T) {
		ret, err := localPeer.Storage().Get(mockKey)
		assert.NoError(t, err)
		assert.NotNil(t, ret)
		assert.False(t, ret[mockKey].Exist)
	})

	t.Run("先存入数据，再获取判断", func(t *testing.T) {
		err = localPeer.Storage().Put(WriteBeatRecord{
			Record: RecordValue{
				CurTimeSec: mockVal,
				Count:      0,
			},
			Key: mockKey,
		})
		assert.NoError(t, err)

		ret, err := localPeer.Storage().Get(mockKey)
		assert.NoError(t, err)
		assert.NotNil(t, ret)
		assert.True(t, ret[mockKey].Exist)
		assert.Equal(t, mockVal, ret[mockKey].Record.CurTimeSec)
	})

	t.Run("删除数据，不存在", func(t *testing.T) {
		err = localPeer.Storage().Del(mockKey)
		assert.NoError(t, err)

		ret, err := localPeer.Storage().Get(mockKey)
		assert.NoError(t, err)
		assert.NotNil(t, ret)
		assert.False(t, ret[mockKey].Exist)
	})
}

func TestRemotePeer(t *testing.T) {
	eventhub.InitEventHub()

	mockPort := uint32(21111)
	mockSvr, err := newMockPolarisGRPCSever(t, mockPort)
	assert.NoError(t, err)
	remotePeer := NewRemotePeerFunc()
	assert.NotNil(t, remotePeer)
	remotePeer.Initialize(Config{
		SoltNum: 1,
	})

	oldCreateBeatClient := CreateBeatClientFunc
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() {
		CreateBeatClientFunc = oldCreateBeatClient
		remotePeer.(*RemotePeer).cancel()
		cancel()
	})

	CreateBeatClientFunc = func(conn *grpc.ClientConn) (service_manage.PolarisHeartbeatGRPCClient, error) {
		return &MockPolarisHeartbeatClient{
			peer: mockSvr.peer,
		}, nil
	}

	ctx = context.WithValue(ctx, ConnectFuncContextKey{}, ConnectPeerFunc(mockSvr.mockRemotePeerConnect))
	ctx = context.WithValue(ctx, PingFuncContextKey{}, PingFunc(func() error {
		t.Logf("debug for peer check ping")
		return nil
	}))
	err = remotePeer.Serve(ctx, nil, "127.0.0.1", mockPort)
	assert.NoError(t, err)

	for {
		if remotePeer.IsAlive() {
			break
		}
		time.Sleep(time.Second)
		t.Logf("wait leader checker ready")
	}

	mockKey := utils.NewUUID()
	mockVal := time.Now().Unix()

	t.Run("获取不存在的数据", func(t *testing.T) {
		ret, err := remotePeer.Storage().Get(mockKey)
		assert.NoError(t, err, err)
		assert.NotNil(t, ret)
		assert.False(t, ret[mockKey].Exist, ret[mockKey])
	})

	t.Run("数据存入后再次取出", func(t *testing.T) {
		err = remotePeer.Storage().Put(WriteBeatRecord{
			Record: RecordValue{
				CurTimeSec: mockVal,
				Count:      0,
			},
			Key: mockKey,
		})
		assert.NoError(t, err)

		ret, err := remotePeer.Storage().Get(mockKey)
		assert.NoError(t, err)
		assert.NotNil(t, ret)
		assert.True(t, ret[mockKey].Exist)
		assert.True(t, mockVal <= ret[mockKey].Record.CurTimeSec)
	})

	t.Run("验证删除场景", func(t *testing.T) {
		err = remotePeer.Storage().Del(mockKey)
		assert.NoError(t, err)

		ret, err := remotePeer.Storage().Get(mockKey)
		assert.NoError(t, err)
		assert.NotNil(t, ret)
		assert.False(t, ret[mockKey].Exist, ret[mockKey])
	})
}
