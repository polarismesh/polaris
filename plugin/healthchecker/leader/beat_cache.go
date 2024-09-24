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
	"strconv"
	"sync"

	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"
	"go.uber.org/zap"

	"github.com/polarismesh/polaris/common/utils"
)

// ReadBeatRecord Heartbeat records read results
type ReadBeatRecord struct {
	Record RecordValue
	Exist  bool
}

// WriteBeatRecord Heartbeat record operation results
type WriteBeatRecord struct {
	Record RecordValue
	Key    string
}

// RecordValue heatrtbeat record value
type RecordValue struct {
	Server     string
	CurTimeSec int64
	Count      int64
}

func (r RecordValue) String() string {
	secStr := strconv.FormatInt(r.CurTimeSec, 10)
	countStr := strconv.FormatInt(r.Count, 10)
	return r.Server + Split + secStr + Split + countStr
}

type (
	// HashFunction hash function to caul record id need locate in SegmentMap
	HashFunction func(string) int
	// RecordSaver beat record saver
	RecordSaver func(req *apiservice.HeartbeatsRequest) error
	// RecordDelter beat record delter
	RecordDelter func(req *apiservice.DelHeartbeatsRequest) error
	// RecordGetter beat record getter
	RecordGetter func(req *apiservice.GetHeartbeatsRequest) (*apiservice.GetHeartbeatsResponse, error)
	// BeatRecordCache Heartbeat data cache
	BeatRecordCache interface {
		// Get get records
		Get(keys ...string) (map[string]*ReadBeatRecord, error)
		// Put put records
		Put(records ...WriteBeatRecord) error
		// Del del records
		Del(keys ...string) error
		// Clean .
		Clean()
		// Snapshot
		Snapshot() map[string]*ReadBeatRecord
		// Ping
		Ping() error
	}
)

// newLocalBeatRecordCache
func newLocalBeatRecordCache(soltNum int32, hashFunc HashFunction) BeatRecordCache {
	if soltNum == 0 {
		soltNum = DefaultSoltNum
	}
	return &LocalBeatRecordCache{
		soltNum:  soltNum,
		hashFunc: hashFunc,
		beatCache: utils.NewSegmentMap[string, RecordValue](int(soltNum), func(k string) int {
			return hashFunc(k)
		}),
	}
}

// LocalBeatRecordCache
type LocalBeatRecordCache struct {
	lock      sync.RWMutex
	soltNum   int32
	hashFunc  HashFunction
	beatCache *utils.SegmentMap[string, RecordValue]
}

func (lc *LocalBeatRecordCache) Ping() error {
	return nil
}

func (lc *LocalBeatRecordCache) Get(keys ...string) (map[string]*ReadBeatRecord, error) {
	lc.lock.RLock()
	defer lc.lock.RUnlock()
	ret := make(map[string]*ReadBeatRecord, len(keys))
	for i := range keys {
		key := keys[i]
		val, ok := lc.beatCache.Get(key)
		ret[key] = &ReadBeatRecord{
			Record: val,
			Exist:  ok,
		}
	}
	return ret, nil
}

func (lc *LocalBeatRecordCache) Put(records ...WriteBeatRecord) error {
	lc.lock.RLock()
	defer lc.lock.RUnlock()
	for i := range records {
		record := records[i]
		if log.DebugEnabled() {
			plog.Debug("receive put action", zap.Any("record", record))
		}
		lc.beatCache.Put(record.Key, record.Record)
	}
	return nil
}

func (lc *LocalBeatRecordCache) Del(keys ...string) error {
	lc.lock.RLock()
	defer lc.lock.RUnlock()
	for i := range keys {
		key := keys[i]
		ok := lc.beatCache.Del(key)
		if log.DebugEnabled() {
			plog.Debug("delete result", zap.String("key", key), zap.Bool("exist", ok))
		}
	}
	return nil
}

func (lc *LocalBeatRecordCache) Clean() {
	lc.lock.Lock()
	defer lc.lock.Unlock()
	lc.beatCache = utils.NewSegmentMap[string, RecordValue](int(lc.soltNum), func(k string) int {
		return lc.hashFunc(k)
	})
}

func (lc *LocalBeatRecordCache) Snapshot() map[string]*ReadBeatRecord {
	lc.lock.RLock()
	defer lc.lock.RUnlock()
	ret := map[string]*ReadBeatRecord{}
	lc.beatCache.Range(func(k string, v RecordValue) {
		ret[k] = &ReadBeatRecord{
			Record: v,
		}
	})
	return ret
}

// newRemoteBeatRecordCache
func newRemoteBeatRecordCache(getter RecordGetter, saver RecordSaver,
	delter RecordDelter, ping func() error) BeatRecordCache {
	return &RemoteBeatRecordCache{
		getter: getter,
		saver:  saver,
		delter: delter,
		ping:   ping,
	}
}

// RemoteBeatRecordCache
type RemoteBeatRecordCache struct {
	saver  RecordSaver
	delter RecordDelter
	getter RecordGetter
	ping   func() error
}

func (rc *RemoteBeatRecordCache) Ping() error {
	return rc.ping()
}

func (rc *RemoteBeatRecordCache) Get(keys ...string) (map[string]*ReadBeatRecord, error) {
	ret := make(map[string]*ReadBeatRecord)
	for i := range keys {
		ret[keys[i]] = &ReadBeatRecord{
			Exist: false,
		}
	}
	resp, err := rc.getter(&apiservice.GetHeartbeatsRequest{
		InstanceIds: keys,
	})
	if err != nil {
		return nil, err
	}
	records := resp.GetRecords()
	for i := range records {
		record := records[i]
		val, ok := ret[record.InstanceId]
		if !ok {
			val.Exist = false
			continue
		}
		val.Exist = record.GetExist()
		val.Record = RecordValue{
			CurTimeSec: record.GetLastHeartbeatSec(),
		}
	}
	return ret, nil
}

func (rc *RemoteBeatRecordCache) Put(records ...WriteBeatRecord) error {
	req := &apiservice.HeartbeatsRequest{
		Heartbeats: make([]*apiservice.InstanceHeartbeat, 0, len(records)),
	}
	for i := range records {
		record := records[i]
		req.Heartbeats = append(req.Heartbeats, &apiservice.InstanceHeartbeat{
			InstanceId: record.Key,
		})
	}
	return rc.saver(req)
}

func (rc *RemoteBeatRecordCache) Del(key ...string) error {
	req := &apiservice.DelHeartbeatsRequest{
		InstanceIds: key,
	}
	return rc.delter(req)
}

func (lc *RemoteBeatRecordCache) Clean() {
	// do nothing
}

func (lc *RemoteBeatRecordCache) Snapshot() map[string]*ReadBeatRecord {
	return map[string]*ReadBeatRecord{}
}
