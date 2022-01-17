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

// groupStore
type groupStore struct {
	handler BoltHandler
}

// AddUserGroup
//  @param group
//  @return error
func (us *groupStore) AddGroup(group *model.UserGroupDetail) error {
	return errors.New("implement me")
}

// UpdateUserGroup
//  @param group
//  @return error
func (us *groupStore) UpdateGroup(group *model.ModifyUserGroup) error {
	return errors.New("implement me")
}

// DeleteUserGroup
//  @param id
//  @return error
func (us *groupStore) DeleteGroup(id string) error {
	return errors.New("implement me")
}

// AddUserGroupRelation
//  @param relations
//  @return error
func (us *groupStore) AddGroupRelation(relations *model.UserGroupRelation) error {
	return errors.New("implement me")
}

// RemoveUserGroupRelation
//  @param relations
//  @return error
func (us *groupStore) RemoveGroupRelation(relations *model.UserGroupRelation) error {
	return errors.New("implement me")
}

// GetUserGroup
//  @param id
//  @return *model.UserGroup
//  @return error
func (us *groupStore) GetGroup(id string) (*model.UserGroup, error) {
	return nil, errors.New("implement me")
}

// GetGroupByName
func (us *groupStore) GetGroupByName(name, owner string) (*model.UserGroup, error) {
	return nil, errors.New("implement me")
}

// GetGroups
func (us *groupStore) GetGroups(filters map[string]string, offset uint32, limit uint32) (uint32, []*model.UserGroup, error) {
	return 0, nil, errors.New("implement me")
}

// ListGroupByUser
//  @receiver us
//  @param filters
//  @param offset
//  @param limit
//  @return uint32
//  @return []*model.UserGroup
//  @return error
func (us *groupStore) ListGroupByUser(filters map[string]string, offset uint32, limit uint32) (uint32, []*model.UserGroup, error) {
	return 0, nil, errors.New("implement me")
}

// ListUserByGroup
//  @receiver us
//  @param filters
//  @param offset
//  @param limit
//  @return uint32
//  @return []*model.UserGroup
//  @return error
func (us *groupStore) ListUserByGroup(filters map[string]string, offset uint32, limit uint32) (uint32, []*model.User, error) {
	return 0, nil, errors.New("implement me")
}

// GetUserGroupsForCache
//  @param mtime
//  @param firstUpdate
//  @return []*model.UserGroupDetail
//  @return error
func (us *groupStore) GetGroupsForCache(mtime time.Time, firstUpdate bool) ([]*model.UserGroupDetail, error) {
	return nil, errors.New("implement me")
}
