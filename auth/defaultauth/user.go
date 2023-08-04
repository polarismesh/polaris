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
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/gogo/protobuf/jsonpb"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	apisecurity "github.com/polarismesh/specification/source/go/api/v1/security"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/model"
	authcommon "github.com/polarismesh/polaris/common/model/auth"
	commonstore "github.com/polarismesh/polaris/common/store"
	commontime "github.com/polarismesh/polaris/common/time"
	"github.com/polarismesh/polaris/common/utils"
)

type (
	// User2Api convert user to api.User
	User2Api func(user *model.User) *apisecurity.User
)

var (

	// UserFilterAttributes 查询用户所能允许的参数查询列表
	UserFilterAttributes = map[string]bool{
		"id":         true,
		"name":       true,
		"owner":      true,
		"source":     true,
		"offset":     true,
		"group_id":   true,
		"limit":      true,
		"hide_admin": true,
	}
)

// CreateUsers 批量创建用户
func (svr *server) CreateUsers(ctx context.Context, req []*apisecurity.User) *apiservice.BatchWriteResponse {
	batchResp := api.NewAuthBatchWriteResponse(apimodel.Code_ExecuteSuccess)

	for i := range req {
		resp := svr.CreateUser(ctx, req[i])
		api.Collect(batchResp, resp)
	}

	return batchResp
}

// CreateUser 创建用户
func (svr *server) CreateUser(ctx context.Context, req *apisecurity.User) *apiservice.Response {
	requestID := utils.ParseRequestID(ctx)
	ownerID := utils.ParseOwnerID(ctx)
	req.Owner = utils.NewStringValue(ownerID)

	if checkErrResp := checkCreateUser(req); checkErrResp != nil {
		return checkErrResp
	}

	// 如果创建的目标账户类型是非子账户，则 ownerId 需要设置为 “”
	if convertCreateUserRole(authcommon.ParseUserRole(ctx)) != model.SubAccountUserRole {
		ownerID = ""
	}

	if ownerID != "" {
		owner, err := svr.storage.GetUser(ownerID)
		if err != nil {
			log.Error("[Auth][User] get owner user", utils.ZapRequestID(requestID), zap.Error(err),
				zap.String("owner", ownerID))
			return api.NewUserResponse(commonstore.StoreCode2APICode(err), req)
		}

		if owner.Name == req.Name.GetValue() {
			log.Error("[Auth][User] create user name is equal owner", utils.ZapRequestID(requestID),
				zap.Error(err), zap.String("name", req.GetName().GetValue()))
			return api.NewUserResponse(apimodel.Code_UserExisted, req)
		}
	}

	// 只有通过 owner + username 才能唯一确定一个用户
	user, err := svr.storage.GetUserByName(req.Name.GetValue(), ownerID)
	if err != nil {
		log.Error("[Auth][User] get user by name and owner", utils.ZapRequestID(requestID),
			zap.Error(err), zap.String("owner", ownerID), zap.String("name", req.GetName().GetValue()))
		return api.NewUserResponse(commonstore.StoreCode2APICode(err), req)
	}
	if user != nil {
		return api.NewUserResponse(apimodel.Code_UserExisted, req)
	}

	return svr.createUser(ctx, req)
}

func (svr *server) createUser(ctx context.Context, req *apisecurity.User) *apiservice.Response {
	requestID := utils.ParseRequestID(ctx)

	data, err := createUserModel(req, authcommon.ParseUserRole(ctx))

	if err != nil {
		log.Error("[Auth][User] create user model", utils.ZapRequestID(requestID), zap.Error(err))
		return api.NewAuthResponse(apimodel.Code_ExecuteException)
	}

	if err := svr.storage.AddUser(data); err != nil {
		log.Error("[Auth][User] add user into store", utils.ZapRequestID(requestID), zap.Error(err))
		return api.NewAuthResponse(commonstore.StoreCode2APICode(err))
	}

	log.Info("[Auth][User] create user", utils.ZapRequestID(requestID),
		zap.String("name", req.Name.GetValue()))
	svr.RecordHistory(userRecordEntry(ctx, req, data, model.OCreate))

	// 去除 owner 信息
	req.Owner = utils.NewStringValue("")
	req.Id = utils.NewStringValue(data.ID)
	return api.NewUserResponse(apimodel.Code_ExecuteSuccess, req)
}

