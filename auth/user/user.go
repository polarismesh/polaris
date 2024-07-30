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

package defaultuser

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gogo/protobuf/jsonpb"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	apisecurity "github.com/polarismesh/specification/source/go/api/v1/security"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

	"github.com/polarismesh/polaris/auth"
	cachetypes "github.com/polarismesh/polaris/cache/api"
	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/model"
	authcommon "github.com/polarismesh/polaris/common/model/auth"
	commonstore "github.com/polarismesh/polaris/common/store"
	commontime "github.com/polarismesh/polaris/common/time"
	"github.com/polarismesh/polaris/common/utils"
)

type (
	// User2Api convert user to api.User
	User2Api func(user *authcommon.User) *apisecurity.User
)

// CreateUsers 批量创建用户
func (svr *Server) CreateUsers(ctx context.Context, req []*apisecurity.User) *apiservice.BatchWriteResponse {
	batchResp := api.NewAuthBatchWriteResponse(apimodel.Code_ExecuteSuccess)

	for i := range req {
		resp := svr.CreateUser(ctx, req[i])
		api.Collect(batchResp, resp)
	}

	return batchResp
}

// CreateUser 创建用户
func (svr *Server) CreateUser(ctx context.Context, req *apisecurity.User) *apiservice.Response {
	ownerID := utils.ParseOwnerID(ctx)
	req.Owner = utils.NewStringValue(ownerID)

	if checkErrResp := checkCreateUser(req); checkErrResp != nil {
		return checkErrResp
	}

	// 如果创建的目标账户类型是非子账户，则 ownerId 需要设置为 “”
	if convertCreateUserRole(authcommon.ParseUserRole(ctx)) != authcommon.SubAccountUserRole {
		ownerID = ""
	}

	if ownerID != "" {
		owner, err := svr.storage.GetUser(ownerID)
		if err != nil {
			log.Error("[Auth][User] get owner user", utils.RequestID(ctx), zap.Error(err), zap.String("owner", ownerID))
			return api.NewUserResponse(commonstore.StoreCode2APICode(err), req)
		}

		if owner.Name == req.Name.GetValue() {
			log.Error("[Auth][User] create user name is equal owner", utils.RequestID(ctx),
				zap.Error(err), zap.String("name", req.GetName().GetValue()))
			return api.NewUserResponse(apimodel.Code_UserExisted, req)
		}
	}

	// 只有通过 owner + username 才能唯一确定一个用户
	user, err := svr.storage.GetUserByName(req.Name.GetValue(), ownerID)
	if err != nil {
		log.Error("[Auth][User] get user by name and owner", utils.RequestID(ctx),
			zap.Error(err), zap.String("owner", ownerID), zap.String("name", req.GetName().GetValue()))
		return api.NewUserResponse(commonstore.StoreCode2APICode(err), req)
	}
	if user != nil {
		return api.NewUserResponse(apimodel.Code_UserExisted, req)
	}

	return svr.createUser(ctx, req)
}

