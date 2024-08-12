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

	"github.com/golang/protobuf/ptypes/wrappers"
	apisecurity "github.com/polarismesh/specification/source/go/api/v1/security"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"

	"github.com/polarismesh/polaris/auth"
	cachetypes "github.com/polarismesh/polaris/cache/api"
	api "github.com/polarismesh/polaris/common/api/v1"
	authcommon "github.com/polarismesh/polaris/common/model/auth"
	"github.com/polarismesh/polaris/store"
)

type (
	PolicyInfoGetter interface {
		GetId() *wrappers.StringValue
		GetName() *wrappers.StringValue
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
	authCtx := authcommon.NewAcquireContext(
		authcommon.WithRequestContext(ctx),
		authcommon.WithOperation(authcommon.Create),
		authcommon.WithModule(authcommon.AuthModule),
		authcommon.WithMethod(authcommon.CreateAuthPolicy),
	)

	if _, err := svr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		resp := api.NewResponseWithMsg(authcommon.ConvertToErrCode(err), err.Error())
		return resp
	}
	return svr.nextSvr.CreateStrategy(authCtx.GetRequestContext(), strategy)
}

// UpdateStrategies 批量更新策略
func (svr *Server) UpdateStrategies(ctx context.Context, reqs []*apisecurity.ModifyAuthStrategy) *apiservice.BatchWriteResponse {
	resources := make([]authcommon.ResourceEntry, 0, len(reqs))
	for i := range reqs {
		entry := authcommon.ResourceEntry{
			Type: apisecurity.ResourceType_PolicyRules,
			ID:   reqs[i].GetId().GetValue(),
		}
		if saveRule := svr.nextSvr.PolicyHelper().GetPolicyRule(reqs[i].GetId().GetValue()); saveRule != nil {
			entry.Metadata = saveRule.Metadata
		}
		resources = append(resources, entry)
	}

	authCtx := authcommon.NewAcquireContext(
		authcommon.WithRequestContext(ctx),
		authcommon.WithOperation(authcommon.Modify),
		authcommon.WithModule(authcommon.AuthModule),
		authcommon.WithMethod(authcommon.UpdateAuthPolicies),
		authcommon.WithAccessResources(map[apisecurity.ResourceType][]authcommon.ResourceEntry{
			apisecurity.ResourceType_PolicyRules: resources,
		}),
	)

	if _, err := svr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		resp := api.NewBatchWriteResponseWithMsg(authcommon.ConvertToErrCode(err), err.Error())
		return resp
	}
	return svr.nextSvr.UpdateStrategies(authCtx.GetRequestContext(), reqs)
}

// DeleteStrategies 删除策略
func (svr *Server) DeleteStrategies(ctx context.Context, reqs []*apisecurity.AuthStrategy) *apiservice.BatchWriteResponse {
	resources := make([]authcommon.ResourceEntry, 0, len(reqs))
	for i := range reqs {
		entry := authcommon.ResourceEntry{
			Type: apisecurity.ResourceType_PolicyRules,
			ID:   reqs[i].GetId().GetValue(),
		}
		if saveRule := svr.nextSvr.PolicyHelper().GetPolicyRule(reqs[i].GetId().GetValue()); saveRule != nil {
			entry.Metadata = saveRule.Metadata
		}
		resources = append(resources, entry)
	}

	authCtx := authcommon.NewAcquireContext(
		authcommon.WithRequestContext(ctx),
		authcommon.WithOperation(authcommon.Delete),
		authcommon.WithModule(authcommon.AuthModule),
		authcommon.WithMethod(authcommon.DeleteAuthPolicies),
		authcommon.WithAccessResources(map[apisecurity.ResourceType][]authcommon.ResourceEntry{
			apisecurity.ResourceType_PolicyRules: resources,
		}),
	)

	if _, err := svr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		resp := api.NewBatchWriteResponseWithMsg(authcommon.ConvertToErrCode(err), err.Error())
		return resp
	}
	return svr.nextSvr.DeleteStrategies(authCtx.GetRequestContext(), reqs)
}

