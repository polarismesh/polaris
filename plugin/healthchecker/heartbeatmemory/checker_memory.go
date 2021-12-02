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
	"github.com/polarismesh/polaris-server/plugin"
	"sync"
)

// 把操作记录记录到日志文件中
const (
	// PluginName plugin name
	PluginName = "heartbeatMemory"
)

// HeartbeatRecord record for heartbeat
type HeartbeatRecord struct {
	Server     string
	CurTimeSec int64
}

// MemoryHealthChecker
type MemoryHealthChecker struct {
	hbRecords *sync.Map
}

// Name
func (r *MemoryHealthChecker) Name() string {
	return PluginName
}

// Initialize
func (r *MemoryHealthChecker) Initialize(c *plugin.ConfigEntry) error {
	r.hbRecords = &sync.Map{}
	return nil
}

// Destroy
func (r *MemoryHealthChecker) Destroy() error {
	return nil
}

// Type type for health check plugin, only one same type plugin is allowed
func (r *MemoryHealthChecker) Type() plugin.HealthCheckType {
	return plugin.HealthCheckerHeartbeat
}

// Report process heartbeat info report
func (r *MemoryHealthChecker) Report(request *plugin.ReportRequest) error {
	record := HeartbeatRecord{
		Server:     request.LocalHost,
		CurTimeSec: request.CurTimeSec,
	}
	r.hbRecords.Store(request.InstanceId, record)
	return nil
}

// Query query the heartbeat time
func (r *MemoryHealthChecker) Query(request *plugin.QueryRequest) (*plugin.QueryResponse, error) {
	recordValue, ok := r.hbRecords.Load(request.InstanceId)
	if !ok {
		return &plugin.QueryResponse{
			LastHeartbeatSec: 0,
		}, nil
	}
	record := recordValue.(HeartbeatRecord)
	return &plugin.QueryResponse{
		Server:           record.Server,
		LastHeartbeatSec: record.CurTimeSec,
	}, nil
}

// Report process the instance check
func (r *MemoryHealthChecker) Check(request *plugin.CheckRequest) (*plugin.CheckResponse, error) {
	queryResp, err := r.Query(&request.QueryRequest)
	if nil != err {
		return nil, err
	}
	lastHeartbeatTime := queryResp.LastHeartbeatSec
	checkResp := &plugin.CheckResponse{
		LastHeartbeatTimeSec: lastHeartbeatTime,
	}
	curTimeSec := request.CurTimeSec()
	if curTimeSec > lastHeartbeatTime {
		if curTimeSec-lastHeartbeatTime >= int64(request.ExpireDurationSec) {
			//心跳超时
			checkResp.Healthy = false
			_ = r.Delete(request.InstanceId)
			return checkResp, nil
		}
	}
	checkResp.Healthy = true
	return checkResp, nil
}

// AddToCheck add the instances to check procedure
func (r *MemoryHealthChecker) AddToCheck(request *plugin.AddCheckRequest) error {
	return nil
}

// AddToCheck add the instances to check procedure
func (r *MemoryHealthChecker) RemoveFromCheck(request *plugin.AddCheckRequest) error {
	return nil
}

// Delete delete the id
func (r *MemoryHealthChecker) Delete(id string) error {
	r.hbRecords.Delete(id)
	return nil
}

func init() {
	d := &MemoryHealthChecker{}
	plugin.RegisterPlugin(d.Name(), d)
}
