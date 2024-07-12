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

	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	apisecurity "github.com/polarismesh/specification/source/go/api/v1/security"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"
	"go.uber.org/zap"

	"github.com/polarismesh/polaris/auth"
	cachetypes "github.com/polarismesh/polaris/cache/api"
	api "github.com/polarismesh/polaris/common/api/v1"
	authcommon "github.com/polarismesh/polaris/common/model/auth"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/store"
)

var (
	// MustOwner 必须超级账户 or 主账户
	MustOwner = true
	// NotOwner 任意账户
	NotOwner = false
	// WriteOp 写操作
	WriteOp = true
	// ReadOp 读操作
	ReadOp = false
)

func NewServer(nextSvr auth.StrategyServer) auth.StrategyServer {
	return &Server{
		nextSvr: nextSvr,
	}
}

type Server struct {
	nextSvr auth.StrategyServer
	userSvr auth.UserServer
}

// Initialize 执行初始化动作
func (svr *Server) Initialize(options *auth.Config, storage store.Store, cacheMgr cachetypes.CacheManager, userSvr auth.UserServer) error {
	svr.userSvr = userSvr
	return svr.nextSvr.Initialize(options, storage, cacheMgr, userSvr)
}

// Name 策略管理server名称
func (svr *Server) Name() string {
	return svr.nextSvr.Name()
}

// CreateStrategy 创建策略
func (svr *Server) CreateStrategy(ctx context.Context, strategy *apisecurity.AuthStrategy) *apiservice.Response {
	ctx, rsp := svr.verifyAuth(ctx, WriteOp, MustOwner)
	if rsp != nil {
		return rsp
	}
	return svr.nextSvr.CreateStrategy(ctx, strategy)
}

// UpdateStrategies 批量更新策略
func (svr *Server) UpdateStrategies(ctx context.Context, reqs []*apisecurity.ModifyAuthStrategy) *apiservice.BatchWriteResponse {
	ctx, rsp := svr.verifyAuth(ctx, WriteOp, MustOwner)
	if rsp != nil {
		resp := api.NewAuthBatchWriteResponse(apimodel.Code_ExecuteSuccess)
		api.Collect(resp, rsp)
		return resp
	}
	return svr.nextSvr.UpdateStrategies(ctx, reqs)
}

// DeleteStrategies 删除策略
func (svr *Server) DeleteStrategies(ctx context.Context, reqs []*apisecurity.AuthStrategy) *apiservice.BatchWriteResponse {
	ctx, rsp := svr.verifyAuth(ctx, WriteOp, MustOwner)
	if rsp != nil {
		resp := api.NewAuthBatchWriteResponse(apimodel.Code_ExecuteSuccess)
		api.Collect(resp, rsp)
		return resp
	}
	return svr.nextSvr.DeleteStrategies(ctx, reqs)
}

// GetStrategies 获取资源列表
// support 1. 支持按照 principal-id + principal-role 进行查询
// support 2. 支持普通的鉴权策略查询
func (svr *Server) GetStrategies(ctx context.Context, query map[string]string) *apiservice.BatchQueryResponse {
	ctx, rsp := svr.verifyAuth(ctx, ReadOp, NotOwner)
	if rsp != nil {
		return api.NewAuthBatchQueryResponseWithMsg(apimodel.Code(rsp.GetCode().Value), rsp.Info.Value)
	}
	return svr.nextSvr.GetStrategies(ctx, query)
}

// GetStrategy 获取策略详细
func (svr *Server) GetStrategy(ctx context.Context, strategy *apisecurity.AuthStrategy) *apiservice.Response {
	ctx, rsp := svr.verifyAuth(ctx, ReadOp, NotOwner)
	if rsp != nil {
		return rsp
	}
	return svr.nextSvr.GetStrategy(ctx, strategy)
}

// GetPrincipalResources 获取某个 principal 的所有可操作资源列表
func (svr *Server) GetPrincipalResources(ctx context.Context, query map[string]string) *apiservice.Response {
	ctx, rsp := svr.verifyAuth(ctx, ReadOp, NotOwner)
	if rsp != nil {
		return rsp
	}
	return svr.nextSvr.GetPrincipalResources(ctx, query)
}

// GetAuthChecker 获取鉴权检查器
func (svr *Server) GetAuthChecker() auth.AuthChecker {
	return svr.nextSvr.GetAuthChecker()
}

// AfterResourceOperation 操作完资源的后置处理逻辑
func (svr *Server) AfterResourceOperation(afterCtx *authcommon.AcquireContext) error {
	return svr.nextSvr.AfterResourceOperation(afterCtx)
}

// verifyAuth 用于 user、group 以及 strategy 模块的鉴权工作检查
func (svr *Server) verifyAuth(ctx context.Context, isWrite bool,
	needOwner bool) (context.Context, *apiservice.Response) {
	reqId := utils.ParseRequestID(ctx)
	authToken := utils.ParseAuthToken(ctx)

	if authToken == "" {
		log.Error("[Auth][Server] auth token is empty", utils.ZapRequestID(reqId))
		return nil, api.NewAuthResponse(apimodel.Code_EmptyAutToken)
	}

	authCtx := authcommon.NewAcquireContext(
		authcommon.WithRequestContext(ctx),
		authcommon.WithModule(authcommon.AuthModule),
	)

	// case 1. 如果 error 不是 token 被禁止的 error，直接返回
	// case 2. 如果 error 是 token 被禁止，按下面情况判断
	// 		i. 如果当前只是一个数据的读取操作，则放通
	// 		ii. 如果当前是一个数据的写操作，则只能允许处于正常的 token 进行操作
	if err := svr.userSvr.CheckCredential(authCtx); err != nil {
		log.Error("[Auth][Server] verify auth token", utils.ZapRequestID(reqId), zap.Error(err))
		return nil, api.NewAuthResponse(apimodel.Code_AuthTokenForbidden)
	}

	attachVal, exist := authCtx.GetAttachment(authcommon.TokenDetailInfoKey)
	if !exist {
		log.Error("[Auth][Server] token detail info not exist", utils.ZapRequestID(reqId))
		return nil, api.NewAuthResponse(apimodel.Code_TokenNotExisted)
	}

	operateInfo := attachVal.(auth.OperatorInfo)
	if isWrite && operateInfo.Disable {
		log.Error("[Auth][Server] token is disabled", utils.ZapRequestID(reqId),
			zap.String("operation", authCtx.GetMethod()))
		return nil, api.NewAuthResponse(apimodel.Code_TokenDisabled)
	}

	if !operateInfo.IsUserToken {
		log.Error("[Auth][Server] only user role can access this API", utils.ZapRequestID(reqId))
		return nil, api.NewAuthResponse(apimodel.Code_OperationRoleForbidden)
	}

	if needOwner && auth.IsSubAccount(operateInfo) {
		log.Error("[Auth][Server] only admin/owner account can access this API", utils.ZapRequestID(reqId))
		return nil, api.NewAuthResponse(apimodel.Code_OperationRoleForbidden)
	}

	return authCtx.GetRequestContext(), nil
}
