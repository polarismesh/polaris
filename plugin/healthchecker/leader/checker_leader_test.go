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
	"github.com/stretchr/testify/assert"

	"github.com/polarismesh/polaris/common/eventhub"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/plugin"
	"github.com/polarismesh/polaris/store"
	"github.com/polarismesh/polaris/store/mock"
)

func TestLeaderHealthChecker_OnEvent(t *testing.T) {
	t.SkipNow()
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

	utils.LocalHost = "127.0.0.2"
	utils.LocalPort = int(mockPort)
	t.Cleanup(func() {
		utils.LocalPort = 8091
		utils.LocalHost = "127.0.0.1"
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
		_, ok := peer.(*RemotePeer)
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
