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

package maintain

import (
	"context"

	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/connlimit"
)

type ConnReq struct {
	Protocol string
	Host     string
	Port     int
	Amount   int
}

type ConnCountResp struct {
	Protocol string
	Total    int32
	Host     map[string]int32
}

type ConnStatsResp struct {
	Protocol        string
	ActiveConnTotal int32
	StatsTotal      int
	StatsSize       int
	Stats           []*connlimit.HostConnStat
}

// MaintainOperateServer Maintain related operation
type MaintainOperateServer interface {

	// GetServerConnections Get connection count
	GetServerConnections(ctx context.Context, req *ConnReq) (*ConnCountResp, error)

	// GetServerConnStats 获取连接缓存里面的统计信息
	GetServerConnStats(ctx context.Context, req *ConnReq) (*ConnStatsResp, error)

	// CloseConnections Close connection by ip
	CloseConnections(ctx context.Context, reqs []ConnReq) error

	// FreeOSMemory Free system memory
	FreeOSMemory(ctx context.Context) error

	// CleanInstance Clean deleted instance
	CleanInstance(ctx context.Context, req *api.Instance) *api.Response

	// GetLastHeartbeat Get last heartbeat
	GetLastHeartbeat(ctx context.Context, req *api.Instance) *api.Response

	// GetLogOutputLevel Get log output level
	GetLogOutputLevel(ctx context.Context) (map[string]string, error)

	// SetLogOutputLevel Set log output level by scope
	SetLogOutputLevel(ctx context.Context, scope string, level string) error
}
