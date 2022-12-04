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

package sqldb

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
)

const (
	TestElectKey = "test-key"
)

func TestMaintainStore_LeaderElection_Follower1(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := NewMockLeaderElectionStore(ctrl)
	mockStore.EXPECT().CheckMtimeExpired(TestElectKey, int32(LeaseTime)).Return(false, nil)

	ctx, cancel := context.WithCancel(context.TODO())
	le := leaderElectionStateMachine{
		electKey:   TestElectKey,
		leStore:    mockStore,
		leaderFlag: 0,
		version:    0,
		ctx:        ctx,
		cancel:     cancel,
	}

	le.tick()
	if le.isLeaderAtomic() {
		t.Error("expect stay follower state")
	}
}

func TestMaintainStore_LeaderElection_Follower2(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := NewMockLeaderElectionStore(ctrl)
	mockStore.EXPECT().CheckMtimeExpired(TestElectKey, int32(LeaseTime)).Return(true, nil)
	mockStore.EXPECT().GetVersion(TestElectKey).Return(int64(0), nil)
	mockStore.EXPECT().CompareAndSwapVersion(TestElectKey, int64(0), int64(1), "127.0.0.1").Return(false, nil)

	ctx, cancel := context.WithCancel(context.TODO())
	le := leaderElectionStateMachine{
		electKey:   TestElectKey,
		leStore:    mockStore,
		leaderFlag: 0,
		version:    0,
		ctx:        ctx,
		cancel:     cancel,
	}

	le.tick()
	if le.isLeaderAtomic() {
		t.Error("expect stay follower state")
	}
}

func TestMaintainStore_LeaderElection_Follower3(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := NewMockLeaderElectionStore(ctrl)
	mockStore.EXPECT().CheckMtimeExpired(TestElectKey, int32(LeaseTime)).Return(false, errors.New("err"))
	ctx, cancel := context.WithCancel(context.TODO())
	le := leaderElectionStateMachine{
		electKey:   TestElectKey,
		leStore:    mockStore,
		leaderFlag: 0,
		version:    0,
		ctx:        ctx,
		cancel:     cancel,
	}

	le.tick()
	if le.isLeaderAtomic() {
		t.Error("expect stay follower state")
	}
}

func TestMaintainStore_LeaderElection_Follower4(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := NewMockLeaderElectionStore(ctrl)
	mockStore.EXPECT().CheckMtimeExpired(TestElectKey, int32(LeaseTime)).Return(true, nil)
	mockStore.EXPECT().GetVersion(TestElectKey).Return(int64(0), errors.New("err"))

	ctx, cancel := context.WithCancel(context.TODO())
	le := leaderElectionStateMachine{
		electKey:   TestElectKey,
		leStore:    mockStore,
		leaderFlag: 0,
		version:    0,
		ctx:        ctx,
		cancel:     cancel,
	}

	le.tick()
	if le.isLeaderAtomic() {
		t.Error("expect stay follower state")
	}
}

func TestMaintainStore_LeaderElection_Follower5(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := NewMockLeaderElectionStore(ctrl)
	mockStore.EXPECT().CheckMtimeExpired(TestElectKey, int32(LeaseTime)).Return(true, nil)
	mockStore.EXPECT().GetVersion(TestElectKey).Return(int64(0), nil)
	mockStore.EXPECT().CompareAndSwapVersion(TestElectKey, int64(0), int64(1), "127.0.0.1").Return(false, errors.New("err"))

	ctx, cancel := context.WithCancel(context.TODO())
	le := leaderElectionStateMachine{
		electKey:   TestElectKey,
		leStore:    mockStore,
		leaderFlag: 0,
		version:    0,
		ctx:        ctx,
		cancel:     cancel,
	}

	le.tick()
	if le.isLeaderAtomic() {
		t.Error("expect stay follower state")
	}
}

func TestMaintainStore_LeaderElection_FollowerToLeader(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := NewMockLeaderElectionStore(ctrl)
	mockStore.EXPECT().CheckMtimeExpired(TestElectKey, int32(LeaseTime)).Return(true, nil)
	mockStore.EXPECT().GetVersion(TestElectKey).Return(int64(42), nil)
	mockStore.EXPECT().CompareAndSwapVersion(TestElectKey, int64(42), int64(43), "127.0.0.1").Return(true, nil)

	ctx, cancel := context.WithCancel(context.TODO())
	le := leaderElectionStateMachine{
		electKey:   TestElectKey,
		leStore:    mockStore,
		leaderFlag: 0,
		version:    0,
		ctx:        ctx,
		cancel:     cancel,
	}

	le.tick()
	if !le.isLeaderAtomic() {
		t.Error("expect to leader state")
	}
	if le.version != 43 {
		t.Errorf("epect version is %d, actual is %d", 43, le.version)
	}
}

