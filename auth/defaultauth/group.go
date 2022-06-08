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

	"go.uber.org/zap"

	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/common/model"
	commontime "github.com/polarismesh/polaris-server/common/time"
	"github.com/polarismesh/polaris-server/common/utils"
)

type (
	// UserGroup2Api is the user group to api
	UserGroup2Api func(user *model.UserGroup) *api.UserGroup
)

var (
	// UserLinkGroupAttributes is the user link group attributes
	UserLinkGroupAttributes = map[string]bool{
		"id":        true,
		"user_id":   true,
		"user_name": true,
		"group_id":  true,
		"name":      true,
		"offset":    true,
		"limit":     true,
	}
)

// CreateGroup create a group
func (svr *server) CreateGroup(ctx context.Context, req *api.UserGroup) *api.Response {
	var (
		requestID  = utils.ParseRequestID(ctx)
		platformID = utils.ParsePlatformID(ctx)
		ownerID    = utils.ParseOwnerID(ctx)
	)

	req.Owner = utils.NewStringValue(ownerID)
	if checkErrResp := svr.checkCreateGroup(ctx, req); checkErrResp != nil {
		return checkErrResp
	}

	// 根据 owner + groupname 确定唯一的用户组信息
	group, err := svr.storage.GetGroupByName(req.Name.GetValue(), ownerID)
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

	req.Id = utils.NewStringValue(data.ID)

	return api.NewGroupResponse(api.ExecuteSuccess, req)
}

// UpdateGroups 批量修改用户组
func (svr *server) UpdateGroups(ctx context.Context, groups []*api.ModifyUserGroup) *api.BatchWriteResponse {
	resp := api.NewBatchWriteResponse(api.ExecuteSuccess)
	for index := range groups {
		req := groups[index]
		ret := svr.UpdateGroup(ctx, req)
		resp.Collect(ret)
	}

	return resp
}

// UpdateGroup 更新用户组
func (svr *server) UpdateGroup(ctx context.Context, req *api.ModifyUserGroup) *api.Response {
	var (
		requestID  = utils.ParseRequestID(ctx)
		platformID = utils.ParsePlatformID(ctx)
	)

	if checkErrResp := svr.checkUpdateGroup(ctx, req); checkErrResp != nil {
		return checkErrResp
	}

	data, errResp := svr.getGroupFromDB(req.Id.GetValue())
	if errResp != nil {
		return errResp
	}

	modifyReq, needUpdate := updateGroupAttribute(ctx, data.UserGroup, req)
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
	svr.RecordHistory(modifyUserGroupRecordEntry(ctx, req, data.UserGroup, model.OUpdateGroup))

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
	var (
		requestID = utils.ParseRequestID(ctx)
		userID    = utils.ParseUserID(ctx)
	)

	group, err := svr.storage.GetGroup(req.GetId().GetValue())
	if err != nil {
		log.AuthScope().Error("get group from store", utils.ZapRequestID(requestID), zap.Error(err))
		return api.NewGroupResponse(api.StoreLayerException, req)
	}
	if group == nil {
		return api.NewGroupResponse(api.ExecuteSuccess, req)
	}

	if utils.ParseUserRole(ctx) != model.AdminUserRole {
		if group.Owner != userID {
			return api.NewResponse(api.NotAllowedAccess)
		}
	}

	if err := svr.storage.DeleteGroup(group); err != nil {
		log.AuthScope().Error("delete group from store", utils.ZapRequestID(requestID), zap.Error(err))
		return api.NewResponseWithMsg(StoreCode2APICode(err), err.Error())
	}

	log.AuthScope().Info("delete group", utils.ZapRequestID(requestID), zap.String("name", req.Name.GetValue()))
	svr.RecordHistory(userGroupRecordEntry(ctx, req, group.UserGroup, model.ODelete))

	return api.NewGroupResponse(api.ExecuteSuccess, req)
}