func (svr *Server) createUser(ctx context.Context, req *apisecurity.User) *apiservice.Response {
	data, err := svr.createUserModel(req, authcommon.ParseUserRole(ctx))
	if err != nil {
		log.Error("[Auth][User] create user model", utils.RequestID(ctx), zap.Error(err))
		return api.NewAuthResponse(apimodel.Code_ExecuteException)
	}

	tx, err := svr.storage.StartTx()
	if err != nil {
		log.Error("[Auth][User] create user begion storage tx", utils.RequestID(ctx), zap.Error(err))
		return api.NewAuthResponse(apimodel.Code_ExecuteException)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if err := svr.storage.AddUser(tx, data); err != nil {
		log.Error("[Auth][User] add user into store", utils.RequestID(ctx), zap.Error(err))
		return api.NewAuthResponse(commonstore.StoreCode2APICode(err))
	}

	if err := svr.policySvr.PolicyHelper().CreatePrincipal(ctx, tx, authcommon.Principal{
		PrincipalID:   data.ID,
		PrincipalType: authcommon.PrincipalUser,
		Owner:         data.Owner,
		Name:          data.Name,
	}); err != nil {
		log.Error("[Auth][User] add user default policy rule", utils.RequestID(ctx), zap.Error(err))
		return api.NewAuthResponse(commonstore.StoreCode2APICode(err))
	}

	if err := tx.Commit(); err != nil {
		log.Error("[Auth][User] create user commit storage tx", utils.RequestID(ctx), zap.Error(err))
		return api.NewAuthResponse(apimodel.Code_ExecuteException)
	}

	log.Info("[Auth][User] create user", utils.RequestID(ctx), zap.String("name", req.GetName().GetValue()))
	svr.RecordHistory(userRecordEntry(ctx, req, data, model.OCreate))

	// 去除 owner 信息
	req.Owner = utils.NewStringValue("")
	req.Id = utils.NewStringValue(data.ID)
	return api.NewUserResponse(apimodel.Code_ExecuteSuccess, req)
}

// UpdateUser 更新用户信息，仅能修改 comment 以及账户密码
func (svr *Server) UpdateUser(ctx context.Context, req *apisecurity.User) *apiservice.Response {
	if checkErrResp := checkUpdateUser(req); checkErrResp != nil {
		return checkErrResp
	}

	user, err := svr.storage.GetUser(req.Id.GetValue())
	if err != nil {
		log.Error("[Auth][User] get user", utils.RequestID(ctx), zap.String("user-id", req.GetId().GetValue()), zap.Error(err))
		return api.NewUserResponse(commonstore.StoreCode2APICode(err), req)
	}
	if user == nil {
		return api.NewUserResponse(apimodel.Code_NotFoundUser, req)
	}

	data, needUpdate, err := updateUserAttribute(user, req)
	if err != nil {
		return api.NewAuthResponseWithMsg(apimodel.Code_ExecuteException, err.Error())
	}

	if !needUpdate {
		log.Info("[Auth][User] update user data no change, no need update", utils.RequestID(ctx), zap.String("user", req.String()))
		return api.NewUserResponse(apimodel.Code_NoNeedUpdate, req)
	}

	if err := svr.storage.UpdateUser(data); err != nil {
		log.Error("[Auth][User] update user from store", utils.RequestID(ctx), zap.Error(err))
		return api.NewAuthResponseWithMsg(commonstore.StoreCode2APICode(err), err.Error())
	}

	log.Info("[Auth][User] update user", utils.RequestID(ctx), zap.String("name", req.GetName().GetValue()))
	svr.RecordHistory(userRecordEntry(ctx, req, user, model.OUpdate))

	return api.NewUserResponse(apimodel.Code_ExecuteSuccess, req)
}

// UpdateUserPassword 更新用户密码信息
func (svr *Server) UpdateUserPassword(ctx context.Context, req *apisecurity.ModifyUserPassword) *apiservice.Response {
	user, err := svr.storage.GetUser(req.Id.GetValue())
	if err != nil {
		log.Error("[Auth][User] get user", utils.RequestID(ctx),
			zap.String("user-id", req.Id.GetValue()), zap.Error(err))
		return api.NewAuthResponse(commonstore.StoreCode2APICode(err))
	}
	if user == nil {
		return api.NewAuthResponse(apimodel.Code_NotFoundUser)
	}

	ignoreOrigin := authcommon.ParseUserRole(ctx) == authcommon.AdminUserRole ||
		authcommon.ParseUserRole(ctx) == authcommon.OwnerUserRole
	data, needUpdate, err := updateUserPasswordAttribute(ignoreOrigin, user, req)
	if err != nil {
		log.Error("[Auth][User] compute user update attribute", zap.Error(err),
			zap.String("user", req.GetId().GetValue()))
		return api.NewAuthResponseWithMsg(apimodel.Code_ExecuteException, err.Error())
	}

	if !needUpdate {
		log.Info("[Auth][User] update user password no change, no need update",
			utils.RequestID(ctx), zap.String("user", req.GetId().GetValue()))
		return api.NewAuthResponse(apimodel.Code_NoNeedUpdate)
	}

	if err := svr.storage.UpdateUser(data); err != nil {
		log.Error("[Auth][User] update user from store", utils.RequestID(ctx),
			zap.Error(err))
		return api.NewAuthResponse(commonstore.StoreCode2APICode(err))
	}

	log.Info("[Auth][User] update user", utils.RequestID(ctx), zap.String("user-id", req.Id.GetValue()))

	return api.NewAuthResponse(apimodel.Code_ExecuteSuccess)
}

// DeleteUsers 批量删除用户
func (svr *Server) DeleteUsers(ctx context.Context, reqs []*apisecurity.User) *apiservice.BatchWriteResponse {
	resp := api.NewAuthBatchWriteResponse(apimodel.Code_ExecuteSuccess)

	for index := range reqs {
		ret := svr.DeleteUser(ctx, reqs[index])
		api.Collect(resp, ret)
	}

	return resp
}

// DeleteUser 删除用户
// Case 1. 删除主账户，主账户不能自己删除自己
// Case 2. 删除主账户，如果主账户下还存在子账户，必须先删除子账户，才能删除主账户
// Case 3. 主账户角色下，只能删除自己创建的子账户
// Case 4. 超级账户角色下，可以删除任意账户
func (svr *Server) DeleteUser(ctx context.Context, req *apisecurity.User) *apiservice.Response {
	user, err := svr.storage.GetUser(req.Id.GetValue())
	if err != nil {
		log.Error("[Auth][User] get user from store", utils.RequestID(ctx), zap.Error(err))
		return api.NewUserResponse(commonstore.StoreCode2APICode(err), req)
	}
	if user == nil {
		return api.NewUserResponse(apimodel.Code_ExecuteSuccess, req)
	}

	if user.ID == utils.ParseOwnerID(ctx) {
		log.Error("[Auth][User] delete user forbidden, can't delete when self is owner",
			utils.RequestID(ctx), zap.String("name", req.Name.GetValue()))
		return api.NewUserResponse(apimodel.Code_NotAllowedAccess, req)
	}
	if user.Type == authcommon.OwnerUserRole {
		count, err := svr.storage.GetSubCount(user)
		if err != nil {
			log.Error("[Auth][User] get user sub-account", zap.String("owner", user.ID),
				utils.RequestID(ctx), zap.Error(err))
			return api.NewUserResponse(commonstore.StoreCode2APICode(err), req)
		}
		if count != 0 {
			log.Error("[Auth][User] delete user but some sub-account existed", zap.String("owner", user.ID))
			return api.NewUserResponse(apimodel.Code_SubAccountExisted, req)
		}
	}
	tx, err := svr.storage.StartTx()
	if err != nil {
		log.Error("[Auth][User] delete user begion storage tx", utils.RequestID(ctx), zap.Error(err))
		return api.NewAuthResponse(apimodel.Code_ExecuteException)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if err := svr.storage.DeleteUser(tx, user); err != nil {
		log.Error("[Auth][User] delete user from store", utils.RequestID(ctx), zap.Error(err))
		return api.NewAuthResponse(commonstore.StoreCode2APICode(err))
	}
	if err := svr.policySvr.PolicyHelper().CleanPrincipal(ctx, tx, authcommon.Principal{
		PrincipalID:   user.ID,
		PrincipalType: authcommon.PrincipalUser,
		Owner:         user.Owner,
	}); err != nil {
		log.Error("[Auth][User] delete user from policy server", utils.RequestID(ctx), zap.Error(err))
		return api.NewAuthResponse(commonstore.StoreCode2APICode(err))
	}
	if err := tx.Commit(); err != nil {
		log.Error("[Auth][User] delete user commit storage tx", utils.RequestID(ctx), zap.Error(err))
		return api.NewAuthResponse(apimodel.Code_ExecuteException)
	}

	log.Info("[Auth][User] delete user", utils.RequestID(ctx), zap.String("name", req.Name.GetValue()))
	svr.RecordHistory(userRecordEntry(ctx, req, user, model.ODelete))

	return api.NewUserResponse(apimodel.Code_ExecuteSuccess, req)
}

// GetUsers 查询用户列表
func (svr *Server) GetUsers(ctx context.Context, filters map[string]string) *apiservice.BatchQueryResponse {
	offset, limit, _ := utils.ParseOffsetAndLimit(filters)

	total, users, err := svr.cacheMgr.User().QueryUsers(ctx, cachetypes.UserSearchArgs{
		Filters: filters,
		Offset:  offset,
		Limit:   limit,
	})
	if err != nil {
		log.Error("[Auth][User] get user from store", utils.RequestID(ctx), zap.Any("req", filters),
			zap.Error(err))
		return api.NewAuthBatchQueryResponse(commonstore.StoreCode2APICode(err))
	}

	resp := api.NewAuthBatchQueryResponse(apimodel.Code_ExecuteSuccess)
	resp.Amount = utils.NewUInt32Value(total)
	resp.Size = utils.NewUInt32Value(uint32(len(users)))
	resp.Users = enhancedUsers2Api(users, user2Api)
	return resp
}

// GetUserToken 获取用户 token
func (svr *Server) GetUserToken(ctx context.Context, req *apisecurity.User) *apiservice.Response {
	var user *authcommon.User
	if req.GetId().GetValue() != "" {
		user = svr.cacheMgr.User().GetUserByID(req.GetId().GetValue())
	} else if req.GetName().GetValue() != "" {
		ownerName := req.GetOwner().GetValue()
		ownerID := utils.ParseOwnerID(ctx)
		if ownerName == "" {
			owner := svr.cacheMgr.User().GetUserByID(ownerID)
			if owner == nil {
				log.Error("[Auth][User] get user's owner not found",
					zap.String("name", req.GetName().GetValue()), zap.String("owner", ownerID))
				return api.NewAuthResponse(apimodel.Code_NotFoundUser)
			}
			ownerName = owner.Name
		}
		user = svr.cacheMgr.User().GetUserByName(req.GetName().GetValue(), ownerName)
	} else {
		return api.NewAuthResponse(apimodel.Code_InvalidParameter)
	}

	if user == nil {
		return api.NewUserResponse(apimodel.Code_NotFoundUser, req)
	}

	out := &apisecurity.User{
		Id:          utils.NewStringValue(user.ID),
		Name:        utils.NewStringValue(user.Name),
		AuthToken:   utils.NewStringValue(user.Token),
		TokenEnable: utils.NewBoolValue(user.TokenEnable),
	}

	return api.NewUserResponse(apimodel.Code_ExecuteSuccess, out)
}

// EnableUserToken 更新用户 token
func (svr *Server) EnableUserToken(ctx context.Context, req *apisecurity.User) *apiservice.Response {
	if checkErrResp := checkUpdateUser(req); checkErrResp != nil {
		return checkErrResp
	}

	user, err := svr.storage.GetUser(req.GetId().GetValue())
	if err != nil {
		log.Error("[Auth][User] get user from store", utils.RequestID(ctx), zap.Error(err))
		return api.NewUserResponse(commonstore.StoreCode2APICode(err), req)
	}
	if user == nil {
		return api.NewUserResponse(apimodel.Code_NotFoundUser, req)
	}

	user.TokenEnable = req.GetTokenEnable().GetValue()

	if err := svr.storage.UpdateUser(user); err != nil {
		log.Error("[Auth][User] update user token into store", utils.RequestID(ctx), zap.Error(err))
		return api.NewAuthResponseWithMsg(commonstore.StoreCode2APICode(err), err.Error())
	}

	log.Info("[Auth][User] update user token", utils.RequestID(ctx),
		zap.String("id", req.GetId().GetValue()), zap.Bool("enable", req.GetTokenEnable().GetValue()))
	svr.RecordHistory(userRecordEntry(ctx, req, user, model.OUpdateToken))

	return api.NewUserResponse(apimodel.Code_ExecuteSuccess, req)
}

// ResetUserToken 重置用户 token
func (svr *Server) ResetUserToken(ctx context.Context, req *apisecurity.User) *apiservice.Response {
	if checkErrResp := checkUpdateUser(req); checkErrResp != nil {
		return checkErrResp
	}

	user, err := svr.storage.GetUser(req.Id.GetValue())
	if err != nil {
		log.Error("[Auth][User] get user from store", utils.RequestID(ctx), zap.Error(err))
		return api.NewUserResponse(commonstore.StoreCode2APICode(err), req)
	}
	if user == nil {
		return api.NewUserResponse(apimodel.Code_NotFoundUser, req)
	}

	newToken, err := createUserToken(user.ID, svr.authOpt.Salt)
	if err != nil {
		log.Error("[Auth][User] update user token", utils.RequestID(ctx), zap.Error(err))
		return api.NewUserResponse(apimodel.Code_ExecuteException, req)
	}

	user.Token = newToken

	if err := svr.storage.UpdateUser(user); err != nil {
		log.Error("[Auth][User] update user token into store", utils.RequestID(ctx), zap.Error(err))
		return api.NewUserResponse(commonstore.StoreCode2APICode(err), req)
	}

	log.Info("[Auth][User] reset user token", utils.RequestID(ctx), zap.String("id", req.GetId().GetValue()))
	svr.RecordHistory(userRecordEntry(ctx, req, user, model.OUpdateToken))

	req.AuthToken = utils.NewStringValue(user.Token)

	return api.NewUserResponse(apimodel.Code_ExecuteSuccess, req)
}

// VerifyCredential 对 token 进行检查验证，并将 verify 过程中解析出的数据注入到 model.AcquireContext 中
// step 1. 首先对 token 进行解析，获取相关的数据信息，注入到整个的 AcquireContext 中
// step 2. 最后对 token 进行一些验证步骤的执行
// step 3. 兜底措施：如果开启了鉴权的非严格模式，则根据错误的类型，判断是否转为匿名用户进行访问
//   - 如果是访问权限控制相关模块（用户、用户组、权限策略），不得转为匿名用户
func (svr *Server) CheckCredential(authCtx *authcommon.AcquireContext) error {
	checkErr := func() error {
		authToken := utils.ParseAuthToken(authCtx.GetRequestContext())
		operator, err := svr.decodeToken(authToken)
		if err != nil {
			log.Error("[Auth][Checker] decode token", utils.RequestID(authCtx.GetRequestContext()), zap.Error(err))
			return authcommon.ErrorTokenInvalid
		}

		ownerId, isOwner, err := svr.checkToken(&operator)
		if err != nil {
			log.Error("[Auth][Checker] check token", utils.RequestID(authCtx.GetRequestContext()), zap.Error(err))
			return err
		}

		operator.OwnerID = ownerId
		ctx := authCtx.GetRequestContext()
		ctx = context.WithValue(ctx, utils.ContextIsOwnerKey, isOwner)
		ctx = context.WithValue(ctx, utils.ContextUserIDKey, operator.OperatorID)
		ctx = context.WithValue(ctx, utils.ContextOwnerIDKey, ownerId)
		authCtx.SetRequestContext(ctx)
		svr.parseOperatorInfo(operator, authCtx)
		if operator.Disable {
			log.Error("[Auth][Checker] token has been set disable", utils.RequestID(authCtx.GetRequestContext()),
				zap.String("operator", operator.String()))
			return authcommon.ErrorTokenDisabled
		}
		return nil
	}()

	if checkErr != nil {
		if !canDowngradeAnonymous(authCtx, checkErr) {
			return checkErr
		}
		log.Warn("[Auth][Checker] parse operator info, downgrade to anonymous", utils.RequestID(authCtx.GetRequestContext()),
			zap.Error(checkErr))
		// 操作者信息解析失败，降级为匿名用户
		authCtx.SetAttachment(authcommon.TokenDetailInfoKey, auth.NewAnonymous())
	}
	return nil
}

func (svr *Server) parseOperatorInfo(operator auth.OperatorInfo, authCtx *authcommon.AcquireContext) {
	ctx := authCtx.GetRequestContext()
	if operator.IsUserToken {
		user := svr.cacheMgr.User().GetUserByID(operator.OperatorID)
		if user != nil {
			operator.Role = user.Type
			ctx = context.WithValue(ctx, utils.ContextOperator, user.Name)
			ctx = context.WithValue(ctx, utils.ContextUserNameKey, user.Name)
			ctx = context.WithValue(ctx, utils.ContextUserRoleIDKey, user.Type)
		}
	} else {
		userGroup := svr.cacheMgr.User().GetGroup(operator.OperatorID)
		if userGroup != nil {
			ctx = context.WithValue(ctx, utils.ContextOperator, userGroup.Name)
			ctx = context.WithValue(ctx, utils.ContextUserNameKey, userGroup.Name)
		}
	}

	authCtx.SetAttachment(authcommon.PrincipalKey, authcommon.Principal{
		PrincipalID: operator.OperatorID,
		PrincipalType: func() authcommon.PrincipalType {
			if operator.IsUserToken {
				return authcommon.PrincipalUser
			}
			return authcommon.PrincipalGroup
		}(),
	})
	authCtx.SetAttachment(authcommon.OperatorRoleKey, operator.Role)
	authCtx.SetAttachment(authcommon.OperatorIDKey, operator.OperatorID)
	authCtx.SetAttachment(authcommon.OperatorOwnerKey, operator)
	authCtx.SetAttachment(authcommon.TokenDetailInfoKey, operator)

	authCtx.SetRequestContext(ctx)
}

func canDowngradeAnonymous(authCtx *authcommon.AcquireContext, err error) bool {
	if authCtx.GetModule() == authcommon.AuthModule || authCtx.GetModule() == authcommon.MaintainModule {
		return false
	}
	if !authCtx.IsAllowAnonymous() {
		return false
	}
	if errors.Is(err, authcommon.ErrorTokenInvalid) {
		return true
	}
	if errors.Is(err, authcommon.ErrorTokenNotExist) {
		return true
	}
	return false
}

// user 数组转为[]*apisecurity.User
func enhancedUsers2Api(users []*authcommon.User, handler User2Api) []*apisecurity.User {
	out := make([]*apisecurity.User, 0, len(users))
	for _, entry := range users {
		outUser := handler(entry)
		out = append(out, outUser)
	}

	return out
}

// model.Service 转为 api.Service
func user2Api(user *authcommon.User) *apisecurity.User {
	if user == nil {
		return nil
	}

	// note: 不包括token，token比较特殊
	out := &apisecurity.User{
		Id:          utils.NewStringValue(user.ID),
		Name:        utils.NewStringValue(user.Name),
		Source:      utils.NewStringValue(user.Source),
		Owner:       utils.NewStringValue(user.Owner),
		TokenEnable: utils.NewBoolValue(user.TokenEnable),
		Comment:     utils.NewStringValue(user.Comment),
		Ctime:       utils.NewStringValue(commontime.Time2String(user.CreateTime)),
		Mtime:       utils.NewStringValue(commontime.Time2String(user.ModifyTime)),
		UserType:    utils.NewStringValue(authcommon.UserRoleNames[user.Type]),
	}

	return out
}

// 生成用户的记录entry
func userRecordEntry(ctx context.Context, req *apisecurity.User, md *authcommon.User,
	operationType model.OperationType) *model.RecordEntry {

	marshaler := jsonpb.Marshaler{}
	detail, _ := marshaler.MarshalToString(req)

	entry := &model.RecordEntry{
		ResourceType:  model.RUser,
		ResourceName:  fmt.Sprintf("%s(%s)", md.Name, md.ID),
		OperationType: operationType,
		Operator:      utils.ParseOperator(ctx),
		Detail:        detail,
		HappenTime:    time.Now(),
	}

	return entry
}

// checkCreateUser 检查创建用户的请求
func checkCreateUser(req *apisecurity.User) *apiservice.Response {
	if req == nil {
		return api.NewUserResponse(apimodel.Code_EmptyRequest, req)
	}

	if err := CheckName(req.Name); err != nil {
		return api.NewUserResponse(apimodel.Code_InvalidUserName, req)
	}

	if err := CheckPassword(req.Password); err != nil {
		return api.NewUserResponse(apimodel.Code_InvalidUserPassword, req)
	}

	if err := CheckOwner(req.Owner); err != nil {
		return api.NewUserResponse(apimodel.Code_InvalidUserOwners, req)
	}
	return nil
}

// checkUpdateUser 检查用户更新请求
func checkUpdateUser(req *apisecurity.User) *apiservice.Response {
	if req == nil {
		return api.NewUserResponse(apimodel.Code_EmptyRequest, req)
	}

	// 如果本次请求需要修改密码的话
	if req.GetPassword() != nil {
		if err := CheckPassword(req.Password); err != nil {
			return api.NewUserResponseWithMsg(apimodel.Code_InvalidUserPassword, err.Error(), req)
		}
	}

	if req.GetId() == nil || req.GetId().GetValue() == "" {
		return api.NewUserResponse(apimodel.Code_BadRequest, req)
	}
	return nil
}

// updateUserAttribute 更新用户属性
func updateUserAttribute(old *authcommon.User, newUser *apisecurity.User) (*authcommon.User, bool, error) {
	var needUpdate = true

	if newUser.Comment != nil && old.Comment != newUser.Comment.GetValue() {
		old.Comment = newUser.Comment.GetValue()
		needUpdate = true
	}
	return old, needUpdate, nil
}

// updateUserAttribute 更新用户密码信息，如果用户的密码被更新
func updateUserPasswordAttribute(
	isAdmin bool, user *authcommon.User, req *apisecurity.ModifyUserPassword) (*authcommon.User, bool, error) {
	needUpdate := false

	if err := CheckPassword(req.NewPassword); err != nil {
		return nil, false, err
	}

	if !isAdmin {
		if req.GetOldPassword().GetValue() == "" {
			return nil, false, errors.New("original password is empty")
		}

		err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.GetOldPassword().GetValue()))
		if err != nil {
			return nil, false, errors.New("original password match failed")
		}
	}

	if req.GetNewPassword().GetValue() != "" {
		pwd, err := bcrypt.GenerateFromPassword([]byte(req.GetNewPassword().GetValue()), bcrypt.DefaultCost)
		if err != nil {
			return nil, false, err
		}
		needUpdate = true
		user.Password = string(pwd)
	}
	return user, needUpdate, nil
}

