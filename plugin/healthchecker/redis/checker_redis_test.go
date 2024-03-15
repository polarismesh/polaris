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

package heartbeatredis

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/polarismesh/polaris/common/redispool"
	commontime "github.com/polarismesh/polaris/common/time"
	"github.com/polarismesh/polaris/plugin"
)

type mockPool struct {
	setValues      map[string]map[string]bool
	itemValues     map[string]string
	compatible     bool
	recoverTimeSec int64
}

// Start 启动ckv连接池工作
func (m *mockPool) Start() {
	m.setValues = make(map[string]map[string]bool)
	m.itemValues = make(map[string]string)
}

// Sdd 使用连接池，向redis发起Sdd请求
func (m *mockPool) Sdd(id string, members []string) *redispool.Resp {
	values, ok := m.setValues[id]
	if !ok {
		values = make(map[string]bool)
		m.setValues[id] = values
	}
	for _, member := range members {
		values[member] = true
	}
	return &redispool.Resp{Compatible: m.compatible}
}

// Srem 使用连接池，向redis发起Srem请求
func (m *mockPool) Srem(id string, members []string) *redispool.Resp {
	values, ok := m.setValues[id]
	if ok {
		for _, member := range members {
			delete(values, member)
		}
	}
	return &redispool.Resp{Compatible: m.compatible}
}

// Get 使用连接池，向redis发起Get请求
func (m *mockPool) Get(id string) *redispool.Resp {
	value, ok := m.itemValues[id]
	return &redispool.Resp{
		Value:      value,
		Exists:     ok,
		Compatible: m.compatible,
	}
}

// Get 使用连接池，向redis发起Get请求
func (m *mockPool) MGet(id []string) *redispool.Resp {
	rsp := &redispool.Resp{
		Values:     make([]interface{}, 0, len(id)),
		Compatible: m.compatible,
	}
	for i := range id {
		value, ok := m.itemValues[id[i]]
		if ok {
			rsp.Values = append(rsp.Values, value)
		} else {
			rsp.Values = append(rsp.Values, nil)
		}
	}
	return rsp
}

// Set 使用连接池，向redis发起Set请求
func (m *mockPool) Set(id string, redisObj redispool.RedisObject) *redispool.Resp {
	value := redisObj.Serialize(m.compatible)
	m.itemValues[id] = value
	return &redispool.Resp{
		Value:      value,
		Exists:     true,
		Compatible: m.compatible,
	}
}

// Del 使用连接池，向redis发起Del请求
func (m *mockPool) Del(id string) *redispool.Resp {
	delete(m.itemValues, id)
	return &redispool.Resp{
		Exists:     true,
		Compatible: m.compatible,
	}
}

// RecoverTimeSec the time second record when recover
func (m *mockPool) RecoverTimeSec() int64 {
	return m.recoverTimeSec
}

func TestReportAndCheck(t *testing.T) {
	pool := &mockPool{}
	checker := &RedisHealthChecker{
		hbPool:    pool,
		checkPool: pool,
	}
	checker.hbPool.Start()
	checker.checkPool.Start()

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
