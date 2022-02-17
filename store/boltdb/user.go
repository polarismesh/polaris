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
	"sort"
	"strings"
	"time"

	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/common/utils"
	"github.com/polarismesh/polaris-server/store"
)

const (
	tblUser              string = "user"
	UserFieldID          string = "ID"
	UserFieldName        string = "Name"
	UserFieldPassword    string = "Password"
	UserFieldOwner       string = "Owner"
	UserFieldSource      string = "Source"
	UserFieldType        string = "Type"
	UserFieldToken       string = "Token"
	UserFieldTokenEnable string = "TokenEnable"
	UserFieldValid       string = "Valid"
	UserFieldComment     string = "Comment"
	UserFieldCreateTime  string = "CreateTime"
	UserFieldModifyTime  string = "ModifyTime"
)

var (
	MultipleUserFound error = errors.New("multiple user found")
)

// userStore
type userStore struct {
	handler BoltHandler
}

// AddUser
func (us *userStore) AddUser(user *model.User) error {

	initUser(user)

	if user.ID == "" || user.Name == "" || user.Source == "" ||
		user.Owner == "" || user.Token == "" {
		return store.NewStatusError(store.EmptyParamsErr, "add user missing some params")
	}

	return us.handler.SaveValue(tblUser, user.ID, user)
}

// UpdateUser
func (us *userStore) UpdateUser(user *model.User) error {
	if user.ID == "" || user.Token == "" {
		return store.NewStatusError(store.EmptyParamsErr, "update user missing some params")
	}

	properties := make(map[string]interface{})
	properties[UserFieldComment] = user.Comment
	properties[UserFieldToken] = user.Token
	properties[UserFieldTokenEnable] = user.TokenEnable
	properties[UserFieldPassword] = user.Password
	properties[UserFieldModifyTime] = time.Now()

	return us.handler.UpdateValue(tblUser, user.ID, properties)
}

// DeleteUser
func (us *userStore) DeleteUser(user *model.User) error {
	if user.ID == "" {
		return store.NewStatusError(store.EmptyParamsErr, "delete user missing some params")
	}

	return us.handler.DeleteValues(tblUser, []string{user.ID}, true)
}

// GetUser
func (us *userStore) GetUser(id string) (*model.User, error) {

	if id == "" {
		return nil, store.NewStatusError(store.EmptyParamsErr, "get user missing some params")
	}

	ret, err := us.handler.LoadValues(tblUser, []string{id}, &model.User{})
	if err != nil {
		return nil, err
	}
	if len(ret) == 0 {
		return nil, nil
	}
	if len(ret) > 1 {
		return nil, MultipleUserFound
	}

	return ret[id].(*model.User), errors.New("implement me")
}

// GetUserByName
func (us *userStore) GetUserByName(name, ownerId string) (*model.User, error) {
	if name == "" {
		return nil, store.NewStatusError(store.EmptyParamsErr, "get user missing some params")
	}

	ret, err := us.handler.LoadValuesByFilter(tblUser, []string{UserFieldName, UserFieldOwner}, &model.User{}, func(m map[string]interface{}) bool {

		saveName, _ := m[UserFieldName].(string)
		saveOwner, _ := m[UserFieldOwner].(string)

		return saveName == name && saveOwner == ownerId
	})
	if err != nil {
		return nil, err
	}
	if len(ret) == 0 {
		return nil, nil
	}
	if len(ret) > 0 {
		return nil, MultipleUserFound
	}

	var id string
	for k := range ret {
		id = k
		break
	}

	return ret[id].(*model.User), nil
}

// GetUserByIds
func (us *userStore) GetUserByIds(ids []string) ([]*model.User, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	ret, err := us.handler.LoadValues(tblUser, ids, &model.User{})
	if err != nil {
		return nil, err
	}
	if len(ret) == 0 {
		return nil, nil
	}

	users := make([]*model.User, 0, len(ids))

	for k := range ret {
		users = append(users, ret[k].(*model.User))
	}

	return users, nil
}

// GetSubCount 获取子账户的个数
func (us *userStore) GetSubCount(user *model.User) (uint32, error) {
	return 0, nil
}

// GetUsers
func (us *userStore) GetUsers(filters map[string]string, offset uint32, limit uint32) (uint32, []*model.User, error) {

	if _, ok := filters["group_id"]; ok {
		return us.getGroupUsers(filters, offset, limit)
	}

	return us.getUsers(filters, offset, limit)
}