// GetGroups 查看用户组
func (svr *server) GetGroups(ctx context.Context, query map[string]string) *api.BatchQueryResponse {
	requestID := utils.ParseRequestID(ctx)

	log.AuthScope().Info("[Auth][Group] origin get groups query params",
		utils.ZapRequestID(requestID), zap.Any("query", query))

	var (
		offset, limit uint32
		err           error
	)

	offset, limit, err = utils.ParseOffsetAndLimit(query)
	if err != nil {
		return api.NewBatchQueryResponse(api.InvalidParameter)
	}

	searchFilters, errResp := parseGroupSearchArgs(ctx, query)
	if errResp != nil {
		return errResp
	}

	total, groups, err := svr.storage.GetGroups(searchFilters, offset, limit)
	if err != nil {
		log.AuthScope().Errorf("[Auth][Group] get groups req(%+v) store err: %s", query, err.Error())
		return api.NewBatchQueryResponse(api.StoreLayerException)
	}

	resp := api.NewBatchQueryResponse(api.ExecuteSuccess)
	resp.Amount = utils.NewUInt32Value(total)
	resp.Size = utils.NewUInt32Value(uint32(len(groups)))
	resp.UserGroups = enhancedGroups2Api(groups, userGroup2Api)

	svr.fillGroupUserCount(resp.UserGroups)

	return resp
}

func parseGroupSearchArgs(ctx context.Context, query map[string]string) (map[string]string, *api.BatchQueryResponse) {
	searchFilters := make(map[string]string, len(query))
	for key, value := range query {
		if _, ok := UserLinkGroupAttributes[key]; !ok {
			log.AuthScope().Errorf("[Auth][Group] get groups attribute(%s) it not allowed", key)
			return nil, api.NewBatchQueryResponseWithMsg(api.InvalidParameter, key+" is not allowed")
		}

		searchFilters[key] = value
	}

	// 如果当前不是管理员角色的话，只能查询该用户所关联的用户组列表以及自己创建的用户组
	if utils.ParseUserRole(ctx) != model.AdminUserRole {
		if !utils.ParseIsOwner(ctx) {
			searchFilters["user_id"] = utils.ParseUserID(ctx)
		}
		searchFilters["owner"] = utils.ParseOwnerID(ctx)
	}

	return searchFilters, nil
}

// GetGroup 查看对应用户组下的用户信息
func (svr *server) GetGroup(ctx context.Context, req *api.UserGroup) *api.Response {
	if req.GetId().GetValue() == "" {
		return api.NewResponse(api.InvalidUserGroupID)
	}

	group, errResp := svr.getGroupFromDB(req.Id.Value)
	if errResp != nil {
		return errResp
	}

	if utils.ParseUserRole(ctx) != model.AdminUserRole {
		userID := utils.ParseUserID(ctx)
		isGroupOwner := group.Owner == userID
		_, find := group.UserIds[userID]
		if !isGroupOwner && !find {
			log.AuthScope().Error("can't see group info", zap.String("user", userID),
				zap.String("group", req.GetId().GetValue()), zap.Bool("group-owner", isGroupOwner),
				zap.Bool("in-group", find))
			return api.NewResponse(api.NotAllowedAccess)
		}
	}

	return api.NewGroupResponse(api.ExecuteSuccess, svr.userGroupDetail2Api(group))
}

// GetGroupToken 查看用户组的token
func (svr *server) GetGroupToken(ctx context.Context, req *api.UserGroup) *api.Response {
	if req.GetId().GetValue() == "" {
		return api.NewResponse(api.InvalidUserGroupID)
	}

	groupCache, errResp := svr.getGroupFromCache(req)
	if errResp != nil {
		return errResp
	}

	if utils.ParseUserRole(ctx) != model.AdminUserRole {
		userID := utils.ParseUserID(ctx)
		isGroupOwner := groupCache.Owner == userID
		_, find := groupCache.UserIds[userID]
		if !isGroupOwner && !find {
			log.AuthScope().Error("can't see group token", zap.String("user", userID),
				zap.String("group", req.GetId().GetValue()), zap.Bool("group-owner", isGroupOwner),
				zap.Bool("in-group", find))
			return api.NewResponse(api.NotAllowedAccess)
		}
	}

	req.AuthToken = utils.NewStringValue(groupCache.Token)
	req.TokenEnable = utils.NewBoolValue(groupCache.TokenEnable)

	return api.NewGroupResponse(api.ExecuteSuccess, req)
}

