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

	"github.com/polarismesh/polaris/auth"
	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/model"
	authcommon "github.com/polarismesh/polaris/common/model/auth"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/store"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	apisecurity "github.com/polarismesh/specification/source/go/api/v1/security"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/wrapperspb"

	cachetypes "github.com/polarismesh/polaris/cache/api"
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

func NewServer(nextSvr auth.UserServer) auth.UserServer {
	return &Server{
		nextSvr: nextSvr,
	}
}

type Server struct {
	nextSvr auth.UserServer
}

// Initialize 初始化
func (svr *Server) Initialize(authOpt *auth.Config, storage store.Store, cacheMgr cachetypes.CacheManager) error {
	return svr.nextSvr.Initialize(authOpt, storage, cacheMgr)
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
func (svr *Server) CheckCredential(authCtx *model.AcquireContext) error {
	return svr.nextSvr.CheckCredential(authCtx)
}

// GetUserHelper
func (svr *Server) GetUserHelper() auth.UserHelper {
	return svr.nextSvr.GetUserHelper()
}

// CreateUsers 批量创建用户
func (svr *Server) CreateUsers(ctx context.Context, users []*apisecurity.User) *apiservice.BatchWriteResponse {
	ctx, rsp := svr.verifyAuth(ctx, WriteOp, MustOwner)
	if rsp != nil {
		resp := api.NewAuthBatchWriteResponse(apimodel.Code_ExecuteSuccess)
		api.Collect(resp, rsp)
		return resp
	}
	return svr.nextSvr.CreateUsers(ctx, users)
}

// UpdateUser 更新用户信息
func (svr *Server) UpdateUser(ctx context.Context, user *apisecurity.User) *apiservice.Response {
	ctx, rsp := svr.verifyAuth(ctx, ReadOp, NotOwner)
	if rsp != nil {
		rsp.User = user
		return rsp
	}
	helper := svr.GetUserHelper()
	targetUser := helper.GetUserByID(ctx, user.GetId().GetValue())
	if !checkUserViewPermission(ctx, targetUser) {
		return api.NewAuthResponse(apimodel.Code_NotAllowedAccess)
	}
	return svr.nextSvr.UpdateUser(ctx, user)
}

// UpdateUserPassword 更新用户密码
func (svr *Server) UpdateUserPassword(ctx context.Context, req *apisecurity.ModifyUserPassword) *apiservice.Response {
	ctx, rsp := svr.verifyAuth(ctx, ReadOp, NotOwner)
	if rsp != nil {
		return rsp
	}
	helper := svr.GetUserHelper()
	targetUser := helper.GetUserByID(ctx, req.GetId().GetValue())
	if !checkUserViewPermission(ctx, targetUser) {
		return api.NewAuthResponse(apimodel.Code_NotAllowedAccess)
	}
	return svr.nextSvr.UpdateUserPassword(ctx, req)
}

// DeleteUsers 批量删除用户
func (svr *Server) DeleteUsers(ctx context.Context, users []*apisecurity.User) *apiservice.BatchWriteResponse {
	ctx, rsp := svr.verifyAuth(ctx, WriteOp, MustOwner)
	if rsp != nil {
		resp := api.NewAuthBatchWriteResponse(apimodel.Code_ExecuteSuccess)
		api.Collect(resp, rsp)
		return resp
	}
	return svr.nextSvr.DeleteUsers(ctx, users)
}

// GetUsers 查询用户列表
func (svr *Server) GetUsers(ctx context.Context, query map[string]string) *apiservice.BatchQueryResponse {
	ctx, rsp := svr.verifyAuth(ctx, ReadOp, NotOwner)
	if rsp != nil {
		return api.NewAuthBatchQueryResponseWithMsg(apimodel.Code(rsp.GetCode().Value), rsp.Info.Value)
	}
	query["hide_admin"] = strconv.FormatBool(true)
	// 如果不是超级管理员，查看数据有限制
	if authcommon.ParseUserRole(ctx) != model.AdminUserRole {
		// 设置 owner 参数，只能查看对应 owner 下的用户
		query["owner"] = utils.ParseOwnerID(ctx)
	}
	return svr.nextSvr.GetUsers(ctx, query)
}

// GetUserToken 获取用户的 token
func (svr *Server) GetUserToken(ctx context.Context, user *apisecurity.User) *apiservice.Response {
	ctx, rsp := svr.verifyAuth(ctx, ReadOp, NotOwner)
	if rsp != nil {
		return rsp
	}
	helper := svr.GetUserHelper()
	targetUser := helper.GetUser(ctx, user)
	if !checkUserViewPermission(ctx, targetUser) {
		return api.NewAuthResponse(apimodel.Code_NotAllowedAccess)
	}
	return svr.nextSvr.GetUserToken(ctx, user)
}

// UpdateUserToken 禁止用户的token使用
func (svr *Server) UpdateUserToken(ctx context.Context, user *apisecurity.User) *apiservice.Response {
	ctx, rsp := svr.verifyAuth(ctx, WriteOp, NotOwner)
	if rsp != nil {
		return rsp
	}
	helper := svr.GetUserHelper()
	targetUser := helper.GetUserByID(ctx, user.GetId().GetValue())
	if !checkUserViewPermission(ctx, targetUser) {
		return api.NewAuthResponse(apimodel.Code_NotAllowedAccess)
	}
	return svr.nextSvr.UpdateUserToken(ctx, user)
}

// ResetUserToken 重置用户的token
func (svr *Server) ResetUserToken(ctx context.Context, user *apisecurity.User) *apiservice.Response {
	ctx, rsp := svr.verifyAuth(ctx, WriteOp, NotOwner)
	if rsp != nil {
		return rsp
	}
	helper := svr.GetUserHelper()
	targetUser := helper.GetUserByID(ctx, user.GetId().GetValue())
	if !checkUserViewPermission(ctx, targetUser) {
		return api.NewAuthResponse(apimodel.Code_NotAllowedAccess)
	}
	return svr.nextSvr.ResetUserToken(ctx, user)
}

// CreateGroup 创建用户组
func (svr *Server) CreateGroup(ctx context.Context, group *apisecurity.UserGroup) *apiservice.Response {
	ctx, rsp := svr.verifyAuth(ctx, WriteOp, MustOwner)
	if rsp != nil {
		return rsp
	}
	return svr.nextSvr.CreateGroup(ctx, group)
}

// UpdateGroups 更新用户组
func (svr *Server) UpdateGroups(ctx context.Context, groups []*apisecurity.ModifyUserGroup) *apiservice.BatchWriteResponse {
	ctx, rsp := svr.verifyAuth(ctx, WriteOp, MustOwner)
	if rsp != nil {
		resp := api.NewAuthBatchWriteResponse(apimodel.Code_ExecuteSuccess)
		api.Collect(resp, rsp)
		return resp
	}

	resp := api.NewAuthBatchWriteResponse(apimodel.Code_ExecuteSuccess)
	for i := range groups {
		item := groups[i]
		rsp := svr.checkUpdateGroup(ctx, item)
		api.Collect(resp, rsp)
	}
	if !api.IsSuccess(resp) {
		return resp
	}

	return svr.nextSvr.UpdateGroups(ctx, groups)
}

// DeleteGroups 批量删除用户组
func (svr *Server) DeleteGroups(ctx context.Context, groups []*apisecurity.UserGroup) *apiservice.BatchWriteResponse {
	ctx, rsp := svr.verifyAuth(ctx, WriteOp, MustOwner)
	if rsp != nil {
		resp := api.NewAuthBatchWriteResponse(apimodel.Code_ExecuteSuccess)
		api.Collect(resp, rsp)
		return resp
	}
	resp := api.NewAuthBatchWriteResponse(apimodel.Code_ExecuteSuccess)
	for i := range groups {
		item := groups[i]
		if !svr.checkGroupViewAuth(ctx, item.GetId().GetValue()) {
			api.Collect(resp, api.NewAuthResponse(apimodel.Code_NotAllowedAccess))
		}
	}
	if !api.IsSuccess(resp) {
		return resp
	}
	return svr.nextSvr.DeleteGroups(ctx, groups)
}

// GetGroups 查询用户组列表（不带用户详细信息）
func (svr *Server) GetGroups(ctx context.Context, query map[string]string) *apiservice.BatchQueryResponse {
	ctx, rsp := svr.verifyAuth(ctx, WriteOp, MustOwner)
	if rsp != nil {
		resp := api.NewAuthBatchQueryResponse(apimodel.Code_ExecuteSuccess)
		api.QueryCollect(resp, rsp)
		return resp
	}

	delete(query, "owner")
	if authcommon.ParseUserRole(ctx) != model.AdminUserRole {
		// step 1: 设置 owner 信息，只能查看归属主帐户下的用户组
		query["owner"] = utils.ParseOwnerID(ctx)
		if authcommon.ParseUserRole(ctx) != model.OwnerUserRole {
			// step 2: 非主帐户，只能查看自己所在的用户组
			if _, ok := query["user_id"]; !ok {
				query["user_id"] = utils.ParseUserID(ctx)
			}
		}
	}

	return svr.nextSvr.GetGroups(ctx, query)
}

// GetGroup 根据用户组信息，查询该用户组下的用户相信
func (svr *Server) GetGroup(ctx context.Context, req *apisecurity.UserGroup) *apiservice.Response {
	ctx, rsp := svr.verifyAuth(ctx, ReadOp, NotOwner)
	if rsp != nil {
		return rsp
	}
	if !svr.checkGroupViewAuth(ctx, req.GetId().GetValue()) {
		return api.NewAuthResponse(apimodel.Code_NotAllowedAccess)
	}
	return svr.nextSvr.GetGroup(ctx, req)
}

// GetGroupToken 获取用户组的 token
func (svr *Server) GetGroupToken(ctx context.Context, group *apisecurity.UserGroup) *apiservice.Response {
	ctx, rsp := svr.verifyAuth(ctx, ReadOp, NotOwner)
	if rsp != nil {
		return rsp
	}
	if !svr.checkGroupViewAuth(ctx, group.GetId().GetValue()) {
		return api.NewAuthResponse(apimodel.Code_NotAllowedAccess)
	}
	return svr.nextSvr.GetGroupToken(ctx, group)
}

// UpdateGroupToken 取消用户组的 token 使用
func (svr *Server) UpdateGroupToken(ctx context.Context, group *apisecurity.UserGroup) *apiservice.Response {
	ctx, rsp := svr.verifyAuth(ctx, WriteOp, MustOwner)
	if rsp != nil {
		return rsp
	}
	saveGroup := svr.GetUserHelper().GetGroup(ctx, &apisecurity.UserGroup{
		Id: wrapperspb.String(group.GetId().GetValue()),
	})
	if saveGroup == nil {
		return api.NewAuthResponse(apimodel.Code_NotFoundUserGroup)
	}
	if authcommon.ParseUserRole(ctx) != model.AdminUserRole {
		if saveGroup.GetOwner().GetValue() != utils.ParseUserID(ctx) {
			return api.NewAuthResponse(apimodel.Code_NotAllowedAccess)
		}
	}
	return svr.nextSvr.UpdateGroupToken(ctx, group)
}

// ResetGroupToken 重置用户组的 token
func (svr *Server) ResetGroupToken(ctx context.Context, group *apisecurity.UserGroup) *apiservice.Response {
	ctx, rsp := svr.verifyAuth(ctx, WriteOp, MustOwner)
	if rsp != nil {
		return rsp
	}
	saveGroup := svr.GetUserHelper().GetGroup(ctx, &apisecurity.UserGroup{
		Id: wrapperspb.String(group.GetId().GetValue()),
	})
	if saveGroup == nil {
		return api.NewAuthResponse(apimodel.Code_NotFoundUserGroup)
	}
	if authcommon.ParseUserRole(ctx) != model.AdminUserRole {
		if saveGroup.GetOwner().GetValue() != utils.ParseUserID(ctx) {
			return api.NewAuthResponse(apimodel.Code_NotAllowedAccess)
		}
	}
	return svr.nextSvr.ResetGroupToken(ctx, group)
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

	authCtx := model.NewAcquireContext(
		model.WithRequestContext(ctx),
		model.WithModule(model.AuthModule),
	)

	// case 1. 如果 error 不是 token 被禁止的 error，直接返回
	// case 2. 如果 error 是 token 被禁止，按下面情况判断
	// 		i. 如果当前只是一个数据的读取操作，则放通
	// 		ii. 如果当前是一个数据的写操作，则只能允许处于正常的 token 进行操作
	if err := svr.CheckCredential(authCtx); err != nil {
		log.Error("[Auth][Server] verify auth token", utils.ZapRequestID(reqId),
			zap.Error(err))
		return nil, api.NewAuthResponse(apimodel.Code_AuthTokenForbidden)
	}

	attachVal, exist := authCtx.GetAttachment(model.TokenDetailInfoKey)
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

// checkUserViewPermission 检查是否可以操作该用户
// Case 1: 如果是自己操作自己，通过
// Case 2: 如果是主账户操作自己的子账户，通过
// Case 3: 如果是超级账户，通过
func checkUserViewPermission(ctx context.Context, user *apisecurity.User) bool {
	role := authcommon.ParseUserRole(ctx)
	if role == model.AdminUserRole {
		log.Debug("check user view permission", utils.RequestID(ctx), zap.Bool("admin", true))
		return true
	}

	userId := utils.ParseUserID(ctx)
	if user.GetId().GetValue() == userId {
		return true
	}

	if user.GetOwner().GetValue() == userId {
		log.Debug("check user view permission", utils.RequestID(ctx),
			zap.Any("user", user), zap.String("owner", user.GetOwner().GetValue()), zap.String("operator", userId))
		return true
	}

	return false
}

// checkUpdateGroup 检查用户组的更新请求
func (svr *Server) checkUpdateGroup(ctx context.Context, req *apisecurity.ModifyUserGroup) *apiservice.Response {
	userId := utils.ParseUserID(ctx)
	isOwner := utils.ParseIsOwner(ctx)
	saveGroup := svr.GetUserHelper().GetGroup(ctx, &apisecurity.UserGroup{
		Id: wrapperspb.String(req.GetId().GetValue()),
	})
	if saveGroup == nil {
		return api.NewAuthResponse(apimodel.Code_NotFoundUserGroup)
	}

	// 满足以下情况才可以进行操作
	// 1.管理员
	// 2.自己在这个用户组里面
	// 3.自己是这个用户组的owner角色
	if authcommon.ParseUserRole(ctx) != model.AdminUserRole {
		inGroup := false
		for i := range saveGroup.GetRelation().GetUsers() {
			if userId == saveGroup.GetRelation().GetUsers()[i].GetId().GetValue() {
				inGroup = true
				break
			}
		}
		if !inGroup && saveGroup.GetOwner().GetValue() != userId {
			return api.NewAuthResponse(apimodel.Code_NotAllowedAccess)
		}
		// 如果当前用户只是在这个组里面，但不是该用户组的owner，那只能添加用户，不能删除用户
		if inGroup && !isOwner && len(req.GetRemoveRelations().GetUsers()) != 0 {
			return api.NewAuthResponseWithMsg(
				apimodel.Code_NotAllowedAccess, "only main account can remove user from usergroup")
		}
	}
	return nil
}

func (svr *Server) checkGroupViewAuth(ctx context.Context, id string) bool {
	saveGroup := svr.GetUserHelper().GetGroup(ctx, &apisecurity.UserGroup{
		Id: wrapperspb.String(id),
	})
	if saveGroup == nil {
		return false
	}

	if authcommon.ParseUserRole(ctx) != model.AdminUserRole {
		userID := utils.ParseUserID(ctx)
		inGroup := svr.GetUserHelper().CheckUserInGroup(ctx, &apisecurity.UserGroup{
			Id: wrapperspb.String(id),
		}, &apisecurity.User{
			Id: wrapperspb.String(userID),
		})
		isGroupOwner := saveGroup.GetOwner().GetValue() == userID
		if !isGroupOwner && !inGroup {
			log.Error("can't see group info", zap.String("user", userID),
				zap.String("group", id), zap.Bool("group-owner", isGroupOwner),
				zap.Bool("in-group", inGroup))
			return false
		}
	}
	return true
}
