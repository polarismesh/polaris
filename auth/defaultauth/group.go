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
	"go.uber.org/zap"
)

type (
	UserGroup2Api func(user *model.UserGroup) *api.UserGroup
)

var (
	UserLinkGroupAttributes = map[string]int{
		"id":         1,
		"user_id":    1,
		"user_name":  1,
		"group_id":   1,
		"group_name": 1,
		"offset":     1,
		"limit":      1,
	}
)

// CreateGroup
func (svr *server) CreateGroup(ctx context.Context, req *api.UserGroup) *api.Response {
	requestID := utils.ParseRequestID(ctx)
	platformID := utils.ParsePlatformID(ctx)
	userId := utils.ParseUserID(ctx)
	ownerId := utils.ParseOwnerID(ctx)

	req.Owner = utils.NewStringValue(userId)

	if checkErrResp := svr.checkCreateGroup(ctx, req); checkErrResp != nil {
		return checkErrResp
	}

	// 根据 owner + groupname 确定唯一的用户组信息
	group, err := svr.storage.GetUserByName(req.Name.GetValue(), ownerId)
	if err != nil {
		log.AuthScope().Error("get group when create", utils.ZapRequestID(requestID),
			utils.ZapPlatformID(platformID), zap.Error(err))
		return api.NewGroupResponse(api.StoreLayerException, req)
	}
	if group != nil {
		return api.NewGroupResponse(api.UserGroupExisted, req)
	}

	data, err := createGroupModel(req)
	if err != nil {
		log.AuthScope().Error("create group model", utils.ZapRequestID(requestID),
			utils.ZapPlatformID(platformID), zap.Error(err))
		return api.NewResponseWithMsg(api.ExecuteException, err.Error())
	}

	if err := svr.storage.AddGroup(data); err != nil {
		log.AuthScope().Error(err.Error(), utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
		return api.NewResponseWithMsg(StoreCode2APICode(err), err.Error())
	}

	log.AuthScope().Info("create group", zap.String("name", req.Name.GetValue()), utils.ZapRequestID(requestID),
		utils.ZapPlatformID(platformID))
	svr.RecordHistory(userGroupRecordEntry(ctx, req, data.UserGroup, model.OCreate))

	return api.NewGroupResponse(api.ExecuteSuccess, req)
}

// UpdateGroup 更新用户组
func (svr *server) UpdateGroup(ctx context.Context, req *api.ModifyUserGroup) *api.Response {
	requestID := utils.ParseRequestID(ctx)
	platformID := utils.ParsePlatformID(ctx)

	if checkErrResp := svr.checkUpdateGroup(ctx, req); checkErrResp != nil {
		return checkErrResp
	}

	data, errResp := svr.getGroupFromDB(requestID, platformID, req.Id.GetValue())
	if errResp != nil {
		return errResp
	}

	modifyReq, needUpdate := updateGroupAttribute(ctx, data, req)

	if !needUpdate {
		log.AuthScope().Info("update group data no change, no need update",
			utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID), zap.String("group", req.String()))
		return api.NewModifyGroupResponse(api.NoNeedUpdate, req)
	}

	if err := svr.storage.UpdateGroup(modifyReq); err != nil {
		log.AuthScope().Error(err.Error(), utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
		return api.NewResponseWithMsg(StoreCode2APICode(err), err.Error())
	}

	log.AuthScope().Info("update group", zap.String("name", data.Name), utils.ZapRequestID(requestID),
		utils.ZapPlatformID(platformID))
	svr.RecordHistory(modifyUserGroupRecordEntry(ctx, req, data, model.OUpdateUserGroup))

	return api.NewModifyGroupResponse(api.ExecuteSuccess, req)
}

// DeleteGroups 批量删除用户组
func (svr *server) DeleteGroups(ctx context.Context, reqs []*api.UserGroup) *api.BatchWriteResponse {
	resp := api.NewBatchWriteResponse(api.ExecuteSuccess)

	for index := range reqs {
		ret := svr.DeleteGroup(ctx, reqs[index])

		resp.Collect(ret)
	}

	return resp
}

// DeleteGroup 删除用户组
func (svr *server) DeleteGroup(ctx context.Context, req *api.UserGroup) *api.Response {
	requestID := utils.ParseRequestID(ctx)
	platformID := utils.ParsePlatformID(ctx)
	isOwner := utils.ParseIsOwner(ctx)
	userId := utils.ParseOwnerID(ctx)

	group, err := svr.storage.GetGroup(req.GetId().GetValue())
	if err != nil {
		log.AuthScope().Error(err.Error(), utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
		return api.NewGroupResponse(api.StoreLayerException, req)
	}
	if group == nil {
		return api.NewGroupResponse(api.ExecuteSuccess, req)
	}

	if !isOwner || (group.Owner != userId) {
		return api.NewResponse(api.NotAllowedAccess)
	}

	if err := svr.storage.DeleteGroup(group.ID); err != nil {
		log.AuthScope().Error(err.Error(), utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
		return api.NewResponseWithMsg(StoreCode2APICode(err), err.Error())
	}

	log.AuthScope().Info("delete group", zap.String("name", req.Name.GetValue()), utils.ZapRequestID(requestID),
		utils.ZapPlatformID(platformID))
	svr.RecordHistory(userGroupRecordEntry(ctx, req, group, model.ODelete))

	return api.NewGroupResponse(api.ExecuteSuccess, req)
}

// ListUserGroups 查看用户组
func (svr *server) GetGroups(ctx context.Context, query map[string]string) *api.BatchQueryResponse {
	searchFilters := make(map[string]string)
	var (
		offset, limit uint32
		err           error
	)

	for key, value := range query {
		if _, ok := UserLinkGroupAttributes[key]; !ok {
			log.AuthScope().Errorf("[Auth][Group] get groups attribute(%s) it not allowed", key)
			return api.NewBatchQueryResponseWithMsg(api.InvalidParameter, key+" is not allowed")
		}
		searchFilters[key] = value
	}

	offset, limit, err = utils.ParseOffsetAndLimit(searchFilters)
	if err != nil {
		return api.NewBatchQueryResponse(api.InvalidParameter)
	}

	total, users, err := svr.storage.GetGroups(searchFilters, offset, limit)
	if err != nil {
		log.AuthScope().Errorf("[Auth][Group] get groups req(%+v) store err: %s", query, err.Error())
		return api.NewBatchQueryResponse(api.StoreLayerException)
	}

	resp := api.NewBatchQueryResponse(api.ExecuteSuccess)
	resp.Amount = utils.NewUInt32Value(total)
	resp.Size = utils.NewUInt32Value(uint32(len(users)))
	resp.UserGroups = enhancedUserGroups2Api(users, userGroup2Api)
	return resp
}

// GetGroupUsers 查看对应用户组下的用户信息
func (svr *server) GetGroupUsers(ctx context.Context, query map[string]string) *api.BatchQueryResponse {
	searchFilters := make(map[string]string)
	var (
		offset, limit uint32
		err           error
	)

	for key, value := range query {
		if _, ok := UserLinkGroupAttributes[key]; !ok {
			log.AuthScope().Errorf("[Auth][Group] get group users attribute(%s) it not allowed", key)
			return api.NewBatchQueryResponseWithMsg(api.InvalidParameter, key+" is not allowed")
		}
		searchFilters[key] = value
	}
	offset, limit, err = utils.ParseOffsetAndLimit(searchFilters)
	if err != nil {
		return api.NewBatchQueryResponse(api.InvalidParameter)
	}

	total, users, err := svr.storage.GetUsers(searchFilters, offset, limit)
	if err != nil {
		log.AuthScope().Errorf("[Auth][Group] get group users req(%+v) store err: %s", query, err.Error())
		return api.NewBatchQueryResponse(api.StoreLayerException)
	}

	resp := api.NewBatchQueryResponse(api.ExecuteSuccess)
	resp.Amount = utils.NewUInt32Value(total)
	resp.Size = utils.NewUInt32Value(uint32(len(users)))
	resp.Users = enhancedUsers2Api(users, user2Api)
	return resp
}

// GetGroupToken 查看用户组的token
func (svr *server) GetGroupToken(ctx context.Context, req *api.UserGroup) *api.Response {
	isOwner := utils.ParseIsOwner(ctx)
	userId := utils.ParseUserID(ctx)
	ownerId := utils.ParseOwnerID(ctx)
	if req.GetId().GetValue() == "" {
		return api.NewResponse(api.InvalidParameter)
	}

	groupCache, errResp := svr.getGroupFromCache(req)
	if errResp != nil {
		return errResp
	}

	if !isOwner {
		if _, find := groupCache.UserIDs[userId]; !find {
			return api.NewResponse(api.NotAllowedAccess)
		}
	} else {
		if groupCache.Owner != ownerId && utils.ParseUserRole(ctx) != model.AdminUserRole {
			return api.NewResponse(api.NotAllowedAccess)
		}
	}

	req.AuthToken = utils.NewStringValue(groupCache.Token)
	req.TokenEnable = utils.NewBoolValue(groupCache.TokenEnable)

	return api.NewGroupResponse(api.ExecuteSuccess, req)
}

// UpdateGroupToken 调整用户组 token 的使用状态 (禁用｜开启)
func (svr *server) UpdateGroupToken(ctx context.Context, req *api.UserGroup) *api.Response {
	requestID := utils.ParseRequestID(ctx)
	platformID := utils.ParsePlatformID(ctx)

	group, errResp := svr.getGroupFromDB(requestID, platformID, req.Id.GetValue())
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

	if err := svr.storage.UpdateGroup(modifyReq); err != nil {
		log.AuthScope().Error(err.Error(), utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
		return api.NewResponseWithMsg(StoreCode2APICode(err), err.Error())
	}

	log.AuthScope().Info("update group token", zap.String("id", req.Id.GetValue()),
		zap.Bool("enable", group.TokenEnable), utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
	svr.RecordHistory(userGroupRecordEntry(ctx, req, group, model.OUpdateToken))

	return api.NewGroupResponse(api.ExecuteSuccess, req)
}

// ResetGroupToken 刷新用户组的token，刷新时会重置 token 的状态为 enable
func (svr *server) ResetGroupToken(ctx context.Context, req *api.UserGroup) *api.Response {
	requestID := utils.ParseRequestID(ctx)
	platformID := utils.ParsePlatformID(ctx)

	group, errResp := svr.getGroupFromDB(requestID, platformID, req.Id.GetValue())
	if errResp != nil {
		return errResp
	}

	isOwner := utils.ParseIsOwner(ctx)
	ownerId := utils.ParseOwnerID(ctx)
	if !isOwner || (group.Owner != ownerId) {
		return api.NewResponse(api.NotAllowedAccess)
	}

	newToken, err := createGroupToken(group.ID)
	if err != nil {
		log.AuthScope().Error("reset group token", utils.ZapRequestID(requestID),
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

	if err := svr.storage.UpdateGroup(modifyReq); err != nil {
		log.AuthScope().Error(err.Error(), utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
		return api.NewResponseWithMsg(StoreCode2APICode(err), err.Error())
	}

	log.AuthScope().Info("reset group token", zap.String("group-id", req.Id.GetValue()),
		utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
	svr.RecordHistory(userGroupRecordEntry(ctx, req, group, model.OUpdate))

	req.AuthToken = utils.NewStringValue(newToken)

	return api.NewGroupResponse(api.ExecuteSuccess, req)
}

// getGroupFromDB 获取用户组
func (svr *server) getGroupFromDB(requestID, platformID string, id string) (*model.UserGroup, *api.Response) {
	group, err := svr.storage.GetGroup(id)
	if err != nil {
		log.Error("get group from store", zap.Error(err))
		return nil, api.NewResponseWithMsg(api.StoreLayerException, err.Error())
	}
	if group == nil {
		return nil, api.NewResponse(api.NotFoundUserGroup)
	}

	return group, nil
}

// getGroupFromCache 从缓存中获取用户组信息数据
func (svr *server) getGroupFromCache(req *api.UserGroup) (*model.UserGroupDetail, *api.Response) {
	group := svr.cacheMgn.User().GetGroup(req.Id.GetValue())
	if group == nil {
		return nil, api.NewGroupResponse(api.NotFoundUserGroup, req)
	}

	return group, nil
}

// preCheckGroupRelation 检查用户-用户组关联关系中，对应的用户信息是否存在，即不能添加一个不存在的用户到用户组
func (svr *server) preCheckGroupRelation(groupId string, req *api.UserGroupRelation) (*model.UserGroupDetail, *api.Response) {
	group := svr.cacheMgn.User().GetGroup(groupId)
	if group == nil {
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
		user := svr.cacheMgn.User().GetUserByID(userId)
		if user == nil {
			return group, api.NewGroupRelationResponse(api.NotFoundUser, req)
		}
	}
	return group, nil
}

// checkCreateGroup 检查创建用户组的请求
func (svr *server) checkCreateGroup(ctx context.Context, req *api.UserGroup) *api.Response {

	if req == nil {
		return api.NewGroupResponse(api.EmptyRequest, req)
	}

	if err := checkOwner(req.Owner); err != nil {
		resp := api.NewGroupResponse(api.InvalidUserGroupOwners, req)
		resp.Info = utils.NewStringValue(err.Error())
		return resp
	}

	userIds := req.GetRelation().GetUserIds()
	for i := range userIds {
		userId := userIds[i]
		user := svr.cacheMgn.User().GetUserByID(userId.GetValue())
		if user == nil {
			return api.NewGroupRelationResponse(api.NotFoundUser, req.GetRelation())
		}
	}

	return nil
}

// checkUpdateGroup 检查用户组的更新请求
func (svr *server) checkUpdateGroup(ctx context.Context, req *api.ModifyUserGroup) *api.Response {
	userId := utils.ParseUserID(ctx)
	isOwner := utils.ParseIsOwner(ctx)

	if req == nil {
		return api.NewModifyGroupResponse(api.EmptyRequest, req)
	}

	if req.Id == nil || req.Id.GetValue() == "" {
		return api.NewModifyGroupResponse(api.InvalidUserGroupID, req)
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

	// 如果当前用户只是在这个组里面，但不是该用户组的owner，那只能添加用户，不能删除用户
	if inGroup && !isOwner && len(req.GetRemoveRelation().UserIds) != 0 {
		return api.NewResponseWithMsg(api.NotAllowedAccess, "only main account can remove user from usergroup")
	}

	return nil
}

// updateGroupAttribute 更新计算用户组更新时的结构体数据，并判断是否需要执行更新操作
func updateGroupAttribute(ctx context.Context, old *model.UserGroup, newUser *api.ModifyUserGroup) (
	*model.ModifyUserGroup, bool) {

	var needUpdate bool = false

	ret := &model.ModifyUserGroup{
		ID:          old.ID,
		Token:       old.Token,
		TokenEnable: old.TokenEnable,
		Comment:     old.Comment,
	}

	// 只有 owner 可以修改这个属性
	if utils.ParseIsOwner(ctx) {
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

// usergroup 数组专为 []*api.UserGroup
func enhancedUserGroups2Api(groups []*model.UserGroup, handler UserGroup2Api) []*api.UserGroup {
	out := make([]*api.UserGroup, 0, len(groups))
	for _, entry := range groups {
		outUser := handler(entry)
		out = append(out, outUser)
	}

	return out
}

// createGroupModel 创建用户组的存储模型
func createGroupModel(req *api.UserGroup) (*model.UserGroupDetail, error) {

	ids := make(map[string]struct{}, len(req.GetRelation().GetUserIds()))
	for index := range req.GetRelation().GetUserIds() {
		ids[req.GetRelation().GetUserIds()[index].GetValue()] = struct{}{}
	}

	group := &model.UserGroupDetail{
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

	newToken, err := createGroupToken(group.ID)
	if err != nil {
		return nil, err
	}

	group.Token = newToken

	return group, nil
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
