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
	"time"
	"unicode/utf8"

	"github.com/golang/protobuf/ptypes/wrappers"
	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/common/model"
	commontime "github.com/polarismesh/polaris-server/common/time"
	"github.com/polarismesh/polaris-server/common/utils"
	"github.com/polarismesh/polaris-server/core/auth/defaultauth/cache"
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
		"id":     1,
		"name":   1,
		"owner":  1,
		"source": 1,
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
		log.Warnf("Not Found History Log Plugin")
	}

	return nil
}

// CreateUser
func (svr *userServer) CreateUser(ctx context.Context, req *api.User) *api.Response {
	requestID := utils.ParseRequestID(ctx)
	platformID := utils.ParsePlatformID(ctx)

	if checkErrResp := checkCreateUser(req); checkErrResp != nil {
		return checkErrResp
	}

	user, err := svr.storage.GetUserByName(req.Name.GetValue())
	if err != nil {
		log.Error(err.Error(), utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
		return api.NewUserResponse(api.StoreLayerException, req)
	}
	if user != nil {
		return api.NewUserResponse(api.ExistedResource, req)
	}

	return svr.createUser(ctx, req)
}

func (svr *userServer) createUser(ctx context.Context, req *api.User) *api.Response {
	requestID := utils.ParseRequestID(ctx)
	platformID := utils.ParsePlatformID(ctx)
	data, err := createUserModel(req)
	if err != nil {
		return api.NewResponseWithMsg(api.ParseException, err.Error())
	}

	newToken, err := CreateToken(data.ID, "")
	if err != nil {
		log.Error(err.Error(), utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
		return api.NewResponseWithMsg(api.ExecuteException, err.Error())
	}

	data.Token = newToken

	if err := svr.storage.AddUser(data); err != nil {
		log.Error(err.Error(), utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
		return api.NewResponseWithMsg(StoreCode2APICode(err), err.Error())
	}

	msg := fmt.Sprintf("create user: name=%v", req.Name)
	log.Info(msg, utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
	svr.RecordHistory(userRecordEntry(ctx, req, data, model.OCreate))

	out := &api.User{
		Id:        utils.NewStringValue(data.Name),
		Name:      req.GetName(),
		AuthToken: utils.NewStringValue(data.Token),
	}

	return api.NewUserResponse(api.ExecuteSuccess, out)
}

// UpdateUser
func (svr *userServer) UpdateUser(ctx context.Context, req *api.User) *api.Response {
	requestID := utils.ParseRequestID(ctx)
	platformID := utils.ParsePlatformID(ctx)
	isOwner := utils.ParseIsOwner(ctx)
	userId := utils.ParseUserID(ctx)

	if checkErrResp := checkUpdateUser(req); checkErrResp != nil {
		return checkErrResp
	}

	user, err := svr.storage.GetUserByName(req.Name.GetValue())
	if err != nil {
		log.Error(err.Error(), utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
		return api.NewUserResponse(api.StoreLayerException, req)
	}
	if user == nil {
		return api.NewUserResponse(api.NotFoundResource, req)
	}

	if userId != user.ID {
		if !isOwner || (user.Owner != userId) {
			return api.NewResponse(api.NotAllowedAccess)
		}
	}

	errResp, needUpdate := diffUserInfo(user, req)
	if errResp != nil {
		return errResp
	}

	if !needUpdate {
		log.Info("update user data no change, no need update",
			utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID), zap.String("user", req.String()))
		return api.NewUserResponse(api.NoNeedUpdate, req)
	}

	if err := svr.storage.UpdateUser(user); err != nil {
		log.Error(err.Error(), utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
		return api.NewResponseWithMsg(StoreCode2APICode(err), err.Error())
	}

	msg := fmt.Sprintf("update user: name=%v", req.Name)
	log.Info(msg, utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
	svr.RecordHistory(userRecordEntry(ctx, req, user, model.OUpdate))

	out := &api.User{
		Id:   utils.NewStringValue(user.ID),
		Name: req.GetName(),
	}

	return api.NewUserResponse(api.ExecuteSuccess, out)
}

// DeleteUser
func (svr *userServer) DeleteUser(ctx context.Context, req *api.User) *api.Response {
	requestID := utils.ParseRequestID(ctx)
	platformID := utils.ParsePlatformID(ctx)
	isOwner := utils.ParseIsOwner(ctx)
	userId := utils.ParseUserID(ctx)

	if checkErrResp := checkUpdateUser(req); checkErrResp != nil {
		return checkErrResp
	}

	user, err := svr.storage.GetUserByName(req.Name.GetValue())
	if err != nil {
		log.Error(err.Error(), utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
		return api.NewUserResponse(api.StoreLayerException, req)
	}
	if user == nil {
		return api.NewUserResponse(api.ExecuteSuccess, req)
	}

	if userId != user.ID {
		if !isOwner || (user.Owner != userId) {
			return api.NewResponse(api.NotAllowedAccess)
		}
	}

	if err := svr.storage.DeleteUser(user.ID); err != nil {
		log.Error(err.Error(), utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
		return api.NewResponseWithMsg(StoreCode2APICode(err), err.Error())
	}

	msg := fmt.Sprintf("delete user: name=%v", req.Name)
	log.Info(msg, utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
	svr.RecordHistory(userRecordEntry(ctx, req, user, model.ODelete))

	out := &api.User{
		Name: req.GetName(),
	}

	return api.NewUserResponse(api.ExecuteSuccess, out)
}

// ListUsers
func (svr *userServer) ListUsers(ctx context.Context, query map[string]string) *api.BatchQueryResponse {
	searchFilters := make(map[string]string)
	var (
		offset, limit uint32
		err           error
	)

	isOwner := utils.ParseIsOwner(ctx)
	userId := utils.ParseUserID(ctx)

	if !isOwner {
		// 就只查询当前该操作者信息
		searchFilters["id"] = userId
		offset = 0
		limit = 1
	} else {
		searchFilters["owmer"] = userId
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
	}

	total, users, err := svr.storage.ListUsers(searchFilters, offset, limit)
	if err != nil {
		log.Errorf("[Auth][User][Query] req(%+v) store err: %s", query, err.Error())
		return api.NewBatchQueryResponse(api.StoreLayerException)
	}

	resp := api.NewBatchQueryResponse(api.ExecuteSuccess)
	resp.Amount = utils.NewUInt32Value(total)
	resp.Size = utils.NewUInt32Value(uint32(len(users)))
	resp.Users = enhancedUsers2Api(users, user2Api)
	return resp
}

// GetUserToken
func (svr *userServer) GetUserToken(ctx context.Context, req *api.User) *api.Response {
	requestID := utils.ParseRequestID(ctx)
	platformID := utils.ParsePlatformID(ctx)
	isOwner := utils.ParseIsOwner(ctx)
	userId := utils.ParseUserID(ctx)

	if checkErrResp := checkCreateUser(req); checkErrResp != nil {
		return checkErrResp
	}

	user, err := svr.storage.GetUser(req.GetId().GetValue())
	if err != nil {
		log.Error(err.Error(), utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
		return api.NewUserResponse(api.StoreLayerException, req)
	}
	if user == nil {
		return api.NewUserResponse(api.NotFoundResource, req)
	}

	if userId != user.ID {
		if !isOwner || (user.Owner != userId) {
			return api.NewResponse(api.NotAllowedAccess)
		}
	}

	if userId != user.ID {
		if !isOwner || (user.Owner != userId) {
			return api.NewResponse(api.NotAllowedAccess)
		}
	}

	out := &api.User{
		Id:        utils.NewStringValue(user.ID),
		Name:      utils.NewStringValue(user.Name),
		AuthToken: utils.NewStringValue(user.Token),
	}

	return api.NewUserResponse(api.ExecuteSuccess, out)
}

// DisableUserToken
func (svr *userServer) DisableUserToken(ctx context.Context, req *api.User) *api.Response {

	return svr.changeUserTokenEnable(ctx, req, true)
}

// EnableUserToken
func (svr *userServer) EnableUserToken(ctx context.Context, req *api.User) *api.Response {

	return svr.changeUserTokenEnable(ctx, req, false)
}

// changeUserTokenEnable
func (svr *userServer) changeUserTokenEnable(ctx context.Context, req *api.User, disable bool) *api.Response {
	requestID := utils.ParseRequestID(ctx)
	platformID := utils.ParsePlatformID(ctx)
	isOwner := utils.ParseIsOwner(ctx)
	userId := utils.ParseUserID(ctx)

	if checkErrResp := checkUpdateUser(req); checkErrResp != nil {
		return checkErrResp
	}

	user, err := svr.storage.GetUserByName(req.Name.GetValue())
	if err != nil {
		log.Error(err.Error(), utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
		return api.NewUserResponse(api.StoreLayerException, req)
	}
	if user == nil {
		return api.NewUserResponse(api.NotFoundResource, req)
	}

	if userId != user.ID {
		if !isOwner || (user.Owner != userId) {
			return api.NewResponse(api.NotAllowedAccess)
		}
	}

	user.TokenEnable = disable

	if err := svr.storage.UpdateUser(user); err != nil {
		log.Error(err.Error(), utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
		return api.NewResponseWithMsg(StoreCode2APICode(err), err.Error())
	}

	msg := fmt.Sprintf("disable user: name=%v token", req.Name)
	log.Info(msg, utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
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

	if checkErrResp := checkUpdateUser(req); checkErrResp != nil {
		return checkErrResp
	}

	user, err := svr.storage.GetUserByName(req.Name.GetValue())
	if err != nil {
		log.Error(err.Error(), utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
		return api.NewUserResponse(api.StoreLayerException, req)
	}
	if user == nil {
		return api.NewUserResponse(api.NotFoundResource, req)
	}
	if userId != user.ID {
		if !isOwner || (user.Owner != userId) {
			return api.NewResponse(api.NotAllowedAccess)
		}
	}

	newToken, err := CreateToken(user.ID, "")
	if err != nil {
		log.Error(err.Error(), utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
		return api.NewResponseWithMsg(api.ExecuteException, err.Error())
	}

	user.Token = newToken

	if err := svr.storage.UpdateUser(user); err != nil {
		log.Error(err.Error(), utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
		return api.NewResponseWithMsg(StoreCode2APICode(err), err.Error())
	}

	msg := fmt.Sprintf("disable user: name=%v token", req.Name)
	log.Info(msg, utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
	svr.RecordHistory(userRecordEntry(ctx, req, user, model.OUpdate))

	out := &api.User{
		Id:   utils.NewStringValue(user.ID),
		Name: req.GetName(),
	}

	return api.NewUserResponse(api.ExecuteSuccess, out)
}

// CreateUserGroup
func (svr *userServer) CreateUserGroup(ctx context.Context, req *api.UserGroup) *api.Response {
	requestID := utils.ParseRequestID(ctx)
	platformID := utils.ParsePlatformID(ctx)

	if checkErrResp := checkCreateUserGroup(req); checkErrResp != nil {
		return checkErrResp
	}

	group, err := svr.storage.GetUserByName(req.Name.GetValue())
	if err != nil {
		log.Error(err.Error(), utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
		return api.NewUserGroupResponse(api.StoreLayerException, req)
	}
	if group != nil {
		return api.NewUserGroupResponse(api.ExistedResource, req)
	}

	isOwner := utils.ParseIsOwner(ctx)
	userId := utils.ParseUserID(ctx)
	if !isOwner || (group.Owner != userId) {
		return api.NewResponse(api.NotAllowedAccess)
	}

	data := createUserGroupModel(req)
	newToken, err := CreateToken(data.ID, "")
	if err != nil {
		log.Error(err.Error(), utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
		return api.NewResponseWithMsg(api.ExecuteException, err.Error())
	}

	data.Token = newToken

	if err := svr.storage.AddUserGroup(data); err != nil {
		log.Error(err.Error(), utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
		return api.NewResponseWithMsg(StoreCode2APICode(err), err.Error())
	}

	msg := fmt.Sprintf("create user group: name=%v", req.Name)
	log.Info(msg, utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
	svr.RecordHistory(userGroupRecordEntry(ctx, req, data, model.OCreate))

	out := &api.UserGroup{
		Id:        utils.NewStringValue(data.ID),
		Name:      req.GetName(),
		AuthToken: utils.NewStringValue(data.Token),
	}

	return api.NewUserGroupResponse(api.ExecuteSuccess, out)
}

// UpdateUserGroup
func (svr *userServer) UpdateUserGroup(ctx context.Context, req *api.UserGroup) *api.Response {
	requestID := utils.ParseRequestID(ctx)
	platformID := utils.ParsePlatformID(ctx)

	if checkErrResp := checkUpdateUserGroup(req); checkErrResp != nil {
		return checkErrResp
	}

	data, errResp := svr.getUserGroupMustExist(requestID, platformID, req)
	if errResp != nil {
		return errResp
	}

	isOwner := utils.ParseIsOwner(ctx)
	userId := utils.ParseUserID(ctx)
	if !isOwner || (data.Owner != userId) {
		return api.NewResponse(api.NotAllowedAccess)
	}

	errResp, needUpdate := diffUserGroupInfo(data, req)
	if errResp != nil {
		return errResp
	}
	if !needUpdate {
		log.Info("update usergroup data no change, no need update",
			utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID), zap.String("usergroup", req.String()))
		return api.NewUserGroupResponse(api.NoNeedUpdate, req)
	}

	if err := svr.storage.UpdateUserGroup(data); err != nil {
		log.Error(err.Error(), utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
		return api.NewResponseWithMsg(StoreCode2APICode(err), err.Error())
	}

	msg := fmt.Sprintf("update user group: name=%v", req.Name)
	log.Info(msg, utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
	svr.RecordHistory(userGroupRecordEntry(ctx, req, data, model.OUpdate))

	out := &api.UserGroup{
		Name:      req.GetName(),
		AuthToken: utils.NewStringValue(data.Token),
	}

	return api.NewUserGroupResponse(api.ExecuteSuccess, out)
}

// DeleteUserGroup
func (svr *userServer) DeleteUserGroup(ctx context.Context, req *api.UserGroup) *api.Response {
	requestID := utils.ParseRequestID(ctx)
	platformID := utils.ParsePlatformID(ctx)

	if checkErrResp := checkUpdateUserGroup(req); checkErrResp != nil {
		return checkErrResp
	}

	group, err := svr.storage.GetUserGroup(req.GetId().GetValue())
	if err != nil {
		log.Error(err.Error(), utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
		return api.NewUserGroupResponse(api.StoreLayerException, req)
	}
	if group == nil {
		return api.NewUserGroupResponse(api.ExecuteSuccess, req)
	}

	isOwner := utils.ParseIsOwner(ctx)
	userId := utils.ParseUserID(ctx)
	if !isOwner || (group.Owner != userId) {
		return api.NewResponse(api.NotAllowedAccess)
	}

	if err := svr.storage.DeleteUserGroup(group.ID); err != nil {
		log.Error(err.Error(), utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
		return api.NewResponseWithMsg(StoreCode2APICode(err), err.Error())
	}

	msg := fmt.Sprintf("delete user group: name=%v", req.Name)
	log.Info(msg, utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
	svr.RecordHistory(userGroupRecordEntry(ctx, req, group, model.ODelete))

	out := &api.UserGroup{
		Name: req.GetName(),
	}

	return api.NewUserGroupResponse(api.ExecuteSuccess, out)
}

// ListUserGroups
func (svr *userServer) ListUserGroups(ctx context.Context, query map[string]string) *api.BatchQueryResponse {
	searchFilters := make(map[string]string)
	var (
		offset, limit uint32
		err           error
	)
	isOwner := utils.ParseIsOwner(ctx)
	userId := utils.ParseUserID(ctx)

	if isOwner {
		searchFilters["owner"] = userId
		for key, value := range query {
			if _, ok := UserFilterAttributes[key]; !ok {
				log.Errorf("[Auth][UserGroup][Query] attribute(%s) it not allowed", key)
				return api.NewBatchQueryResponseWithMsg(api.InvalidParameter, key+" is not allowed")
			}
			searchFilters[key] = value
		}

		offset, limit, err = utils.ParseOffsetAndLimit(searchFilters)
		if err != nil {
			return api.NewBatchQueryResponse(api.InvalidParameter)
		}

	} else {

	}

	total, users, err := svr.storage.ListUserGroups(searchFilters, offset, limit)
	if err != nil {
		log.Errorf("[Auth][UserGroup][Query] req(%+v) store err: %s", query, err.Error())
		return api.NewBatchQueryResponse(api.StoreLayerException)
	}

	resp := api.NewBatchQueryResponse(api.ExecuteSuccess)
	resp.Amount = utils.NewUInt32Value(total)
	resp.Size = utils.NewUInt32Value(uint32(len(users)))
	resp.UserGroups = enhancedUserGroups2Api(users, userGroup2Api)
	return resp
}

// GetUserGroupToken
func (svr *userServer) GetUserGroupToken(ctx context.Context, req *api.UserGroup) *api.Response {
	if checkErrResp := checkUpdateUserGroup(req); checkErrResp != nil {
		return checkErrResp
	}

	groupCache, err := svr.getUserGroupFromCache(req)
	if err != nil {
		return err
	}

	isOwner := utils.ParseIsOwner(ctx)
	userId := utils.ParseUserID(ctx)
	if !isOwner {
		find := false
		for i := range groupCache.UserIDs {
			if userId == groupCache.UserIDs[i] {
				find = true
			}
		}
		if !find {
			return api.NewResponse(api.NotAllowedAccess)
		}
	} else {
		if groupCache.Owner != userId {
			return api.NewResponse(api.NotAllowedAccess)
		}
	}

	out := &api.UserGroup{
		Id:        utils.NewStringValue(groupCache.ID),
		Name:      utils.NewStringValue(groupCache.Name),
		AuthToken: utils.NewStringValue(groupCache.Token),
	}

	return api.NewUserGroupResponse(api.ExecuteSuccess, out)
}

// DisableUserGroupToken
func (svr *userServer) DisableUserGroupToken(ctx context.Context, req *api.UserGroup) *api.Response {

	return svr.changeUserGroupTokenEnable(ctx, req, true)
}

// EnableUserGroupToken
func (svr *userServer) EnableUserGroupToken(ctx context.Context, req *api.UserGroup) *api.Response {

	return svr.changeUserGroupTokenEnable(ctx, req, false)
}

// changeUserGroupTokenEnable
func (svr *userServer) changeUserGroupTokenEnable(ctx context.Context, req *api.UserGroup, disable bool) *api.Response {
	requestID := utils.ParseRequestID(ctx)
	platformID := utils.ParsePlatformID(ctx)

	if checkErrResp := checkUpdateUserGroup(req); checkErrResp != nil {
		return checkErrResp
	}

	group, err := svr.getUserGroupMustExist(requestID, platformID, req)
	if err != nil {
		return err
	}

	isOwner := utils.ParseIsOwner(ctx)
	userId := utils.ParseUserID(ctx)
	if !isOwner || (group.Owner != userId) {
		return api.NewResponse(api.NotAllowedAccess)
	}

	group.TokenEnable = disable

	if err := svr.storage.UpdateUserGroup(group); err != nil {
		log.Error(err.Error(), utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
		return api.NewResponseWithMsg(StoreCode2APICode(err), err.Error())
	}

	msg := fmt.Sprintf("disable usergroup: name=%v token", req.Name)
	log.Info(msg, utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
	svr.RecordHistory(userGroupRecordEntry(ctx, req, group, model.OUpdate))

	out := &api.User{
		Id:   utils.NewStringValue(group.ID),
		Name: req.GetName(),
	}

	return api.NewUserResponse(api.ExecuteSuccess, out)
}

// RefreshUserGroupToken
func (svr *userServer) RefreshUserGroupToken(ctx context.Context, req *api.UserGroup) *api.Response {
	requestID := utils.ParseRequestID(ctx)
	platformID := utils.ParsePlatformID(ctx)

	if checkErrResp := checkUpdateUserGroup(req); checkErrResp != nil {
		return checkErrResp
	}

	group, errResp := svr.getUserGroupMustExist(requestID, platformID, req)
	if errResp != nil {
		return errResp
	}

	isOwner := utils.ParseIsOwner(ctx)
	userId := utils.ParseUserID(ctx)
	if !isOwner || (group.Owner != userId) {
		return api.NewResponse(api.NotAllowedAccess)
	}

	newToken, err := CreateToken("", group.ID)
	if err != nil {
		log.Error(err.Error(), utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
		return api.NewResponseWithMsg(api.ExecuteException, err.Error())
	}

	group.Token = newToken

	if err := svr.storage.UpdateUserGroup(group); err != nil {
		log.Error(err.Error(), utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
		return api.NewResponseWithMsg(StoreCode2APICode(err), err.Error())
	}

	msg := fmt.Sprintf("refresh usergroup: name=%v token", req.Name)
	log.Info(msg, utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
	svr.RecordHistory(userGroupRecordEntry(ctx, req, group, model.OUpdate))

	out := &api.User{
		Id:   utils.NewStringValue(group.ID),
		Name: req.GetName(),
	}

	return api.NewUserResponse(api.ExecuteSuccess, out)
}

// BatchAddUserToGroup
func (svr *userServer) BatchAddUserToGroup(ctx context.Context, req *api.UserGroupRelation) *api.BatchWriteResponse {

	return svr.batchOperateUserFromGroup(ctx, req, false)
}

// BatchRemoveUserFromGroup
func (svr *userServer) BatchRemoveUserFromGroup(ctx context.Context, req *api.UserGroupRelation) *api.BatchWriteResponse {

	return svr.batchOperateUserFromGroup(ctx, req, true)
}

func (svr *userServer) batchOperateUserFromGroup(ctx context.Context, req *api.UserGroupRelation, remove bool) *api.BatchWriteResponse {
	resps := api.NewBatchWriteResponse(api.ExecuteSuccess)

	requestID := utils.ParseRequestID(ctx)
	platformID := utils.ParsePlatformID(ctx)
	isOwner := utils.ParseIsOwner(ctx)
	ownerId := utils.ParseUserID(ctx)

	if !isOwner {
		resps.Collect(api.NewResponse(api.NotAllowedAccess))
		return resps
	}

	if checkErrResp := svr.preCheckGroupRelation(req); checkErrResp != nil {
		return checkErrResp
	}

	userIds := req.UserIds
	for i := range userIds {
		userId := userIds[i]
		user := svr.userCache.GetUser(userId.GetValue())
		if user == nil {
			resps.Collect(api.NewUserGroupRelationResponse(api.NotFoundResource, req))
			return resps
		}

		if user.Owner != ownerId {
			resps.Collect(api.NewResponse(api.NotAllowedAccess))
			return resps
		}
	}

	data := createUserGroupRelationModel(req)

	if remove {
		if err := svr.storage.RemoveUserGroupRelation(data); err != nil {
			log.Error(err.Error(), utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
			resps.Collect(api.NewResponseWithMsg(StoreCode2APICode(err), err.Error()))
			return resps
		}

		msg := fmt.Sprintf("batch remove user to group: req(%+v)", req)
		log.Info(msg, utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
	} else {
		if err := svr.storage.AddUserGroupRelation(data); err != nil {
			log.Error(err.Error(), utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
			resps.Collect(api.NewResponseWithMsg(StoreCode2APICode(err), err.Error()))
			return resps
		}

		msg := fmt.Sprintf("batch add user to group: req(%+v)", req)
		log.Info(msg, utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
	}

	resps.Collect(api.NewUserGroupRelationResponse(api.ExecuteSuccess, req))
	return resps
}

func (svr *userServer) getUserGroupMustExist(requestID, platformID string, req *api.UserGroup) (*model.UserGroup, *api.Response) {
	group, err := svr.storage.GetUserGroup(req.Id.GetValue())
	if err != nil {
		log.Error(err.Error(), utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
		return nil, api.NewUserGroupResponse(api.StoreLayerException, req)
	}
	if group == nil {
		return nil, api.NewUserGroupResponse(api.NotFoundResource, req)
	}

	return group, nil
}

func (svr *userServer) getUserGroupFromCache(req *api.UserGroup) (*model.UserGroupDetail, *api.Response) {
	group := svr.userCache.GetUserGroup(req.Id.GetValue())
	if group == nil {
		return nil, api.NewUserGroupResponse(api.NotFoundResource, req)
	}

	return group, nil
}

func (svr *userServer) preCheckGroupRelation(req *api.UserGroupRelation) *api.BatchWriteResponse {
	ok, err := svr.checkGroupExist(req.GetGroupId().GetValue())
	if err != nil {
		log.Errorf("[Auth][UserGroupRelation][Query] check group(%s) exist by store err: %s", req.GetGroupId().GetValue(), err.Error())
		return api.NewBatchWriteResponse(api.StoreLayerException)
	}
	if !ok {
		log.Errorf("[Auth][UserGroupRelation][Query] usergroup=%s not exist", req.GroupId)
		return api.NewBatchWriteResponse(api.NotFoundResource)
	}

	// check users is all exist
	uids := make([]string, len(req.UserIds))
	for i := range req.UserIds {
		uids[i] = req.UserIds[i].GetValue()
	}
	uids = utils.StringSliceDeDuplication(uids)
	users, err := svr.storage.GetUserByIDS(uids)
	if err != nil {
		log.Errorf("[Auth][UserGroupRelation][Query] req(%+v) store err: %s", req, err.Error())
		return api.NewBatchWriteResponse(api.StoreLayerException)
	}
	if len(users) != len(uids) {
		return api.NewBatchWriteResponse(api.InvalidParameter)
	}

	return nil
}

// checkGroupExist 检查用户组是否存在
func (svr *userServer) checkGroupExist(groupId string) (bool, error) {
	group, err := svr.storage.GetUserGroup(groupId)
	if err != nil {
		return false, err
	}
	return group != nil, nil
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

// user 数组转为[]*api.User
func enhancedUsers2Api(users []*model.User, handler User2Api) []*api.User {
	out := make([]*api.User, 0, len(users))
	for _, entry := range users {
		outUser := handler(entry)
		out = append(out, outUser)
	}

	return out
}

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
		Id:      utils.NewStringValue(user.ID),
		Name:    utils.NewStringValue(user.Name),
		Source:  utils.NewStringValue(user.Source),
		Owner:   utils.NewStringValue(user.Owner),
		Comment: utils.NewStringValue(user.Comment),
		Ctime:   utils.NewStringValue(commontime.Time2String(user.CreateTime)),
		Mtime:   utils.NewStringValue(commontime.Time2String(user.ModifyTime)),
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
		Id:      utils.NewStringValue(group.ID),
		Name:    utils.NewStringValue(group.Name),
		Owner:   utils.NewStringValue(group.Owner),
		Comment: utils.NewStringValue(group.Comment),
		Ctime:   utils.NewStringValue(commontime.Time2String(group.CreateTime)),
		Mtime:   utils.NewStringValue(commontime.Time2String(group.ModifyTime)),
	}

	return out
}

// 生成服务的记录entry
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

// 生成服务的记录entry
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

// 生成服务的记录entry
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

func checkCreateUser(req *api.User) *api.Response {
	if req == nil {
		return api.NewUserResponse(api.EmptyRequest, req)
	}

	if err := checkName(req.Name); err != nil {
		return api.NewUserResponse(api.InvalidUserName, req)
	}

	if err := checkOwner(req.Owner); err != nil {
		return api.NewUserResponse(api.InvalidUserOwners, req)
	}

	return nil
}

func checkUpdateUser(req *api.User) *api.Response {
	if req == nil {
		return api.NewUserResponse(api.EmptyRequest, req)
	}

	if req.GetId() == nil || req.GetId().GetValue() == "" {
		return api.NewUserResponse(api.BadRequest, req)
	}

	return nil
}

func diffUserInfo(old *model.User, newUser *api.User) (*api.Response, bool) {
	var needUpdate bool = true

	pwd, err := bcrypt.GenerateFromPassword([]byte(newUser.GetPassword().GetValue()), bcrypt.DefaultCost)
	if err != nil {
		return api.NewResponseWithMsg(api.ExecuteException, err.Error()), false
	}

	if old.Comment != newUser.Comment.GetValue() {
		needUpdate = true
	}

	if string(pwd) != old.Password {
		needUpdate = true
		old.Password = string(pwd)
	}

	return nil, needUpdate
}

func createUserModel(req *api.User) (*model.User, error) {
	return &model.User{
		ID:         utils.NewUUID(),
		Name:       req.GetName().GetValue(),
		Password:   req.GetPassword().GetValue(),
		Owner:      req.GetOwner().GetValue(),
		Source:     req.GetSource().GetValue(),
		Valid:      true,
		Comment:    req.GetComment().GetValue(),
		CreateTime: time.Time{},
		ModifyTime: time.Time{},
	}, nil
}

// ============ user group ============

func diffUserGroupInfo(old *model.UserGroup, newUser *api.UserGroup) (*api.Response, bool) {
	var needUpdate bool = true

	if old.Comment != newUser.Comment.GetValue() {
		needUpdate = true
	}

	return nil, needUpdate
}

// checkCreateUserGroup
func checkCreateUserGroup(req *api.UserGroup) *api.Response {
	if req == nil {
		return api.NewUserGroupResponse(api.EmptyRequest, req)
	}

	if err := checkOwner(req.Owner); err != nil {
		return api.NewUserGroupResponse(api.InvalidUserGroupOwners, req)
	}

	return nil
}

func checkUpdateUserGroup(req *api.UserGroup) *api.Response {
	if req == nil {
		return api.NewUserGroupResponse(api.EmptyRequest, req)
	}

	if req.Id == nil || req.Id.GetValue() == "" {
		return api.NewUserGroupResponse(api.InvalidUserGroupID, req)
	}

	return nil
}

func checkQueryUserGroup(req *api.UserGroup) *api.Response {
	return nil
}

func createUserGroupModel(req *api.UserGroup) *model.UserGroup {
	return nil
}

// ============ user group relation ============

func createUserGroupRelationModel(req *api.UserGroupRelation) *model.UserGroupRelation {
	return nil
}

// ============ common ============

func checkName(name *wrappers.StringValue) *api.Response {
	return nil
}

func checkOwner(owner *wrappers.StringValue) error {
	if owner == nil {
		return errors.New("nil")
	}

	if owner.GetValue() == "" {
		return errors.New("empty")
	}

	if utf8.RuneCountInString(owner.GetValue()) > utils.MaxOwnersLength {
		return errors.New("owners too long")
	}

	return nil
}
