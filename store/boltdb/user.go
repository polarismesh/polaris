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
	"time"

	bolt "go.etcd.io/bbolt"
	"go.uber.org/zap"

	authcommon "github.com/polarismesh/polaris/common/model/auth"
	"github.com/polarismesh/polaris/store"
)

const (
	// 用户数据 scope
	tblUser string = "user"

	// UserFieldID 用户ID字段
	UserFieldID string = "ID"
	// UserFieldName 用户名字段
	UserFieldName string = "Name"
	// UserFieldPassword 用户密码字段
	UserFieldPassword string = "Password"
	// UserFieldOwner 用户Owner字段
	UserFieldOwner string = "Owner"
	// UserFieldSource 用户来源字段
	UserFieldSource string = "Source"
	// UserFieldType 用户类型字段
	UserFieldType string = "Type"
	// UserFieldToken 用户Token字段
	UserFieldToken string = "Token"
	// UserFieldTokenEnable 用户Token是否可用字段
	UserFieldTokenEnable string = "TokenEnable"
	// UserFieldValid 用户逻辑删除字段
	UserFieldValid string = "Valid"
	// UserFieldComment 用户备注字段
	UserFieldComment string = "Comment"
	// UserFieldCreateTime 用户创建时间字段
	UserFieldCreateTime string = "CreateTime"
	// UserFieldModifyTime 用户修改时间字段
	UserFieldModifyTime string = "ModifyTime"
	// UserFieldMobile 用户手机号信息
	UserFieldMobile string = "Mobile"
	// UserFieldEmail 用户邮箱信息
	UserFieldEmail string = "Email"
)

var (
	// ErrMultipleUserFound 多个用户
	ErrMultipleUserFound = errors.New("multiple user found")
)

// userStore
type userStore struct {
	handler BoltHandler
}

// GetMainUser .
func (u *userStore) GetMainUser() (*authcommon.User, error) {
	users, err := u.handler.LoadValuesAll(tblUser, &userForStore{})
	if err != nil {
		log.Error("[Store][User] get main user", zap.Error(err))
		return nil, store.Error(err)
	}

	for i := range users {
		user := users[i].(*userForStore)
		if user.Type == int(authcommon.OwnerUserRole) {
			return converToUserModel(user), nil
		}
	}

	return nil, nil
}

// AddUser 添加用户
func (us *userStore) AddUser(tx store.Tx, user *authcommon.User) error {

	initUser(user)

	if user.ID == "" || user.Name == "" || user.Source == "" || user.Token == "" {
		return store.NewStatusError(store.EmptyParamsErr, "add user missing some params")
	}

	dbTx := tx.GetDelegateTx().(*bolt.Tx)
	owner := user.Owner
	if owner == "" {
		owner = user.ID
	}
	// 添加用户信息
	if err := saveValue(dbTx, tblUser, user.ID, converToUserStore(user)); err != nil {
		log.Error("[Store][User] save user fail", zap.Error(err), zap.String("name", user.Name))
		return err
	}
	return nil
}

// UpdateUser
func (us *userStore) UpdateUser(user *authcommon.User) error {
	if user.ID == "" || user.Token == "" {
		return store.NewStatusError(store.EmptyParamsErr, "update user missing some params")
	}

	properties := make(map[string]interface{})
	properties[UserFieldComment] = user.Comment
	properties[UserFieldToken] = user.Token
	properties[UserFieldTokenEnable] = user.TokenEnable
	properties[UserFieldEmail] = user.Email
	properties[UserFieldMobile] = user.Mobile
	properties[UserFieldPassword] = user.Password
	properties[UserFieldModifyTime] = time.Now()

	err := us.handler.UpdateValue(tblUser, user.ID, properties)
	if err != nil {
		log.Error("[Store][User] update user fail", zap.Error(err), zap.String("id", user.ID))
		return err
	}

	return nil
}

// DeleteUser 删除用户
func (us *userStore) DeleteUser(tx store.Tx, user *authcommon.User) error {
	if user.ID == "" {
		return store.NewStatusError(store.EmptyParamsErr, "delete user missing some params")
	}
	dbTx := tx.GetDelegateTx().(*bolt.Tx)

	properties := make(map[string]interface{})
	properties[UserFieldValid] = false
	properties[UserFieldModifyTime] = time.Now()

	if err := updateValue(dbTx, tblUser, user.ID, properties); err != nil {
		log.Error("[Store][User] delete user by id", zap.Error(err), zap.String("id", user.ID))
		return err
	}
	return nil
}

// GetUser 获取用户
func (us *userStore) GetUser(id string) (*authcommon.User, error) {
	if id == "" {
		return nil, store.NewStatusError(store.EmptyParamsErr, "get user missing some params")
	}

	proxy, err := us.handler.StartTx()
	if err != nil {
		return nil, err
	}
	tx := proxy.GetDelegateTx().(*bolt.Tx)
	defer func() {
		_ = tx.Rollback()
	}()

	return us.getUser(tx, id)
}

// GetUser 获取用户
func (us *userStore) getUser(tx *bolt.Tx, id string) (*authcommon.User, error) {
	if id == "" {
		return nil, store.NewStatusError(store.EmptyParamsErr, "get user missing id params")
	}

	ret := make(map[string]interface{})
	if err := loadValues(tx, tblUser, []string{id}, &userForStore{}, ret); err != nil {
		log.Error("[Store][User] get user by id", zap.Error(err), zap.String("id", id))
		return nil, err
	}
	if len(ret) == 0 {
		return nil, nil
	}
	user := ret[id].(*userForStore)
	if !user.Valid {
		return nil, nil
	}

	return converToUserModel(user), nil
}

