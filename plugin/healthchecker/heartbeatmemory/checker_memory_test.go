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
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/polarismesh/polaris/plugin"
)

func TestMemoryHealthChecker_Query(t *testing.T) {
	mhc := MemoryHealthChecker{
		hbRecords: new(sync.Map),
	}
	test := HeartbeatRecord{
		Server:     "127.0.0.1",
		CurTimeSec: 1,
	}
	mhc.hbRecords.Store("key", test)

	queryRequest := plugin.QueryRequest{
		InstanceId: "key",
		Host:       "127.0.0.2",
		Port:       80,
		Healthy:    true,
	}
	qr, err := mhc.Query(&queryRequest)
	if err != nil {
		t.Error(err)
	}
	if qr.Server != "127.0.0.1" {
		t.Error()
	}
	if qr.LastHeartbeatSec != 1 {
		t.Error()
	}
}

func TestMemoryHealthChecker_Check(t *testing.T) {
	mhc := MemoryHealthChecker{
		hbRecords: new(sync.Map),
	}
	test := HeartbeatRecord{
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

	test = HeartbeatRecord{
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
