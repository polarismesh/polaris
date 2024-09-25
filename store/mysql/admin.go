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
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/polarismesh/polaris/common/eventhub"
	"github.com/polarismesh/polaris/common/model/admin"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/store"
)

const (
	TickTime  = 2
	LeaseTime = 10
)

// adminStore implement adminStore interface
type adminStore struct {
	master  *BaseDB
	leStore LeaderElectionStore
	leMap   map[string]*leaderElectionStateMachine
	mutex   sync.Mutex
}

func newAdminStore(master *BaseDB) *adminStore {
	return &adminStore{
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
	CheckMtimeExpired(key string, leaseTime int32) (string, bool, error)
	// ListLeaderElections list all leaderelection
	ListLeaderElections() ([]*admin.LeaderElection, error)
}

// leaderElectionStore
type leaderElectionStore struct {
	master *BaseDB
}

// CreateLeaderElection insert election key into leader table
func (l *leaderElectionStore) CreateLeaderElection(key string) error {
	log.Debugf("[Store][database] create leader election (%s)", key)
	return l.master.processWithTransaction("createLeaderElection", func(tx *BaseTx) error {
		mainStr := "insert ignore into leader_election (elect_key, leader) values (?, ?)"
		if _, err := tx.Exec(mainStr, key, ""); err != nil {
			log.Errorf("[Store][database] create leader election (%s), err: %s", key, err.Error())
		}

		if err := tx.Commit(); err != nil {
			log.Errorf("[Store][database] create leader election (%s) commit tx err: %s", key, err.Error())
			return err
		}
		return nil
	})
}

// GetVersion get the version from election
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

// CompareAndSwapVersion compare key version and update
func (l *leaderElectionStore) CompareAndSwapVersion(key string, curVersion int64, newVersion int64,
	leader string) (bool, error) {
	var rows int64
	err := l.master.processWithTransaction("compareAndSwapVersion", func(tx *BaseTx) error {
		log.Debugf("[Store][database] compare and swap version (%s, %d, %d, %s)", key, curVersion, newVersion, leader)
		mainStr := "update leader_election set leader = ?, version = ? where elect_key = ? and version = ?"
		result, err := tx.Exec(mainStr, leader, newVersion, key, curVersion)
		if err != nil {
			log.Errorf("[Store][database] compare and swap version (%s), err: %s", key, err.Error())
			return store.Error(err)
		}
		tRows, err := result.RowsAffected()
		if err != nil {
			log.Errorf("[Store][database] compare and swap version (%s), get RowsAffected err: %s", key, err.Error())
			return store.Error(err)
		}

		if err := tx.Commit(); err != nil {
			log.Errorf("[Store][database] create leader election (%s) commit tx err: %s", key, err.Error())
			return err
		}

		rows = tRows
		return nil
	})
	return rows > 0, err
}

// CheckMtimeExpired check last modify time expired
func (l *leaderElectionStore) CheckMtimeExpired(key string, leaseTime int32) (string, bool, error) {
	log.Debugf("[Store][database] check mtime expired (%s, %d)", key, leaseTime)
	mainStr := "select leader, FROM_UNIXTIME(UNIX_TIMESTAMP(SYSDATE())) - mtime from leader_election where elect_key = ?"

	var (
		leader   string
		diffTime int32
	)
	err := l.master.DB.QueryRow(mainStr, key).Scan(&leader, &diffTime)
	if err != nil {
		log.Errorf("[Store][database] check mtime expired (%s), err: %s", key, err.Error())
	}
	return leader, (diffTime > leaseTime), store.Error(err)
}

// ListLeaderElections list the election records
func (l *leaderElectionStore) ListLeaderElections() ([]*admin.LeaderElection, error) {
	log.Info("[Store][database] list leader election")
	mainStr := "select elect_key, leader, UNIX_TIMESTAMP(ctime), UNIX_TIMESTAMP(mtime) from leader_election"

	rows, err := l.master.Query(mainStr)
	if err != nil {
		log.Errorf("[Store][database] list leader election query err: %s", err.Error())
		return nil, store.Error(err)
	}

	return fetchLeaderElectionRows(rows)
}

func fetchLeaderElectionRows(rows *sql.Rows) ([]*admin.LeaderElection, error) {
	if rows == nil {
		return nil, nil
	}
	defer rows.Close()

	var out []*admin.LeaderElection

	for rows.Next() {
		space := &admin.LeaderElection{}
		if err := rows.Scan(&space.ElectKey, &space.Host, &space.Ctime, &space.Mtime); err != nil {
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
	leader           string
}

// isLeader
func isLeader(flag int32) bool {
	return flag > 0
}

// mainLoop
func (le *leaderElectionStateMachine) mainLoop() {
	le.changeToFollower("")
	log.Infof("[Store][database] leader election started (%s)", le.electKey)
	ticker := time.NewTicker(TickTime * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			le.tick()
		case <-le.ctx.Done():
			log.Infof("[Store][database] leader election stopped (%s)", le.electKey)
			le.changeToFollower("")
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
			le.changeToFollower("")
			le.setReleaseTickLimit()
			return
		}
		success, err := le.heartbeat()
		if err == nil && success {
			return
		}
		if err != nil {
			log.Errorf("[Store][database] leader heartbeat err (%v), change to follower state (%s)", err, le.electKey)
		}
		if !success && err == nil {
			log.Infof("[Store][database] leader heartbeat abort, change to follower state (%s)", le.electKey)
		}
	}
	leader, dead, err := le.checkLeaderDead()
	if err != nil {
		log.Errorf("[Store][database] check leader dead err (%s), stay follower state (%s)",
			err.Error(), le.electKey)
		return
	}
	if !dead {
		// 自己之前是 leader，并且租期还没过，调整自己为 leader
		if leader == utils.LocalHost {
			le.changeToLeader()
		}
		// leader 信息出现变化，发布leader信息变化通知
		if le.leader != leader {
			le.changeToFollower(leader)
		}
		return
	}
	success, err := le.elect()
	if err != nil {
		log.Errorf("[Store][database] elect leader err (%s), stay follower state (%s)", err.Error(), le.electKey)
		return
	}
	if success {
		le.changeToLeader()
	}
}

func (le *leaderElectionStateMachine) publishLeaderChangeEvent() {
	_ = eventhub.Publish(eventhub.LeaderChangeEventTopic, store.LeaderChangeEvent{
		Key:        le.electKey,
		Leader:     le.isLeader(),
		LeaderHost: le.leader,
	})
}

// changeToLeader
func (le *leaderElectionStateMachine) changeToLeader() {
	log.Infof("[Store][database] change from follower to leader (%s)", le.electKey)
	atomic.StoreInt32(&le.leaderFlag, 1)
	le.leader = utils.LocalHost
	le.publishLeaderChangeEvent()
}

// changeToFollower
func (le *leaderElectionStateMachine) changeToFollower(leader string) {
	log.Infof("[Store][database] change from leader(%s) to follower (%s)", leader, le.electKey)
	atomic.StoreInt32(&le.leaderFlag, 0)
	le.leader = leader
	le.publishLeaderChangeEvent()
}

// checkLeaderDead
func (le *leaderElectionStateMachine) checkLeaderDead() (string, bool, error) {
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

// StartLeaderElection start the election procedure
func (m *adminStore) StartLeaderElection(key string) error {
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

// StopLeaderElections stop the election procedure
func (m *adminStore) StopLeaderElections() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	for k, le := range m.leMap {
		le.cancel()
		delete(m.leMap, k)
	}
}

// IsLeader check leader
func (m *adminStore) IsLeader(key string) bool {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	le, ok := m.leMap[key]
	if !ok {
		return false
	}
	return le.isLeaderAtomic()
}

// ListLeaderElections list election records
func (m *adminStore) ListLeaderElections() ([]*admin.LeaderElection, error) {
	return m.leStore.ListLeaderElections()
}

// ReleaseLeaderElection release election lock
func (m *adminStore) ReleaseLeaderElection(key string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	le, ok := m.leMap[key]
	if !ok {
		return fmt.Errorf("LeaderElection(%s) not started", key)
	}

	le.setReleaseSignal()
	return nil
}

// BatchCleanDeletedInstances batch clean soft deleted instances
func (m *adminStore) BatchCleanDeletedInstances(timeout time.Duration, batchSize uint32) (uint32, error) {
	log.Infof("[Store][database] batch clean soft deleted instances(%d)", batchSize)
	var rowsAffected int64
	err := m.master.processWithTransaction("batchCleanDeletedInstances", func(tx *BaseTx) error {
		// 查询出需要清理的实例 ID 信息
		loadWaitDel := "SELECT id FROM instance WHERE flag = 1 AND " +
			"mtime <= FROM_UNIXTIME(UNIX_TIMESTAMP(SYSDATE()) - ?) LIMIT ?"
		rows, err := tx.Query(loadWaitDel, int32(timeout.Seconds()), batchSize)
		if err != nil {
			log.Errorf("[Store][database] batch clean soft deleted instances(%d), err: %s", batchSize, err.Error())
			return store.Error(err)
		}
		waitDelIds := make([]interface{}, 0, batchSize)
		defer func() {
			_ = rows.Close()
		}()

		placeholders := make([]string, 0, batchSize)
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				log.Errorf("[Store][database] scan deleted instances id, err: %s", err.Error())
				return store.Error(err)
			}
			waitDelIds = append(waitDelIds, id)
			placeholders = append(placeholders, "?")
		}

		if len(waitDelIds) == 0 {
			return nil
		}
		inSql := strings.Join(placeholders, ",")

		cleanMetaStr := fmt.Sprintf("delete from instance_metadata where id in (%s)", inSql)
		if _, err := tx.Exec(cleanMetaStr, waitDelIds...); err != nil {
			log.Errorf("[Store][database] batch clean soft deleted instances(%d), err: %s", batchSize, err.Error())
			return store.Error(err)
		}

		cleanCheckStr := fmt.Sprintf("delete from health_check where id in (%s)", inSql)
		if _, err := tx.Exec(cleanCheckStr, waitDelIds...); err != nil {
			log.Errorf("[Store][database] batch clean soft deleted instances(%d), err: %s", batchSize, err.Error())
			return store.Error(err)
		}

		cleanInsStr := fmt.Sprintf("delete from instance where flag = 1 and id in (%s)", inSql)
		result, err := tx.Exec(cleanInsStr, waitDelIds...)
		if err != nil {
			log.Errorf("[Store][database] batch clean soft deleted instances(%d), err: %s", batchSize, err.Error())
			return store.Error(err)
		}

		tRows, err := result.RowsAffected()
		if err != nil {
			log.Warnf("[Store][database] batch clean soft deleted instances(%d), get RowsAffected err: %s",
				batchSize, err.Error())
			return store.Error(err)
		}

		if err := tx.Commit(); err != nil {
			log.Errorf("[Store][database] batch clean soft deleted instances(%d) commit tx err: %s",
				batchSize, err.Error())
			return err
		}

		rowsAffected = tRows
		return nil
	})
	return uint32(rowsAffected), err
}

func (m *adminStore) GetUnHealthyInstances(timeout time.Duration, limit uint32) ([]string, error) {
	log.Infof("[Store][database] get unhealthy instances which mtime timeout %s (%d)", timeout, limit)
	queryStr := "select id from instance where flag=0 and enable_health_check=1 and health_status=0 " +
		"and mtime < FROM_UNIXTIME(UNIX_TIMESTAMP(SYSDATE()) - ?) limit ?"
	rows, err := m.master.Query(queryStr, int32(timeout.Seconds()), limit)
	if err != nil {
		log.Errorf("[Store][database] get unhealthy instances, err: %s", err.Error())
		return nil, store.Error(err)
	}

	var instanceIds []string
	defer rows.Close()
	for rows.Next() {
		var id string
		err := rows.Scan(&id)
		if err != nil {
			log.Errorf("[Store][database] fetch unhealthy instance rows, err: %s", err.Error())
			return nil, store.Error(err)
		}
		instanceIds = append(instanceIds, id)
	}
	if err := rows.Err(); err != nil {
		log.Errorf("[Store][database] fetch unhealthy instance rows next, err: %s", err.Error())
		return nil, store.Error(err)
	}

	return instanceIds, nil
}

// BatchCleanDeletedClients batch clean soft deleted clients
func (m *adminStore) BatchCleanDeletedClients(timeout time.Duration, batchSize uint32) (uint32, error) {
	log.Infof("[Store][database] batch clean soft deleted clients(%d)", batchSize)
	var rows int64
	err := m.master.processWithTransaction("batchCleanDeletedClients", func(tx *BaseTx) error {
		mainStr := "delete from client where flag = 1 limit ?"
		result, err := tx.Exec(mainStr, int32(timeout.Seconds()), batchSize)
		if err != nil {
			log.Errorf("[Store][database] batch clean soft deleted clients(%d), err: %s", batchSize, err.Error())
			return store.Error(err)
		}

		tRows, err := result.RowsAffected()
		if err != nil {
			log.Warnf("[Store][database] batch clean soft deleted clients(%d), get RowsAffected err: %s",
				batchSize, err.Error())
			return store.Error(err)
		}

		if err := tx.Commit(); err != nil {
			log.Errorf("[Store][database] batch clean soft deleted clients(%d) commit tx err: %s",
				batchSize, err.Error())
			return err
		}

		rows = tRows
		return nil
	})
	return uint32(rows), err
}

// BatchCleanDeletedServices batch clean soft deleted clients
func (m *adminStore) BatchCleanDeletedServices(timeout time.Duration, batchSize uint32) (uint32, error) {
	return 0, nil
}

// BatchCleanDeletedRules batch clean soft deleted clients
func (m *adminStore) BatchCleanDeletedRules(rule string, timeout time.Duration, batchSize uint32) (uint32, error) {
	return 0, nil
}

// BatchCleanDeletedConfigFiles batch clean soft deleted clients
func (m *adminStore) BatchCleanDeletedConfigFiles(timeout time.Duration, batchSize uint32) (uint32, error) {
	return 0, nil
}
