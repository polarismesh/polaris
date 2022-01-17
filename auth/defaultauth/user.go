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
	"time"

	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/common/model"
	commontime "github.com/polarismesh/polaris-server/common/time"
	"github.com/polarismesh/polaris-server/common/utils"
	"github.com/polarismesh/polaris-server/plugin"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

type (
	User2Api func(user *model.User) *api.User
)

var (
	UserFilterAttributes = map[string]int{
		"id":     1,
		"name":   1,
		"owner":  1,
		"source": 1,
		"offset": 1,
		"limit":  1,
	}
)

// initialize
func (svr *server) initialize() error {
	// 获取History插件，注意：插件的配置在bootstrap已经设置好
	svr.history = plugin.GetHistory()
	if svr.history == nil {
		log.GetAuthLogger().Warnf("Not Found History Log Plugin")
	}

	return nil
}

// CreateUsers 批量创建用户
func (svr *server) CreateUsers(ctx context.Context, req []*api.User) *api.BatchWriteResponse {
	batchResp := api.NewBatchWriteResponse(api.ExecuteSuccess)

	for i := range req {
		user := req[i]
		resp := svr.CreateUser(ctx, user)
		batchResp.Collect(resp)
	}

	return batchResp
}

// CreateUser 创建用户
func (svr *server) CreateUser(ctx context.Context, req *api.User) *api.Response {
	requestID := utils.ParseRequestID(ctx)
	platformID := utils.ParsePlatformID(ctx)
	ownerId := utils.ParseOwnerID(ctx)

	req.Owner = utils.NewStringValue(ownerId)

	if checkErrResp := checkCreateUser(req); checkErrResp != nil {
		return checkErrResp
	}

	// 如果创建的目标账户类型是非子账户，则 ownerId 需要设置为 “”
	if converCreateUserRole(utils.ParseUserRole(ctx)) != model.SubAccountUserRole {
		ownerId = ""
	}

	// 只有通过 owner + username 才能唯一确定一个用户
	user, err := svr.storage.GetUserByName(req.Name.GetValue(), ownerId)
	if err != nil {
		log.Error(err.Error(), utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
		return api.NewUserResponse(api.StoreLayerException, req)
	}
	if user != nil {
		return api.NewUserResponse(api.UserExisted, req)
	}

	return svr.createUser(ctx, req)
}

func (svr *server) createUser(ctx context.Context, req *api.User) *api.Response {
	requestID := utils.ParseRequestID(ctx)
	platformID := utils.ParsePlatformID(ctx)

	role := utils.ParseUserRole(ctx)
	data, err := createUserModel(req, role)

	if err != nil {
		log.GetAuthLogger().Error("create user model", utils.ZapRequestID(requestID),
			utils.ZapPlatformID(platformID), zap.Error(err))
		return api.NewResponseWithMsg(api.ExecuteException, err.Error())
	}

	if err := svr.storage.AddUser(data); err != nil {
		log.GetAuthLogger().Error(err.Error(), utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
		return api.NewResponseWithMsg(StoreCode2APICode(err), err.Error())
	}

	log.GetAuthLogger().Info("create user", zap.String("name", req.Name.GetValue()), utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
	svr.RecordHistory(userRecordEntry(ctx, req, data, model.OCreate))

	// 去除 owner 信息
	req.Owner = utils.NewStringValue("")
	return api.NewUserResponse(api.ExecuteSuccess, req)
}

// UpdateUser 更新用户信息，仅能修改 comment 以及账户密码
func (svr *server) UpdateUser(ctx context.Context, req *api.User) *api.Response {
	requestID := utils.ParseRequestID(ctx)
	platformID := utils.ParsePlatformID(ctx)
	isOwner := utils.ParseIsOwner(ctx)
	userId := utils.ParseUserID(ctx)
	ownerId := utils.ParseOwnerID(ctx)

	if checkErrResp := checkUpdateUser(req); checkErrResp != nil {
		return checkErrResp
	}

	user, err := svr.storage.GetUser(req.Id.GetValue())
	if err != nil {
		log.GetAuthLogger().Error(err.Error(), utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
		return api.NewUserResponse(api.StoreLayerException, req)
	}
	if user == nil {
		return api.NewUserResponse(api.NotFoundUser, req)
	}

	if userId != user.ID && (!isOwner || (user.Owner != ownerId)) {
		return api.NewResponse(api.NotAllowedAccess)
	}

	data, err, needUpdate := updateUserAttribute(user, req)
	if err != nil {
		return api.NewResponseWithMsg(api.ExecuteException, err.Error())
	}

	if !needUpdate {
		log.GetAuthLogger().Info("update user data no change, no need update",
			utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID), zap.String("user", req.String()))
		return api.NewUserResponse(api.NoNeedUpdate, req)
	}

	if err := svr.storage.UpdateUser(data); err != nil {
		log.GetAuthLogger().Error(err.Error(), utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
		return api.NewResponseWithMsg(StoreCode2APICode(err), err.Error())
	}

	log.GetAuthLogger().Info("update user", zap.String("name", req.Name.GetValue()), utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
	svr.RecordHistory(userRecordEntry(ctx, req, user, model.OUpdate))

	return api.NewUserResponse(api.ExecuteSuccess, req)
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
func (svr *server) DeleteUser(ctx context.Context, req *api.User) *api.Response {
	requestID := utils.ParseRequestID(ctx)
	platformID := utils.ParsePlatformID(ctx)
	isOwner := utils.ParseIsOwner(ctx)
	userId := utils.ParseUserID(ctx)
	ownerId := utils.ParseOwnerID(ctx)

	if checkErrResp := checkUpdateUser(req); checkErrResp != nil {
		return checkErrResp
	}

	user, err := svr.storage.GetUser(req.Id.GetValue())
	if err != nil {
		log.GetAuthLogger().Error(err.Error(), utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
		return api.NewUserResponse(api.StoreLayerException, req)
	}
	if user == nil {
		return api.NewUserResponse(api.ExecuteSuccess, req)
	}

	if userId != user.ID && (!isOwner || (user.Owner != ownerId)) {
		return api.NewResponse(api.NotAllowedAccess)
	}

	if err := svr.storage.DeleteUser(user.ID); err != nil {
		log.GetAuthLogger().Error(err.Error(), utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
		return api.NewResponseWithMsg(StoreCode2APICode(err), err.Error())
	}

	log.GetAuthLogger().Info("delete user", zap.String("name", req.Name.GetValue()), utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
	svr.RecordHistory(userRecordEntry(ctx, req, user, model.ODelete))

	return api.NewUserResponse(api.ExecuteSuccess, req)
}

// GetUsers 查询用户列表
func (svr *server) GetUsers(ctx context.Context, query map[string]string) *api.BatchQueryResponse {
	searchFilters := make(map[string]string)
	var (
		offset, limit uint32
		err           error
	)

	for key, value := range query {
		if _, ok := UserFilterAttributes[key]; !ok {
			log.Errorf("[Auth][User][Query] attribute(%s) it not allowed", key)
			return api.NewBatchQueryResponseWithMsg(api.InvalidParameter, key+" is not allowed")
		}
		searchFilters[key] = value
	}

	// 如果不是超级管理员，查看数据有限制
	if utils.ParseUserRole(ctx) != model.AdminUserRole {
		// 设置 owner 参数，只能查看对应 owner 下的用户
		searchFilters["owner"] = utils.ParseOwnerID(ctx)
	}

	offset, limit, err = utils.ParseOffsetAndLimit(searchFilters)
	if err != nil {
		return api.NewBatchQueryResponse(api.InvalidParameter)
	}

	total, users, err := svr.storage.GetUsers(searchFilters, offset, limit)
	if err != nil {
		log.GetAuthLogger().Error("[Auth][User][Query] ", zap.Any("req", query), zap.String("store err", err.Error()))
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
	isOwner := utils.ParseIsOwner(ctx)
	userId := utils.ParseUserID(ctx)
	ownerId := utils.ParseOwnerID(ctx)

	if req.GetId().GetValue() == "" {
		return api.NewResponse(api.InvalidParameter)
	}

	user := svr.cacheMgn.User().GetUserByID(req.GetId().GetValue())

	if user == nil {
		return api.NewUserResponse(api.NotFoundUser, req)
	}

	if userId != user.ID && !isOwner && (user.Owner != ownerId) {
		return api.NewResponse(api.NotAllowedAccess)
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
	platformID := utils.ParsePlatformID(ctx)
	isOwner := utils.ParseIsOwner(ctx)
	userId := utils.ParseUserID(ctx)
	ownerId := utils.ParseOwnerID(ctx)

	if checkErrResp := checkUpdateUser(req); checkErrResp != nil {
		return checkErrResp
	}

	user, err := svr.storage.GetUser(req.Id.GetValue())
	if err != nil {
		log.GetAuthLogger().Error(err.Error(), utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
		return api.NewUserResponse(api.StoreLayerException, req)
	}
	if user == nil {
		return api.NewUserResponse(api.NotFoundUser, req)
	}

	if userId != user.ID && (!isOwner || (user.Owner != ownerId)) {
		return api.NewResponse(api.NotAllowedAccess)
	}

	user.TokenEnable = req.TokenEnable.GetValue()

	if err := svr.storage.UpdateUser(user); err != nil {
		log.GetAuthLogger().Error(err.Error(), utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
		return api.NewResponseWithMsg(StoreCode2APICode(err), err.Error())
	}

	log.GetAuthLogger().Info("update user token", zap.String("id", req.Id.GetValue()), zap.Bool("enable", req.TokenEnable.GetValue()),
		utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
	svr.RecordHistory(userRecordEntry(ctx, req, user, model.OUpdate))

	return api.NewUserResponse(api.ExecuteSuccess, req)
}

// ResetUserToken 重置用户 token
func (svr *server) ResetUserToken(ctx context.Context, req *api.User) *api.Response {
	requestID := utils.ParseRequestID(ctx)
	platformID := utils.ParsePlatformID(ctx)
	isOwner := utils.ParseIsOwner(ctx)
	userId := utils.ParseUserID(ctx)
	ownerId := utils.ParseOwnerID(ctx)

	if checkErrResp := checkUpdateUser(req); checkErrResp != nil {
		return checkErrResp
	}

	user, err := svr.storage.GetUser(req.Id.GetValue())
	if err != nil {
		log.GetAuthLogger().Error(err.Error(), utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
		return api.NewUserResponse(api.StoreLayerException, req)
	}
	if user == nil {
		return api.NewUserResponse(api.NotFoundUser, req)
	}
	if userId != user.ID && (!isOwner || (user.Owner != ownerId)) {
		return api.NewResponse(api.NotAllowedAccess)
	}

	newToken, err := createUserToken(user.ID)
	if err != nil {
		log.GetAuthLogger().Error(err.Error(), utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
		return api.NewResponseWithMsg(api.ExecuteException, err.Error())
	}

	user.Token = newToken
	user.TokenEnable = true

	if err := svr.storage.UpdateUser(user); err != nil {
		log.GetAuthLogger().Error(err.Error(), utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
		return api.NewResponseWithMsg(StoreCode2APICode(err), err.Error())
	}

	log.GetAuthLogger().Info("reset user token", zap.String("id", req.Id.GetValue()),
		utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
	svr.RecordHistory(userRecordEntry(ctx, req, user, model.OUpdate))

	req.AuthToken = utils.NewStringValue(user.Token)

	return api.NewUserResponse(api.ExecuteSuccess, req)
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
	}

	return out
}

// 生成用户的记录entry
func userRecordEntry(ctx context.Context, req *api.User, md *model.User,
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
		return api.NewUserResponseWithMsg(api.InvalidUserName, err.Error(), req)
	}

	if err := checkPassword(req.Password); err != nil {
		return api.NewUserResponseWithMsg(api.InvalidUserPassword, err.Error(), req)
	}

	if err := checkOwner(req.Owner); err != nil {
		return api.NewUserResponse(api.InvalidUserOwners, req)
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
func updateUserAttribute(old *model.User, newUser *api.User) (*model.User, error, bool) {
	var needUpdate bool = true

	if newUser.GetPassword().GetValue() != "" {
		pwd, err := bcrypt.GenerateFromPassword([]byte(newUser.GetPassword().GetValue()), bcrypt.DefaultCost)
		if err != nil {
			return nil, err, false
		}
		needUpdate = true
		old.Password = string(pwd)
	}

	if old.Comment != newUser.Comment.GetValue() {
		needUpdate = true
	}

	return old, nil, needUpdate
}

// createUserModel 创建用户模型
func createUserModel(req *api.User, role model.UserRoleType) (*model.User, error) {
	pwd, err := bcrypt.GenerateFromPassword([]byte(req.GetPassword().GetValue()), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &model.User{
		ID:         utils.NewUUID(),
		Name:       req.GetName().GetValue(),
		Password:   string(pwd),
		Owner:      req.GetOwner().GetValue(),
		Source:     req.GetSource().GetValue(),
		Valid:      true,
		Type:       converCreateUserRole(role),
		Comment:    req.GetComment().GetValue(),
		CreateTime: time.Time{},
		ModifyTime: time.Time{},
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

// converCreateUserRole 转换为创建的目标用户的用户角色类型
func converCreateUserRole(role model.UserRoleType) model.UserRoleType {
	if role == model.AdminUserRole {
		return model.OwnerUserRole
	}
	if role == model.OwnerUserRole {
		return model.SubAccountUserRole
	}

	return model.SubAccountUserRole
}