// UpdateUser 更新用户信息，仅能修改 comment 以及账户密码
func (svr *server) UpdateUser(ctx context.Context, req *apisecurity.User) *apiservice.Response {
	requestID := utils.ParseRequestID(ctx)

	if checkErrResp := checkUpdateUser(req); checkErrResp != nil {
		return checkErrResp
	}

	user, err := svr.storage.GetUser(req.Id.GetValue())
	if err != nil {
		log.Error("[Auth][User] get user", utils.ZapRequestID(requestID),
			zap.String("user-id", req.Id.GetValue()), zap.Error(err))
		return api.NewUserResponse(commonstore.StoreCode2APICode(err), req)
	}
	if user == nil {
		return api.NewUserResponse(apimodel.Code_NotFoundUser, req)
	}

	if !checkUserViewPermission(ctx, user) {
		return api.NewAuthResponse(apimodel.Code_NotAllowedAccess)
	}

	data, needUpdate, err := updateUserAttribute(user, req)
	if err != nil {
		return api.NewAuthResponseWithMsg(apimodel.Code_ExecuteException, err.Error())
	}

	if !needUpdate {
		log.Info("[Auth][User] update user data no change, no need update",
			utils.ZapRequestID(requestID), zap.String("user", req.String()))
		return api.NewUserResponse(apimodel.Code_NoNeedUpdate, req)
	}

	if err := svr.storage.UpdateUser(data); err != nil {
		log.Error("[Auth][User] update user from store", utils.ZapRequestID(requestID),
			zap.Error(err))
		return api.NewAuthResponseWithMsg(commonstore.StoreCode2APICode(err), err.Error())
	}

	log.Info("[Auth][User] update user", utils.ZapRequestID(requestID),
		zap.String("name", req.Name.GetValue()))
	svr.RecordHistory(userRecordEntry(ctx, req, user, model.OUpdate))

	return api.NewUserResponse(apimodel.Code_ExecuteSuccess, req)
}

// UpdateUserPassword 更新用户密码信息
func (svr *server) UpdateUserPassword(ctx context.Context, req *apisecurity.ModifyUserPassword) *apiservice.Response {
	requestID := utils.ParseRequestID(ctx)

	user, err := svr.storage.GetUser(req.Id.GetValue())
	if err != nil {
		log.Error("[Auth][User] get user", utils.ZapRequestID(requestID),
			zap.String("user-id", req.Id.GetValue()), zap.Error(err))
		return api.NewAuthResponse(commonstore.StoreCode2APICode(err))
	}
	if user == nil {
		return api.NewAuthResponse(apimodel.Code_NotFoundUser)
	}

	if !checkUserViewPermission(ctx, user) {
		return api.NewAuthResponse(apimodel.Code_NotAllowedAccess)
	}

	ignoreOrigin := authcommon.ParseUserRole(ctx) == model.AdminUserRole ||
		authcommon.ParseUserRole(ctx) == model.OwnerUserRole
	data, needUpdate, err := updateUserPasswordAttribute(ignoreOrigin, user, req)
	if err != nil {
		log.Error("[Auth][User] compute user update attribute", zap.Error(err),
			zap.String("user", req.GetId().GetValue()))
		return api.NewAuthResponseWithMsg(apimodel.Code_ExecuteException, err.Error())
	}

	if !needUpdate {
		log.Info("[Auth][User] update user password no change, no need update",
			utils.ZapRequestID(requestID), zap.String("user", req.GetId().GetValue()))
		return api.NewAuthResponse(apimodel.Code_NoNeedUpdate)
	}

	if err := svr.storage.UpdateUser(data); err != nil {
		log.Error("[Auth][User] update user from store", utils.ZapRequestID(requestID),
			zap.Error(err))
		return api.NewAuthResponse(commonstore.StoreCode2APICode(err))
	}

	log.Info("[Auth][User] update user", utils.ZapRequestID(requestID),
		zap.String("user-id", req.Id.GetValue()))

	return api.NewAuthResponse(apimodel.Code_ExecuteSuccess)
}

