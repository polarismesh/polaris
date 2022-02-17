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

package boltdb

import (
	"errors"
	"time"

	"github.com/polarismesh/polaris-server/common/model"
)

var (
	MultipleGroupFound error = errors.New("multiple group found")
)

const (
	tblGroup string = "group"

	GroupFieldModifyTime string = "ModifyTime"
)

// groupStore
type groupStore struct {
	handler BoltHandler
}

// AddUserGroup
func (us *groupStore) AddGroup(group *model.UserGroupDetail) error {
	return nil
}

// UpdateUserGroup
func (us *groupStore) UpdateGroup(group *model.ModifyUserGroup) error {
	return nil
}

// DeleteUserGroup
func (us *groupStore) DeleteGroup(group *model.UserGroupDetail) error {
	return nil
}

// AddUserGroupRelation
func (us *groupStore) AddGroupRelation(relations *model.UserGroupRelation) error {
	return nil
}

// RemoveUserGroupRelation
func (us *groupStore) RemoveGroupRelation(relations *model.UserGroupRelation) error {
	return nil
}

// GetUserGroup
func (us *groupStore) GetGroup(id string) (*model.UserGroupDetail, error) {
	return nil, nil
}

// GetGroupByName
func (us *groupStore) GetGroupByName(name, owner string) (*model.UserGroup, error) {
	return nil, nil
}

// GetGroups
func (us *groupStore) GetGroups(filters map[string]string, offset uint32, limit uint32) (uint32, []*model.UserGroup, error) {
	return 0, nil, nil
}

// ListGroupByUser
func (us *groupStore) ListGroupByUser(filters map[string]string, offset uint32, limit uint32) (uint32, []*model.UserGroup, error) {
	return 0, nil, nil
}

// ListUserByGroup
func (us *groupStore) ListUserByGroup(filters map[string]string, offset uint32, limit uint32) (uint32, []*model.User, error) {
	return 0, nil, nil
}

// GetUserGroupsForCache
func (us *groupStore) GetGroupsForCache(mtime time.Time, firstUpdate bool) ([]*model.UserGroupDetail, error) {
	ret, err := us.handler.LoadValuesByFilter(tblGroup, []string{GroupFieldModifyTime}, &model.UserGroupDetail{},
		func(m map[string]interface{}) bool {
			mt := m[GroupFieldModifyTime].(time.Time)
			isAfter := mt.After(mtime)
			return isAfter
		})
	if err != nil {
		return nil, err
	}

	groups := make([]*model.UserGroupDetail, 0, len(ret))

	for k := range ret {
		val := ret[k]
		groups = append(groups, val.(*model.UserGroupDetail))
	}

	return groups, nil
}
