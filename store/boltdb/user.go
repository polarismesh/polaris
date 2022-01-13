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

// userStore
type userStore struct {
	handler BoltHandler
}

// AddUser
//  @param user
//  @return error
func (us *userStore) AddUser(user *model.User) error {
	return errors.New("implement me")
}

// UpdateUser
//  @param user
//  @return error
func (us *userStore) UpdateUser(user *model.User) error {
	return errors.New("implement me")
}

// DeleteUser
//  @param id
//  @return error
func (us *userStore) DeleteUser(id string) error {
	return errors.New("implement me")
}

// GetUser
//  @param id
//  @return *model.User
//  @return error
func (us *userStore) GetUser(id string) (*model.User, error) {
	return nil, errors.New("implement me")
}

// GetUserByName
//  @receiver us
//  @param name
//  @param ownerId
//  @return *model.User
//  @return error
func (us *userStore) GetUserByName(name, ownerId string) (*model.User, error) {
	return nil, errors.New("implement me")
}

// GetUserByIDS
//  @param ids
//  @return []*model.User
//  @return error
func (us *userStore) GetUserByIDS(ids []string) ([]*model.User, error) {
	return nil, errors.New("implement me")
}

// ListUsers
//  @param filters
//  @param offset
//  @param limit
//  @return uint32
//  @return []*model.User
//  @return error
func (us *userStore) ListUsers(filters map[string]string, offset uint32, limit uint32) (uint32, []*model.User, error) {
	return 0, nil, errors.New("implement me")
}

// GetUsersForCache
//  @param mtime
//  @param firstUpdate
//  @return []*model.User
//  @return error
func (us *userStore) GetUsersForCache(mtime time.Time, firstUpdate bool) ([]*model.User, error) {
	return nil, errors.New("implement me")
}

// AddUserGroup
//  @param group
//  @return error
func (us *userStore) AddUserGroup(group *model.UserGroupDetail) error {
	return errors.New("implement me")
}

// UpdateUserGroup
//  @param group
//  @return error
func (us *userStore) UpdateUserGroup(group *model.ModifyUserGroup) error {
	return errors.New("implement me")
}

// DeleteUserGroup
//  @param id
//  @return error
func (us *userStore) DeleteUserGroup(id string) error {
	return errors.New("implement me")
}

// AddUserGroupRelation
//  @param relations
//  @return error
func (us *userStore) AddUserGroupRelation(relations *model.UserGroupRelation) error {
	return errors.New("implement me")
}

// RemoveUserGroupRelation
//  @param relations
//  @return error
func (us *userStore) RemoveUserGroupRelation(relations *model.UserGroupRelation) error {
	return errors.New("implement me")
}

// GetUserGroup
//  @param id
//  @return *model.UserGroup
//  @return error
func (us *userStore) GetUserGroup(id string) (*model.UserGroup, error) {
	return nil, errors.New("implement me")
}

// GetUserGroupByName
//  @param name
//  @return *model.UserGroup
//  @return error
func (us *userStore) GetUserGroupByName(name string) (*model.UserGroup, error) {
	return nil, errors.New("implement me")
}

// ListUserGroups
//  @param filters
//  @param offset
//  @param limit
//  @return uint32
//  @return []*model.UserGroup
//  @return error
func (us *userStore) ListUserGroups(filters map[string]string, offset uint32, limit uint32) (uint32, []*model.UserGroup, error) {
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
func (us *userStore) ListUserByGroup(filters map[string]string, offset uint32, limit uint32) (uint32, []*model.User, error) {
	return 0, nil, errors.New("implement me")
}

// GetUserGroupsForCache
//  @param mtime
//  @param firstUpdate
//  @return []*model.UserGroupDetail
//  @return error
func (us *userStore) GetUserGroupsForCache(mtime time.Time, firstUpdate bool) ([]*model.UserGroupDetail, error) {
	return nil, errors.New("implement me")
}
