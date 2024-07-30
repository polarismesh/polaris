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
	"strconv"

	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	apisecurity "github.com/polarismesh/specification/source/go/api/v1/security"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"

	"github.com/polarismesh/polaris/auth"
	cachetypes "github.com/polarismesh/polaris/cache/api"
	api "github.com/polarismesh/polaris/common/api/v1"
	authcommon "github.com/polarismesh/polaris/common/model/auth"
	authmodel "github.com/polarismesh/polaris/common/model/auth"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/store"
)

func NewServer(nextSvr auth.UserServer) auth.UserServer {
	return &Server{
		nextSvr: nextSvr,
	}
}

type Server struct {
	nextSvr   auth.UserServer
	policySvr auth.StrategyServer
}

// Initialize 初始化
func (svr *Server) Initialize(authOpt *auth.Config, storage store.Store, policyMgr auth.StrategyServer, cacheMgr cachetypes.CacheManager) error {
	return svr.nextSvr.Initialize(authOpt, storage, policyMgr, cacheMgr)
}

// Name 用户数据管理server名称
func (svr *Server) Name() string {
	return svr.nextSvr.Name()
}

// Login 登录动作
func (svr *Server) Login(req *apisecurity.LoginRequest) *apiservice.Response {
	return svr.nextSvr.Login(req)
}

// CheckCredential 检查当前操作用户凭证
func (svr *Server) CheckCredential(authCtx *authmodel.AcquireContext) error {
	return svr.nextSvr.CheckCredential(authCtx)
}

// GetUserHelper
func (svr *Server) GetUserHelper() auth.UserHelper {
	return svr.nextSvr.GetUserHelper()
}

// CreateUsers 批量创建用户
func (svr *Server) CreateUsers(ctx context.Context, users []*apisecurity.User) *apiservice.BatchWriteResponse {
	authCtx := authcommon.NewAcquireContext(
		authcommon.WithRequestContext(ctx),
		authcommon.WithOperation(authcommon.Create),
		authcommon.WithModule(authcommon.AuthModule),
		authcommon.WithMethod(authcommon.CreateUsers),
	)

	if _, err := svr.policySvr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewBatchWriteResponseWithMsg(authcommon.ConvertToErrCode(err), err.Error())
	}
	return svr.nextSvr.CreateUsers(authCtx.GetRequestContext(), users)
}

// UpdateUser 更新用户信息
func (svr *Server) UpdateUser(ctx context.Context, user *apisecurity.User) *apiservice.Response {
	helper := svr.nextSvr.GetUserHelper()
	saveUser := helper.GetUserByID(ctx, user.GetId().GetValue())
	if saveUser == nil {
		return api.NewResponse(apimodel.Code_NotFoundUser)
	}

	authCtx := authcommon.NewAcquireContext(
		authcommon.WithRequestContext(ctx),
		authcommon.WithOperation(authcommon.Modify),
		authcommon.WithModule(authcommon.AuthModule),
		authcommon.WithMethod(authcommon.UpdateUser),
		authcommon.WithAccessResources(map[apisecurity.ResourceType][]authmodel.ResourceEntry{
			apisecurity.ResourceType_Users: {
				authmodel.ResourceEntry{
					ID:       user.GetId().GetValue(),
					Type:     apisecurity.ResourceType_Users,
					Metadata: saveUser.Metadata,
				},
			},
		}),
	)

	if _, err := svr.policySvr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewResponseWithMsg(authcommon.ConvertToErrCode(err), err.Error())
	}
	return svr.nextSvr.UpdateUser(authCtx.GetRequestContext(), user)
}

// UpdateUserPassword 更新用户密码
func (svr *Server) UpdateUserPassword(ctx context.Context, req *apisecurity.ModifyUserPassword) *apiservice.Response {
	helper := svr.nextSvr.GetUserHelper()
	saveUser := helper.GetUserByID(ctx, req.GetId().GetValue())
	if saveUser == nil {
		return api.NewResponse(apimodel.Code_NotFoundUser)
	}

	authCtx := authcommon.NewAcquireContext(
		authcommon.WithRequestContext(ctx),
		authcommon.WithOperation(authcommon.Modify),
		authcommon.WithModule(authcommon.AuthModule),
		authcommon.WithMethod(authcommon.UpdateUserPassword),
		authcommon.WithAccessResources(map[apisecurity.ResourceType][]authmodel.ResourceEntry{
			apisecurity.ResourceType_Users: {
				authmodel.ResourceEntry{
					ID:       req.GetId().GetValue(),
					Type:     apisecurity.ResourceType_Users,
					Metadata: saveUser.Metadata,
				},
			},
		}),
	)

	if _, err := svr.policySvr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewResponseWithMsg(authcommon.ConvertToErrCode(err), err.Error())
	}
	return svr.nextSvr.UpdateUserPassword(authCtx.GetRequestContext(), req)
}

