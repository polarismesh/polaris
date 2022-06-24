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
	"strconv"
	"time"

	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/common/model"
	commontime "github.com/polarismesh/polaris-server/common/time"
	"github.com/polarismesh/polaris-server/common/utils"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

type (
	// User2Api convert user to api.User
	User2Api func(user *model.User) *api.User
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
func (svr *server) CreateUsers(ctx context.Context, req []*api.User) *api.BatchWriteResponse {
	batchResp := api.NewBatchWriteResponse(api.ExecuteSuccess)

	for i := range req {
		resp := svr.CreateUser(ctx, req[i])
		batchResp.Collect(resp)
	}

	return batchResp
}

// CreateUser 创建用户
func (svr *server) CreateUser(ctx context.Context, req *api.User) *api.Response {
	requestID := utils.ParseRequestID(ctx)
	ownerID := utils.ParseOwnerID(ctx)
	req.Owner = utils.NewStringValue(ownerID)

	if checkErrResp := checkCreateUser(req); checkErrResp != nil {
		return checkErrResp
	}

	// 如果创建的目标账户类型是非子账户，则 ownerId 需要设置为 “”
	if convertCreateUserRole(utils.ParseUserRole(ctx)) != model.SubAccountUserRole {
		ownerID = ""
	}

	// 只有通过 owner + username 才能唯一确定一个用户
	user, err := svr.storage.GetUserByName(req.Name.GetValue(), ownerID)
	if err != nil {
		log.Error("[Auth][User] get user by name and owner", utils.ZapRequestID(requestID),
			zap.Error(err), zap.String("name", req.GetName().GetValue()))
		return api.NewUserResponse(api.StoreLayerException, req)
	}
	if user != nil {
		return api.NewUserResponse(api.UserExisted, req)
	}

	return svr.createUser(ctx, req)
}

func (svr *server) createUser(ctx context.Context, req *api.User) *api.Response {
	requestID := utils.ParseRequestID(ctx)

	data, err := createUserModel(req, utils.ParseUserRole(ctx))

	if err != nil {
		log.AuthScope().Error("[Auth][User] create user model", utils.ZapRequestID(requestID), zap.Error(err))
		return api.NewResponse(api.ExecuteException)
	}

	if err := svr.storage.AddUser(data); err != nil {
		log.AuthScope().Error("[Auth][User] add user into store", utils.ZapRequestID(requestID), zap.Error(err))
		return api.NewResponse(StoreCode2APICode(err))
	}

	log.AuthScope().Info("[Auth][User] create user", utils.ZapRequestID(requestID),
		zap.String("name", req.Name.GetValue()))
	svr.RecordHistory(userRecordEntry(ctx, req, data, model.OCreate))

	// 去除 owner 信息
	req.Owner = utils.NewStringValue("")
	req.Id = utils.NewStringValue(data.ID)
	return api.NewUserResponse(api.ExecuteSuccess, req)
}

// UpdateUser 更新用户信息，仅能修改 comment 以及账户密码
func (svr *server) UpdateUser(ctx context.Context, req *api.User) *api.Response {
	requestID := utils.ParseRequestID(ctx)

	if checkErrResp := checkUpdateUser(req); checkErrResp != nil {
		return checkErrResp
	}

	user, err := svr.storage.GetUser(req.Id.GetValue())
	if err != nil {
		log.AuthScope().Error("[Auth][User] get user", utils.ZapRequestID(requestID),
			zap.String("user-id", req.Id.GetValue()), zap.Error(err))
		return api.NewUserResponse(api.StoreLayerException, req)
	}
	if user == nil {
		return api.NewUserResponse(api.NotFoundUser, req)
	}

	if !checkUserViewPermission(ctx, user) {
		return api.NewResponse(api.NotAllowedAccess)
	}

	data, needUpdate, err := updateUserAttribute(user, req)
	if err != nil {
		return api.NewResponseWithMsg(api.ExecuteException, err.Error())
	}

	if !needUpdate {
		log.AuthScope().Info("[Auth][User] update user data no change, no need update",
			utils.ZapRequestID(requestID), zap.String("user", req.String()))
		return api.NewUserResponse(api.NoNeedUpdate, req)
	}

	if err := svr.storage.UpdateUser(data); err != nil {
		log.AuthScope().Error("[Auth][User] update user from store", utils.ZapRequestID(requestID),
			zap.Error(err))
		return api.NewResponseWithMsg(StoreCode2APICode(err), err.Error())
	}

	log.AuthScope().Info("[Auth][User] update user", utils.ZapRequestID(requestID),
		zap.String("name", req.Name.GetValue()))
	svr.RecordHistory(userRecordEntry(ctx, req, user, model.OUpdate))

	return api.NewUserResponse(api.ExecuteSuccess, req)
}

// UpdateUserPassword 更新用户密码信息
func (svr *server) UpdateUserPassword(ctx context.Context, req *api.ModifyUserPassword) *api.Response {
	requestID := utils.ParseRequestID(ctx)

	user, err := svr.storage.GetUser(req.Id.GetValue())
	if err != nil {
		log.AuthScope().Error("[Auth][User] get user", utils.ZapRequestID(requestID),
			zap.String("user-id", req.Id.GetValue()), zap.Error(err))
		return api.NewResponse(api.StoreLayerException)
	}
	if user == nil {
		return api.NewResponse(api.NotFoundUser)
	}

	if !checkUserViewPermission(ctx, user) {
		return api.NewResponse(api.NotAllowedAccess)
	}

	ignoreOrign := utils.ParseUserRole(ctx) == model.AdminUserRole || utils.ParseUserRole(ctx) == model.OwnerUserRole
	data, needUpdate, err := updateUserPasswordAttribute(ignoreOrign, user, req)
	if err != nil {
		log.AuthScope().Error("[Auth][User] compute user update attribute", zap.Error(err),
			zap.String("user", req.GetId().GetValue()))
		return api.NewResponseWithMsg(api.ExecuteException, err.Error())
	}

	if !needUpdate {
		log.AuthScope().Info("[Auth][User] update user password no change, no need update",
			utils.ZapRequestID(requestID), zap.String("user", req.GetId().GetValue()))
		return api.NewResponse(api.NoNeedUpdate)
	}

	if err := svr.storage.UpdateUser(data); err != nil {
		log.AuthScope().Error("[Auth][User] update user from store", utils.ZapRequestID(requestID),
			zap.Error(err))
		return api.NewResponse(StoreCode2APICode(err))
	}

	log.AuthScope().Info("[Auth][User] update user", utils.ZapRequestID(requestID),
		zap.String("user-id", req.Id.GetValue()))

	return api.NewResponse(api.ExecuteSuccess)
}

// DeleteUsers 批量删除用户
func (svr *server) DeleteUsers(ctx context.Context, reqs []*api.User) *api.BatchWriteResponse {
	resp := api.NewBatchWriteResponse(api.ExecuteSuccess)

	for index := range reqs {
		ret := svr.DeleteUser(ctx, reqs[index])
		resp.Collect(ret)
	}

	return resp

}

// DeleteUser 删除用户
// Case 1. 删除主账户，主账户不能自己删除自己
// Case 2. 删除主账户，如果主账户下还存在子账户，必须先删除子账户，才能删除主账户
// Case 3. 主账户角色下，只能删除自己创建的子账户
// Case 4. 超级账户角色下，可以删除任意账户
func (svr *server) DeleteUser(ctx context.Context, req *api.User) *api.Response {
	requestID := utils.ParseRequestID(ctx)

	user, err := svr.storage.GetUser(req.Id.GetValue())
	if err != nil {
		log.AuthScope().Error("[Auth][User] get user from store", utils.ZapRequestID(requestID), zap.Error(err))
		return api.NewUserResponse(api.StoreLayerException, req)
	}
	if user == nil {
		return api.NewUserResponse(api.ExecuteSuccess, req)
	}

	if !checkUserViewPermission(ctx, user) {
		log.AuthScope().Error("[Auth][User] delete user forbidden", utils.ZapRequestID(requestID),
			zap.String("name", req.GetName().GetValue()))
		return api.NewUserResponse(api.NotAllowedAccess, req)
	}
	if user.ID == utils.ParseOwnerID(ctx) {
		log.AuthScope().Error("[Auth][User] delete user forbidden, can't delete when self is owner",
			utils.ZapRequestID(requestID), zap.String("name", req.Name.GetValue()))
		return api.NewUserResponse(api.NotAllowedAccess, req)
	}
	if user.Type == model.OwnerUserRole {
		count, err := svr.storage.GetSubCount(user)
		if err != nil {
			return api.NewUserResponse(api.StoreLayerException, req)
		}
		if count != 0 {
			log.AuthScope().Error("[Auth][User] delete user but some sub-account existed", zap.String("owner", user.ID))
			return api.NewUserResponse(api.SubAccountExisted, req)
		}
	}

	if err := svr.storage.DeleteUser(user); err != nil {
		log.AuthScope().Error("[Auth][User] delete user from store", utils.ZapRequestID(requestID), zap.Error(err))
		return api.NewResponse(StoreCode2APICode(err))
	}

	log.AuthScope().Info("[Auth][User] delete user", utils.ZapRequestID(requestID),
		zap.String("name", req.Name.GetValue()))
	svr.RecordHistory(userRecordEntry(ctx, req, user, model.ODelete))

	return api.NewUserResponse(api.ExecuteSuccess, req)
}

// GetUsers 查询用户列表
func (svr *server) GetUsers(ctx context.Context, query map[string]string) *api.BatchQueryResponse {
	requestID := utils.ParseRequestID(ctx)

	log.AuthScope().Debug("[Auth][User] origin get users query params",
		utils.ZapRequestID(requestID), zap.Any("query", query))

	var (
		offset, limit uint32
		err           error
	)

	searchFilters := make(map[string]string, len(query)+1)
	for key, value := range query {
		if _, ok := UserFilterAttributes[key]; !ok {
			log.AuthScope().Errorf("[Auth][User] attribute(%s) it not allowed", key)
			return api.NewBatchQueryResponseWithMsg(api.InvalidParameter, key+" is not allowed")
		}

		searchFilters[key] = value
	}

	searchFilters["hide_admin"] = strconv.FormatBool(true)

	// 如果不是超级管理员，查看数据有限制
	if utils.ParseUserRole(ctx) != model.AdminUserRole {
		// 设置 owner 参数，只能查看对应 owner 下的用户
		searchFilters["owner"] = utils.ParseOwnerID(ctx)
	}

	var (
		total uint32
		users []*model.User
	)

	offset, limit, err = utils.ParseOffsetAndLimit(searchFilters)
	if err != nil {
		return api.NewBatchQueryResponse(api.InvalidParameter)
	}

	total, users, err = svr.storage.GetUsers(searchFilters, offset, limit)
	if err != nil {
		log.AuthScope().Error("[Auth][User] get user from store", zap.Any("req", searchFilters),
			zap.Error(err))
		return api.NewBatchQueryResponse(api.StoreLayerException)
	}

	resp := api.NewBatchQueryResponse(api.ExecuteSuccess)
	resp.Amount = utils.NewUInt32Value(total)
	resp.Size = utils.NewUInt32Value(uint32(len(users)))
	resp.Users = enhancedUsers2Api(users, user2Api)
	return resp
}

// GetUserToken 获取用户 token
func (svr *server) GetUserToken(ctx context.Context, req *api.User) *api.Response {
	var user *model.User

	if req.GetId().GetValue() != "" {
		user = svr.cacheMgn.User().GetUserByID(req.GetId().GetValue())
	} else if req.GetName().GetValue() != "" {
		ownerName := req.GetOwner().GetValue()
		ownerID := utils.ParseOwnerID(ctx)
		if ownerName == "" {
			owner := svr.cacheMgn.User().GetUserByID(ownerID)
			if owner == nil {
				log.AuthScope().Error("[Auth][User] get user's owner not found",
					zap.String("name", req.GetName().GetValue()), zap.String("owner", ownerID))
				return api.NewResponse(api.NotFoundUser)
			}
			ownerName = owner.Name
		}
		user = svr.cacheMgn.User().GetUserByName(req.GetName().GetValue(), ownerName)
	} else {
		return api.NewResponse(api.InvalidParameter)
	}

	if user == nil {
		return api.NewUserResponse(api.NotFoundUser, req)
	}

	if !checkUserViewPermission(ctx, user) {
		return api.NewUserResponse(api.NotAllowedAccess, req)
	}

	out := &api.User{
		Id:          utils.NewStringValue(user.ID),
		Name:        utils.NewStringValue(user.Name),
		AuthToken:   utils.NewStringValue(user.Token),
		TokenEnable: utils.NewBoolValue(user.TokenEnable),
	}

	return api.NewUserResponse(api.ExecuteSuccess, out)
}

// UpdateUserToken 更新用户 token
func (svr *server) UpdateUserToken(ctx context.Context, req *api.User) *api.Response {
	requestID := utils.ParseRequestID(ctx)

	if checkErrResp := checkUpdateUser(req); checkErrResp != nil {
		return checkErrResp
	}

	user, err := svr.storage.GetUser(req.Id.GetValue())
	if err != nil {
		log.AuthScope().Error("[Auth][User] get user from store", utils.ZapRequestID(requestID), zap.Error(err))
		return api.NewUserResponse(api.StoreLayerException, req)
	}
	if user == nil {
		return api.NewUserResponse(api.NotFoundUser, req)
	}

	if !checkUserViewPermission(ctx, user) {
		return api.NewUserResponse(api.NotAllowedAccess, req)
	}

	if utils.ParseUserRole(ctx) != model.AdminUserRole {
		if user.Type != model.SubAccountUserRole {
			return api.NewUserResponseWithMsg(api.NotAllowedAccess, "only disable sub-account token", req)
		}
	}

	user.TokenEnable = req.TokenEnable.GetValue()

	if err := svr.storage.UpdateUser(user); err != nil {
		log.AuthScope().Error("[Auth][User] update user token into store",
			utils.ZapRequestID(requestID), zap.Error(err))
		return api.NewResponseWithMsg(StoreCode2APICode(err), err.Error())
	}

	log.AuthScope().Info("[Auth][User] update user token", utils.ZapRequestID(requestID),
		zap.String("id", req.Id.GetValue()), zap.Bool("enable", req.TokenEnable.GetValue()))
	svr.RecordHistory(userRecordEntry(ctx, req, user, model.OUpdate))

	return api.NewUserResponse(api.ExecuteSuccess, req)
}

// ResetUserToken 重置用户 token
func (svr *server) ResetUserToken(ctx context.Context, req *api.User) *api.Response {
	requestID := utils.ParseRequestID(ctx)

	if checkErrResp := checkUpdateUser(req); checkErrResp != nil {
		return checkErrResp
	}

	user, err := svr.storage.GetUser(req.Id.GetValue())
	if err != nil {
		log.AuthScope().Error("[Auth][User] get user from store", utils.ZapRequestID(requestID), zap.Error(err))
		return api.NewUserResponse(api.StoreLayerException, req)
	}
	if user == nil {
		return api.NewUserResponse(api.NotFoundUser, req)
	}

	if !checkUserViewPermission(ctx, user) {
		return api.NewUserResponse(api.NotAllowedAccess, req)
	}

	newToken, err := createUserToken(user.ID)
	if err != nil {
		log.AuthScope().Error("[Auth][User] update user token", utils.ZapRequestID(requestID), zap.Error(err))
		return api.NewUserResponse(api.ExecuteException, req)
	}

	user.Token = newToken

	if err := svr.storage.UpdateUser(user); err != nil {
		log.AuthScope().Error("[Auth][User] update user token into store", utils.ZapRequestID(requestID), zap.Error(err))
		return api.NewUserResponse(StoreCode2APICode(err), req)
	}

	log.AuthScope().Info("[Auth][User] reset user token", utils.ZapRequestID(requestID),
		zap.String("id", req.Id.GetValue()))
	svr.RecordHistory(userRecordEntry(ctx, req, user, model.OUpdate))

	req.AuthToken = utils.NewStringValue(user.Token)

	return api.NewUserResponse(api.ExecuteSuccess, req)
}

// checkUserViewPermission 检查是否可以操作该用户
// Case 1: 如果是自己操作自己，通过
// Case 2: 如果是主账户操作自己的子账户，通过
// Case 3: 如果是超级账户，通过
func checkUserViewPermission(ctx context.Context, user *model.User) bool {
	role := utils.ParseUserRole(ctx)
	if role == model.AdminUserRole {
		log.AuthScope().Debug("check user view permission", utils.ZapRequestIDByCtx(ctx), zap.Bool("admin", true))
		return true
	}

	userId := utils.ParseUserID(ctx)
	if user.ID == userId {
		return true
	}

	if user.Owner == userId {
		log.AuthScope().Debug("check user view permission", utils.ZapRequestIDByCtx(ctx),
			zap.Any("user", user), zap.String("owner", user.Owner), zap.String("operator", userId))
		return true
	}

	return false
}

// user 数组转为[]*api.User
func enhancedUsers2Api(users []*model.User, handler User2Api) []*api.User {
	out := make([]*api.User, 0, len(users))
	for _, entry := range users {
		outUser := handler(entry)
		out = append(out, outUser)
	}

	return out
}

// model.Service 转为 api.Service
func user2Api(user *model.User) *api.User {
	if user == nil {
		return nil
	}

	// note: 不包括token，token比较特殊
	out := &api.User{
		Id:          utils.NewStringValue(user.ID),
		Name:        utils.NewStringValue(user.Name),
		Source:      utils.NewStringValue(user.Source),
		Owner:       utils.NewStringValue(user.Owner),
		TokenEnable: utils.NewBoolValue(user.TokenEnable),
		Comment:     utils.NewStringValue(user.Comment),
		Ctime:       utils.NewStringValue(commontime.Time2String(user.CreateTime)),
		Mtime:       utils.NewStringValue(commontime.Time2String(user.ModifyTime)),
		Mobile:      utils.NewStringValue(user.Mobile),
		Email:       utils.NewStringValue(user.Email),
		UserType:    utils.NewStringValue(model.UserRoleNames[user.Type]),
	}

	return out
}

// 生成用户的记录entry
func userRecordEntry(ctx context.Context, _ *api.User, md *model.User,
	operationType model.OperationType) *model.RecordEntry {
	entry := &model.RecordEntry{
		ResourceType:  model.RUser,
		Username:      md.Name,
		OperationType: operationType,
		Operator:      utils.ParseOperator(ctx),
		CreateTime:    time.Now(),
	}

	return entry
}

// checkCreateUser 检查创建用户的请求
func checkCreateUser(req *api.User) *api.Response {
	if req == nil {
		return api.NewUserResponse(api.EmptyRequest, req)
	}

	if err := checkName(req.Name); err != nil {
		return api.NewUserResponse(api.InvalidUserName, req)
	}

	if err := checkPassword(req.Password); err != nil {
		return api.NewUserResponse(api.InvalidUserPassword, req)
	}

	if err := checkOwner(req.Owner); err != nil {
		return api.NewUserResponse(api.InvalidUserOwners, req)
	}

	if err := checkMobile(req.Mobile); err != nil {
		return api.NewUserResponse(api.InvalidUserMobile, req)
	}

	if err := checkEmail(req.Email); err != nil {
		return api.NewUserResponse(api.InvalidUserEmail, req)
	}

	return nil
}

// checkUpdateUser 检查用户更新请求
func checkUpdateUser(req *api.User) *api.Response {
	if req == nil {
		return api.NewUserResponse(api.EmptyRequest, req)
	}

	// 如果本次请求需要修改密码的话
	if req.GetPassword() != nil {
		if err := checkPassword(req.Password); err != nil {
			return api.NewUserResponseWithMsg(api.InvalidUserPassword, err.Error(), req)
		}
	}

	if req.GetId() == nil || req.GetId().GetValue() == "" {
		return api.NewUserResponse(api.BadRequest, req)
	}

	return nil
}

// updateUserAttribute 更新用户属性
func updateUserAttribute(old *model.User, newUser *api.User) (*model.User, bool, error) {
	var needUpdate = true

	if newUser.Comment != nil && old.Comment != newUser.Comment.GetValue() {
		old.Comment = newUser.Comment.GetValue()
		needUpdate = true
	}

	if newUser.Mobile != nil && old.Mobile != newUser.Mobile.GetValue() {
		old.Mobile = newUser.Mobile.GetValue()
		needUpdate = true
	}

	if newUser.Email != nil && old.Email != newUser.Email.GetValue() {
		old.Email = newUser.Email.GetValue()
		needUpdate = true
	}

	return old, needUpdate, nil
}

// updateUserAttribute 更新用户密码信息，如果用户的密码被更新
func updateUserPasswordAttribute(isAdmin bool, user *model.User, req *api.ModifyUserPassword) (*model.User, bool, error) {
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
func createUserModel(req *api.User, role model.UserRoleType) (*model.User, error) {
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
		Mobile:      req.GetMobile().GetValue(),
		Email:       req.GetEmail().GetValue(),
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
