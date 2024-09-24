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
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/metadata"

	"github.com/polarismesh/polaris/common/eventhub"
	commontime "github.com/polarismesh/polaris/common/time"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/plugin"
	"github.com/polarismesh/polaris/store"
	"github.com/polarismesh/polaris/store/mock"
)

func TestLeaderHealthChecker_Function(t *testing.T) {
	ctrl := gomock.NewController(t)
	eventhub.InitEventHub()
	t.Cleanup(func() {
		ctrl.Finish()
	})
	mockStore := mock.NewMockStore(ctrl)
	mockStore.EXPECT().StartLeaderElection(gomock.Any()).Return(nil).AnyTimes()
	checker := &LeaderHealthChecker{
		self: NewLocalPeerFunc(),
		s:    mockStore,
		conf: &Config{
			SoltNum: 0,
		},
	}
	err := checker.Initialize(&plugin.ConfigEntry{
		Option: map[string]interface{}{},
	})
	assert.NoError(t, err)

	t.Run("follower_case", func(t *testing.T) {
		atomic.StoreInt32(&checker.leader, 0)

		t.Run("duplicate_request", func(t *testing.T) {
			ctx := metadata.NewIncomingContext(context.Background(), metadata.New(map[string]string{
				sendResource: utils.LocalHost,
			}))
			t.Run("report", func(t *testing.T) {
				err := checker.Report(ctx, &plugin.ReportRequest{})
				assert.ErrorIs(t, err, ErrorRedirectOnlyOnce)
			})
			t.Run("query", func(t *testing.T) {
				_, err := checker.Query(ctx, &plugin.QueryRequest{})
				assert.ErrorIs(t, err, ErrorRedirectOnlyOnce)
			})
			t.Run("batch-query", func(t *testing.T) {
				_, err := checker.BatchQuery(ctx, &plugin.BatchQueryRequest{})
				assert.ErrorIs(t, err, ErrorRedirectOnlyOnce)
			})
			t.Run("delete", func(t *testing.T) {
				err := checker.Delete(ctx, "")
				assert.ErrorIs(t, err, ErrorRedirectOnlyOnce)
			})
		})
	})

	t.Run("not-initialize", func(t *testing.T) {
		ctx := context.Background()
		atomic.StoreInt32(&checker.initialize, 0)
		t.Run("report", func(t *testing.T) {
			err := checker.Report(ctx, &plugin.ReportRequest{})
			assert.ErrorIs(t, err, ErrorLeaderNotInitialize)
		})
		t.Run("query", func(t *testing.T) {
			_, err := checker.Query(ctx, &plugin.QueryRequest{})
			assert.ErrorIs(t, err, ErrorLeaderNotInitialize)
		})
		t.Run("batch-query", func(t *testing.T) {
			_, err := checker.BatchQuery(ctx, &plugin.BatchQueryRequest{})
			assert.ErrorIs(t, err, ErrorLeaderNotInitialize)
		})
		t.Run("delete", func(t *testing.T) {
			err := checker.Delete(ctx, "")
			assert.ErrorIs(t, err, ErrorLeaderNotInitialize)
		})
	})
}