// DeleteUsers 批量删除用户
func (svr *Server) DeleteUsers(ctx context.Context, users []*apisecurity.User) *apiservice.BatchWriteResponse {
	helper := svr.nextSvr.GetUserHelper()
	resources := make([]authcommon.ResourceEntry, 0, len(users))
	for i := range users {
		saveUser := helper.GetUserByID(ctx, users[i].GetId().GetValue())
		if saveUser == nil {
			return api.NewBatchWriteResponse(apimodel.Code_NotFoundUser)
		}
		resources = append(resources, authmodel.ResourceEntry{
			ID:       users[i].GetId().GetValue(),
			Type:     apisecurity.ResourceType_Users,
			Metadata: saveUser.Metadata,
		})
	}
	authCtx := authcommon.NewAcquireContext(
		authcommon.WithRequestContext(ctx),
		authcommon.WithOperation(authcommon.Delete),
		authcommon.WithModule(authcommon.AuthModule),
		authcommon.WithMethod(authcommon.DeleteUsers),
		authcommon.WithAccessResources(map[apisecurity.ResourceType][]authmodel.ResourceEntry{
			apisecurity.ResourceType_Users: resources,
		}),
	)
	if _, err := svr.policySvr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewBatchWriteResponseWithMsg(authcommon.ConvertToErrCode(err), err.Error())
	}
	return svr.nextSvr.DeleteUsers(authCtx.GetRequestContext(), users)
}

// GetUsers 查询用户列表
func (svr *Server) GetUsers(ctx context.Context, query map[string]string) *apiservice.BatchQueryResponse {
	authCtx := authcommon.NewAcquireContext(
		authcommon.WithRequestContext(ctx),
		authcommon.WithOperation(authcommon.Read),
		authcommon.WithModule(authcommon.AuthModule),
		authcommon.WithMethod(authcommon.DescribeUsers),
	)
	if _, err := svr.policySvr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewBatchQueryResponseWithMsg(authcommon.ConvertToErrCode(err), err.Error())
	}
	ctx = authCtx.GetRequestContext()
	query["hide_admin"] = strconv.FormatBool(true)
	// 如果不是超级管理员，查看数据有限制
	if authcommon.ParseUserRole(ctx) != authmodel.AdminUserRole {
		// 设置 owner 参数，只能查看对应 owner 下的用户
		query["owner"] = utils.ParseOwnerID(ctx)
	}

	cachetypes.AppendUserPredicate(ctx, func(ctx context.Context, u *authcommon.User) bool {
		return svr.policySvr.GetAuthChecker().ResourcePredicate(authCtx, &authmodel.ResourceEntry{
			Type:     apisecurity.ResourceType_Users,
			ID:       u.ID,
			Metadata: u.Metadata,
		})
	})

	return svr.nextSvr.GetUsers(authCtx.GetRequestContext(), query)
}

// GetUserToken 获取用户的 token
func (svr *Server) GetUserToken(ctx context.Context, user *apisecurity.User) *apiservice.Response {
	helper := svr.nextSvr.GetUserHelper()
	saveUser := helper.GetUserByID(ctx, user.GetId().GetValue())
	if saveUser == nil {
		return api.NewResponse(apimodel.Code_NotFoundUser)
	}
	authCtx := authcommon.NewAcquireContext(
		authcommon.WithRequestContext(ctx),
		authcommon.WithOperation(authcommon.Read),
		authcommon.WithModule(authcommon.AuthModule),
		authcommon.WithMethod(authcommon.DescribeUserToken),
		authcommon.WithAccessResources(map[apisecurity.ResourceType][]authmodel.ResourceEntry{
			apisecurity.ResourceType_Users: {
				authmodel.ResourceEntry{
					ID:       user.GetId().GetValue(),
					Type:     apisecurity.ResourceType_Users,
					Metadata: saveUser.Metadata,
				},
			},
		}),
	)

	if _, err := svr.policySvr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewResponseWithMsg(authcommon.ConvertToErrCode(err), err.Error())
	}
	return svr.nextSvr.GetUserToken(authCtx.GetRequestContext(), user)
}