// createUserModel 创建用户模型
func (svr *Server) createUserModel(req *apisecurity.User, role authcommon.UserRoleType) (*authcommon.User, error) {
	pwd, err := bcrypt.GenerateFromPassword([]byte(req.GetPassword().GetValue()), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	id := utils.NewUUID()
	if req.GetId().GetValue() != "" {
		id = req.GetId().GetValue()
	}

	user := &authcommon.User{
		ID:          id,
		Name:        req.GetName().GetValue(),
		Password:    string(pwd),
		Owner:       req.GetOwner().GetValue(),
		Source:      req.GetSource().GetValue(),
		Valid:       true,
		Type:        convertCreateUserRole(role),
		Comment:     req.GetComment().GetValue(),
		CreateTime:  time.Now(),
		ModifyTime:  time.Now(),
		TokenEnable: true,
	}

	// 如果不是子账户的话，owner 就是自己
	if user.Type != authcommon.SubAccountUserRole {
		user.Owner = ""
	}

	newToken, err := createUserToken(user.ID, svr.authOpt.Salt)
	if err != nil {
		return nil, err
	}

	user.Token = newToken

	return user, nil
}

// convertCreateUserRole 转换为创建的目标用户的用户角色类型
func convertCreateUserRole(role authcommon.UserRoleType) authcommon.UserRoleType {
	if role == authcommon.AdminUserRole {
		return authcommon.OwnerUserRole
	}

	if role == authcommon.OwnerUserRole {
		return authcommon.SubAccountUserRole
	}

	return authcommon.SubAccountUserRole
}
