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

	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/model/admin"
	authcommon "github.com/polarismesh/polaris/common/model/auth"
	"github.com/polarismesh/polaris/common/utils"
)

var _ AdminOperateServer = (*serverAuthAbility)(nil)

func (s *serverAuthAbility) HasMainUser(ctx context.Context) (bool, error) {
	return false, nil
}

func (s *serverAuthAbility) InitMainUser(ctx context.Context, user apisecurity.User) error {
	return nil
}

func (svr *serverAuthAbility) GetServerConnections(ctx context.Context, req *admin.ConnReq) (*admin.ConnCountResp, error) {
	authCtx := svr.collectMaintainAuthContext(ctx, authcommon.Read, authcommon.DescribeServerConnections)
	if _, err := svr.strategyMgn.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return nil, err
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return svr.targetServer.GetServerConnections(ctx, req)
}

func (svr *serverAuthAbility) GetServerConnStats(ctx context.Context, req *admin.ConnReq) (*admin.ConnStatsResp, error) {
	authCtx := svr.collectMaintainAuthContext(ctx, authcommon.Read, authcommon.DescribeServerConnStats)
	if _, err := svr.strategyMgn.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return nil, err
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return svr.targetServer.GetServerConnStats(ctx, req)
}

func (svr *serverAuthAbility) CloseConnections(ctx context.Context, reqs []admin.ConnReq) error {
	authCtx := svr.collectMaintainAuthContext(ctx, authcommon.Delete, authcommon.CloseConnections)
	if _, err := svr.strategyMgn.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return err
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return svr.targetServer.CloseConnections(ctx, reqs)
}

func (svr *serverAuthAbility) FreeOSMemory(ctx context.Context) error {
	authCtx := svr.collectMaintainAuthContext(ctx, authcommon.Modify, authcommon.FreeOSMemory)
	if _, err := svr.strategyMgn.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return err
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return svr.targetServer.FreeOSMemory(ctx)
}

func (svr *serverAuthAbility) CleanInstance(ctx context.Context, req *apiservice.Instance) *apiservice.Response {
	authCtx := svr.collectMaintainAuthContext(ctx, authcommon.Delete, authcommon.CleanInstance)
	if _, err := svr.strategyMgn.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewResponseWithMsg(convertToErrCode(err), err.Error())
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return svr.targetServer.CleanInstance(ctx, req)
}

func (svr *serverAuthAbility) BatchCleanInstances(ctx context.Context, batchSize uint32) (uint32, error) {
	authCtx := svr.collectMaintainAuthContext(ctx, authcommon.Delete, authcommon.BatchCleanInstances)
	if _, err := svr.strategyMgn.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return 0, err
	}

	return svr.targetServer.BatchCleanInstances(ctx, batchSize)
}

func (svr *serverAuthAbility) GetLastHeartbeat(ctx context.Context, req *apiservice.Instance) *apiservice.Response {
	authCtx := svr.collectMaintainAuthContext(ctx, authcommon.Read, authcommon.DescribeInstanceLastHeartbeat)
	if _, err := svr.strategyMgn.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewResponseWithMsg(convertToErrCode(err), err.Error())
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return svr.targetServer.GetLastHeartbeat(ctx, req)
}

func (svr *serverAuthAbility) GetLogOutputLevel(ctx context.Context) ([]admin.ScopeLevel, error) {
	authCtx := svr.collectMaintainAuthContext(ctx, authcommon.Read, authcommon.DescribeGetLogOutputLevel)
	if _, err := svr.strategyMgn.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return nil, err
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return svr.targetServer.GetLogOutputLevel(ctx)
}

func (svr *serverAuthAbility) SetLogOutputLevel(ctx context.Context, scope string, level string) error {
	authCtx := svr.collectMaintainAuthContext(ctx, authcommon.Modify, authcommon.UpdateLogOutputLevel)
	if _, err := svr.strategyMgn.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return err
	}

	return svr.targetServer.SetLogOutputLevel(ctx, scope, level)
}

func (svr *serverAuthAbility) ListLeaderElections(ctx context.Context) ([]*admin.LeaderElection, error) {
	authCtx := svr.collectMaintainAuthContext(ctx, authcommon.Read, authcommon.DescribeLeaderElections)
	if _, err := svr.strategyMgn.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return nil, err
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return svr.targetServer.ListLeaderElections(ctx)
}

func (svr *serverAuthAbility) ReleaseLeaderElection(ctx context.Context, electKey string) error {
	authCtx := svr.collectMaintainAuthContext(ctx, authcommon.Modify, authcommon.ReleaseLeaderElection)
	if _, err := svr.strategyMgn.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return err
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return svr.targetServer.ReleaseLeaderElection(ctx, electKey)
}

func (svr *serverAuthAbility) GetCMDBInfo(ctx context.Context) ([]model.LocationView, error) {
	authCtx := svr.collectMaintainAuthContext(ctx, authcommon.Read, authcommon.DescribeCMDBInfo)
	if _, err := svr.strategyMgn.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return nil, err
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return svr.targetServer.GetCMDBInfo(ctx)
}