// UpdateUserToken 禁止用户的token使用
func (svr *Server) EnableUserToken(ctx context.Context, user *apisecurity.User) *apiservice.Response {
	helper := svr.nextSvr.GetUserHelper()
	saveUser := helper.GetUserByID(ctx, user.GetId().GetValue())
	if saveUser == nil {
		return api.NewResponse(apimodel.Code_NotFoundUser)
	}
	authCtx := authcommon.NewAcquireContext(
		authcommon.WithRequestContext(ctx),
		authcommon.WithOperation(authcommon.Modify),
		authcommon.WithModule(authcommon.AuthModule),
		authcommon.WithMethod(authcommon.EnableUserToken),
		authcommon.WithAccessResources(map[apisecurity.ResourceType][]authmodel.ResourceEntry{
			apisecurity.ResourceType_Users: {
				authmodel.ResourceEntry{
					ID:       user.GetId().GetValue(),
					Type:     apisecurity.ResourceType_Users,
					Metadata: saveUser.Metadata,
				},
			},
		}),
	)

	if _, err := svr.policySvr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewResponseWithMsg(authcommon.ConvertToErrCode(err), err.Error())
	}
	return svr.nextSvr.EnableUserToken(ctx, user)
}

// ResetUserToken 重置用户的token
func (svr *Server) ResetUserToken(ctx context.Context, user *apisecurity.User) *apiservice.Response {
	helper := svr.nextSvr.GetUserHelper()
	saveUser := helper.GetUserByID(ctx, user.GetId().GetValue())
	if saveUser == nil {
		return api.NewResponse(apimodel.Code_NotFoundUser)
	}
	authCtx := authcommon.NewAcquireContext(
		authcommon.WithRequestContext(ctx),
		authcommon.WithOperation(authcommon.Modify),
		authcommon.WithModule(authcommon.AuthModule),
		authcommon.WithMethod(authcommon.ResetUserToken),
		authcommon.WithAccessResources(map[apisecurity.ResourceType][]authmodel.ResourceEntry{
			apisecurity.ResourceType_Users: {
				authmodel.ResourceEntry{
					ID:       user.GetId().GetValue(),
					Type:     apisecurity.ResourceType_Users,
					Metadata: saveUser.Metadata,
				},
			},
		}),
	)

	if _, err := svr.policySvr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewResponseWithMsg(authcommon.ConvertToErrCode(err), err.Error())
	}
	return svr.nextSvr.ResetUserToken(ctx, user)
}

// CreateGroup 创建用户组
func (svr *Server) CreateGroup(ctx context.Context, group *apisecurity.UserGroup) *apiservice.Response {
	authCtx := authcommon.NewAcquireContext(
		authcommon.WithRequestContext(ctx),
		authcommon.WithOperation(authcommon.Create),
		authcommon.WithModule(authcommon.AuthModule),
		authcommon.WithMethod(authcommon.CreateUserGroup),
	)

	if _, err := svr.policySvr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewResponseWithMsg(authcommon.ConvertToErrCode(err), err.Error())
	}
	return svr.nextSvr.CreateGroup(authCtx.GetRequestContext(), group)
}

// UpdateGroups 更新用户组
func (svr *Server) UpdateGroups(ctx context.Context, groups []*apisecurity.ModifyUserGroup) *apiservice.BatchWriteResponse {
	helper := svr.nextSvr.GetUserHelper()
	resources := make([]authcommon.ResourceEntry, 0, len(groups))
	for i := range groups {
		saveGroup := helper.GetGroup(ctx, &apisecurity.UserGroup{Id: groups[i].GetId()})
		if saveGroup == nil {
			return api.NewBatchWriteResponse(apimodel.Code_NotFoundUserGroup)
		}
		resources = append(resources, authmodel.ResourceEntry{
			Type:     apisecurity.ResourceType_UserGroups,
			ID:       groups[i].GetId().GetValue(),
			Metadata: saveGroup.Metadata,
		})
	}

	authCtx := authcommon.NewAcquireContext(
		authcommon.WithRequestContext(ctx),
		authcommon.WithOperation(authcommon.Modify),
		authcommon.WithModule(authcommon.AuthModule),
		authcommon.WithMethod(authcommon.UpdateUserGroups),
		authcommon.WithAccessResources(map[apisecurity.ResourceType][]authmodel.ResourceEntry{
			apisecurity.ResourceType_UserGroups: resources,
		}),
	)

	if _, err := svr.policySvr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewBatchWriteResponseWithMsg(authcommon.ConvertToErrCode(err), err.Error())
	}
	return svr.nextSvr.UpdateGroups(authCtx.GetRequestContext(), groups)
}

