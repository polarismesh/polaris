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
	"database/sql"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/polarismesh/polaris/common/eventhub"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/store"
)

const (
	TickTime  = 2
	LeaseTime = 10
)

// maintainStore implement MaintainStore interface
type maintainStore struct {
	master  *BaseDB
	leStore LeaderElectionStore
	leMap   map[string]*leaderElectionStateMachine
	mutex   sync.Mutex
}

func newMaintainStore(master *BaseDB) *maintainStore {
	return &maintainStore{
		master:  master,
		leStore: &leaderElectionStore{master: master},
		leMap:   make(map[string]*leaderElectionStateMachine),
	}
}

// LeaderElectionStore store inteface
type LeaderElectionStore interface {
	// CreateLeaderElection
	CreateLeaderElection(key string) error
	// GetVersion get current version
	GetVersion(key string) (int64, error)
	// CompareAndSwapVersion cas version
	CompareAndSwapVersion(key string, curVersion int64, newVersion int64, leader string) (bool, error)
	// CheckMtimeExpired check mtime expired
	CheckMtimeExpired(key string, leaseTime int32) (bool, error)
	// ListLeaderElections list all leaderelection
	ListLeaderElections() ([]*model.LeaderElection, error)
}

// leaderElectionStore
type leaderElectionStore struct {
	master *BaseDB
}

// CreateLeaderElection
func (l *leaderElectionStore) CreateLeaderElection(key string) error {
	log.Debugf("[Store][database] create leader election (%s)", key)
	mainStr := "insert ignore into leader_election (elect_key, leader) values (?, ?)"

	_, err := l.master.Exec(mainStr, key, "")
	if err != nil {
		log.Errorf("[Store][database] create leader election (%s), err: %s", key, err.Error())
	}
	return err
}

// GetVersion
func (l *leaderElectionStore) GetVersion(key string) (int64, error) {
	log.Debugf("[Store][database] get version (%s)", key)
	mainStr := "select version from leader_election where elect_key = ?"

	var count int64
	err := l.master.DB.QueryRow(mainStr, key).Scan(&count)
	if err != nil {
		log.Errorf("[Store][database] get version (%s), err: %s", key, err.Error())
	}
	return count, store.Error(err)
}