// TestLeaderHealthChecker_Switch 测试 Leader 事件切换
func TestLeaderHealthChecker_Switch(t *testing.T) {
	ctrl := gomock.NewController(t)
	eventhub.InitEventHub()
	t.Cleanup(func() {
		ctrl.Finish()
	})
	mockStore := mock.NewMockStore(ctrl)
	mockStore.EXPECT().StartLeaderElection(gomock.Any()).Return(nil).AnyTimes()

	checker := &LeaderHealthChecker{
		self: NewLocalPeerFunc(),
		s:    mockStore,
		conf: &Config{
			SoltNum: 0,
		},
	}
	err := checker.Initialize(&plugin.ConfigEntry{
		Option: map[string]interface{}{},
	})
	assert.NoError(t, err)

	mockPort := uint32(28888)
	_, err = newMockPolarisGRPCSever(t, mockPort)
	assert.NoError(t, err)

	oldNewRemoteFunc := NewRemotePeerFunc
	t.Cleanup(func() {
		NewRemotePeerFunc = oldNewRemoteFunc
	})
	// 这里要模拟一个远程节点，规避 RemotePeer 内部的真实逻辑
	NewRemotePeerFunc = func() Peer {
		return &MockPeerImpl{}
	}

	oldHost := utils.LocalHost
	oldPort := utils.LocalPort
	utils.LocalHost = "127.0.0.2"
	utils.LocalPort = int(mockPort)
	t.Cleanup(func() {
		utils.LocalPort = oldPort
		utils.LocalHost = oldHost
	})

	t.Run("initialize-self-is-follower", func(t *testing.T) {
		checker.OnEvent(context.Background(), store.LeaderChangeEvent{
			Key:        electionKey,
			Leader:     false,
			LeaderHost: "127.0.0.1",
		})

		assert.True(t, checker.isInitialize())
		assert.False(t, checker.isLeader())

		skipRet := checker.skipCheck(utils.NewUUID(), 15)
		assert.True(t, skipRet)
		time.Sleep(15 * time.Second)
		skipRet = checker.skipCheck(utils.NewUUID(), 15)
		assert.False(t, skipRet)

		peer := checker.findLeaderPeer()
		assert.NotNil(t, peer)
		_, ok := peer.(*MockPeerImpl)
		assert.True(t, ok)
	})

	t.Run("initialize-self-become-leader", func(t *testing.T) {
		checker.OnEvent(context.Background(), store.LeaderChangeEvent{
			Key:        electionKey,
			Leader:     true,
			LeaderHost: "127.0.0.2",
		})

		assert.True(t, checker.isInitialize())
		assert.True(t, checker.isLeader())
		assert.Nil(t, checker.remote)

		skipRet := checker.skipCheck(utils.NewUUID(), 15)
		assert.True(t, skipRet)
		time.Sleep(15 * time.Second)
		skipRet = checker.skipCheck(utils.NewUUID(), 15)
		assert.False(t, skipRet)

		peer := checker.findLeaderPeer()
		assert.NotNil(t, peer)
		_, ok := peer.(*LocalPeer)
		assert.True(t, ok)
	})
}

func TestLeaderHealthChecker_Normal(t *testing.T) {
	ctrl := gomock.NewController(t)
	eventhub.InitEventHub()
	t.Cleanup(func() {
		ctrl.Finish()
	})
	mockStore := mock.NewMockStore(ctrl)
	mockStore.EXPECT().StartLeaderElection(gomock.Any()).Return(nil).AnyTimes()
	checker := &LeaderHealthChecker{
		self: NewLocalPeerFunc(),
		s:    mockStore,
		conf: &Config{
			SoltNum: 0,
		},
		// 设置为 Leader
		leader: 1,
		// 设置为已经初始化结束
		initialize: 1,
	}
	err := checker.Initialize(&plugin.ConfigEntry{
		Option: map[string]interface{}{},
	})
	assert.NoError(t, err)

	mockInstanceId := "mockInstanceId"
	mockInstanceHost := "1.1.1.1"
	mockInstancePort := uint32(8080)

	t.Run("report", func(t *testing.T) {
		err := checker.Report(context.Background(), &plugin.ReportRequest{
			QueryRequest: plugin.QueryRequest{
				InstanceId: mockInstanceId,
				Host:       mockInstanceHost,
				Port:       mockInstancePort,
			},
			LocalHost:  utils.LocalHost,
			CurTimeSec: commontime.CurrentMillisecond(),
		})
		assert.NoError(t, err)

		t.Run("abnormal-query", func(t *testing.T) {
			rsp, err := checker.Query(context.Background(), &plugin.QueryRequest{
				InstanceId: mockInstanceId + "noExist",
				Host:       mockInstanceHost,
				Port:       mockInstancePort,
			})
			assert.NoError(t, err)
			assert.False(t, rsp.Exists)
		})

		t.Run("normal-query", func(t *testing.T) {
			rsp, err := checker.Query(context.Background(), &plugin.QueryRequest{
				InstanceId: mockInstanceId,
				Host:       mockInstanceHost,
				Port:       mockInstancePort,
			})
			assert.NoError(t, err)
			assert.True(t, rsp.Exists)
			assert.True(t, rsp.LastHeartbeatSec <= commontime.CurrentMillisecond())
		})

		err = checker.Delete(context.Background(), mockInstanceId)
		assert.NoError(t, err)

		t.Run("query-should-noexist", func(t *testing.T) {
			rsp, err := checker.Query(context.Background(), &plugin.QueryRequest{
				InstanceId: mockInstanceId,
				Host:       mockInstanceHost,
				Port:       mockInstancePort,
			})
			assert.NoError(t, err)
			assert.False(t, rsp.Exists)
		})
	})
}