// DeleteGroups 批量删除用户组
func (svr *Server) DeleteGroups(ctx context.Context, groups []*apisecurity.UserGroup) *apiservice.BatchWriteResponse {
	helper := svr.nextSvr.GetUserHelper()
	resources := make([]authcommon.ResourceEntry, 0, len(groups))
	for i := range groups {
		saveGroup := helper.GetGroup(ctx, &apisecurity.UserGroup{Id: groups[i].GetId()})
		if saveGroup == nil {
			return api.NewBatchWriteResponse(apimodel.Code_NotFoundUserGroup)
		}
		resources = append(resources, authmodel.ResourceEntry{
			ID: groups[i].GetId().GetValue(),
		})
	}

	authCtx := authcommon.NewAcquireContext(
		authcommon.WithRequestContext(ctx),
		authcommon.WithOperation(authcommon.Modify),
		authcommon.WithModule(authcommon.AuthModule),
		authcommon.WithMethod(authcommon.DeleteUserGroups),
		authcommon.WithAccessResources(map[apisecurity.ResourceType][]authmodel.ResourceEntry{
			apisecurity.ResourceType_UserGroups: resources,
		}),
	)

	if _, err := svr.policySvr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewBatchWriteResponseWithMsg(authcommon.ConvertToErrCode(err), err.Error())
	}
	return svr.nextSvr.DeleteGroups(ctx, groups)
}

// GetGroups 查询用户组列表（不带用户详细信息）
func (svr *Server) GetGroups(ctx context.Context, query map[string]string) *apiservice.BatchQueryResponse {
	authCtx := authcommon.NewAcquireContext(
		authcommon.WithRequestContext(ctx),
		authcommon.WithOperation(authcommon.Read),
		authcommon.WithModule(authcommon.AuthModule),
		authcommon.WithMethod(authcommon.DescribeUserGroups),
	)
	if _, err := svr.policySvr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewBatchQueryResponseWithMsg(authcommon.ConvertToErrCode(err), err.Error())
	}
	ctx = authCtx.GetRequestContext()
	if authcommon.ParseUserRole(ctx) != authmodel.AdminUserRole {
		// step 1: 设置 owner 信息，只能查看归属主帐户下的用户组
		query["owner"] = utils.ParseOwnerID(ctx)
	}

	cachetypes.AppendUserPredicate(ctx, func(ctx context.Context, u *authcommon.User) bool {
		return svr.policySvr.GetAuthChecker().ResourcePredicate(authCtx, &authmodel.ResourceEntry{
			Type:     apisecurity.ResourceType_UserGroups,
			ID:       u.ID,
			Metadata: u.Metadata,
		})
	})
	delete(query, "owner")
	return svr.nextSvr.GetGroups(ctx, query)
}

// GetGroup 根据用户组信息，查询该用户组下的用户相信
func (svr *Server) GetGroup(ctx context.Context, req *apisecurity.UserGroup) *apiservice.Response {
	helper := svr.nextSvr.GetUserHelper()
	saveGroup := helper.GetGroup(ctx, req)
	if saveGroup == nil {
		return api.NewResponse(apimodel.Code_NotFoundUserGroup)
	}
	authCtx := authcommon.NewAcquireContext(
		authcommon.WithRequestContext(ctx),
		authcommon.WithOperation(authcommon.Read),
		authcommon.WithModule(authcommon.AuthModule),
		authcommon.WithMethod(authcommon.DescribeUserGroupDetail),
		authcommon.WithAccessResources(map[apisecurity.ResourceType][]authmodel.ResourceEntry{
			apisecurity.ResourceType_UserGroups: {
				authmodel.ResourceEntry{
					Type:     apisecurity.ResourceType_UserGroups,
					ID:       req.GetId().GetValue(),
					Metadata: saveGroup.Metadata,
				},
			},
		}),
	)

	if _, err := svr.policySvr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewResponseWithMsg(authcommon.ConvertToErrCode(err), err.Error())
	}
	return svr.nextSvr.GetGroup(ctx, req)
}

