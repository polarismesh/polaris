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
	"fmt"
	"time"

	"github.com/polarismesh/polaris-server/cache"
	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/common/model"
	commontime "github.com/polarismesh/polaris-server/common/time"
	"github.com/polarismesh/polaris-server/common/utils"
	"github.com/polarismesh/polaris-server/plugin"
	"github.com/polarismesh/polaris-server/store"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

type (
	User2Api      func(user *model.User) *api.User
	UserGroup2Api func(user *model.UserGroup) *api.UserGroup
)

var (
	UserFilterAttributes = map[string]int{
		"id":       1,
		"name":     1,
		"owner":    1,
		"source":   1,
		"group_id": 1,
		"offset":   1,
		"limit":    1,
	}

	UserLinkGroupAttributes = map[string]int{
		"id":         1,
		"group_name": 1,
		"offset":     1,
		"limit":      1,
	}
)

// UserServer 用户数据管理 server
type userServer struct {
	storage   store.Store
	history   plugin.History
	userCache cache.UserCache
}

// newUserServer
func newUserServer(s store.Store, userCache cache.UserCache) (*userServer, error) {
	svr := &userServer{
		storage:   s,
		userCache: userCache,
	}

	return svr, svr.initialize()
}

// initialize
func (svr *userServer) initialize() error {
	// 获取History插件，注意：插件的配置在bootstrap已经设置好
	svr.history = plugin.GetHistory()
	if svr.history == nil {
		log.GetAuthLogger().Warnf("Not Found History Log Plugin")
	}

	return nil
}

// CreateUsers
func (svr *userServer) CreateUsers(ctx context.Context, req []*api.User) *api.BatchWriteResponse {
	batchResp := api.NewBatchWriteResponse(api.ExecuteSuccess)

	for i := range req {
		user := req[i]
		resp := svr.CreateUser(ctx, user)
		batchResp.Collect(resp)
	}

	return batchResp
}

