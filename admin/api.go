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

package admin

import (
	"context"

	apisecurity "github.com/polarismesh/specification/source/go/api/v1/security"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"

	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/model/admin"
)

// AdminOperateServer Maintain related operation
type AdminOperateServer interface {
	// GetServerConnections Get connection count
	GetServerConnections(ctx context.Context, req *admin.ConnReq) (*admin.ConnCountResp, error)
	// GetServerConnStats 获取连接缓存里面的统计信息
	GetServerConnStats(ctx context.Context, req *admin.ConnReq) (*admin.ConnStatsResp, error)
	// CloseConnections Close connection by ip
	CloseConnections(ctx context.Context, reqs []admin.ConnReq) error
	// FreeOSMemory Free system memory
	FreeOSMemory(ctx context.Context) error
	// CleanInstance Clean deleted instance
	CleanInstance(ctx context.Context, req *apiservice.Instance) *apiservice.Response
	// BatchCleanInstances Batch clean deleted instances
	BatchCleanInstances(ctx context.Context, batchSize uint32) (uint32, error)
	// GetLastHeartbeat Get last heartbeat
	GetLastHeartbeat(ctx context.Context, req *apiservice.Instance) *apiservice.Response
	// GetLogOutputLevel Get log output level
	GetLogOutputLevel(ctx context.Context) ([]admin.ScopeLevel, error)
	// SetLogOutputLevel Set log output level by scope
	SetLogOutputLevel(ctx context.Context, scope string, level string) error
	// ListLeaderElections
	ListLeaderElections(ctx context.Context) ([]*admin.LeaderElection, error)
	// ReleaseLeaderElection
	ReleaseLeaderElection(ctx context.Context, electKey string) error
	// GetCMDBInfo get cmdb info
	GetCMDBInfo(ctx context.Context) ([]model.LocationView, error)
	// InitMainUser
	InitMainUser(ctx context.Context, user apisecurity.User) error
}