// GetUserByName 获取用户
func (us *userStore) GetUserByName(name, ownerId string) (*authcommon.User, error) {
	if name == "" {
		return nil, store.NewStatusError(store.EmptyParamsErr, "get user missing name params")
	}
	fields := []string{UserFieldName, UserFieldOwner, UserFieldValid}
	ret, err := us.handler.LoadValuesByFilter(tblUser, fields, &userForStore{},
		func(m map[string]interface{}) bool {
			valid, ok := m[UserFieldValid].(bool)
			if ok && !valid {
				return false
			}
			saveName, _ := m[UserFieldName].(string)
			saveOwner, _ := m[UserFieldOwner].(string)
			return saveName == name && saveOwner == ownerId
		})
	if err != nil {
		log.Error("[Store][User] get user by name", zap.Error(err), zap.String("name", name),
			zap.String("owner", ownerId))
		return nil, err
	}
	if len(ret) == 0 {
		return nil, nil
	}
	if len(ret) > 1 {
		return nil, ErrMultipleUserFound
	}

	var id string
	for k := range ret {
		id = k
		break
	}

	user := ret[id].(*userForStore)
	if !user.Valid {
		return nil, nil
	}

	return converToUserModel(user), nil
}

// GetUserByIds 通过用户ID批量获取用户
func (us *userStore) GetUserByIds(ids []string) ([]*authcommon.User, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	ret, err := us.handler.LoadValues(tblUser, ids, &userForStore{})
	if err != nil {
		log.Error("[Store][User] get user by ids", zap.Error(err), zap.Any("ids", ids))
		return nil, err
	}
	if len(ret) == 0 {
		return nil, nil
	}

	users := make([]*authcommon.User, 0, len(ids))
	for k := range ret {
		user := ret[k].(*userForStore)
		if !user.Valid {
			continue
		}
		users = append(users, converToUserModel(user))
	}

	return users, nil
}

// GetSubCount 获取子账户的个数
func (us *userStore) GetSubCount(user *authcommon.User) (uint32, error) {
	ownerId := user.ID
	ret, err := us.handler.LoadValuesByFilter(tblUser, []string{UserFieldOwner, UserFieldValid}, &userForStore{},
		func(m map[string]interface{}) bool {
			valid, ok := m[UserFieldValid].(bool)
			if ok && !valid {
				return false
			}

			saveOwner, _ := m[UserFieldOwner].(string)
			return saveOwner == ownerId
		})

	if err != nil {
		log.Error("[Store][User] get user sub count", zap.Error(err), zap.String("id", user.ID))
		return 0, err
	}

	return uint32(len(ret)), nil
}

// GetMoreUsers 获取所有用户信息
func (us *userStore) GetMoreUsers(mtime time.Time, firstUpdate bool) ([]*authcommon.User, error) {
	fields := []string{UserFieldModifyTime, UserFieldValid}
	ret, err := us.handler.LoadValuesByFilter(tblUser, fields, &userForStore{},
		func(m map[string]interface{}) bool {
			if firstUpdate {
				valid, _ := m[UserFieldValid].(bool)
				if !valid {
					return false
				}
			}
			mt := m[UserFieldModifyTime].(time.Time)
			isBefore := mt.Before(mtime)
			return !isBefore
		})
	if err != nil {
		log.Error("[Store][User] get users for cache", zap.Error(err))
		return nil, err
	}

	users := make([]*authcommon.User, 0, len(ret))
	for k := range ret {
		val := ret[k]
		users = append(users, converToUserModel(val.(*userForStore)))
	}

	return users, nil
}

// doPage 进行分页
func doUserPage(ret map[string]interface{}, offset, limit uint32) []*authcommon.User {
	users := make([]*authcommon.User, 0, len(ret))
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
		users = append(users, converToUserModel(ret[k].(*userForStore)))
	}

	sort.Slice(users, func(i, j int) bool {
		return users[i].ModifyTime.After(users[j].ModifyTime)
	})

	return users[beginIndex:endIndex]
}

func converToUserStore(user *authcommon.User) *userForStore {
	return &userForStore{
		ID:          user.ID,
		Name:        user.Name,
		Password:    user.Password,
		Owner:       user.Owner,
		Source:      user.Source,
		Type:        int(user.Type),
		Token:       user.Token,
		TokenEnable: user.TokenEnable,
		Valid:       user.Valid,
		Comment:     user.Comment,
		CreateTime:  user.CreateTime,
		ModifyTime:  user.ModifyTime,
	}
}

func converToUserModel(user *userForStore) *authcommon.User {
	return &authcommon.User{
		ID:          user.ID,
		Name:        user.Name,
		Password:    user.Password,
		Owner:       user.Owner,
		Source:      user.Source,
		Type:        authcommon.UserRoleType(user.Type),
		Token:       user.Token,
		TokenEnable: user.TokenEnable,
		Valid:       user.Valid,
		Comment:     user.Comment,
		CreateTime:  user.CreateTime,
		ModifyTime:  user.ModifyTime,
	}
}

func initUser(user *authcommon.User) {
	if user != nil {
		tn := time.Now()
		user.Valid = true
		user.CreateTime = tn
		user.ModifyTime = tn
	}
}

type userForStore struct {
	ID          string
	Name        string
	Password    string
	Owner       string
	Source      string
	Type        int
	Mobile      string
	Email       string
	Token       string
	TokenEnable bool
	Valid       bool
	Comment     string
	CreateTime  time.Time
	ModifyTime  time.Time
}