// UpdateGroupToken 调整用户组 token 的使用状态 (禁用｜开启)
func (svr *server) UpdateGroupToken(ctx context.Context, req *api.UserGroup) *api.Response {
	var (
		requestID      = utils.ParseRequestID(ctx)
		platformID     = utils.ParsePlatformID(ctx)
		group, errResp = svr.getGroupFromDB(req.Id.GetValue())
	)

	if errResp != nil {
		return errResp
	}

	if utils.ParseUserRole(ctx) != model.AdminUserRole {
		userID := utils.ParseUserID(ctx)
		if group.Owner != userID {
			return api.NewResponse(api.NotAllowedAccess)
		}
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
	svr.RecordHistory(userGroupRecordEntry(ctx, req, group.UserGroup, model.OUpdateToken))

	return api.NewGroupResponse(api.ExecuteSuccess, req)
}

// ResetGroupToken 刷新用户组的token
func (svr *server) ResetGroupToken(ctx context.Context, req *api.UserGroup) *api.Response {
	var (
		requestID      = utils.ParseRequestID(ctx)
		platformID     = utils.ParsePlatformID(ctx)
		group, errResp = svr.getGroupFromDB(req.Id.GetValue())
	)

	if errResp != nil {
		return errResp
	}

	if !utils.ParseIsOwner(ctx) || (group.Owner != utils.ParseUserID(ctx)) {
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
	svr.RecordHistory(userGroupRecordEntry(ctx, req, group.UserGroup, model.OUpdate))

	req.AuthToken = utils.NewStringValue(newToken)

	return api.NewGroupResponse(api.ExecuteSuccess, req)
}

// getGroupFromDB 获取用户组
func (svr *server) getGroupFromDB(id string) (*model.UserGroupDetail, *api.Response) {
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
func (svr *server) preCheckGroupRelation(groupID string, req *api.UserGroupRelation) (*model.UserGroupDetail,
	*api.Response) {
	group := svr.cacheMgn.User().GetGroup(groupID)
	if group == nil {
		return nil, api.NewResponse(api.NotFoundUserGroup)
	}

	// 检查该关系中所有的用户是否存在
	uIDs := make([]string, len(req.GetUsers()))
	for i := range req.GetUsers() {
		uIDs[i] = req.GetUsers()[i].GetId().GetValue()
	}

	uIDs = utils.StringSliceDeDuplication(uIDs)
	for i := range uIDs {
		user := svr.cacheMgn.User().GetUserByID(uIDs[i])
		if user == nil {
			return group, api.NewGroupRelationResponse(api.NotFoundUser, req)
		}
	}

	return group, nil
}

// checkCreateGroup 检查创建用户组的请求
func (svr *server) checkCreateGroup(_ context.Context, req *api.UserGroup) *api.Response {
	if req == nil {
		return api.NewGroupResponse(api.EmptyRequest, req)
	}

	users := req.GetRelation().GetUsers()
	for i := range users {
		user := svr.cacheMgn.User().GetUserByID(users[i].GetId().GetValue())
		if user == nil {
			return api.NewGroupRelationResponse(api.NotFoundUser, req.GetRelation())
		}
	}

	return nil
}

// checkUpdateGroup 检查用户组的更新请求
func (svr *server) checkUpdateGroup(ctx context.Context, req *api.ModifyUserGroup) *api.Response {
	userID := utils.ParseUserID(ctx)
	isOwner := utils.ParseIsOwner(ctx)

	if req == nil {
		return api.NewModifyGroupResponse(api.EmptyRequest, req)
	}

	if req.Id == nil || req.Id.GetValue() == "" {
		return api.NewModifyGroupResponse(api.InvalidUserGroupID, req)
	}

	group, checkErrResp := svr.preCheckGroupRelation(req.GetId().GetValue(), req.GetAddRelations())
	if checkErrResp != nil {
		return checkErrResp
	}

	// 满足以下情况才可以进行操作
	// 1.管理员
	// 2.自己在这个用户组里面
	// 3.自己是这个用户组的owner角色
	if utils.ParseUserRole(ctx) != model.AdminUserRole {
		_, inGroup := group.UserIds[userID]
		if !inGroup && group.Owner != userID {
			return api.NewResponse(api.NotAllowedAccess)
		}

		// 如果当前用户只是在这个组里面，但不是该用户组的owner，那只能添加用户，不能删除用户
		if inGroup && !isOwner && len(req.GetRemoveRelations().GetUsers()) != 0 {
			return api.NewResponseWithMsg(api.NotAllowedAccess, "only main account can remove user from usergroup")
		}
	}

	return nil
}

func (svr *server) fillGroupUserCount(groups []*api.UserGroup) {
	groupCache := svr.cacheMgn.User()

	for index := range groups {
		group := groups[index]
		cacheVal := groupCache.GetGroup(group.Id.Value)
		if cacheVal == nil {
			group.UserCount = utils.NewUInt32Value(0)
		} else {
			group.UserCount = utils.NewUInt32Value(uint32(len(cacheVal.UserIds)))
		}

	}

}

// updateGroupAttribute 更新计算用户组更新时的结构体数据，并判断是否需要执行更新操作
func updateGroupAttribute(ctx context.Context, old *model.UserGroup, newUser *api.ModifyUserGroup) (
	*model.ModifyUserGroup, bool) {
	var needUpdate bool
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
	if len(newUser.GetAddRelations().GetUsers()) != 0 {
		needUpdate = true
		ids := make([]string, 0, len(newUser.GetAddRelations().GetUsers()))
		for index := range newUser.GetAddRelations().GetUsers() {
			ids = append(ids, newUser.GetAddRelations().GetUsers()[index].GetId().GetValue())
		}
		ret.AddUserIds = ids
	}

	if len(newUser.GetRemoveRelations().GetUsers()) != 0 {
		needUpdate = true
		ids := make([]string, 0, len(newUser.GetRemoveRelations().GetUsers()))
		for index := range newUser.GetRemoveRelations().GetUsers() {
			ids = append(ids, newUser.GetRemoveRelations().GetUsers()[index].GetId().GetValue())
		}
		ret.RemoveUserIds = ids
	}

	return ret, needUpdate
}

// enhancedGroups2Api 数组专为 []*api.UserGroup
func enhancedGroups2Api(groups []*model.UserGroup, handler UserGroup2Api) []*api.UserGroup {
	out := make([]*api.UserGroup, 0, len(groups))
	for k := range groups {
		out = append(out, handler(groups[k]))
	}

	return out
}

// createGroupModel 创建用户组的存储模型
func createGroupModel(req *api.UserGroup) (*model.UserGroupDetail, error) {
	ids := make(map[string]struct{}, len(req.GetRelation().GetUsers()))
	for index := range req.GetRelation().GetUsers() {
		ids[req.GetRelation().GetUsers()[index].GetId().GetValue()] = struct{}{}
	}

	group := &model.UserGroupDetail{
		UserGroup: &model.UserGroup{
			ID:          utils.NewUUID(),
			Name:        req.GetName().GetValue(),
			Owner:       req.GetOwner().GetValue(),
			TokenEnable: true,
			Valid:       true,
			Comment:     req.GetComment().GetValue(),
			CreateTime:  time.Now(),
			ModifyTime:  time.Now(),
		},
		UserIds: ids,
	}

	newToken, err := createGroupToken(group.ID)
	if err != nil {
		return nil, err
	}

	group.Token = newToken

	return group, nil
}

// model.UserGroup 转为 api.UserGroup
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

// model.UserGroupDetail 转为 api.UserGroup，并且主动填充 user 的信息数据
func (svr *server) userGroupDetail2Api(group *model.UserGroupDetail) *api.UserGroup {
	if group == nil {
		return nil
	}

	users := make([]*api.User, 0, len(group.UserIds))
	for id := range group.UserIds {
		user := svr.cacheMgn.User().GetUserByID(id)
		users = append(users, &api.User{
			Id:          utils.NewStringValue(user.ID),
			Name:        utils.NewStringValue(user.Name),
			Source:      utils.NewStringValue(user.Source),
			Comment:     utils.NewStringValue(user.Comment),
			TokenEnable: utils.NewBoolValue(user.TokenEnable),
			Ctime:       utils.NewStringValue(commontime.Time2String(user.CreateTime)),
			Mtime:       utils.NewStringValue(commontime.Time2String(user.ModifyTime)),
		})
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
		Relation: &api.UserGroupRelation{
			Users: users,
		},
		UserCount: utils.NewUInt32Value(uint32(len(users))),
	}

	return out
}

// userGroupRecordEntry 生成用户组的记录entry
func userGroupRecordEntry(ctx context.Context, _ *api.UserGroup, md *model.UserGroup,
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
func modifyUserGroupRecordEntry(ctx context.Context, _ *api.ModifyUserGroup, md *model.UserGroup,
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
func userRelationRecordEntry(ctx context.Context, _ *api.UserGroupRelation, md *model.UserGroup,
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