// GetStrategies 获取资源列表
// support 1. 支持按照 principal-id + principal-role 进行查询
// support 2. 支持普通的鉴权策略查询
func (svr *Server) GetStrategies(ctx context.Context, query map[string]string) *apiservice.BatchQueryResponse {
	authCtx := authcommon.NewAcquireContext(
		authcommon.WithRequestContext(ctx),
		authcommon.WithOperation(authcommon.Read),
		authcommon.WithModule(authcommon.AuthModule),
		authcommon.WithMethod(authcommon.DescribeAuthPolicies),
	)

	checker := svr.GetAuthChecker()
	if _, err := checker.CheckConsolePermission(authCtx); err != nil {
		return api.NewAuthBatchQueryResponseWithMsg(authcommon.ConvertToErrCode(err), err.Error())
	}

	ctx = cachetypes.AppendAuthPolicyPredicate(ctx, func(ctx context.Context, sd *authcommon.StrategyDetail) bool {
		return checker.ResourcePredicate(authCtx, &authcommon.ResourceEntry{
			Type:     apisecurity.ResourceType_PolicyRules,
			ID:       sd.ID,
			Metadata: sd.Metadata,
		})
	})

	return svr.nextSvr.GetStrategies(ctx, query)
}

// GetStrategy 获取策略详细
func (svr *Server) GetStrategy(ctx context.Context, strategy *apisecurity.AuthStrategy) *apiservice.Response {
	entry := authcommon.ResourceEntry{
		Type: apisecurity.ResourceType_PolicyRules,
		ID:   strategy.GetId().GetValue(),
	}
	saveRule := svr.nextSvr.PolicyHelper().GetPolicyRule(strategy.GetId().GetValue())
	if saveRule != nil {
		entry.Metadata = saveRule.Metadata
	}

	authCtx := authcommon.NewAcquireContext(
		authcommon.WithRequestContext(ctx),
		authcommon.WithOperation(authcommon.Read),
		authcommon.WithModule(authcommon.AuthModule),
		authcommon.WithMethod(authcommon.DescribeAuthPolicyDetail),
		authcommon.WithAccessResources(map[apisecurity.ResourceType][]authcommon.ResourceEntry{
			apisecurity.ResourceType_PolicyRules: {entry},
		}),
	)

	checker := svr.GetAuthChecker()

	if _, err := checker.CheckConsolePermission(authCtx); err != nil {
		return api.NewResponseWithMsg(authcommon.ConvertToErrCode(err), err.Error())
	}
	return svr.nextSvr.GetStrategy(authCtx.GetRequestContext(), strategy)
}

// GetPrincipalResources 获取某个 principal 的所有可操作资源列表
func (svr *Server) GetPrincipalResources(ctx context.Context, query map[string]string) *apiservice.Response {
	authCtx := authcommon.NewAcquireContext(
		authcommon.WithRequestContext(ctx),
		authcommon.WithOperation(authcommon.Read),
		authcommon.WithModule(authcommon.AuthModule),
		authcommon.WithMethod(authcommon.DescribePrincipalResources),
	)

	checker := svr.GetAuthChecker()

	if _, err := checker.CheckConsolePermission(authCtx); err != nil {
		return api.NewResponseWithMsg(authcommon.ConvertToErrCode(err), err.Error())
	}
	return svr.nextSvr.GetPrincipalResources(authCtx.GetRequestContext(), query)
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
	authCtx := authcommon.NewAcquireContext(
		authcommon.WithRequestContext(ctx),
		authcommon.WithOperation(authcommon.Create),
		authcommon.WithModule(authcommon.AuthModule),
		authcommon.WithMethod(authcommon.CreateAuthRoles),
	)

	if _, err := svr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		resp := api.NewBatchWriteResponseWithMsg(authcommon.ConvertToErrCode(err), err.Error())
		return resp
	}
	return svr.nextSvr.CreateRoles(authCtx.GetRequestContext(), reqs)
}

