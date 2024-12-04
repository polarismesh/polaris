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

package auth

import (
	"context"

	apisecurity "github.com/polarismesh/specification/source/go/api/v1/security"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"

	"github.com/polarismesh/polaris/admin"
	"github.com/polarismesh/polaris/auth"
	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/model"
	admincommon "github.com/polarismesh/polaris/common/model/admin"
	authcommon "github.com/polarismesh/polaris/common/model/auth"
	"github.com/polarismesh/polaris/common/utils"
)

var _ admin.AdminOperateServer = (*Server)(nil)

// Server 带有鉴权能力的 maintainServer
type Server struct {
	nextSvr   admin.AdminOperateServer
	userSvr   auth.UserServer
	policySvr auth.StrategyServer
}

func NewServer(nextSvr admin.AdminOperateServer,
	userSvr auth.UserServer, policySvr auth.StrategyServer) admin.AdminOperateServer {
	proxy := &Server{
		nextSvr:   nextSvr,
		userSvr:   userSvr,
		policySvr: policySvr,
	}

	return proxy
}

func (svr *Server) collectMaintainAuthContext(ctx context.Context, resourceOp authcommon.ResourceOperation,
	methodName authcommon.ServerFunctionName) *authcommon.AcquireContext {
	return authcommon.NewAcquireContext(
		authcommon.WithRequestContext(ctx),
		authcommon.WithOperation(resourceOp),
		authcommon.WithModule(authcommon.MaintainModule),
		authcommon.WithMethod(methodName),
	)
}

func (s *Server) HasMainUser(ctx context.Context) *apiservice.Response {
	return s.nextSvr.HasMainUser(ctx)
}

func (s *Server) InitMainUser(ctx context.Context, user *apisecurity.User) *apiservice.Response {
	return s.nextSvr.InitMainUser(ctx, user)
}

func (svr *Server) GetServerConnections(ctx context.Context, req *admincommon.ConnReq) (*admincommon.ConnCountResp, error) {
	authCtx := svr.collectMaintainAuthContext(ctx, authcommon.Read, authcommon.DescribeServerConnections)
	if _, err := svr.policySvr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return nil, err
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return svr.nextSvr.GetServerConnections(ctx, req)
}

func (svr *Server) GetServerConnStats(ctx context.Context, req *admincommon.ConnReq) (*admincommon.ConnStatsResp, error) {
	authCtx := svr.collectMaintainAuthContext(ctx, authcommon.Read, authcommon.DescribeServerConnStats)
	if _, err := svr.policySvr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return nil, err
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return svr.nextSvr.GetServerConnStats(ctx, req)
}

func (svr *Server) CloseConnections(ctx context.Context, reqs []admincommon.ConnReq) error {
	authCtx := svr.collectMaintainAuthContext(ctx, authcommon.Delete, authcommon.CloseConnections)
	if _, err := svr.policySvr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return err
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return svr.nextSvr.CloseConnections(ctx, reqs)
}

func (svr *Server) FreeOSMemory(ctx context.Context) error {
	authCtx := svr.collectMaintainAuthContext(ctx, authcommon.Modify, authcommon.FreeOSMemory)
	if _, err := svr.policySvr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return err
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return svr.nextSvr.FreeOSMemory(ctx)
}

func (svr *Server) CleanInstance(ctx context.Context, req *apiservice.Instance) *apiservice.Response {
	authCtx := svr.collectMaintainAuthContext(ctx, authcommon.Delete, authcommon.CleanInstance)
	if _, err := svr.policySvr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewResponse(authcommon.ConvertToErrCode(err))
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return svr.nextSvr.CleanInstance(ctx, req)
}

func (svr *Server) BatchCleanInstances(ctx context.Context, batchSize uint32) (uint32, error) {
	authCtx := svr.collectMaintainAuthContext(ctx, authcommon.Delete, authcommon.BatchCleanInstances)
	if _, err := svr.policySvr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return 0, err
	}

	return svr.nextSvr.BatchCleanInstances(ctx, batchSize)
}

func (svr *Server) GetLastHeartbeat(ctx context.Context, req *apiservice.Instance) *apiservice.Response {
	authCtx := svr.collectMaintainAuthContext(ctx, authcommon.Read, authcommon.DescribeInstanceLastHeartbeat)
	if _, err := svr.policySvr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewResponse(authcommon.ConvertToErrCode(err))
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return svr.nextSvr.GetLastHeartbeat(ctx, req)
}

func (svr *Server) GetLogOutputLevel(ctx context.Context) ([]admincommon.ScopeLevel, error) {
	authCtx := svr.collectMaintainAuthContext(ctx, authcommon.Read, authcommon.DescribeGetLogOutputLevel)
	if _, err := svr.policySvr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return nil, err
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return svr.nextSvr.GetLogOutputLevel(ctx)
}

func (svr *Server) SetLogOutputLevel(ctx context.Context, scope string, level string) error {
	authCtx := svr.collectMaintainAuthContext(ctx, authcommon.Modify, authcommon.UpdateLogOutputLevel)
	if _, err := svr.policySvr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return err
	}

	return svr.nextSvr.SetLogOutputLevel(ctx, scope, level)
}

func (svr *Server) ListLeaderElections(ctx context.Context) ([]*admincommon.LeaderElection, error) {
	authCtx := svr.collectMaintainAuthContext(ctx, authcommon.Read, authcommon.DescribeLeaderElections)
	if _, err := svr.policySvr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return nil, err
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return svr.nextSvr.ListLeaderElections(ctx)
}

func (svr *Server) ReleaseLeaderElection(ctx context.Context, electKey string) error {
	authCtx := svr.collectMaintainAuthContext(ctx, authcommon.Modify, authcommon.ReleaseLeaderElection)
	if _, err := svr.policySvr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return err
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return svr.nextSvr.ReleaseLeaderElection(ctx, electKey)
}

func (svr *Server) GetCMDBInfo(ctx context.Context) ([]model.LocationView, error) {
	authCtx := svr.collectMaintainAuthContext(ctx, authcommon.Read, authcommon.DescribeCMDBInfo)
	if _, err := svr.policySvr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return nil, err
	}

	ctx = authCtx.GetRequestContext()
	ctx = context.WithValue(ctx, utils.ContextAuthContextKey, authCtx)

	return svr.nextSvr.GetCMDBInfo(ctx)
}

// GetServerFunctions .
func (svr *Server) GetServerFunctions(ctx context.Context) []authcommon.ServerFunctionGroup {
	return svr.nextSvr.GetServerFunctions(ctx)
}