// CreateUser
func (svr *userServer) CreateUser(ctx context.Context, req *api.User) *api.Response {
	requestID := utils.ParseRequestID(ctx)
	platformID := utils.ParsePlatformID(ctx)
	ownerId := utils.ParseOwnerID(ctx)

	req.Owner = utils.NewStringValue(ownerId)

	if checkErrResp := checkCreateUser(req); checkErrResp != nil {
		return checkErrResp
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

func (svr *userServer) createUser(ctx context.Context, req *api.User) *api.Response {
	requestID := utils.ParseRequestID(ctx)
	platformID := utils.ParsePlatformID(ctx)

	role := utils.ParseUserRole(ctx)
	data, err := createUserModel(req, role)
	if err != nil {
		return api.NewResponseWithMsg(api.ParseException, err.Error())
	}

	newToken, err := CreateUserToken(data.ID)
	if err != nil {
		log.GetAuthLogger().Error(err.Error(), utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
		return api.NewResponseWithMsg(api.ExecuteException, err.Error())
	}

	data.Token = newToken

	if err := svr.storage.AddUser(data); err != nil {
		log.GetAuthLogger().Error(err.Error(), utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
		return api.NewResponseWithMsg(StoreCode2APICode(err), err.Error())
	}

	log.GetAuthLogger().Info("create user", zap.String("name", req.Name.GetValue()), utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
	svr.RecordHistory(userRecordEntry(ctx, req, data, model.OCreate))

	return api.NewUserResponse(api.ExecuteSuccess, req)
}

// UpdateUser 更新用户信息，仅能修改 comment 以及账户密码
func (svr *userServer) UpdateUser(ctx context.Context, req *api.User) *api.Response {
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

// DeleteUser 删除用户
func (svr *userServer) DeleteUser(ctx context.Context, req *api.User) *api.Response {
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

// ListUsers 查询用户列表
func (svr *userServer) ListUsers(ctx context.Context, query map[string]string) *api.BatchQueryResponse {
	searchFilters := make(map[string]string)
	var (
		offset, limit uint32
		err           error
	)

	// 如果不是超级管理员，查看数据有限制
	if utils.ParseUserRole(ctx) != model.AdminUserRole {
		// 设置 owner 参数，只能查看对应 owner 下的用户
		searchFilters["owner"] = utils.ParseOwnerID(ctx)
	}
	for key, value := range query {
		if _, ok := UserFilterAttributes[key]; !ok {
			log.Errorf("[Auth][User][Query] attribute(%s) it not allowed", key)
			return api.NewBatchQueryResponseWithMsg(api.InvalidParameter, key+" is not allowed")
		}
		searchFilters[key] = value
	}

	offset, limit, err = utils.ParseOffsetAndLimit(searchFilters)
	if err != nil {
		return api.NewBatchQueryResponse(api.InvalidParameter)
	}

	total, users, err := svr.storage.ListUsers(searchFilters, offset, limit)
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

// GetUserToken
func (svr *userServer) GetUserToken(ctx context.Context, filter map[string]string) *api.Response {
	isOwner := utils.ParseIsOwner(ctx)
	userId := utils.ParseUserID(ctx)
	ownerId := utils.ParseOwnerID(ctx)

	reqUserId := filter["id"]
	if reqUserId == "" {
		return api.NewResponse(api.InvalidParameter)
	}

	user := svr.userCache.GetUserByID(reqUserId)

	if user == nil {
		return api.NewUserResponse(api.NotFoundUser, &api.User{Id: utils.NewStringValue(reqUserId)})
	}

	if userId != user.ID && (!isOwner || (user.Owner != ownerId)) {
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

// changeUserTokenEnable
func (svr *userServer) ChangeUserTokenStatus(ctx context.Context, req *api.User) *api.Response {
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

	log.GetAuthLogger().Info("change user token status", zap.String("id", req.Id.GetValue()), zap.Bool("enable", req.TokenEnable.GetValue()),
		utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
	svr.RecordHistory(userRecordEntry(ctx, req, user, model.OUpdate))

	out := &api.User{
		Id:   utils.NewStringValue(user.ID),
		Name: req.GetName(),
	}

	return api.NewUserResponse(api.ExecuteSuccess, out)
}

// RefreshUserToken
func (svr *userServer) RefreshUserToken(ctx context.Context, req *api.User) *api.Response {
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

	newToken, err := CreateUserToken(user.ID)
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

	log.GetAuthLogger().Info("refresh user token", zap.String("id", req.Id.GetValue()),
		utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
	svr.RecordHistory(userRecordEntry(ctx, req, user, model.OUpdate))

	out := &api.User{
		Id:        utils.NewStringValue(user.ID),
		AuthToken: utils.NewStringValue(user.Token),
		Name:      req.GetName(),
	}

	return api.NewUserResponse(api.ExecuteSuccess, out)
}

// CreateUserGroup
func (svr *userServer) CreateUserGroup(ctx context.Context, req *api.UserGroup) *api.Response {
	requestID := utils.ParseRequestID(ctx)
	platformID := utils.ParsePlatformID(ctx)
	userId := utils.ParseUserID(ctx)
	ownerId := utils.ParseOwnerID(ctx)

	req.Owner = utils.NewStringValue(userId)

	if checkErrResp := svr.checkCreateUserGroup(ctx, req); checkErrResp != nil {
		return checkErrResp
	}

	// 根据 owner + groupname 确定唯一的用户组信息
	group, err := svr.storage.GetUserByName(req.Name.GetValue(), ownerId)
	if err != nil {
		log.GetAuthLogger().Error(err.Error(), utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
		return api.NewUserGroupResponse(api.StoreLayerException, req)
	}
	if group != nil {
		return api.NewUserGroupResponse(api.UserGroupExisted, req)
	}

	data := createUserGroupModel(req)
	newToken, err := CreateUserGroupToken(data.ID)
	if err != nil {
		log.GetAuthLogger().Error(err.Error(), utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
		return api.NewResponseWithMsg(api.ExecuteException, err.Error())
	}

	data.Token = newToken

	if err := svr.storage.AddUserGroup(data); err != nil {
		log.GetAuthLogger().Error(err.Error(), utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
		return api.NewResponseWithMsg(StoreCode2APICode(err), err.Error())
	}

	log.GetAuthLogger().Info("create usergroup", zap.String("name", req.Name.GetValue()), utils.ZapRequestID(requestID),
		utils.ZapPlatformID(platformID))
	svr.RecordHistory(userGroupRecordEntry(ctx, req, data.UserGroup, model.OCreate))

	return api.NewUserGroupResponse(api.ExecuteSuccess, req)
}

// UpdateUserGroup 更新用户组
func (svr *userServer) UpdateUserGroup(ctx context.Context, req *api.ModifyUserGroup) *api.Response {
	requestID := utils.ParseRequestID(ctx)
	platformID := utils.ParsePlatformID(ctx)

	if checkErrResp := svr.checkUpdateUserGroup(ctx, req); checkErrResp != nil {
		return checkErrResp
	}

	data, errResp := svr.getUserGroupFromDB(requestID, platformID, req.Id.GetValue())
	if errResp != nil {
		return errResp
	}

	modifyReq, needUpdate := updateUserGroupAttribute(ctx, data, req)

	if !needUpdate {
		log.GetAuthLogger().Info("update usergroup data no change, no need update",
			utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID), zap.String("usergroup", req.String()))
		return api.NewModifyUserGroupResponse(api.NoNeedUpdate, req)
	}

	if err := svr.storage.UpdateUserGroup(modifyReq); err != nil {
		log.GetAuthLogger().Error(err.Error(), utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
		return api.NewResponseWithMsg(StoreCode2APICode(err), err.Error())
	}

	log.GetAuthLogger().Info("update user group", zap.String("name", req.Name.GetValue()), utils.ZapRequestID(requestID),
		utils.ZapPlatformID(platformID))
	svr.RecordHistory(modifyUserGroupRecordEntry(ctx, req, data, model.OUpdateUserGroup))

	return api.NewModifyUserGroupResponse(api.ExecuteSuccess, req)
}

// DeleteUserGroup 删除用户组
func (svr *userServer) DeleteUserGroup(ctx context.Context, req *api.UserGroup) *api.Response {
	requestID := utils.ParseRequestID(ctx)
	platformID := utils.ParsePlatformID(ctx)
	isOwner := utils.ParseIsOwner(ctx)
	userId := utils.ParseOwnerID(ctx)

	group, err := svr.storage.GetUserGroup(req.GetId().GetValue())
	if err != nil {
		log.GetAuthLogger().Error(err.Error(), utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
		return api.NewUserGroupResponse(api.StoreLayerException, req)
	}
	if group == nil {
		return api.NewUserGroupResponse(api.ExecuteSuccess, req)
	}

	if !isOwner || (group.Owner != userId) {
		return api.NewResponse(api.NotAllowedAccess)
	}

	if err := svr.storage.DeleteUserGroup(group.ID); err != nil {
		log.GetAuthLogger().Error(err.Error(), utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
		return api.NewResponseWithMsg(StoreCode2APICode(err), err.Error())
	}

	log.GetAuthLogger().Info("delete user group", zap.String("name", req.Name.GetValue()), utils.ZapRequestID(requestID),
		utils.ZapPlatformID(platformID))
	svr.RecordHistory(userGroupRecordEntry(ctx, req, group, model.ODelete))

	return api.NewUserGroupResponse(api.ExecuteSuccess, req)
}

// ListUserGroups 查看用户组
func (svr *userServer) ListGroups(ctx context.Context, query map[string]string) *api.BatchQueryResponse {
	searchFilters := make(map[string]string)
	var (
		offset, limit uint32
		err           error
	)

	for key, value := range query {
		if _, ok := UserFilterAttributes[key]; !ok {
			log.GetAuthLogger().Errorf("[Auth][UserGroup][ListUserGroups] attribute(%s) it not allowed", key)
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

	total, users, err := svr.storage.ListUserGroups(searchFilters, offset, limit)
	if err != nil {
		log.GetAuthLogger().Errorf("[Auth][UserGroup][ListUserGroups] req(%+v) store err: %s", query, err.Error())
		return api.NewBatchQueryResponse(api.StoreLayerException)
	}

	resp := api.NewBatchQueryResponse(api.ExecuteSuccess)
	resp.Amount = utils.NewUInt32Value(total)
	resp.Size = utils.NewUInt32Value(uint32(len(users)))
	resp.UserGroups = enhancedUserGroups2Api(users, userGroup2Api)
	return resp
}

// ListUserByGroup 查看对应用户组下的用户信息
func (svr *userServer) ListUserByGroup(ctx context.Context, query map[string]string) *api.BatchQueryResponse {
	searchFilters := make(map[string]string)
	var (
		offset, limit uint32
		err           error
	)

	for key, value := range query {
		if _, ok := UserFilterAttributes[key]; !ok {
			log.GetAuthLogger().Errorf("[Auth][UserGroup][ListUserByGroup] attribute(%s) it not allowed", key)
			return api.NewBatchQueryResponseWithMsg(api.InvalidParameter, key+" is not allowed")
		}
		searchFilters[key] = value
	}
	offset, limit, err = utils.ParseOffsetAndLimit(searchFilters)
	if err != nil {
		return api.NewBatchQueryResponse(api.InvalidParameter)
	}

	searchFilters["owner"] = utils.ParseOwnerID(ctx)

	total, users, err := svr.storage.ListUserByGroup(searchFilters, offset, limit)
	if err != nil {
		log.GetAuthLogger().Errorf("[Auth][UserGroup][ListUserByGroup] req(%+v) store err: %s", query, err.Error())
		return api.NewBatchQueryResponse(api.StoreLayerException)
	}

	resp := api.NewBatchQueryResponse(api.ExecuteSuccess)
	resp.Amount = utils.NewUInt32Value(total)
	resp.Size = utils.NewUInt32Value(uint32(len(users)))
	resp.Users = enhancedUsers2Api(users, user2Api)
	return resp
}

// ListUserLinkGroups 查看某个用户所在的所有用户组列表信息
func (svr *userServer) ListUserLinkGroups(ctx context.Context, query map[string]string) *api.BatchQueryResponse {
	searchFilters := make(map[string]string)
	var (
		offset, limit uint32
		err           error
	)

	for key, value := range query {
		if _, ok := UserLinkGroupAttributes[key]; !ok {
			log.GetAuthLogger().Errorf("[Auth][UserGroup][ListUserLinkGroups] attribute(%s) it not allowed", key)
			return api.NewBatchQueryResponseWithMsg(api.InvalidParameter, key+" is not allowed")
		}
		searchFilters[key] = value
	}
	offset, limit, err = utils.ParseOffsetAndLimit(searchFilters)
	if err != nil {
		return api.NewBatchQueryResponse(api.InvalidParameter)
	}

	// searchFilters["owner"] = utils.ParseOwnerID(ctx)

	total, groups, err := svr.storage.ListGroupByUser(searchFilters, offset, limit)
	if err != nil {
		log.GetAuthLogger().Errorf("[Auth][UserGroup][ListUserLinkGroups] req(%+v) store err: %s", query, err.Error())
		return api.NewBatchQueryResponse(api.StoreLayerException)
	}

	resp := api.NewBatchQueryResponse(api.ExecuteSuccess)
	resp.Amount = utils.NewUInt32Value(total)
	resp.Size = utils.NewUInt32Value(uint32(len(groups)))
	resp.UserGroups = enhancedUserGroups2Api(groups, userGroup2Api)
	return resp
}

// GetUserGroupToken 查看用户组的token
func (svr *userServer) GetUserGroupToken(ctx context.Context, filter map[string]string) *api.Response {
	isOwner := utils.ParseIsOwner(ctx)
	userId := utils.ParseUserID(ctx)
	ownerId := utils.ParseOwnerID(ctx)
	reqGroupId := filter["id"]
	if reqGroupId == "" {
		return api.NewResponse(api.InvalidParameter)
	}

	groupCache, errResp := svr.getUserGroupFromCache(&api.UserGroup{Id: utils.NewStringValue(reqGroupId)})
	if errResp != nil {
		return errResp
	}

	if !isOwner {
		_, find := groupCache.UserIDs[userId]
		if !find {
			return api.NewResponse(api.NotAllowedAccess)
		}
	} else {
		if groupCache.Owner != ownerId || utils.ParseUserRole(ctx) != model.AdminUserRole {
			return api.NewResponse(api.NotAllowedAccess)
		}
	}

	out := &api.UserGroup{
		Id:          utils.NewStringValue(groupCache.ID),
		Name:        utils.NewStringValue(groupCache.Name),
		AuthToken:   utils.NewStringValue(groupCache.Token),
		TokenEnable: utils.NewBoolValue(groupCache.TokenEnable),
	}

	return api.NewUserGroupResponse(api.ExecuteSuccess, out)
}

// ChangeUserGroupTokenStatus 调整用户组 token 的使用状态 (禁用｜开启)
func (svr *userServer) ChangeUserGroupTokenStatus(ctx context.Context, req *api.UserGroup) *api.Response {
	requestID := utils.ParseRequestID(ctx)
	platformID := utils.ParsePlatformID(ctx)

	group, errResp := svr.getUserGroupFromDB(requestID, platformID, req.Id.GetValue())
	if errResp != nil {
		return errResp
	}

	isOwner := utils.ParseIsOwner(ctx)
	ownerId := utils.ParseOwnerID(ctx)
	if !isOwner || (group.Owner != ownerId) {
		return api.NewResponse(api.NotAllowedAccess)
	}

	group.TokenEnable = req.TokenEnable.GetValue()

	modifyReq := &model.ModifyUserGroup{
		ID:          group.ID,
		Owner:       group.Owner,
		Token:       group.Token,
		TokenEnable: group.TokenEnable,
		Comment:     group.Comment,
	}

	if err := svr.storage.UpdateUserGroup(modifyReq); err != nil {
		log.GetAuthLogger().Error(err.Error(), utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
		return api.NewResponseWithMsg(StoreCode2APICode(err), err.Error())
	}

	log.GetAuthLogger().Info("change usergroup token status", zap.String("id", req.Id.GetValue()),
		zap.Bool("enable", group.TokenEnable), utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
	svr.RecordHistory(userGroupRecordEntry(ctx, req, group, model.OUpdateToken))

	return api.NewUserGroupResponse(api.ExecuteSuccess, req)
}

// RefreshUserGroupToken 刷新用户组的token，刷新时会重置 token 的状态为 enable
func (svr *userServer) RefreshUserGroupToken(ctx context.Context, req *api.UserGroup) *api.Response {
	requestID := utils.ParseRequestID(ctx)
	platformID := utils.ParsePlatformID(ctx)

	group, errResp := svr.getUserGroupFromDB(requestID, platformID, req.Id.GetValue())
	if errResp != nil {
		return errResp
	}

	isOwner := utils.ParseIsOwner(ctx)
	ownerId := utils.ParseOwnerID(ctx)
	if !isOwner || (group.Owner != ownerId) {
		return api.NewResponse(api.NotAllowedAccess)
	}

	newToken, err := CreateUserGroupToken(group.ID)
	if err != nil {
		log.GetAuthLogger().Error("re-create usergroup token", utils.ZapRequestID(requestID),
			utils.ZapPlatformID(platformID), zap.Error(err))
		return api.NewResponseWithMsg(api.ExecuteException, err.Error())
	}

	group.Token = newToken
	modifyReq := &model.ModifyUserGroup{
		ID:          group.ID,
		Owner:       group.Owner,
		Token:       group.Token,
		TokenEnable: group.TokenEnable,
		Comment:     group.Comment,
	}

	if err := svr.storage.UpdateUserGroup(modifyReq); err != nil {
		log.GetAuthLogger().Error(err.Error(), utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
		return api.NewResponseWithMsg(StoreCode2APICode(err), err.Error())
	}

	log.GetAuthLogger().Info("refresh usergroup token", zap.String("group-id", req.Id.GetValue()),
		utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
	svr.RecordHistory(userGroupRecordEntry(ctx, req, group, model.OUpdate))

	req.AuthToken = utils.NewStringValue(newToken)

	return api.NewUserGroupResponse(api.ExecuteSuccess, req)
}

// getUserGroupFromDB 获取用户组
func (svr *userServer) getUserGroupFromDB(requestID, platformID string, id string) (*model.UserGroup, *api.Response) {
	group, err := svr.storage.GetUserGroup(id)
	if err != nil {
		log.Error(err.Error(), utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
		return nil, api.NewResponseWithMsg(api.StoreLayerException, err.Error())
	}
	if group == nil {
		return nil, api.NewResponse(api.NotFoundUserGroup)
	}

	return group, nil
}

// getUserGroupFromCache 从缓存中获取用户组信息数据
func (svr *userServer) getUserGroupFromCache(req *api.UserGroup) (*model.UserGroupDetail, *api.Response) {
	group := svr.userCache.GetUserGroup(req.Id.GetValue())
	if group == nil {
		return nil, api.NewUserGroupResponse(api.NotFoundUserGroup, req)
	}

	return group, nil
}

// preCheckGroupRelation 检查用户-用户组关联关系中，对应的用户信息是否存在，即不能添加一个不存在的用户到用户组
func (svr *userServer) preCheckGroupRelation(groupId string, req *api.UserGroupRelation) (*model.UserGroupDetail, *api.Response) {
	group, err := svr.checkGroupExist(groupId)
	if err != nil {
		log.GetAuthLogger().Error("[Auth][UserGroupRelation][preCheckGroupRelation] check group exist",
			zap.String("group-id", groupId), zap.Error(err))
		return nil, api.NewResponse(api.StoreLayerException)
	}
	if group == nil {
		log.GetAuthLogger().Error("[Auth][UserGroupRelation][preCheckGroupRelation] not exist",
			zap.String("group-id", groupId))
		return nil, api.NewResponse(api.NotFoundUserGroup)
	}

	// 检查该关系中所有的用户是否存在
	uids := make([]string, len(req.UserIds))
	for i := range req.UserIds {
		uids[i] = req.UserIds[i].GetValue()
	}
	uids = utils.StringSliceDeDuplication(uids)
	for i := range uids {
		userId := uids[i]
		user := svr.userCache.GetUserByID(userId)
		if user == nil {
			return group, api.NewUserGroupRelationResponse(api.NotFoundUser, req)
		}
	}
	return group, nil
}

// checkGroupExist 检查用户组是否存在
func (svr *userServer) checkGroupExist(groupId string) (*model.UserGroupDetail, error) {
	group := svr.userCache.GetUserGroup(groupId)
	if group == nil {
		return nil, ErrorNoUserGroup
	}
	return group, nil
}

// RecordHistory server对外提供history插件的简单封装
func (svr *userServer) RecordHistory(entry *model.RecordEntry) {
	// 如果插件没有初始化，那么不记录history
	if svr.history == nil {
		return
	}
	// 如果数据为空，则不需要打印了
	if entry == nil {
		return
	}

	// 调用插件记录history
	svr.history.Record(entry)
}

// checkCreateUserGroup 检查创建用户组的请求
func (svr *userServer) checkCreateUserGroup(ctx context.Context, req *api.UserGroup) *api.Response {

	if req == nil {
		return api.NewUserGroupResponse(api.EmptyRequest, req)
	}

	ownerId := utils.ParseOwnerID(ctx)

	if err := checkOwner(req.Owner); err != nil {
		resp := api.NewUserGroupResponse(api.InvalidUserGroupOwners, req)
		resp.Info = utils.NewStringValue(err.Error())
		return resp
	}

	userIds := req.GetRelation().GetUserIds()
	for i := range userIds {
		userId := userIds[i]
		user := svr.userCache.GetUserByID(userId.GetValue())
		if user == nil {
			return api.NewUserGroupRelationResponse(api.NotFoundUser, req.GetRelation())
		}

		if user.Owner != ownerId {
			return api.NewResponseWithMsg(api.NotAllowedAccess, fmt.Sprintf("user=(%s) owner not equal", userId))
		}
	}

	return nil
}

// checkUpdateUserGroup 检查用户组的更新请求
func (svr *userServer) checkUpdateUserGroup(ctx context.Context, req *api.ModifyUserGroup) *api.Response {
	userId := utils.ParseUserID(ctx)
	isOwner := utils.ParseIsOwner(ctx)

	if req == nil {
		return api.NewModifyUserGroupResponse(api.EmptyRequest, req)
	}

	if req.Id == nil || req.Id.GetValue() == "" {
		return api.NewModifyUserGroupResponse(api.InvalidUserGroupID, req)
	}

	group, checkErrResp := svr.preCheckGroupRelation(req.GetId().GetValue(), req.AddRelation)
	if checkErrResp != nil {
		return checkErrResp
	}

	// 满足以下情况才可以进行操作
	// 1. 自己在这个用户组里面
	// 2. 自己是这个用户组的owner角色
	_, inGroup := group.UserIDs[userId]
	if !inGroup && (!isOwner || group.Owner != userId) {
		return api.NewResponse(api.NotAllowedAccess)
	}

	// 如果当前用户只是在这个组里面，但是不是 owner 角色，那只能添加用户，不能删除用户
	if inGroup && !isOwner && len(req.GetRemoveRelation().UserIds) != 0 {
		return api.NewResponseWithMsg(api.NotAllowedAccess, "only main account can remove user from usergroup")
	}

	return nil
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

// usergroup 数组专为 []*api.UserGroup
func enhancedUserGroups2Api(groups []*model.UserGroup, handler UserGroup2Api) []*api.UserGroup {
	out := make([]*api.UserGroup, 0, len(groups))
	for _, entry := range groups {
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

// model.Service 转为 api.Service
func userGroup2Api(group *model.UserGroup) *api.UserGroup {
	if group == nil {
		return nil
	}

	// note: 不包括token，token比较特殊
	out := &api.UserGroup{
		Id:          utils.NewStringValue(group.ID),
		Name:        utils.NewStringValue(group.Name),
		Owner:       utils.NewStringValue(group.Owner),
		TokenEnable: utils.NewBoolValue(group.TokenEnable),
		Comment:     utils.NewStringValue(group.Comment),
		Ctime:       utils.NewStringValue(commontime.Time2String(group.CreateTime)),
		Mtime:       utils.NewStringValue(commontime.Time2String(group.ModifyTime)),
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

// 生成用户组的记录entry
func userGroupRecordEntry(ctx context.Context, req *api.UserGroup, md *model.UserGroup,
	operationType model.OperationType) *model.RecordEntry {
	entry := &model.RecordEntry{
		ResourceType:  model.RUserGroup,
		UserGroup:     md.Name,
		OperationType: operationType,
		Operator:      utils.ParseOperator(ctx),
		CreateTime:    time.Now(),
	}

	return entry
}

// 生成修改用户组的记录entry
func modifyUserGroupRecordEntry(ctx context.Context, req *api.ModifyUserGroup, md *model.UserGroup,
	operationType model.OperationType) *model.RecordEntry {
	entry := &model.RecordEntry{
		ResourceType:  model.RUserGroup,
		UserGroup:     md.Name,
		OperationType: operationType,
		Operator:      utils.ParseOperator(ctx),
		CreateTime:    time.Now(),
	}

	return entry
}

// 生成用户-用户组关联关系的记录entry
func userRelationRecordEntry(ctx context.Context, req *api.UserGroupRelation, md *model.UserGroup,
	operationType model.OperationType) *model.RecordEntry {
	entry := &model.RecordEntry{
		ResourceType:  model.RUserGroupRelation,
		UserGroup:     md.Name,
		OperationType: operationType,
		Operator:      utils.ParseOperator(ctx),
		CreateTime:    time.Now(),
	}

	return entry
}

// ============ user ============

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
		user.Owner = user.ID
	}

	return user, nil
}

// ============ user group ============

// updateUserGroupAttribute 更新计算用户组更新时的结构体数据，并判断是否需要执行更新操作
func updateUserGroupAttribute(ctx context.Context, old *model.UserGroup, newUser *api.ModifyUserGroup) (*model.ModifyUserGroup, bool) {
	var needUpdate bool = false

	ret := &model.ModifyUserGroup{
		ID:          old.ID,
		Token:       old.Token,
		TokenEnable: old.TokenEnable,
		Comment:     old.Comment,
	}

	isOwner := utils.ParseIsOwner(ctx)

	// 只有 owner 可以修改这个属性
	if isOwner {
		if newUser.Comment.GetValue() != "" && old.Comment != newUser.Comment.GetValue() {
			needUpdate = true
			ret.Comment = newUser.Comment.GetValue()
		}
	}

	// 用户组成员变更计算
	if len(newUser.AddRelation.GetUserIds()) != 0 {
		needUpdate = true
		ids := make([]string, 0, len(newUser.AddRelation.GetUserIds()))
		for index := range newUser.AddRelation.GetUserIds() {
			ids = append(ids, newUser.AddRelation.GetUserIds()[index].GetValue())
		}
		ret.AddUserIds = ids
	}

	if len(newUser.RemoveRelation.GetUserIds()) != 0 {
		needUpdate = true
		ids := make([]string, 0, len(newUser.RemoveRelation.GetUserIds()))
		for index := range newUser.RemoveRelation.GetUserIds() {
			ids = append(ids, newUser.RemoveRelation.GetUserIds()[index].GetValue())
		}
		ret.RemoveUserIds = ids
	}

	return ret, needUpdate
}

// createUserGroupModel 创建用户组的存储模型
func createUserGroupModel(req *api.UserGroup) *model.UserGroupDetail {

	ids := make(map[string]struct{}, len(req.GetRelation().GetUserIds()))
	for index := range req.GetRelation().GetUserIds() {
		ids[req.GetRelation().GetUserIds()[index].GetValue()] = struct{}{}
	}

	return &model.UserGroupDetail{
		UserGroup: &model.UserGroup{
			ID:          utils.NewUUID(),
			Name:        req.GetName().GetValue(),
			Owner:       req.GetOwner().GetValue(),
			Token:       utils.NewUUID(),
			TokenEnable: true,
			Valid:       true,
			Comment:     req.GetComment().GetValue(),
			CreateTime:  time.Now(),
			ModifyTime:  time.Now(),
		},
		UserIDs: ids,
	}
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