// UpdateRoles 批量更新角色
func (svr *Server) UpdateRoles(ctx context.Context, reqs []*apisecurity.Role) *apiservice.BatchWriteResponse {
	resources := make([]authcommon.ResourceEntry, 0, len(reqs))
	for i := range reqs {
		entry := authcommon.ResourceEntry{
			Type: apisecurity.ResourceType_Roles,
			ID:   reqs[i].GetId(),
		}
		if saveRule := svr.nextSvr.PolicyHelper().GetRole(reqs[i].GetId()); saveRule != nil {
			entry.Metadata = saveRule.Metadata
		}
		resources = append(resources, entry)
	}

	authCtx := authcommon.NewAcquireContext(
		authcommon.WithRequestContext(ctx),
		authcommon.WithOperation(authcommon.Modify),
		authcommon.WithModule(authcommon.AuthModule),
		authcommon.WithMethod(authcommon.UpdateAuthRoles),
		authcommon.WithAccessResources(map[apisecurity.ResourceType][]authcommon.ResourceEntry{
			apisecurity.ResourceType_Roles: resources,
		}),
	)

	if _, err := svr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		resp := api.NewBatchWriteResponseWithMsg(authcommon.ConvertToErrCode(err), err.Error())
		return resp
	}
	return svr.nextSvr.UpdateRoles(authCtx.GetRequestContext(), reqs)
}

// DeleteRoles 批量删除角色
func (svr *Server) DeleteRoles(ctx context.Context, reqs []*apisecurity.Role) *apiservice.BatchWriteResponse {
	resources := make([]authcommon.ResourceEntry, 0, len(reqs))
	for i := range reqs {
		entry := authcommon.ResourceEntry{
			Type: apisecurity.ResourceType_Roles,
			ID:   reqs[i].GetId(),
		}
		if saveRule := svr.nextSvr.PolicyHelper().GetRole(reqs[i].GetId()); saveRule != nil {
			entry.Metadata = saveRule.Metadata
		}
		resources = append(resources, entry)
	}

	authCtx := authcommon.NewAcquireContext(
		authcommon.WithRequestContext(ctx),
		authcommon.WithOperation(authcommon.Modify),
		authcommon.WithModule(authcommon.AuthModule),
		authcommon.WithMethod(authcommon.DeleteAuthRoles),
		authcommon.WithAccessResources(map[apisecurity.ResourceType][]authcommon.ResourceEntry{
			apisecurity.ResourceType_Roles: resources,
		}),
	)

	if _, err := svr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		resp := api.NewBatchWriteResponseWithMsg(authcommon.ConvertToErrCode(err), err.Error())
		return resp
	}
	return svr.nextSvr.DeleteRoles(authCtx.GetRequestContext(), reqs)
}

// GetRoles 查询角色列表
func (svr *Server) GetRoles(ctx context.Context, query map[string]string) *apiservice.BatchQueryResponse {
	authCtx := authcommon.NewAcquireContext(
		authcommon.WithRequestContext(ctx),
		authcommon.WithOperation(authcommon.Read),
		authcommon.WithModule(authcommon.AuthModule),
		authcommon.WithMethod(authcommon.DescribeAuthRoles),
	)

	if _, err := svr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewAuthBatchQueryResponseWithMsg(authcommon.ConvertToErrCode(err), err.Error())
	}

	checker := svr.GetAuthChecker()
	ctx = cachetypes.AppendAuthRolePredicate(ctx, func(ctx context.Context, sd *authcommon.Role) bool {
		return checker.ResourcePredicate(authCtx, &authcommon.ResourceEntry{
			Type:     apisecurity.ResourceType_Roles,
			ID:       sd.ID,
			Metadata: sd.Metadata,
		})
	})

	return svr.nextSvr.GetRoles(ctx, query)
}
