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
	commonstore "github.com/polarismesh/polaris/common/store"
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
	storage  store.Store
	cacheMgr cachetypes.CacheManager
	nextSvr  auth.StrategyServer
	userSvr  auth.UserServer
}

// PolicyHelper implements auth.StrategyServer.
func (svr *Server) PolicyHelper() auth.PolicyHelper {
	return svr.nextSvr.PolicyHelper()
}

// Initialize 执行初始化动作
func (svr *Server) Initialize(options *auth.Config, storage store.Store, cacheMgr cachetypes.CacheManager, userSvr auth.UserServer) error {
	svr.userSvr = userSvr
	svr.cacheMgr = cacheMgr
	svr.storage = storage
	return svr.nextSvr.Initialize(options, storage, cacheMgr, userSvr)
}

// Name 策略管理server名称
func (svr *Server) Name() string {
	return svr.nextSvr.Name()
}

// CreateStrategy 创建策略
func (svr *Server) CreateStrategy(ctx context.Context, req *apisecurity.AuthStrategy) *apiservice.Response {
	if err := svr.checkCreateStrategy(req); err != nil {
		return err
	}
	return svr.nextSvr.CreateStrategy(ctx, req)
}

// UpdateStrategies 批量更新策略
func (svr *Server) UpdateStrategies(ctx context.Context, reqs []*apisecurity.ModifyAuthStrategy) *apiservice.BatchWriteResponse {
	batchResp := api.NewBatchWriteResponse(apimodel.Code_ExecuteSuccess)
	for i := range reqs {
		var rsp *apiservice.Response
		strategy, err := svr.storage.GetStrategyDetail(reqs[i].GetId().GetValue())
		if err != nil {
			log.Error("[Auth][Strategy] get strategy from store", utils.RequestID(ctx), zap.Error(err))
			rsp = api.NewModifyAuthStrategyResponse(commonstore.StoreCode2APICode(err), reqs[i])
		}
		if strategy == nil {
			continue
		} else {
			rsp = svr.checkUpdateStrategy(ctx, reqs[i], strategy)
		}
		api.Collect(batchResp, rsp)
	}
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

// checkCreateStrategy 检查创建鉴权策略的请求
func (svr *Server) checkCreateStrategy(req *apisecurity.AuthStrategy) *apiservice.Response {
	// 检查名称信息
	if err := CheckName(req.GetName()); err != nil {
		return api.NewAuthStrategyResponse(apimodel.Code_InvalidUserName, req)
	}
	// 检查用户是否存在
	if err := svr.checkUserExist(convertPrincipalsToUsers(req.GetPrincipals())); err != nil {
		return api.NewAuthStrategyResponse(apimodel.Code_NotFoundUser, req)
	}
	// 检查用户组是否存在
	if err := svr.checkGroupExist(convertPrincipalsToGroups(req.GetPrincipals())); err != nil {
		return api.NewAuthStrategyResponse(apimodel.Code_NotFoundUserGroup, req)
	}
	// 检查资源是否存在
	if errResp := svr.checkResourceExist(req.GetResources()); errResp != nil {
		return errResp
	}
	return nil
}

// checkUpdateStrategy 检查更新鉴权策略的请求
// Case 1. 修改的是默认鉴权策略的话，只能修改资源，不能添加用户 or 用户组
// Case 2. 鉴权策略只能被自己的 owner 对应的用户修改
func (svr *Server) checkUpdateStrategy(ctx context.Context, req *apisecurity.ModifyAuthStrategy,
	saved *authcommon.StrategyDetail) *apiservice.Response {
	if saved.Default {
		if len(req.AddPrincipals.Users) != 0 ||
			len(req.AddPrincipals.Groups) != 0 ||
			len(req.RemovePrincipals.Groups) != 0 ||
			len(req.RemovePrincipals.Users) != 0 {
			return api.NewModifyAuthStrategyResponse(apimodel.Code_NotAllowModifyDefaultStrategyPrincipal, req)
		}

		// 主账户的默认策略禁止编辑
		if len(saved.Principals) == 1 && saved.Principals[0].PrincipalType == authcommon.PrincipalUser {
			if saved.Principals[0].PrincipalID == utils.ParseOwnerID(ctx) {
				return api.NewAuthResponse(apimodel.Code_NotAllowModifyOwnerDefaultStrategy)
			}
		}
	}

	// 检查用户是否存在
	if err := svr.checkUserExist(convertPrincipalsToUsers(req.GetAddPrincipals())); err != nil {
		return api.NewModifyAuthStrategyResponse(apimodel.Code_NotFoundUser, req)
	}

	// 检查用户组是否存
	if err := svr.checkGroupExist(convertPrincipalsToGroups(req.GetAddPrincipals())); err != nil {
		return api.NewModifyAuthStrategyResponse(apimodel.Code_NotFoundUserGroup, req)
	}

	// 检查资源是否存在
	if errResp := svr.checkResourceExist(req.GetAddResources()); errResp != nil {
		return errResp
	}
	return nil
}

// checkUserExist 检查用户是否存在
func (svr *Server) checkUserExist(users []*apisecurity.User) error {
	if len(users) == 0 {
		return nil
	}
	return svr.userSvr.GetUserHelper().CheckUsersExist(context.TODO(), users)
}

// checkUserGroupExist 检查用户组是否存在
func (svr *Server) checkGroupExist(groups []*apisecurity.UserGroup) error {
	if len(groups) == 0 {
		return nil
	}
	return svr.userSvr.GetUserHelper().CheckGroupsExist(context.TODO(), groups)
}

// checkResourceExist 检查资源是否存在
func (svr *Server) checkResourceExist(resources *apisecurity.StrategyResources) *apiservice.Response {
	namespaces := resources.GetNamespaces()

	nsCache := svr.cacheMgr.Namespace()
	for index := range namespaces {
		val := namespaces[index]
		if val.GetId().GetValue() == "*" {
			break
		}
		if ns := nsCache.GetNamespace(val.GetId().GetValue()); ns == nil {
			return api.NewAuthResponse(apimodel.Code_NotFoundNamespace)
		}
	}

	services := resources.GetServices()
	svcCache := svr.cacheMgr.Service()
	for index := range services {
		val := services[index]
		if val.GetId().GetValue() == "*" {
			break
		}
		if svc := svcCache.GetServiceByID(val.GetId().GetValue()); svc == nil {
			return api.NewAuthResponse(apimodel.Code_NotFoundService)
		}
	}

	return nil
}

func convertPrincipalsToUsers(principals *apisecurity.Principals) []*apisecurity.User {
	if principals == nil {
		return make([]*apisecurity.User, 0)
	}

	users := make([]*apisecurity.User, 0, len(principals.Users))
	for k := range principals.GetUsers() {
		user := principals.GetUsers()[k]
		users = append(users, &apisecurity.User{
			Id: user.Id,
		})
	}

	return users
}

func convertPrincipalsToGroups(principals *apisecurity.Principals) []*apisecurity.UserGroup {
	if principals == nil {
		return make([]*apisecurity.UserGroup, 0)
	}

	groups := make([]*apisecurity.UserGroup, 0, len(principals.Groups))
	for k := range principals.GetGroups() {
		group := principals.GetGroups()[k]
		groups = append(groups, &apisecurity.UserGroup{
			Id: group.Id,
		})
	}

	return groups
}
