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

package paramcheck

import (
	"context"
	"strconv"

	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	apisecurity "github.com/polarismesh/specification/source/go/api/v1/security"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"
	"go.uber.org/zap"

	"github.com/polarismesh/polaris/auth"
	cachetypes "github.com/polarismesh/polaris/cache/api"
	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/log"
	authcommon "github.com/polarismesh/polaris/common/model/auth"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/store"
)

var (
	// StrategyFilterAttributes strategy filter attributes
	StrategyFilterAttributes = map[string]bool{
		"id":             true,
		"name":           true,
		"owner":          true,
		"offset":         true,
		"limit":          true,
		"principal_id":   true,
		"principal_type": true,
		"res_id":         true,
		"res_type":       true,
		"default":        true,
		"show_detail":    true,
	}
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

// PolicyHelper implements auth.StrategyServer.
func (svr *Server) PolicyHelper() auth.PolicyHelper {
	return svr.nextSvr.PolicyHelper()
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
	return svr.nextSvr.CreateStrategy(ctx, strategy)
}

// UpdateStrategies 批量更新策略
func (svr *Server) UpdateStrategies(ctx context.Context, reqs []*apisecurity.ModifyAuthStrategy) *apiservice.BatchWriteResponse {
	return svr.nextSvr.UpdateStrategies(ctx, reqs)
}

// DeleteStrategies 删除策略
func (svr *Server) DeleteStrategies(ctx context.Context, reqs []*apisecurity.AuthStrategy) *apiservice.BatchWriteResponse {
	return svr.nextSvr.DeleteStrategies(ctx, reqs)
}

// GetStrategies 获取资源列表
// support 1. 支持按照 principal-id + principal-role 进行查询
// support 2. 支持普通的鉴权策略查询
func (svr *Server) GetStrategies(ctx context.Context, query map[string]string) *apiservice.BatchQueryResponse {
	log.Debug("[Auth][Strategy] origin get strategies query params", utils.RequestID(ctx), zap.Any("query", query))

	searchFilters := make(map[string]string, len(query))
	for key, value := range query {
		if _, ok := StrategyFilterAttributes[key]; !ok {
			log.Errorf("[Auth][Strategy] get strategies attribute(%s) it not allowed", key)
			return api.NewAuthBatchQueryResponseWithMsg(apimodel.Code_InvalidParameter, key+" is not allowed")
		}
		searchFilters[key] = value
	}

	offset, limit, err := utils.ParseOffsetAndLimit(searchFilters)

	if err != nil {
		return api.NewAuthBatchQueryResponse(apimodel.Code_InvalidParameter)
	}
	searchFilters["offset"] = strconv.FormatUint(uint64(offset), 10)
	searchFilters["limit"] = strconv.FormatUint(uint64(limit), 10)
	return svr.nextSvr.GetStrategies(ctx, query)
}

// GetStrategy 获取策略详细
func (svr *Server) GetStrategy(ctx context.Context, strategy *apisecurity.AuthStrategy) *apiservice.Response {
	return svr.nextSvr.GetStrategy(ctx, strategy)
}

// GetPrincipalResources 获取某个 principal 的所有可操作资源列表
func (svr *Server) GetPrincipalResources(ctx context.Context, query map[string]string) *apiservice.Response {
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

// CreateRoles 批量创建角色
func (svr *Server) CreateRoles(ctx context.Context, reqs []*apisecurity.Role) *apiservice.BatchWriteResponse {
	return svr.nextSvr.CreateRoles(ctx, reqs)
}

// UpdateRoles 批量更新角色
func (svr *Server) UpdateRoles(ctx context.Context, reqs []*apisecurity.Role) *apiservice.BatchWriteResponse {
	return svr.nextSvr.UpdateRoles(ctx, reqs)
}

// DeleteRoles 批量删除角色
func (svr *Server) DeleteRoles(ctx context.Context, reqs []*apisecurity.Role) *apiservice.BatchWriteResponse {
	return svr.nextSvr.DeleteRoles(ctx, reqs)
}

// GetRoles 查询角色列表
func (svr *Server) GetRoles(ctx context.Context, query map[string]string) *apiservice.BatchQueryResponse {
	return svr.nextSvr.GetRoles(ctx, query)
}
