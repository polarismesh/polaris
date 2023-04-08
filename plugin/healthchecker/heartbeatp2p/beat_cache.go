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

package heartbeatp2p

import (
	"strconv"
	"strings"
	"sync"

	"github.com/polarismesh/polaris/common/utils"
)

// ReadBeatRecord
type ReadBeatRecord struct {
	Record RecordValue
	Exist  bool
}

// WriteBeatRecord
type WriteBeatRecord struct {
	Record RecordValue
	Key    string
}

// RecordValue
type RecordValue struct {
	Server     string
	CurTimeSec int64
	Count      int64
}

// ParseRecordValue
func ParseRecordValue(s string) (*RecordValue, bool) {
	infos := strings.Split(s, Split)
	if len(infos) < 3 {
		return nil, false
	}

	sec, err := strconv.ParseInt(infos[1], 10, 64)
	if err != nil {
		return nil, false
	}
	count, err := strconv.ParseInt(infos[2], 10, 64)
	if err != nil {
		return nil, false
	}

	return &RecordValue{
		Server:     infos[0],
		CurTimeSec: sec,
		Count:      count,
	}, true
}

func (r RecordValue) String() string {
	secStr := strconv.FormatInt(r.CurTimeSec, 10)
	countStr := strconv.FormatInt(r.Count, 10)
	return r.Server + Split + secStr + Split + countStr
}

type (
	// HashFunction
	HashFunction func(string) int
	// RecordSaver
	RecordSaver func(req *PutRecordsRequest)
	// RecordDelter
	RecordDelter func(req *DelRecordsRequest)
	// RecordGetter
	RecordGetter func(req *GetRecordsRequest) *GetRecordsResponse
	// BeatRecordCache
	BeatRecordCache interface {
		// Get
		Get(key ...string) map[string]*ReadBeatRecord
		// Put
		Put(records ...WriteBeatRecord)
		// Del
		Del(key ...string)
	}
)

func newLocalBeatRecordCache(soltNum int32, hashFunc HashFunction) BeatRecordCache {
	soltLocks := make([]*sync.RWMutex, 0, soltNum)
	solts := make([]map[string]RecordValue, 0, soltNum)
	for i := 0; i < int(soltNum); i++ {
		soltLocks = append(soltLocks, &sync.RWMutex{})
		solts = append(solts, map[string]RecordValue{})
	}

	return &LocalBeatRecordCache{
		beatCache: utils.NewSegmentMap[string, RecordValue](int(soltNum), func(k string) int {
			return hashFunc(k)
		}),
	}
}

// LocalBeatRecordCache
type LocalBeatRecordCache struct {
	beatCache *utils.SegmentMap[string, RecordValue]
}

func (lc *LocalBeatRecordCache) Get(keys ...string) map[string]*ReadBeatRecord {
	ret := make(map[string]*ReadBeatRecord, len(keys))
	for i := range keys {
		key := keys[i]
		val, ok := lc.beatCache.Get(key)
		ret[key] = &ReadBeatRecord{
			Record: val,
			Exist:  ok,
		}
	}
	return ret
}

func (lc *LocalBeatRecordCache) Put(records ...WriteBeatRecord) {
	for i := range records {
		record := records[i]
		lc.beatCache.Put(record.Key, record.Record)
	}
}

func (lc *LocalBeatRecordCache) Del(keys ...string) {
	for i := range keys {
		key := keys[i]
		lc.beatCache.Del(key)
	}
}

func newRemoteBeatRecordCache(getter RecordGetter, saver RecordSaver,
	delter RecordDelter) BeatRecordCache {
	return &RemoteBeatRecordCache{
		getter: getter,
		saver:  saver,
		delter: delter,
	}
}

// RemoteBeatRecordCache
type RemoteBeatRecordCache struct {
	saver  RecordSaver
	delter RecordDelter
	getter RecordGetter
}

func (rc *RemoteBeatRecordCache) Get(keys ...string) map[string]*ReadBeatRecord {
	req := &GetRecordsRequest{
		Keys: keys,
	}
	resp := rc.getter(req)
	ret := make(map[string]*ReadBeatRecord)
	for i := range keys {
		ret[keys[i]] = &ReadBeatRecord{
			Exist: false,
		}
	}
	records := resp.GetRecords()
	for i := range records {
		record := records[i]
		val, ok := ret[record.Key]
		if !ok {
			continue
		}
		val.Exist = true
		recordVal, ok := ParseRecordValue(record.Value)
		if ok {
			val.Record = *recordVal
		}
	}
	return ret
}

func (rc *RemoteBeatRecordCache) Put(records ...WriteBeatRecord) {
	req := &PutRecordsRequest{
		Records: make([]*HeartbeatRecord, 0, len(records)),
	}
	for i := range records {
		record := records[i]
		req.Records = append(req.Records, &HeartbeatRecord{
			Key:   record.Key,
			Value: req.String(),
		})
	}
	rc.saver(req)
}

func (rc *RemoteBeatRecordCache) Del(key ...string) {
	req := &DelRecordsRequest{
		Keys: key,
	}
	rc.delter(req)
}