// CompareAndSwapVersion
func (l *leaderElectionStore) CompareAndSwapVersion(key string, curVersion int64, newVersion int64,
	leader string) (bool, error) {

	log.Debugf("[Store][database] compare and swap version (%s, %d, %d, %s)", key, curVersion, newVersion, leader)
	mainStr := "update leader_election set leader = ?, version = ? where elect_key = ? and version = ?"
	result, err := l.master.DB.Exec(mainStr, leader, newVersion, key, curVersion)
	if err != nil {
		log.Errorf("[Store][database] compare and swap version (%s), err: %s", key, err.Error())
		return false, store.Error(err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		log.Errorf("[Store][database] compare and swap version (%s), get RowsAffected err: %s", key, err.Error())
		return false, store.Error(err)
	}
	return (rows > 0), nil
}

// CheckMtimeExpired
func (l *leaderElectionStore) CheckMtimeExpired(key string, leaseTime int32) (bool, error) {
	log.Debugf("[Store][database] check mtime expired (%s, %d)", key, leaseTime)
	mainStr := "select count(1) from leader_election where elect_key = ? and mtime < " +
		" FROM_UNIXTIME(UNIX_TIMESTAMP(SYSDATE()) - ?)"

	var count int32
	err := l.master.DB.QueryRow(mainStr, key, leaseTime).Scan(&count)
	if err != nil {
		log.Errorf("[Store][database] check mtime expired (%s), err: %s", key, err.Error())
	}
	return (count > 0), store.Error(err)
}

// ListLeaderElection
func (l *leaderElectionStore) ListLeaderElections() ([]*model.LeaderElection, error) {
	log.Info("[Store][database] list leader election")
	mainStr := "select elect_key, leader, UNIX_TIMESTAMP(ctime), UNIX_TIMESTAMP(mtime) from leader_election"

	rows, err := l.master.Query(mainStr)
	if err != nil {
		log.Errorf("[Store][database] list leader election query err: %s", err.Error())
		return nil, store.Error(err)
	}

	return fetchLeaderElectionRows(rows)
}

func fetchLeaderElectionRows(rows *sql.Rows) ([]*model.LeaderElection, error) {
	if rows == nil {
		return nil, nil
	}
	defer rows.Close()

	var out []*model.LeaderElection

	for rows.Next() {
		space := &model.LeaderElection{}
		err := rows.Scan(
			&space.ElectKey,
			&space.Host,
			&space.Ctime,
			&space.Mtime)
		if err != nil {
			log.Errorf("[Store][database] fetch leader election rows scan err: %s", err.Error())
			return nil, err
		}

		space.CreateTime = time.Unix(space.Ctime, 0)
		space.ModifyTime = time.Unix(space.Mtime, 0)
		space.Valid = checkLeaderValid(space.Mtime)
		out = append(out, space)
	}
	if err := rows.Err(); err != nil {
		log.Errorf("[Store][database] fetch leader election rows next err: %s", err.Error())
		return nil, err
	}

	return out, nil
}

func checkLeaderValid(mtime int64) bool {
	delta := time.Now().Unix() - mtime
	return delta <= LeaseTime
}

// leaderElectionStateMachine
type leaderElectionStateMachine struct {
	electKey         string
	leStore          LeaderElectionStore
	leaderFlag       int32
	version          int64
	ctx              context.Context
	cancel           context.CancelFunc
	releaseSignal    int32
	releaseTickLimit int32
}

// isLeader
func isLeader(flag int32) bool {
	return flag > 0
}

// mainLoop
func (le *leaderElectionStateMachine) mainLoop() {
	log.Infof("[Store][database] leader election started (%s)", le.electKey)
	ticker := time.NewTicker(TickTime * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			le.tick()
		case <-le.ctx.Done():
			log.Infof("[Store][database] leader election stopped (%s)", le.electKey)
			le.changeToFollower()
			return
		}
	}
}

// tick
func (le *leaderElectionStateMachine) tick() {
	if le.checkReleaseTickLimit() {
		log.Infof("[Store][database] abandon leader election in this tick (%s)", le.electKey)
		return
	}
	shouldRelease := le.checkAndClearReleaseSignal()

	if le.isLeader() {
		if shouldRelease {
			log.Infof("[Store][database] release leader election (%s)", le.electKey)
			le.changeToFollower()
			le.setReleaseTickLimit()
			return
		}
		r, err := le.heartbeat()
		if err != nil {
			log.Errorf("[Store][database] leader heartbeat err (%s), change to follower state (%s)",
				err.Error(), le.electKey)
			le.changeToFollower()
			return
		}
		if !r {
			le.changeToFollower()
		}
	} else {
		dead, err := le.checkLeaderDead()
		if err != nil {
			log.Errorf("[Store][database] check leader dead err (%s), stay follower state (%s)",
				err.Error(), le.electKey)
			return
		}
		if !dead {
			return
		}
		r, err := le.elect()
		if err != nil {
			log.Errorf("[Store][database] elect leader err (%s), stay follower state (%s)", err.Error(), le.electKey)
			return
		}
		if r {
			le.changeToLeader()
		}
	}
}

func (le *leaderElectionStateMachine) publishLeaderChangeEvent() {
	eventhub.Publish(eventhub.LeaderChangeEventTopic, store.LeaderChangeEvent{Key: le.electKey, Leader: le.isLeader()})
}

// changeToLeader
func (le *leaderElectionStateMachine) changeToLeader() {
	log.Infof("[Store][database] change from follower to leader (%s)", le.electKey)
	atomic.StoreInt32(&le.leaderFlag, 1)
	le.publishLeaderChangeEvent()
}

// changeToFollower
func (le *leaderElectionStateMachine) changeToFollower() {
	log.Infof("[Store][database] change from leader to follower (%s)", le.electKey)
	atomic.StoreInt32(&le.leaderFlag, 0)
	le.publishLeaderChangeEvent()
}

// checkLeaderDead
func (le *leaderElectionStateMachine) checkLeaderDead() (bool, error) {
	return le.leStore.CheckMtimeExpired(le.electKey, LeaseTime)
}