// GetGroupToken 获取用户组的 token
func (svr *Server) GetGroupToken(ctx context.Context, group *apisecurity.UserGroup) *apiservice.Response {
	helper := svr.nextSvr.GetUserHelper()
	saveGroup := helper.GetGroup(ctx, group)
	if saveGroup == nil {
		return api.NewResponse(apimodel.Code_NotFoundUserGroup)
	}
	authCtx := authcommon.NewAcquireContext(
		authcommon.WithRequestContext(ctx),
		authcommon.WithOperation(authcommon.Read),
		authcommon.WithModule(authcommon.AuthModule),
		authcommon.WithMethod(authcommon.DescribeUserGroupToken),
		authcommon.WithAccessResources(map[apisecurity.ResourceType][]authmodel.ResourceEntry{
			apisecurity.ResourceType_UserGroups: {
				authmodel.ResourceEntry{
					ID:       group.GetId().GetValue(),
					Type:     apisecurity.ResourceType_UserGroups,
					Metadata: saveGroup.Metadata,
				},
			},
		}),
	)

	if _, err := svr.policySvr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewResponseWithMsg(authcommon.ConvertToErrCode(err), err.Error())
	}
	return svr.nextSvr.GetGroupToken(ctx, group)
}

// EnableGroupToken 取消用户组的 token 使用
func (svr *Server) EnableGroupToken(ctx context.Context, group *apisecurity.UserGroup) *apiservice.Response {
	helper := svr.nextSvr.GetUserHelper()
	saveGroup := helper.GetGroup(ctx, group)
	if saveGroup == nil {
		return api.NewResponse(apimodel.Code_NotFoundUserGroup)
	}
	authCtx := authcommon.NewAcquireContext(
		authcommon.WithRequestContext(ctx),
		authcommon.WithOperation(authcommon.Modify),
		authcommon.WithModule(authcommon.AuthModule),
		authcommon.WithMethod(authcommon.EnableUserGroupToken),
		authcommon.WithAccessResources(map[apisecurity.ResourceType][]authmodel.ResourceEntry{
			apisecurity.ResourceType_UserGroups: {
				authmodel.ResourceEntry{
					ID:       group.GetId().GetValue(),
					Type:     apisecurity.ResourceType_UserGroups,
					Metadata: saveGroup.Metadata,
				},
			},
		}),
	)

	if _, err := svr.policySvr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewResponseWithMsg(authcommon.ConvertToErrCode(err), err.Error())
	}
	return svr.nextSvr.EnableGroupToken(ctx, group)
}

// ResetGroupToken 重置用户组的 token
func (svr *Server) ResetGroupToken(ctx context.Context, group *apisecurity.UserGroup) *apiservice.Response {
	helper := svr.nextSvr.GetUserHelper()
	saveGroup := helper.GetGroup(ctx, group)
	if saveGroup == nil {
		return api.NewResponse(apimodel.Code_NotFoundUserGroup)
	}
	authCtx := authcommon.NewAcquireContext(
		authcommon.WithRequestContext(ctx),
		authcommon.WithOperation(authcommon.Modify),
		authcommon.WithModule(authcommon.AuthModule),
		authcommon.WithMethod(authcommon.ResetUserGroupToken),
		authcommon.WithAccessResources(map[apisecurity.ResourceType][]authmodel.ResourceEntry{
			apisecurity.ResourceType_UserGroups: {
				authmodel.ResourceEntry{
					ID:       group.GetId().GetValue(),
					Type:     apisecurity.ResourceType_UserGroups,
					Metadata: saveGroup.Metadata,
				},
			},
		}),
	)

	if _, err := svr.policySvr.GetAuthChecker().CheckConsolePermission(authCtx); err != nil {
		return api.NewResponseWithMsg(authcommon.ConvertToErrCode(err), err.Error())
	}
	return svr.nextSvr.ResetGroupToken(ctx, group)
}
