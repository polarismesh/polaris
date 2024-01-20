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

package defaultauth

import (
	"context"

	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	apisecurity "github.com/polarismesh/specification/source/go/api/v1/security"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"

	"github.com/polarismesh/polaris/auth"
	cachetypes "github.com/polarismesh/polaris/cache/api"
	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/plugin"
	"github.com/polarismesh/polaris/store"
)

func NewStrategyAuthAbility(authMgn *DefaultAuthChecker, target *Server) *StrategyAuthAbility {
	return &StrategyAuthAbility{
		authMgn: authMgn,
		target:  target,
	}
}

type StrategyAuthAbility struct {
	authMgn *DefaultAuthChecker
	target  *Server
}

// Initialize 执行初始化动作
func (svr *StrategyAuthAbility) Initialize(authOpt *auth.Config, storage store.Store,
	cacheMgn cachetypes.CacheManager) error {
	_ = cacheMgn.OpenResourceCache(cachetypes.ConfigEntry{
		Name: cachetypes.StrategyRuleName,
	})
	var (
		history = plugin.GetHistory()
		authMgn = &DefaultAuthChecker{}
	)
	if err := authMgn.Initialize(authOpt, storage, cacheMgn); err != nil {
		return err
	}

	svr.authMgn = authMgn
	svr.target = &Server{
		storage:  storage,
		history:  history,
		cacheMgn: cacheMgn,
		authMgn:  authMgn,
	}
	return nil
}

// Name of the user operator plugin
func (svr *StrategyAuthAbility) Name() string {
	return auth.DefaultStrategyMgnPluginName
}

// CreateStrategy creates a new strategy.
func (svr *StrategyAuthAbility) CreateStrategy(
	ctx context.Context, strategy *apisecurity.AuthStrategy) *apiservice.Response {
	ctx, rsp := verifyAuth(ctx, WriteOp, MustOwner, svr.authMgn)
	if rsp != nil {
		rsp.AuthStrategy = strategy
		return rsp
	}

	return svr.target.CreateStrategy(ctx, strategy)
}

// UpdateStrategies update a strategy.
func (svr *StrategyAuthAbility) UpdateStrategies(ctx context.Context,
	reqs []*apisecurity.ModifyAuthStrategy) *apiservice.BatchWriteResponse {
	ctx, rsp := verifyAuth(ctx, WriteOp, MustOwner, svr.authMgn)
	if rsp != nil {
		resp := api.NewAuthBatchWriteResponse(apimodel.Code_ExecuteSuccess)
		api.Collect(resp, rsp)
		return resp
	}

	return svr.target.UpdateStrategies(ctx, reqs)
}

// DeleteStrategies delete strategy.
func (svr *StrategyAuthAbility) DeleteStrategies(ctx context.Context,
	reqs []*apisecurity.AuthStrategy) *apiservice.BatchWriteResponse {
	ctx, rsp := verifyAuth(ctx, WriteOp, MustOwner, svr.authMgn)
	if rsp != nil {
		resp := api.NewAuthBatchWriteResponse(apimodel.Code_ExecuteSuccess)
		api.Collect(resp, rsp)
		return resp
	}

	return svr.target.DeleteStrategies(ctx, reqs)
}

// GetStrategies get strategy list .
func (svr *StrategyAuthAbility) GetStrategies(ctx context.Context,
	query map[string]string) *apiservice.BatchQueryResponse {
	ctx, rsp := verifyAuth(ctx, ReadOp, NotOwner, svr.authMgn)
	if rsp != nil {
		return api.NewAuthBatchQueryResponseWithMsg(apimodel.Code(rsp.GetCode().Value), rsp.Info.Value)
	}

	return svr.target.GetStrategies(ctx, query)
}

// GetStrategy get strategy.
func (svr *StrategyAuthAbility) GetStrategy(
	ctx context.Context, strategy *apisecurity.AuthStrategy) *apiservice.Response {
	ctx, rsp := verifyAuth(ctx, ReadOp, NotOwner, svr.authMgn)
	if rsp != nil {
		return rsp
	}

	return svr.target.GetStrategy(ctx, strategy)
}

// GetPrincipalResources get principal resources.
func (svr *StrategyAuthAbility) GetPrincipalResources(ctx context.Context, query map[string]string) *apiservice.Response {
	ctx, rsp := verifyAuth(ctx, ReadOp, NotOwner, svr.authMgn)
	if rsp != nil {
		return rsp
	}

	return svr.target.GetPrincipalResources(ctx, query)
}

// GetAuthChecker 获取鉴权管理器
func (svr *StrategyAuthAbility) GetAuthChecker() auth.AuthChecker {
	return svr.authMgn
}

// AfterResourceOperation is called after resource operation
func (svr *StrategyAuthAbility) AfterResourceOperation(afterCtx *model.AcquireContext) error {
	return svr.target.AfterResourceOperation(afterCtx)
}