// elect
func (le *leaderElectionStateMachine) elect() (bool, error) {
	curVersion, err := le.leStore.GetVersion(le.electKey)
	if err != nil {
		return false, err
	}
	le.version = curVersion + 1
	return le.leStore.CompareAndSwapVersion(le.electKey, curVersion, le.version, utils.LocalHost)
}

// heartbeat
func (le *leaderElectionStateMachine) heartbeat() (bool, error) {
	curVersion := le.version
	le.version = curVersion + 1
	return le.leStore.CompareAndSwapVersion(le.electKey, curVersion, le.version, utils.LocalHost)
}

// isLeader
func (le *leaderElectionStateMachine) isLeader() bool {
	return isLeader(le.leaderFlag)
}

// isLeaderAtomic
func (le *leaderElectionStateMachine) isLeaderAtomic() bool {
	return isLeader(atomic.LoadInt32(&le.leaderFlag))
}

func (le *leaderElectionStateMachine) setReleaseSignal() {
	atomic.StoreInt32(&le.releaseSignal, 1)
}

func (le *leaderElectionStateMachine) checkAndClearReleaseSignal() bool {
	return atomic.CompareAndSwapInt32(&le.releaseSignal, 1, 0)
}

func (le *leaderElectionStateMachine) checkReleaseTickLimit() bool {
	if le.releaseTickLimit > 0 {
		le.releaseTickLimit = le.releaseTickLimit - 1
		return true
	}
	return false
}

func (le *leaderElectionStateMachine) setReleaseTickLimit() {
	le.releaseTickLimit = LeaseTime / TickTime * 3
}

// StartLeaderElection
func (m *maintainStore) StartLeaderElection(key string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	_, ok := m.leMap[key]
	if ok {
		return nil
	}

	ctx, cancel := context.WithCancel(context.TODO())
	le := &leaderElectionStateMachine{
		electKey:         key,
		leStore:          m.leStore,
		leaderFlag:       0,
		version:          0,
		ctx:              ctx,
		cancel:           cancel,
		releaseSignal:    0,
		releaseTickLimit: 0,
	}
	err := le.leStore.CreateLeaderElection(key)
	if err != nil {
		return store.Error(err)
	}

	m.leMap[key] = le
	go le.mainLoop()
	return nil
}

// StopLeaderElections
func (m *maintainStore) StopLeaderElections() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	for k, le := range m.leMap {
		le.cancel()
		delete(m.leMap, k)
	}
}

// IsLeader
func (maintain *maintainStore) IsLeader(key string) bool {
	maintain.mutex.Lock()
	defer maintain.mutex.Unlock()
	le, ok := maintain.leMap[key]
	if !ok {
		return false
	}
	return le.isLeaderAtomic()
}

// ListLeaderElections
func (maintain *maintainStore) ListLeaderElections() ([]*model.LeaderElection, error) {
	return maintain.leStore.ListLeaderElections()
}

// ReleaseLeaderElection
func (maintain *maintainStore) ReleaseLeaderElection(key string) error {
	maintain.mutex.Lock()
	defer maintain.mutex.Unlock()
	le, ok := maintain.leMap[key]
	if !ok {
		return fmt.Errorf("LeaderElection(%s) not started", key)
	}

	le.setReleaseSignal()
	return nil
}

// BatchCleanDeletedInstances batch clean soft deleted instances
func (maintain *maintainStore) BatchCleanDeletedInstances(batchSize uint32) (uint32, error) {
	log.Infof("[Store][database] batch clean soft deleted instances(%d)", batchSize)
	mainStr := "delete from instance where flag = 1 limit ?"
	result, err := maintain.master.Exec(mainStr, batchSize)
	if err != nil {
		log.Errorf("[Store][database] batch clean soft deleted instances(%d), err: %s", batchSize, err.Error())
		return 0, store.Error(err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		log.Warnf("[Store][database] batch clean soft deleted instances(%d), get RowsAffected err: %s",
			batchSize, err.Error())
		return 0, store.Error(err)
	}

	return uint32(rows), nil
}
