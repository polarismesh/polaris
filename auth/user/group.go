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
	"fmt"
	"time"

	"github.com/gogo/protobuf/jsonpb"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	apisecurity "github.com/polarismesh/specification/source/go/api/v1/security"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"
	"go.uber.org/zap"

	cachetypes "github.com/polarismesh/polaris/cache/api"
	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/model"
	authcommon "github.com/polarismesh/polaris/common/model/auth"
	commonstore "github.com/polarismesh/polaris/common/store"
	commontime "github.com/polarismesh/polaris/common/time"
	"github.com/polarismesh/polaris/common/utils"
)

type (
	// UserGroup2Api is the user group to api
	UserGroup2Api func(user *authcommon.UserGroup) *apisecurity.UserGroup
)

// CreateGroup create a group
func (svr *Server) CreateGroup(ctx context.Context, req *apisecurity.UserGroup) *apiservice.Response {
	ownerID := utils.ParseOwnerID(ctx)
	req.Owner = utils.NewStringValue(ownerID)
	if rsp := svr.preCheckGroupRelation(req.GetRelation()); rsp != nil {
		return rsp
	}

	// 根据 owner + groupname 确定唯一的用户组信息
	group, err := svr.storage.GetGroupByName(req.GetName().GetValue(), ownerID)
	if err != nil {
		log.Error("get group when create", utils.RequestID(ctx), zap.Error(err))
		return api.NewGroupResponse(commonstore.StoreCode2APICode(err), req)
	}

	if group != nil {
		return api.NewGroupResponse(apimodel.Code_UserGroupExisted, req)
	}

	data, err := svr.createGroupModel(req)
	if err != nil {
		log.Error("create group model", utils.RequestID(ctx), zap.Error(err))
		return api.NewAuthResponseWithMsg(apimodel.Code_ExecuteException, err.Error())
	}

	tx, err := svr.storage.StartTx()
	if err != nil {
		log.Error("[Auth][User] create user_group begion storage tx", utils.RequestID(ctx), zap.Error(err))
		return api.NewAuthResponse(apimodel.Code_ExecuteException)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if err := svr.storage.AddGroup(tx, data); err != nil {
		log.Error(err.Error(), utils.RequestID(ctx))
		return api.NewAuthResponseWithMsg(commonstore.StoreCode2APICode(err), err.Error())
	}
	if err := svr.policySvr.PolicyHelper().CreatePrincipal(ctx, tx, authcommon.Principal{
		PrincipalID:   data.ID,
		PrincipalType: authcommon.PrincipalGroup,
		Owner:         data.Owner,
		Name:          data.Name,
	}); err != nil {
		log.Error("[Auth][User] add user_group default policy rule", utils.RequestID(ctx), zap.Error(err))
		return api.NewAuthResponse(commonstore.StoreCode2APICode(err))
	}
	if err := tx.Commit(); err != nil {
		log.Error("[Auth][User] create user_group commit storage tx", utils.RequestID(ctx), zap.Error(err))
		return api.NewAuthResponse(apimodel.Code_ExecuteException)
	}

	log.Info("create group", zap.String("name", req.Name.GetValue()), utils.RequestID(ctx))
	svr.RecordHistory(userGroupRecordEntry(ctx, req, data.UserGroup, model.OCreate))

	req.Id = utils.NewStringValue(data.ID)
	return api.NewGroupResponse(apimodel.Code_ExecuteSuccess, req)
}

// UpdateGroups 批量修改用户组
func (svr *Server) UpdateGroups(
	ctx context.Context, groups []*apisecurity.ModifyUserGroup) *apiservice.BatchWriteResponse {
	resp := api.NewAuthBatchWriteResponse(apimodel.Code_ExecuteSuccess)
	for index := range groups {
		ret := svr.UpdateGroup(ctx, groups[index])
		api.Collect(resp, ret)
	}
	return resp
}

// UpdateGroup 更新用户组
func (svr *Server) UpdateGroup(ctx context.Context, req *apisecurity.ModifyUserGroup) *apiservice.Response {
	if checkErrResp := svr.checkUpdateGroup(ctx, req); checkErrResp != nil {
		return checkErrResp
	}

	data, errResp := svr.getGroupFromDB(req.Id.GetValue())
	if errResp != nil {
		return errResp
	}

	modifyReq, needUpdate := UpdateGroupAttribute(ctx, data.UserGroup, req)
	if !needUpdate {
		log.Info("update group data no change, no need update",
			utils.RequestID(ctx), zap.String("group", req.String()))
		return api.NewModifyGroupResponse(apimodel.Code_NoNeedUpdate, req)
	}

	if err := svr.storage.UpdateGroup(modifyReq); err != nil {
		log.Error("update group", zap.Error(err), utils.RequestID(ctx))
		return api.NewAuthResponseWithMsg(commonstore.StoreCode2APICode(err), err.Error())
	}

	log.Info("update group", zap.String("name", data.Name), utils.RequestID(ctx))
	svr.RecordHistory(modifyUserGroupRecordEntry(ctx, req, data.UserGroup, model.OUpdateGroup))

	return api.NewModifyGroupResponse(apimodel.Code_ExecuteSuccess, req)
}

// DeleteGroups 批量删除用户组
func (svr *Server) DeleteGroups(ctx context.Context, reqs []*apisecurity.UserGroup) *apiservice.BatchWriteResponse {
	resp := api.NewAuthBatchWriteResponse(apimodel.Code_ExecuteSuccess)
	for index := range reqs {
		ret := svr.DeleteGroup(ctx, reqs[index])
		api.Collect(resp, ret)
	}

	return resp
}

// DeleteGroup 删除用户组
func (svr *Server) DeleteGroup(ctx context.Context, req *apisecurity.UserGroup) *apiservice.Response {
	group, err := svr.storage.GetGroup(req.GetId().GetValue())
	if err != nil {
		log.Error("get group from store", utils.RequestID(ctx), zap.Error(err))
		return api.NewGroupResponse(commonstore.StoreCode2APICode(err), req)
	}
	if group == nil {
		return api.NewGroupResponse(apimodel.Code_ExecuteSuccess, req)
	}
	tx, err := svr.storage.StartTx()
	if err != nil {
		log.Error("[Auth][User] delete user_group begion storage tx", utils.RequestID(ctx), zap.Error(err))
		return api.NewAuthResponse(apimodel.Code_ExecuteException)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if err := svr.storage.DeleteGroup(tx, group); err != nil {
		log.Error("delete group from store", utils.RequestID(ctx), zap.Error(err))
		return api.NewAuthResponseWithMsg(commonstore.StoreCode2APICode(err), err.Error())
	}
	if err := svr.policySvr.PolicyHelper().CleanPrincipal(ctx, tx, authcommon.Principal{
		PrincipalID:   group.ID,
		PrincipalType: authcommon.PrincipalGroup,
		Owner:         group.Owner,
	}); err != nil {
		log.Error("[Auth][User] delete user_group from policy server", utils.RequestID(ctx), zap.Error(err))
		return api.NewAuthResponse(commonstore.StoreCode2APICode(err))
	}

	log.Info("delete group", utils.RequestID(ctx), zap.String("name", req.Name.GetValue()))
	svr.RecordHistory(userGroupRecordEntry(ctx, req, group.UserGroup, model.ODelete))

	return api.NewGroupResponse(apimodel.Code_ExecuteSuccess, req)
}

// GetGroups 查看用户组
func (svr *Server) GetGroups(ctx context.Context, filters map[string]string) *apiservice.BatchQueryResponse {
	offset, limit, _ := utils.ParseOffsetAndLimit(filters)
	total, groups, err := svr.cacheMgr.User().QueryUserGroups(ctx, cachetypes.UserGroupSearchArgs{
		Filters: filters,
		Offset:  offset,
		Limit:   limit,
	})
	if err != nil {
		log.Error("[Auth][Group] list user_group from store", utils.RequestID(ctx),
			zap.Any("filters", filters), zap.Error(err))
		return api.NewAuthBatchQueryResponse(commonstore.StoreCode2APICode(err))
	}

	resp := api.NewAuthBatchQueryResponse(apimodel.Code_ExecuteSuccess)
	resp.Amount = utils.NewUInt32Value(total)
	resp.Size = utils.NewUInt32Value(uint32(len(groups)))
	resp.UserGroups = enhancedGroups2Api(groups, userGroup2Api)

	for index := range resp.UserGroups {
		group := resp.UserGroups[index]
		cacheVal := svr.cacheMgr.User().GetGroup(group.Id.Value)
		if cacheVal == nil {
			group.UserCount = utils.NewUInt32Value(0)
		} else {
			group.UserCount = utils.NewUInt32Value(uint32(len(cacheVal.UserIds)))
		}
	}
	return resp
}

// GetGroup 查看对应用户组下的用户信息
func (svr *Server) GetGroup(ctx context.Context, req *apisecurity.UserGroup) *apiservice.Response {
	if req.GetId().GetValue() == "" {
		return api.NewAuthResponse(apimodel.Code_InvalidUserGroupID)
	}
	group, errResp := svr.getGroupFromDB(req.Id.Value)
	if errResp != nil {
		return errResp
	}
	return api.NewGroupResponse(apimodel.Code_ExecuteSuccess, svr.userGroupDetail2Api(group))
}

// GetGroupToken 查看用户组的token
func (svr *Server) GetGroupToken(ctx context.Context, req *apisecurity.UserGroup) *apiservice.Response {
	if req.GetId().GetValue() == "" {
		return api.NewAuthResponse(apimodel.Code_InvalidUserGroupID)
	}

	group := svr.cacheMgr.User().GetGroup(req.Id.GetValue())
	if group == nil {
		return api.NewGroupResponse(apimodel.Code_NotFoundUserGroup, req)
	}

	req.AuthToken = utils.NewStringValue(group.Token)
	req.TokenEnable = utils.NewBoolValue(group.TokenEnable)

	return api.NewGroupResponse(apimodel.Code_ExecuteSuccess, req)
}

// EnableGroupToken 调整用户组 token 的使用状态 (禁用｜开启)
func (svr *Server) EnableGroupToken(ctx context.Context, req *apisecurity.UserGroup) *apiservice.Response {
	group, errResp := svr.getGroupFromDB(req.Id.GetValue())
	if errResp != nil {
		return errResp
	}

	group.TokenEnable = req.TokenEnable.GetValue()

	modifyReq := &authcommon.ModifyUserGroup{
		ID:          group.ID,
		Owner:       group.Owner,
		Token:       group.Token,
		TokenEnable: group.TokenEnable,
		Comment:     group.Comment,
	}

	if err := svr.storage.UpdateGroup(modifyReq); err != nil {
		log.Error(err.Error(), utils.RequestID(ctx))
		return api.NewAuthResponseWithMsg(commonstore.StoreCode2APICode(err), err.Error())
	}

	log.Info("update group token", zap.String("id", req.Id.GetValue()),
		zap.Bool("enable", group.TokenEnable), utils.RequestID(ctx))
	svr.RecordHistory(userGroupRecordEntry(ctx, req, group.UserGroup, model.OUpdateToken))

	return api.NewGroupResponse(apimodel.Code_ExecuteSuccess, req)
}

// ResetGroupToken 刷新用户组的token
func (svr *Server) ResetGroupToken(ctx context.Context, req *apisecurity.UserGroup) *apiservice.Response {
	var (
		requestID      = utils.ParseRequestID(ctx)
		platformID     = utils.ParsePlatformID(ctx)
		group, errResp = svr.getGroupFromDB(req.Id.GetValue())
	)

	if errResp != nil {
		return errResp
	}

	if !utils.ParseIsOwner(ctx) || (group.Owner != utils.ParseUserID(ctx)) {
		return api.NewAuthResponse(apimodel.Code_NotAllowedAccess)
	}

	newToken, err := createGroupToken(group.ID, svr.authOpt.Salt)
	if err != nil {
		log.Error("reset group token", utils.ZapRequestID(requestID),
			utils.ZapPlatformID(platformID), zap.Error(err))
		return api.NewAuthResponseWithMsg(apimodel.Code_ExecuteException, err.Error())
	}

	group.Token = newToken
	modifyReq := &authcommon.ModifyUserGroup{
		ID:          group.ID,
		Owner:       group.Owner,
		Token:       group.Token,
		TokenEnable: group.TokenEnable,
		Comment:     group.Comment,
	}

	if err := svr.storage.UpdateGroup(modifyReq); err != nil {
		log.Error(err.Error(), utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
		return api.NewAuthResponseWithMsg(commonstore.StoreCode2APICode(err), err.Error())
	}

	log.Info("reset group token", zap.String("group-id", req.Id.GetValue()),
		utils.ZapRequestID(requestID), utils.ZapPlatformID(platformID))
	svr.RecordHistory(userGroupRecordEntry(ctx, req, group.UserGroup, model.OUpdate))

	req.AuthToken = utils.NewStringValue(newToken)

	return api.NewGroupResponse(apimodel.Code_ExecuteSuccess, req)
}

// getGroupFromDB 获取用户组
func (svr *Server) getGroupFromDB(id string) (*authcommon.UserGroupDetail, *apiservice.Response) {
	group, err := svr.storage.GetGroup(id)
	if err != nil {
		log.Error("get group from store", zap.Error(err))
		return nil, api.NewAuthResponseWithMsg(commonstore.StoreCode2APICode(err), err.Error())
	}
	if group == nil {
		return nil, api.NewAuthResponse(apimodel.Code_NotFoundUserGroup)
	}
	return group, nil
}

// preCheckGroupRelation 检查用户-用户组关联关系中，对应的用户信息是否存在，即不能添加一个不存在的用户到用户组
func (svr *Server) preCheckGroupRelation(req *apisecurity.UserGroupRelation) *apiservice.Response {
	// 检查该关系中所有的用户是否存在
	uIDs := make([]string, len(req.GetUsers()))
	for i := range req.GetUsers() {
		uIDs[i] = req.GetUsers()[i].GetId().GetValue()
	}

	uIDs = utils.StringSliceDeDuplication(uIDs)
	for i := range uIDs {
		user := svr.cacheMgr.User().GetUserByID(uIDs[i])
		if user == nil {
			return api.NewGroupRelationResponse(apimodel.Code_NotFoundUser, req)
		}
	}

	return nil
}

// checkUpdateGroup 检查用户组的更新请求
func (svr *Server) checkUpdateGroup(ctx context.Context, req *apisecurity.ModifyUserGroup) *apiservice.Response {
	if req == nil {
		return api.NewModifyGroupResponse(apimodel.Code_EmptyRequest, req)
	}
	if req.Id == nil || req.Id.GetValue() == "" {
		return api.NewModifyGroupResponse(apimodel.Code_InvalidUserGroupID, req)
	}
	if rsp := svr.preCheckGroupRelation(req.GetAddRelations()); rsp != nil {
		return rsp
	}
	return nil
}

// UpdateGroupAttribute 更新计算用户组更新时的结构体数据，并判断是否需要执行更新操作
func UpdateGroupAttribute(ctx context.Context, old *authcommon.UserGroup, newUser *apisecurity.ModifyUserGroup) (
	*authcommon.ModifyUserGroup, bool) {
	var (
		needUpdate bool
		ret        = &authcommon.ModifyUserGroup{
			ID:          old.ID,
			Token:       old.Token,
			TokenEnable: old.TokenEnable,
			Comment:     old.Comment,
		}
	)

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

// enhancedGroups2Api 数组专为 []*apisecurity.UserGroup
func enhancedGroups2Api(groups []*authcommon.UserGroupDetail, handler UserGroup2Api) []*apisecurity.UserGroup {
	out := make([]*apisecurity.UserGroup, 0, len(groups))
	for k := range groups {
		out = append(out, handler(groups[k].UserGroup))
	}

	return out
}

// createGroupModel 创建用户组的存储模型
func (svr *Server) createGroupModel(req *apisecurity.UserGroup) (group *authcommon.UserGroupDetail, err error) {
	ids := make(map[string]struct{}, len(req.GetRelation().GetUsers()))
	for index := range req.GetRelation().GetUsers() {
		ids[req.GetRelation().GetUsers()[index].GetId().GetValue()] = struct{}{}
	}

	group = &authcommon.UserGroupDetail{
		UserGroup: &authcommon.UserGroup{
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

	if group.Token, err = createGroupToken(group.ID, svr.authOpt.Salt); err != nil {
		return nil, err
	}
	return group, nil
}

// model.UserGroup 转为 api.UserGroup
func userGroup2Api(group *authcommon.UserGroup) *apisecurity.UserGroup {
	if group == nil {
		return nil
	}

	// note: 不包括token，token比较特殊
	out := &apisecurity.UserGroup{
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
func (svr *Server) userGroupDetail2Api(group *authcommon.UserGroupDetail) *apisecurity.UserGroup {
	if group == nil {
		return nil
	}

	users := make([]*apisecurity.User, 0, len(group.UserIds))
	for id := range group.UserIds {
		user := svr.cacheMgr.User().GetUserByID(id)
		users = append(users, &apisecurity.User{
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
	out := &apisecurity.UserGroup{
		Id:          utils.NewStringValue(group.ID),
		Name:        utils.NewStringValue(group.Name),
		Owner:       utils.NewStringValue(group.Owner),
		TokenEnable: utils.NewBoolValue(group.TokenEnable),
		Comment:     utils.NewStringValue(group.Comment),
		Ctime:       utils.NewStringValue(commontime.Time2String(group.CreateTime)),
		Mtime:       utils.NewStringValue(commontime.Time2String(group.ModifyTime)),
		Relation: &apisecurity.UserGroupRelation{
			Users: users,
		},
		UserCount: utils.NewUInt32Value(uint32(len(users))),
	}

	return out
}

// userGroupRecordEntry 生成用户组的记录entry
func userGroupRecordEntry(ctx context.Context, req *apisecurity.UserGroup, md *authcommon.UserGroup,
	operationType model.OperationType) *model.RecordEntry {

	marshaler := jsonpb.Marshaler{}
	datail, _ := marshaler.MarshalToString(req)

	entry := &model.RecordEntry{
		ResourceType:  model.RUserGroup,
		ResourceName:  fmt.Sprintf("%s(%s)", md.Name, md.ID),
		OperationType: operationType,
		Operator:      utils.ParseOperator(ctx),
		Detail:        datail,
		HappenTime:    time.Now(),
	}

	return entry
}

// 生成修改用户组的记录entry
func modifyUserGroupRecordEntry(ctx context.Context, req *apisecurity.ModifyUserGroup, md *authcommon.UserGroup,
	operationType model.OperationType) *model.RecordEntry {

	marshaler := jsonpb.Marshaler{}
	detail, _ := marshaler.MarshalToString(req)

	entry := &model.RecordEntry{
		ResourceType:  model.RUserGroup,
		ResourceName:  fmt.Sprintf("%s(%s)", md.Name, md.ID),
		OperationType: operationType,
		Operator:      utils.ParseOperator(ctx),
		Detail:        detail,
		HappenTime:    time.Now(),
	}

	return entry
}

// 生成用户-用户组关联关系的记录entry
func userRelationRecordEntry(ctx context.Context, req *apisecurity.UserGroupRelation, md *authcommon.UserGroup,
	operationType model.OperationType) *model.RecordEntry {

	marshaler := jsonpb.Marshaler{}
	detail, _ := marshaler.MarshalToString(req)

	entry := &model.RecordEntry{
		ResourceType:  model.RUserGroupRelation,
		ResourceName:  fmt.Sprintf("%s(%s)", md.Name, md.ID),
		OperationType: operationType,
		Operator:      utils.ParseOperator(ctx),
		Detail:        detail,
		HappenTime:    time.Now(),
	}

	return entry
}

func defaultUserGroupPolicy(u *authcommon.UserGroupDetail) *authcommon.StrategyDetail {
	// Create the user's default weight policy
	return &authcommon.StrategyDetail{
		ID:        utils.NewUUID(),
		Name:      authcommon.BuildDefaultStrategyName(authcommon.PrincipalGroup, u.Name),
		Action:    apisecurity.AuthAction_READ_WRITE.String(),
		Default:   true,
		Owner:     u.Owner,
		Revision:  utils.NewUUID(),
		Resources: []authcommon.StrategyResource{},
		Valid:     true,
		Comment:   "Default Strategy",
	}
}
