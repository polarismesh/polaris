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

	"github.com/polarismesh/polaris/common/utils"
	"go.uber.org/zap"
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

// ParseRecordValue parse string to RecordValue
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
	// HashFunction hash function to caul record id need locate in SegmentMap
	HashFunction func(string) int
	// RecordSaver beat record saver
	RecordSaver func(req *PutRecordsRequest)
	// RecordDelter beat record delter
	RecordDelter func(req *DelRecordsRequest)
	// RecordGetter beat record getter
	RecordGetter func(req *GetRecordsRequest) *GetRecordsResponse
	// BeatRecordCache Heartbeat data cache
	BeatRecordCache interface {
		// Get get records
		Get(key ...string) map[string]*ReadBeatRecord
		// Put put records
		Put(records ...WriteBeatRecord)
		// Del del records
		Del(key ...string)
	}
)

// newLocalBeatRecordCache
func newLocalBeatRecordCache(soltNum int32, hashFunc HashFunction) BeatRecordCache {
	if soltNum == 0 {
		soltNum = DefaultSoltNum
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
	log.Debug("receive get action", zap.Any("keys", keys))
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
	log.Debug("receive put action", zap.Any("records", records))
	for i := range records {
		record := records[i]
		lc.beatCache.Put(record.Key, record.Record)
	}
}

func (lc *LocalBeatRecordCache) Del(keys ...string) {
	log.Debug("receive del action", zap.Strings("keys", keys))
	for i := range keys {
		key := keys[i]
		ok := lc.beatCache.Del(key)
		if log.DebugEnabled() {
			log.Debug("delete result", zap.String("key", key), zap.Bool("exist", ok))
		}
	}
}

// newRemoteBeatRecordCache
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
			val.Exist = false
			continue
		}
		val.Exist = record.Exist
		if recordVal, ok := ParseRecordValue(record.Value); ok {
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
			Value: record.Record.String(),
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
