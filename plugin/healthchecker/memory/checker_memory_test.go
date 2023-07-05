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

package heartbeatmemory

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	commontime "github.com/polarismesh/polaris/common/time"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/plugin"
)

func TestMemoryHealthChecker_Query(t *testing.T) {
	mhc := MemoryHealthChecker{
		hbRecords: utils.NewSyncMap[string, *HeartbeatRecord](),
	}
	id := "key1"
	reportRequest := &plugin.ReportRequest{
		QueryRequest: plugin.QueryRequest{
			InstanceId: id,
		},
		LocalHost:  "127.0.0.1",
		CurTimeSec: 1,
		Count:      5,
	}
	err := mhc.Report(context.Background(), reportRequest)
	assert.Nil(t, err)

	queryRequest := plugin.QueryRequest{
		InstanceId: id,
	}
	qr, err := mhc.Query(context.Background(), &queryRequest)
	assert.Nil(t, err)
	assert.Equal(t, reportRequest.LocalHost, qr.Server)
	assert.Equal(t, reportRequest.Count, qr.Count)
	assert.Equal(t, reportRequest.CurTimeSec, qr.LastHeartbeatSec)
}

func TestMemoryHealthChecker_Check(t *testing.T) {
	mhc := MemoryHealthChecker{
		hbRecords: utils.NewSyncMap[string, *HeartbeatRecord](),
	}
	test := &HeartbeatRecord{
		Server:     "127.0.0.1",
		CurTimeSec: 1,
	}
	mhc.hbRecords.Store("key", test)

	queryRequest := plugin.CheckRequest{
		QueryRequest: plugin.QueryRequest{
			InstanceId: "key",
			Host:       "127.0.0.2",
			Port:       80,
			Healthy:    true,
		},
		CurTimeSec: func() int64 {
			return time.Now().Unix()
		},
		ExpireDurationSec: 15,
	}
	qr, err := mhc.Check(&queryRequest)
	assert.NoError(t, err)
	assert.False(t, qr.StayUnchanged)

	queryRequest = plugin.CheckRequest{
		QueryRequest: plugin.QueryRequest{
			InstanceId: "key",
			Host:       "127.0.0.2",
			Port:       80,
			Healthy:    false,
		},
		CurTimeSec: func() int64 {
			return time.Now().Unix()
		},
		ExpireDurationSec: 15,
	}
	qr, err = mhc.Check(&queryRequest)
	assert.NoError(t, err)
	assert.True(t, qr.StayUnchanged)

	test = &HeartbeatRecord{
		Server:     "127.0.0.1",
		CurTimeSec: time.Now().Unix(),
	}
	mhc.hbRecords.Store("key", test)

	queryRequest = plugin.CheckRequest{
		QueryRequest: plugin.QueryRequest{
			InstanceId: "key",
			Host:       "127.0.0.2",
			Port:       80,
			Healthy:    false,
		},
		CurTimeSec: func() int64 {
			return time.Now().Unix()
		},
		ExpireDurationSec: 15,
	}
	qr, err = mhc.Check(&queryRequest)
	assert.NoError(t, err)
	assert.False(t, qr.StayUnchanged)
}

func TestReportAndCheck(t *testing.T) {
	checker := MemoryHealthChecker{
		hbRecords: utils.NewSyncMap[string, *HeartbeatRecord](),
	}
	startTime := commontime.CurrentMillisecond() / 1000
	instanceId := "testId"
	host := "localhost"
	var count int64
	var port uint32 = 8888
	reportReq := &plugin.ReportRequest{
		QueryRequest: plugin.QueryRequest{
			InstanceId: instanceId,
			Host:       host,
			Port:       port,
		},
		LocalHost:  "127.0.0.1",
		CurTimeSec: startTime,
		Count:      atomic.AddInt64(&count, 1),
	}
	err := checker.Report(context.Background(), reportReq)
	assert.Nil(t, err)

	queryResp, err := checker.Query(context.Background(), &reportReq.QueryRequest)
	assert.Nil(t, err)
	assert.Equal(t, reportReq.CurTimeSec, queryResp.LastHeartbeatSec)

	// after 3 seconds
	curTimeSec := startTime + 3
	checkReq := &plugin.CheckRequest{
		QueryRequest: plugin.QueryRequest{
			InstanceId: instanceId,
			Host:       host,
			Port:       port,
			Healthy:    true,
		},
		ExpireDurationSec: 2,
		CurTimeSec: func() int64 {
			return curTimeSec
		},
	}
	resp, err := checker.Check(checkReq)
	assert.Nil(t, err)
	assert.False(t, resp.StayUnchanged)
	assert.False(t, resp.Healthy)

	reportReq.CurTimeSec = curTimeSec
	reportReq.Count = atomic.AddInt64(&count, 1)
	err = checker.Report(context.Background(), reportReq)
	assert.Nil(t, err)

	time.Sleep(3 * time.Second)
	checker.Suspend()
	startTime = commontime.CurrentMillisecond() / 1000
	reportReq.CurTimeSec = startTime
	reportReq.Count = atomic.AddInt64(&count, 1)
	err = checker.Report(context.Background(), reportReq)
	assert.Nil(t, err)

	checkReq = &plugin.CheckRequest{
		QueryRequest: plugin.QueryRequest{
			InstanceId: instanceId,
			Host:       host,
			Port:       port,
			Healthy:    true,
		},
		ExpireDurationSec: 2,
		CurTimeSec: func() int64 {
			return startTime
		},
	}
	resp, err = checker.Check(checkReq)
	assert.Nil(t, err)
	assert.True(t, resp.StayUnchanged)

	// after 4 seconds
	time.Sleep(4 * time.Second)
	checkReq = &plugin.CheckRequest{
		QueryRequest: plugin.QueryRequest{
			InstanceId: instanceId,
			Host:       host,
			Port:       port,
			Healthy:    true,
		},
		ExpireDurationSec: 2,
		CurTimeSec: func() int64 {
			return commontime.CurrentMillisecond() / 1000
		},
	}
	resp, err = checker.Check(checkReq)
	assert.Nil(t, err)
	assert.False(t, resp.StayUnchanged)
	assert.False(t, resp.Healthy)
}