// DeleteUsers 批量删除用户
func (svr *server) DeleteUsers(ctx context.Context, reqs []*apisecurity.User) *apiservice.BatchWriteResponse {
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
func (svr *server) DeleteUser(ctx context.Context, req *apisecurity.User) *apiservice.Response {
	requestID := utils.ParseRequestID(ctx)
	user, err := svr.storage.GetUser(req.Id.GetValue())
	if err != nil {
		log.Error("[Auth][User] get user from store", utils.ZapRequestID(requestID), zap.Error(err))
		return api.NewUserResponse(commonstore.StoreCode2APICode(err), req)
	}
	if user == nil {
		return api.NewUserResponse(apimodel.Code_ExecuteSuccess, req)
	}

	if !checkUserViewPermission(ctx, user) {
		log.Error("[Auth][User] delete user forbidden", utils.ZapRequestID(requestID),
			zap.String("name", req.GetName().GetValue()))
		return api.NewUserResponse(apimodel.Code_NotAllowedAccess, req)
	}
	if user.ID == utils.ParseOwnerID(ctx) {
		log.Error("[Auth][User] delete user forbidden, can't delete when self is owner",
			utils.ZapRequestID(requestID), zap.String("name", req.Name.GetValue()))
		return api.NewUserResponse(apimodel.Code_NotAllowedAccess, req)
	}
	if user.Type == model.OwnerUserRole {
		count, err := svr.storage.GetSubCount(user)
		if err != nil {
			log.Error("[Auth][User] get user sub-account", zap.String("owner", user.ID),
				utils.ZapRequestID(requestID), zap.Error(err))
			return api.NewUserResponse(commonstore.StoreCode2APICode(err), req)
		}
		if count != 0 {
			log.Error("[Auth][User] delete user but some sub-account existed", zap.String("owner", user.ID))
			return api.NewUserResponse(apimodel.Code_SubAccountExisted, req)
		}
	}

	if err := svr.storage.DeleteUser(user); err != nil {
		log.Error("[Auth][User] delete user from store", utils.ZapRequestID(requestID), zap.Error(err))
		return api.NewAuthResponse(commonstore.StoreCode2APICode(err))
	}

	log.Info("[Auth][User] delete user", utils.ZapRequestID(requestID),
		zap.String("name", req.Name.GetValue()))
	svr.RecordHistory(userRecordEntry(ctx, req, user, model.ODelete))

	return api.NewUserResponse(apimodel.Code_ExecuteSuccess, req)
}

// GetUsers 查询用户列表
func (svr *server) GetUsers(ctx context.Context, query map[string]string) *apiservice.BatchQueryResponse {
	requestID := utils.ParseRequestID(ctx)
	log.Debug("[Auth][User] origin get users query params",
		utils.ZapRequestID(requestID), zap.Any("query", query))

	var (
		offset, limit uint32
		err           error
		searchFilters = make(map[string]string, len(query)+1)
	)

	for key, value := range query {
		if _, ok := UserFilterAttributes[key]; !ok {
			log.Errorf("[Auth][User] attribute(%s) it not allowed", key)
			return api.NewAuthBatchQueryResponseWithMsg(apimodel.Code_InvalidParameter, key+" is not allowed")
		}

		searchFilters[key] = value
	}

	searchFilters["hide_admin"] = strconv.FormatBool(true)
	// 如果不是超级管理员，查看数据有限制
	if authcommon.ParseUserRole(ctx) != model.AdminUserRole {
		// 设置 owner 参数，只能查看对应 owner 下的用户
		searchFilters["owner"] = utils.ParseOwnerID(ctx)
	}

	var (
		total uint32
		users []*model.User
	)

	offset, limit, err = utils.ParseOffsetAndLimit(searchFilters)
	if err != nil {
		return api.NewAuthBatchQueryResponse(apimodel.Code_InvalidParameter)
	}

	total, users, err = svr.storage.GetUsers(searchFilters, offset, limit)
	if err != nil {
		log.Error("[Auth][User] get user from store", zap.Any("req", searchFilters),
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
func (svr *server) GetUserToken(ctx context.Context, req *apisecurity.User) *apiservice.Response {
	var user *model.User
	if req.GetId().GetValue() != "" {
		user = svr.cacheMgn.User().GetUserByID(req.GetId().GetValue())
	} else if req.GetName().GetValue() != "" {
		ownerName := req.GetOwner().GetValue()
		ownerID := utils.ParseOwnerID(ctx)
		if ownerName == "" {
			owner := svr.cacheMgn.User().GetUserByID(ownerID)
			if owner == nil {
				log.Error("[Auth][User] get user's owner not found",
					zap.String("name", req.GetName().GetValue()), zap.String("owner", ownerID))
				return api.NewAuthResponse(apimodel.Code_NotFoundUser)
			}
			ownerName = owner.Name
		}
		user = svr.cacheMgn.User().GetUserByName(req.GetName().GetValue(), ownerName)
	} else {
		return api.NewAuthResponse(apimodel.Code_InvalidParameter)
	}

	if user == nil {
		return api.NewUserResponse(apimodel.Code_NotFoundUser, req)
	}

	if !checkUserViewPermission(ctx, user) {
		return api.NewUserResponse(apimodel.Code_NotAllowedAccess, req)
	}

	out := &apisecurity.User{
		Id:          utils.NewStringValue(user.ID),
		Name:        utils.NewStringValue(user.Name),
		AuthToken:   utils.NewStringValue(user.Token),
		TokenEnable: utils.NewBoolValue(user.TokenEnable),
	}

	return api.NewUserResponse(apimodel.Code_ExecuteSuccess, out)
}

// UpdateUserToken 更新用户 token
func (svr *server) UpdateUserToken(ctx context.Context, req *apisecurity.User) *apiservice.Response {
	requestID := utils.ParseRequestID(ctx)
	if checkErrResp := checkUpdateUser(req); checkErrResp != nil {
		return checkErrResp
	}

	user, err := svr.storage.GetUser(req.Id.GetValue())
	if err != nil {
		log.Error("[Auth][User] get user from store", utils.ZapRequestID(requestID), zap.Error(err))
		return api.NewUserResponse(commonstore.StoreCode2APICode(err), req)
	}
	if user == nil {
		return api.NewUserResponse(apimodel.Code_NotFoundUser, req)
	}

	if !checkUserViewPermission(ctx, user) {
		return api.NewUserResponse(apimodel.Code_NotAllowedAccess, req)
	}

	if authcommon.ParseUserRole(ctx) != model.AdminUserRole {
		if user.Type != model.SubAccountUserRole {
			return api.NewUserResponseWithMsg(apimodel.Code_NotAllowedAccess, "only disable sub-account token", req)
		}
	}

	user.TokenEnable = req.TokenEnable.GetValue()

	if err := svr.storage.UpdateUser(user); err != nil {
		log.Error("[Auth][User] update user token into store",
			utils.ZapRequestID(requestID), zap.Error(err))
		return api.NewAuthResponseWithMsg(commonstore.StoreCode2APICode(err), err.Error())
	}

	log.Info("[Auth][User] update user token", utils.ZapRequestID(requestID),
		zap.String("id", req.Id.GetValue()), zap.Bool("enable", req.TokenEnable.GetValue()))
	svr.RecordHistory(userRecordEntry(ctx, req, user, model.OUpdateToken))

	return api.NewUserResponse(apimodel.Code_ExecuteSuccess, req)
}

// ResetUserToken 重置用户 token
func (svr *server) ResetUserToken(ctx context.Context, req *apisecurity.User) *apiservice.Response {
	requestID := utils.ParseRequestID(ctx)
	if checkErrResp := checkUpdateUser(req); checkErrResp != nil {
		return checkErrResp
	}

	user, err := svr.storage.GetUser(req.Id.GetValue())
	if err != nil {
		log.Error("[Auth][User] get user from store", utils.ZapRequestID(requestID), zap.Error(err))
		return api.NewUserResponse(commonstore.StoreCode2APICode(err), req)
	}
	if user == nil {
		return api.NewUserResponse(apimodel.Code_NotFoundUser, req)
	}

	if !checkUserViewPermission(ctx, user) {
		return api.NewUserResponse(apimodel.Code_NotAllowedAccess, req)
	}

	newToken, err := createUserToken(user.ID)
	if err != nil {
		log.Error("[Auth][User] update user token", utils.ZapRequestID(requestID), zap.Error(err))
		return api.NewUserResponse(apimodel.Code_ExecuteException, req)
	}

	user.Token = newToken

	if err := svr.storage.UpdateUser(user); err != nil {
		log.Error("[Auth][User] update user token into store", utils.ZapRequestID(requestID), zap.Error(err))
		return api.NewUserResponse(commonstore.StoreCode2APICode(err), req)
	}

	log.Info("[Auth][User] reset user token", utils.ZapRequestID(requestID),
		zap.String("id", req.Id.GetValue()))
	svr.RecordHistory(userRecordEntry(ctx, req, user, model.OUpdateToken))

	req.AuthToken = utils.NewStringValue(user.Token)

	return api.NewUserResponse(apimodel.Code_ExecuteSuccess, req)
}

// checkUserViewPermission 检查是否可以操作该用户
// Case 1: 如果是自己操作自己，通过
// Case 2: 如果是主账户操作自己的子账户，通过
// Case 3: 如果是超级账户，通过
func checkUserViewPermission(ctx context.Context, user *model.User) bool {
	role := authcommon.ParseUserRole(ctx)
	if role == model.AdminUserRole {
		log.Debug("check user view permission", utils.RequestID(ctx), zap.Bool("admin", true))
		return true
	}

	userId := utils.ParseUserID(ctx)
	if user.ID == userId {
		return true
	}

	if user.Owner == userId {
		log.Debug("check user view permission", utils.RequestID(ctx),
			zap.Any("user", user), zap.String("owner", user.Owner), zap.String("operator", userId))
		return true
	}

	return false
}

// user 数组转为[]*apisecurity.User
func enhancedUsers2Api(users []*model.User, handler User2Api) []*apisecurity.User {
	out := make([]*apisecurity.User, 0, len(users))
	for _, entry := range users {
		outUser := handler(entry)
		out = append(out, outUser)
	}

	return out
}

// model.Service 转为 api.Service
func user2Api(user *model.User) *apisecurity.User {
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
		UserType:    utils.NewStringValue(model.UserRoleNames[user.Type]),
	}

	return out
}

// 生成用户的记录entry
func userRecordEntry(ctx context.Context, req *apisecurity.User, md *model.User,
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

	if err := checkName(req.Name); err != nil {
		return api.NewUserResponse(apimodel.Code_InvalidUserName, req)
	}

	if err := checkPassword(req.Password); err != nil {
		return api.NewUserResponse(apimodel.Code_InvalidUserPassword, req)
	}

	if err := checkOwner(req.Owner); err != nil {
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
		if err := checkPassword(req.Password); err != nil {
			return api.NewUserResponseWithMsg(apimodel.Code_InvalidUserPassword, err.Error(), req)
		}
	}

	if req.GetId() == nil || req.GetId().GetValue() == "" {
		return api.NewUserResponse(apimodel.Code_BadRequest, req)
	}
	return nil
}

// updateUserAttribute 更新用户属性
func updateUserAttribute(old *model.User, newUser *apisecurity.User) (*model.User, bool, error) {
	var needUpdate = true

	if newUser.Comment != nil && old.Comment != newUser.Comment.GetValue() {
		old.Comment = newUser.Comment.GetValue()
		needUpdate = true
	}
	return old, needUpdate, nil
}

// updateUserAttribute 更新用户密码信息，如果用户的密码被更新
func updateUserPasswordAttribute(
	isAdmin bool, user *model.User, req *apisecurity.ModifyUserPassword) (*model.User, bool, error) {
	needUpdate := false

	if err := checkPassword(req.NewPassword); err != nil {
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
		// newToken, err := createUserToken(user.ID)
		// if err != nil {
		// 	return nil, false, err
		// }
		// user.Token = newToken
	}

	return user, needUpdate, nil
}

// createUserModel 创建用户模型
func createUserModel(req *apisecurity.User, role model.UserRoleType) (*model.User, error) {
	pwd, err := bcrypt.GenerateFromPassword([]byte(req.GetPassword().GetValue()), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	id := utils.NewUUID()
	if req.GetId().GetValue() != "" {
		id = req.GetId().GetValue()
	}

	user := &model.User{
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
	if user.Type != model.SubAccountUserRole {
		user.Owner = ""
	}

	newToken, err := createUserToken(user.ID)
	if err != nil {
		return nil, err
	}

	user.Token = newToken

	return user, nil
}

// convertCreateUserRole 转换为创建的目标用户的用户角色类型
func convertCreateUserRole(role model.UserRoleType) model.UserRoleType {
	if role == model.AdminUserRole {
		return model.OwnerUserRole
	}

	if role == model.OwnerUserRole {
		return model.SubAccountUserRole
	}

	return model.SubAccountUserRole
}