func TestMaintainStore_LeaderElection_Leader1(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := NewMockLeaderElectionStore(ctrl)
	mockStore.EXPECT().CompareAndSwapVersion(TestElectKey, int64(42), int64(43), "127.0.0.1").Return(true, nil)

	ctx, cancel := context.WithCancel(context.TODO())
	le := leaderElectionStateMachine{
		electKey:   TestElectKey,
		leStore:    mockStore,
		leaderFlag: 1,
		version:    42,
		ctx:        ctx,
		cancel:     cancel,
	}

	le.tick()
	if !le.isLeaderAtomic() {
		t.Error("expect stay leader state")
	}
	if le.version != 43 {
		t.Errorf("epect version is %d, actual is %d", 43, le.version)
	}
}

func TestMaintainStore_LeaderElection_LeaderToFollower1(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := NewMockLeaderElectionStore(ctrl)
	mockStore.EXPECT().CompareAndSwapVersion(TestElectKey, int64(42), int64(43), "127.0.0.1").Return(true, errors.New("err"))

	ctx, cancel := context.WithCancel(context.TODO())
	le := leaderElectionStateMachine{
		electKey:   TestElectKey,
		leStore:    mockStore,
		leaderFlag: 1,
		version:    42,
		ctx:        ctx,
		cancel:     cancel,
	}

	le.tick()
	if le.isLeaderAtomic() {
		t.Error("expect to follower state")
	}
}

func TestMaintainStore_LeaderElection_LeaderToFollower2(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := NewMockLeaderElectionStore(ctrl)
	mockStore.EXPECT().CompareAndSwapVersion(TestElectKey, int64(42), int64(43), "127.0.0.1").Return(false, nil)

	ctx, cancel := context.WithCancel(context.TODO())
	le := leaderElectionStateMachine{
		electKey:   TestElectKey,
		leStore:    mockStore,
		leaderFlag: 1,
		version:    42,
		ctx:        ctx,
		cancel:     cancel,
	}

	le.tick()
	if le.isLeaderAtomic() {
		t.Error("expect to follower state")
	}
}

func TestMaintainStore_StartLeaderElection1(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := NewMockLeaderElectionStore(ctrl)
	mockStore.EXPECT().CreateLeaderElection(TestElectKey).Return(errors.New("err"))

	m := &maintainStore{
		leStore: mockStore,
		leMap:   make(map[string]*leaderElectionStateMachine),
	}

	err := m.StartLeaderElection(TestElectKey)
	if err == nil {
		t.Errorf("should start failed")
	}
	_, ok := m.leMap[TestElectKey]
	if ok {
		t.Errorf("should not in map")
	}
}

func TestMaintainStore_StartLeaderElection2(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := NewMockLeaderElectionStore(ctrl)
	mockStore.EXPECT().CreateLeaderElection(TestElectKey).Return(nil)

	m := &maintainStore{
		leStore: mockStore,
		leMap:   make(map[string]*leaderElectionStateMachine),
	}

	err := m.StartLeaderElection(TestElectKey)
	if err != nil {
		t.Errorf("should start success")
	}
	le, ok := m.leMap[TestElectKey]
	if !ok {
		t.Errorf("should in map")
	}

	le.cancel()
}

func TestMaintainStore_StartLeaderElection3(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := NewMockLeaderElectionStore(ctrl)
	mockStore.EXPECT().CreateLeaderElection(TestElectKey).Return(nil)

	m := &maintainStore{
		leStore: mockStore,
		leMap:   make(map[string]*leaderElectionStateMachine),
	}

	err := m.StartLeaderElection(TestElectKey)
	if err != nil {
		t.Errorf("should start success")
	}
	le, ok := m.leMap[TestElectKey]
	if !ok {
		t.Errorf("should in map")
	}

	err = m.StartLeaderElection(TestElectKey)
	if err != nil {
		t.Errorf("expect no err if already started")
	}

	le.cancel()
}
