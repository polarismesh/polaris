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

	apisecurity "github.com/polarismesh/specification/source/go/api/v1/security"
	"go.uber.org/zap"

	"github.com/polarismesh/polaris/common/utils"
)

var (
	ErrGroupNotExist = errors.New("group not exist")
	ErrUserNotExist  = errors.New("user not exist")
)

type DefaultUserHelper struct {
	svr *Server
}

// CheckUserInGroup 检查用户是否在用户组中
func (helper *DefaultUserHelper) CheckUserInGroup(ctx context.Context,
	group *apisecurity.UserGroup, user *apisecurity.User) bool {

	cacheMgr := helper.svr.cacheMgr
	return cacheMgr.User().IsUserInGroup(user.GetId().GetValue(), group.GetId().GetValue())
}

// CheckGroupsExist 批量检查用户组是否存在
func (helper *DefaultUserHelper) CheckGroupsExist(ctx context.Context, groups []*apisecurity.UserGroup) error {
	cacheMgr := helper.svr.cacheMgr
	for i := range groups {
		item := groups[i]
		if ret := cacheMgr.User().GetGroup(item.GetId().GetValue()); ret == nil {
			return ErrGroupNotExist
		}
	}
	return nil
}

// CheckUsersExist 批量检查用户是否存在
func (helper *DefaultUserHelper) CheckUsersExist(ctx context.Context, users []*apisecurity.User) error {
	cacheMgr := helper.svr.cacheMgr
	for i := range users {
		item := users[i]
		if ret := cacheMgr.User().GetUserByID(item.GetId().GetValue()); ret == nil {
			return ErrUserNotExist
		}
	}
	return nil
}

// GetUserOwnGroup 查询某个用户所在的所有用户组
func (helper *DefaultUserHelper) GetUserOwnGroup(ctx context.Context, user *apisecurity.User) []*apisecurity.UserGroup {
	cacheMgr := helper.svr.cacheMgr

	ids := cacheMgr.User().GetUserLinkGroupIds(user.GetId().GetValue())
	groups := make([]*apisecurity.UserGroup, 0, len(ids))
	for i := range ids {
		item := ids[i]
		group := cacheMgr.User().GetGroup(item)
		if group != nil {
			groups = append(groups, group.ToSpec())
		}
	}
	return groups
}

// GetUser 查询用户信息
func (helper *DefaultUserHelper) GetUser(ctx context.Context, user *apisecurity.User) *apisecurity.User {
	cacheMgr := helper.svr.cacheMgr
	if user.GetName().GetValue() == "" {
		return cacheMgr.User().GetUserByID(user.GetId().GetValue()).ToSpec()
	}
	owner := cacheMgr.User().GetUserByID(utils.ParseUserID(ctx))
	if owner == nil {
		return nil
	}
	return cacheMgr.User().GetUserByName(user.GetName().GetValue(), owner.Name).ToSpec()
}

func (helper *DefaultUserHelper) GetUserByID(ctx context.Context, id string) *apisecurity.User {
	cacheMgr := helper.svr.cacheMgr
	saveUser := cacheMgr.User().GetUserByID(id)
	if saveUser == nil {
		saveUser, _ = helper.svr.storage.GetUser(id)
	}
	return saveUser.ToSpec()
}

// GetGroup 查询用户组信息
func (helper *DefaultUserHelper) GetGroup(ctx context.Context, req *apisecurity.UserGroup) *apisecurity.UserGroup {
	cacheMgr := helper.svr.cacheMgr
	saveVal := cacheMgr.User().GetGroup(req.GetId().GetValue())
	if saveVal != nil {
		return saveVal.ToSpec()
	}
	// 从数据库在获取一次
	saveVal, err := helper.svr.storage.GetGroup(req.GetId().GetValue())
	if err != nil {
		log.Error("[Auth][UserHelper] get user_group from store", zap.String("id", req.GetId().GetValue()),
			zap.Error(err))
		return nil
	}
	return saveVal.ToSpec()
}
