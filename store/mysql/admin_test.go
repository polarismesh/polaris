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
	"os"
	"testing"

	"github.com/golang/mock/gomock"

	"github.com/polarismesh/polaris/common/eventhub"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/store/mock"
)

const (
	TestElectKey = "test-key"
)

func setup() {
	eventhub.InitEventHub()
}

func teardown() {
}

func TestAdminStore_LeaderElection_Follower1(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mock.NewMockLeaderElectionStore(ctrl)
	mockStore.EXPECT().CheckMtimeExpired(TestElectKey, int32(LeaseTime)).Return("127.0.0.2", false, nil)

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

func TestAdminStore_LeaderElection_Follower2(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mock.NewMockLeaderElectionStore(ctrl)
	mockStore.EXPECT().CheckMtimeExpired(TestElectKey, int32(LeaseTime)).Return(utils.LocalHost, true, nil)
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

func TestAdminStore_LeaderElection_Follower3(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mock.NewMockLeaderElectionStore(ctrl)
	mockStore.EXPECT().CheckMtimeExpired(TestElectKey, int32(LeaseTime)).Return(utils.LocalHost, false, errors.New("err"))
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

func TestAdminStore_LeaderElection_Follower4(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mock.NewMockLeaderElectionStore(ctrl)
	mockStore.EXPECT().CheckMtimeExpired(TestElectKey, int32(LeaseTime)).Return(utils.LocalHost, true, nil)
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

func TestAdminStore_LeaderElection_Follower5(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mock.NewMockLeaderElectionStore(ctrl)
	mockStore.EXPECT().CheckMtimeExpired(TestElectKey, int32(LeaseTime)).Return(utils.LocalHost, true, nil)
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

func TestAdminStore_LeaderElection_FollowerToLeader(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mock.NewMockLeaderElectionStore(ctrl)
	mockStore.EXPECT().CheckMtimeExpired(TestElectKey, int32(LeaseTime)).Return(utils.LocalHost, true, nil)
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

func TestAdminStore_LeaderElection_Leader1(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mock.NewMockLeaderElectionStore(ctrl)
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

func TestAdminStore_LeaderElection_LeaderToFollower1(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mock.NewMockLeaderElectionStore(ctrl)
	mockStore.EXPECT().CheckMtimeExpired(gomock.Any(), gomock.Any()).Return("127.0.0.2", false, nil)
	mockStore.EXPECT().CompareAndSwapVersion(TestElectKey, int64(42), int64(43), "127.0.0.1").Return(false, errors.New("err"))

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

func TestAdminStore_LeaderElection_LeaderToFollower2(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mock.NewMockLeaderElectionStore(ctrl)
	mockStore.EXPECT().CheckMtimeExpired(gomock.Any(), gomock.Any()).Return("127.0.0.2", false, nil)
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

func TestAdminStore_StartLeaderElection1(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mock.NewMockLeaderElectionStore(ctrl)
	mockStore.EXPECT().CreateLeaderElection(TestElectKey).Return(errors.New("err"))

	m := &adminStore{
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

func TestAdminStore_StartLeaderElection2(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mock.NewMockLeaderElectionStore(ctrl)
	mockStore.EXPECT().CreateLeaderElection(TestElectKey).Return(nil)

	m := &adminStore{
		leStore: mockStore,
		leMap:   make(map[string]*leaderElectionStateMachine),
	}

	err := m.StartLeaderElection(TestElectKey)
	if err != nil {
		t.Errorf("should start success")
	}
	_, ok := m.leMap[TestElectKey]
	if !ok {
		t.Errorf("should in map")
	}

	m.StopLeaderElections()
}

func TestAdminStore_StartLeaderElection3(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mock.NewMockLeaderElectionStore(ctrl)
	mockStore.EXPECT().CreateLeaderElection(TestElectKey).Return(nil)

	m := &adminStore{
		leStore: mockStore,
		leMap:   make(map[string]*leaderElectionStateMachine),
	}

	err := m.StartLeaderElection(TestElectKey)
	if err != nil {
		t.Errorf("should start success")
	}
	_, ok := m.leMap[TestElectKey]
	if !ok {
		t.Errorf("should in map")
	}

	err = m.StartLeaderElection(TestElectKey)
	if err != nil {
		t.Errorf("expect no err if already started")
	}

	m.StopLeaderElections()
}

func TestAdminStore_ReleaseLeaderElection1(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mock.NewMockLeaderElectionStore(ctrl)
	mockStore.EXPECT().CompareAndSwapVersion(TestElectKey, int64(42), int64(43), "127.0.0.1").Return(true, nil)

	ctx, cancel := context.WithCancel(context.TODO())
	le := leaderElectionStateMachine{
		electKey:         TestElectKey,
		leStore:          mockStore,
		leaderFlag:       1,
		version:          42,
		ctx:              ctx,
		cancel:           cancel,
		releaseSignal:    0,
		releaseTickLimit: 0,
	}

	le.tick()
	if !le.isLeaderAtomic() {
		t.Error("expect stay leader state")
	}

	le.setReleaseSignal()
	le.tick()
	if le.isLeaderAtomic() {
		t.Error("expect to follower state")
	}

	limit := le.releaseTickLimit
	for i := 0; i < int(limit); i++ {
		le.tick()
		if le.isLeaderAtomic() {
			t.Error("expect stay follower state")
		}
	}

	mockStore.EXPECT().CheckMtimeExpired(TestElectKey, int32(LeaseTime)).Return(utils.LocalHost, true, nil)
	mockStore.EXPECT().GetVersion(TestElectKey).Return(int64(101), nil)
	mockStore.EXPECT().CompareAndSwapVersion(TestElectKey, int64(101), int64(102), "127.0.0.1").Return(true, nil)

	le.tick()
	if !le.isLeaderAtomic() {
		t.Error("expect to leader state")
	}
}

func TestAdminStore_ReleaseLeaderElection2(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mock.NewMockLeaderElectionStore(ctrl)
	mockStore.EXPECT().CreateLeaderElection(TestElectKey).Return(nil)

	m := &adminStore{
		leStore: mockStore,
		leMap:   make(map[string]*leaderElectionStateMachine),
	}

	err := m.ReleaseLeaderElection(TestElectKey)
	if err == nil {
		t.Error("expect err when release not existed key")
	}

	_ = m.StartLeaderElection(TestElectKey)
	err = m.ReleaseLeaderElection(TestElectKey)
	if err != nil {
		t.Errorf("unexpect err: %v", err)
	}

	m.StopLeaderElections()
}

func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	teardown()
	os.Exit(code)
}