// getUsers
// "name":   1,
// "owner":  1,
// "source": 1,
func (us *userStore) getUsers(filters map[string]string, offset uint32, limit uint32) (uint32, []*model.User, error) {

	fields := []string{UserFieldName, UserFieldOwner, UserFieldSource, UserFieldValid}

	ret, err := us.handler.LoadValuesByFilter(tblUser, fields, &model.User{},
		func(m map[string]interface{}) bool {

			valid, ok := m[UserFieldValid].(bool)
			if ok && !valid {
				return false
			}

			saveName, _ := m[UserFieldName].(string)
			saveOwner, _ := m[UserFieldOwner].(string)
			saveSource, _ := m[UserFieldSource].(string)

			if name, ok := filters["name"]; ok {
				if utils.IsWildName(name) {
					if !strings.Contains(saveName, name[:len(name)-1]) {
						return false
					}
				} else {
					if saveName != name {
						return false
					}
				}
			}

			if owner, ok := filters["owner"]; ok {
				if owner != saveOwner {
					return false
				}
			}

			if source, ok := filters["source"]; ok {
				if source != saveSource {
					return false
				}
			}

			return true
		})

	if err != nil {
		return 0, nil, err
	}
	if len(ret) == 0 {
		return 0, nil, nil
	}

	return uint32(len(ret)), doPage(ret, offset, limit), nil

}

func (us *userStore) getGroupUsers(filters map[string]string, offset uint32, limit uint32) (uint32, []*model.User, error) {

	groupId := filters["group_id"]
	delete(filters, "group_id")

	ret, err := us.handler.LoadValues(tblGroup, []string{groupId}, &model.UserGroupDetail{})
	if err != nil {
		return 0, nil, err
	}
	if len(ret) == 0 {
		return 0, nil, nil
	}
	if len(ret) > 1 {
		return 0, nil, MultipleGroupFound
	}
	group := ret[groupId].(*model.UserGroupDetail)

	userIds := make([]string, 0, len(group.UserIds))
	for k := range group.UserIds {
		userIds = append(userIds, k)
	}

	ret, err = us.handler.LoadValues(tblUser, userIds, &model.User{})
	if err != nil {
		return 0, nil, err
	}

	predicate := func(user *model.User) bool {
		if name, ok := filters["name"]; ok {
			if utils.IsWildName(name) {
				if !strings.Contains(user.Name, name[:len(name)-1]) {
					return false
				}
			} else {
				if user.Name != name {
					return false
				}
			}
		}

		if owner, ok := filters["owner"]; ok {
			if owner != user.Owner {
				return false
			}
		}

		if source, ok := filters["source"]; ok {
			if source != user.Source {
				return false
			}
		}

		return true
	}

	users := make(map[string]interface{})
	for k := range ret {
		val := ret[k]
		if predicate(val.(*model.User)) {
			users[k] = val.(*model.User)
		}
	}

	return uint32(len(ret)), doPage(users, offset, limit), err
}

// GetUsersForCache
func (us *userStore) GetUsersForCache(mtime time.Time, firstUpdate bool) ([]*model.User, error) {
	ret, err := us.handler.LoadValuesByFilter(tblUser, []string{UserFieldModifyTime}, &model.User{},
		func(m map[string]interface{}) bool {
			mt := m[UserFieldModifyTime].(time.Time)
			isAfter := mt.After(mtime)
			return isAfter
		})
	if err != nil {
		return nil, err
	}

	users := make([]*model.User, 0, len(ret))

	for k := range ret {
		val := ret[k]
		users = append(users, val.(*model.User))
	}

	return users, nil
}

// GetUserRelationGroupCount
func (us *userStore) GetUserRelationGroupCount(userId string) (uint32, error) {
	return 0, nil
}

// doPage 进行分页
func doPage(ret map[string]interface{}, offset, limit uint32) []*model.User {

	users := make([]*model.User, 0, len(ret))

	beginIndex := offset
	endIndex := beginIndex + limit
	totalCount := uint32(len(ret))

	if totalCount == 0 {
		return users
	}
	if beginIndex >= endIndex {
		return users
	}
	if beginIndex >= totalCount {
		return users
	}
	if endIndex > totalCount {
		endIndex = totalCount
	}
	for k := range ret {
		users = append(users, ret[k].(*model.User))
	}

	sort.Slice(users, func(i, j int) bool {
		return users[i].ModifyTime.After(users[j].ModifyTime)
	})

	return users[beginIndex:endIndex]

}

func initUser(user *model.User) {
	if user != nil {
		user.Valid = true
		user.CreateTime = time.Now()
		user.ModifyTime = time.Now()
	}
}
