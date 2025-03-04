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
	"fmt"
	"time"

	bolt "go.etcd.io/bbolt"
	"go.uber.org/zap"

	authcommon "github.com/polarismesh/polaris/common/model/auth"
	"github.com/polarismesh/polaris/store"
)

var (
	// ErrorMultipleGroupFound is returned when multiple groups are found.
	ErrorMultipleGroupFound error = errors.New("multiple group found")
	// ErrorGroupNotFound is returned when a group is not found.
	ErrorGroupNotFound error = errors.New("usergroup not found")
)

const (
	tblGroup string = "group"

	GroupFieldID          string = "ID"
	GroupFieldName        string = "Name"
	GroupFieldOwner       string = "Owner"
	GroupFieldToken       string = "Token"
	GroupFieldTokenEnable string = "TokenEnable"
	GroupFieldValid       string = "Valid"
	GroupFieldComment     string = "Comment"
	GroupFieldCreateTime  string = "CreateTime"
	GroupFieldModifyTime  string = "ModifyTime"
	GroupFieldUserIds     string = "UserIds"
)

type groupForStore struct {
	ID          string
	Name        string
	Owner       string
	Token       string
	TokenEnable bool
	Valid       bool
	Comment     string
	CreateTime  time.Time
	ModifyTime  time.Time
	UserIds     map[string]string
}

// groupStore
type groupStore struct {
	handler BoltHandler
}

// AddGroup add a group
func (gs *groupStore) AddGroup(tx store.Tx, group *authcommon.UserGroupDetail) error {
	if group.ID == "" || group.Name == "" || group.Token == "" {
		return store.NewStatusError(store.EmptyParamsErr, fmt.Sprintf(
			"add usergroup missing some params, groupId is %s, name is %s", group.ID, group.Name))
	}

	dbTx := tx.GetDelegateTx().(*bolt.Tx)

	if err := gs.cleanInValidGroup(dbTx, group.Name, group.Owner); err != nil {
		log.Error("[Store][Group] clean invalid usergroup", zap.Error(err),
			zap.String("name", group.Name), zap.String("owner", group.Owner))
		return err
	}

	group.Valid = true
	group.CreateTime = time.Now()
	group.ModifyTime = group.CreateTime

	data := convertForGroupStore(group)

	if err := saveValue(dbTx, tblGroup, data.ID, data); err != nil {
		log.Error("[Store][Group] save usergroup", zap.Error(err),
			zap.String("name", group.Name), zap.String("owner", group.Owner))

		return err
	}
	return nil
}

// UpdateGroup update a group
func (gs *groupStore) UpdateGroup(group *authcommon.ModifyUserGroup) error {
	if group.ID == "" {
		return store.NewStatusError(store.EmptyParamsErr, fmt.Sprintf(
			"update usergroup missing some params, groupId is %s", group.ID))
	}

	return gs.updateGroup(group)
}

func (gs *groupStore) updateGroup(group *authcommon.ModifyUserGroup) error {
	proxy, err := gs.handler.StartTx()
	if err != nil {
		return err
	}
	tx := proxy.GetDelegateTx().(*bolt.Tx)

	defer func() {
		_ = tx.Rollback()
	}()

	values := make(map[string]interface{})

	if err := loadValues(tx, tblGroup, []string{group.ID}, &groupForStore{}, values); err != nil {
		log.Error("[Store][Group] get usergroup by id", zap.Error(err), zap.String("id", group.ID))
	}

	if len(values) == 0 {
		return ErrorGroupNotFound
	}

	if len(values) > 1 {
		return ErrorMultipleGroupFound
	}

	var ret *authcommon.UserGroupDetail
	for _, v := range values {
		ret = convertForGroupDetail(v.(*groupForStore))
		break
	}

	ret.Comment = group.Comment
	ret.Token = group.Token
	ret.TokenEnable = group.TokenEnable
	ret.ModifyTime = time.Now()

	updateGroupRelation(ret, group)

	if err := saveValue(tx, tblGroup, ret.ID, convertForGroupStore(ret)); err != nil {
		log.Error("[Store][Group] update usergroup", zap.Error(err), zap.String("id", ret.ID))
		return err
	}

	if err := tx.Commit(); err != nil {
		log.Error("[Store][Group] update usergroup tx commit",
			zap.Error(err), zap.String("id", ret.ID))
		return err
	}
	return nil
}

// updateGroupRelation 更新用户组的关联关系数据
func updateGroupRelation(group *authcommon.UserGroupDetail, modify *authcommon.ModifyUserGroup) {
	for i := range modify.AddUserIds {
		group.UserIds[modify.AddUserIds[i]] = struct{}{}
	}

	for i := range modify.RemoveUserIds {
		delete(group.UserIds, modify.RemoveUserIds[i])
	}
}

// DeleteGroup 删除用户组
func (gs *groupStore) DeleteGroup(tx store.Tx, group *authcommon.UserGroupDetail) error {
	if group.ID == "" {
		return store.NewStatusError(store.EmptyParamsErr, fmt.Sprintf(
			"delete usergroup missing some params, groupId is %s", group.ID))
	}
	dbTx := tx.GetDelegateTx().(*bolt.Tx)

	properties := make(map[string]interface{})
	properties[GroupFieldValid] = false
	properties[GroupFieldModifyTime] = time.Now()

	if err := updateValue(dbTx, tblGroup, group.ID, properties); err != nil {
		log.Error("[Store][Group] remove usergroup", zap.Error(err), zap.String("id", group.ID))
		return store.Error(err)
	}
	return nil
}

// GetGroup get a group
func (gs *groupStore) GetGroup(groupID string) (*authcommon.UserGroupDetail, error) {
	if groupID == "" {
		return nil, store.NewStatusError(store.EmptyParamsErr, fmt.Sprintf(
			"get usergroup missing some params, groupID is %s", groupID))
	}

	values, err := gs.handler.LoadValues(tblGroup, []string{groupID}, &groupForStore{})
	if err != nil {
		log.Error("[Store][Group] get usergroup by id", zap.Error(err), zap.String("id", groupID))
		return nil, err
	}

	if len(values) == 0 {
		return nil, nil
	}

	if len(values) > 1 {
		return nil, ErrorMultipleGroupFound
	}

	var ret *authcommon.UserGroupDetail
	for _, v := range values {
		ret = convertForGroupDetail(v.(*groupForStore))
		break
	}

	if ret.Valid {
		return ret, nil
	}

	return nil, nil
}

// GetGroupByName get a group by name
func (gs *groupStore) GetGroupByName(name, owner string) (*authcommon.UserGroup, error) {
	if name == "" || owner == "" {
		return nil, store.NewStatusError(store.EmptyParamsErr, fmt.Sprintf(
			"get usergroup missing some params, name=%s, owner=%s", name, owner))
	}

	fields := []string{GroupFieldName, GroupFieldOwner, GroupFieldValid}
	values, err := gs.handler.LoadValuesByFilter(tblGroup, fields, &groupForStore{},
		func(m map[string]interface{}) bool {
			valid, ok := m[GroupFieldValid].(bool)
			if ok && !valid {
				return false
			}

			saveName := m[GroupFieldName]
			saveOwner := m[GroupFieldOwner]

			return saveName == name && saveOwner == owner
		})

	if err != nil {
		return nil, err
	}

	if len(values) == 0 {
		return nil, nil
	}

	if len(values) > 1 {
		return nil, ErrorMultipleGroupFound
	}
	var ret *authcommon.UserGroupDetail
	for _, v := range values {
		ret = convertForGroupDetail(v.(*groupForStore))
		break
	}
	return ret.UserGroup, nil
}

// GetMoreGroups 查询用户分组数据，主要用于Cache更新
func (gs *groupStore) GetMoreGroups(mtime time.Time, firstUpdate bool) ([]*authcommon.UserGroupDetail, error) {
	ret, err := gs.handler.LoadValuesByFilter(tblGroup, []string{GroupFieldModifyTime}, &groupForStore{},
		func(m map[string]interface{}) bool {
			mt := m[GroupFieldModifyTime].(time.Time)
			isAfter := !mt.Before(mtime)
			return isAfter
		})
	if err != nil {
		return nil, err
	}

	groups := make([]*authcommon.UserGroupDetail, 0, len(ret))

	for k := range ret {
		val := ret[k]
		groups = append(groups, convertForGroupDetail(val.(*groupForStore)))
	}

	return groups, nil
}

// cleanInValidGroup 清理无效的用户组数据
func (gs *groupStore) cleanInValidGroup(tx *bolt.Tx, name, owner string) error {
	log.Infof("[Store][User] clean usergroup(%s)", name)

	fields := []string{GroupFieldName, GroupFieldValid, GroupFieldOwner}

	values := make(map[string]interface{})

	err := loadValuesByFilter(tx, tblGroup, fields, &groupForStore{},
		func(m map[string]interface{}) bool {
			valid, ok := m[GroupFieldValid].(bool)
			// 如果数据是 valid 的，则不能被清理
			if ok && valid {
				return false
			}

			saveName := m[GroupFieldName]
			saveOwner := m[GroupFieldOwner]

			return saveName == name && saveOwner == owner
		}, values)

	if err != nil {
		return err
	}

	if len(values) == 0 {
		return nil
	}

	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}

	return deleteValues(tx, tblGroup, keys)
}

func convertForGroupStore(group *authcommon.UserGroupDetail) *groupForStore {

	userIds := make(map[string]string, len(group.UserIds))

	for id := range group.UserIds {
		userIds[id] = ""
	}

	return &groupForStore{
		ID:          group.ID,
		Name:        group.Name,
		Owner:       group.Owner,
		Token:       group.Token,
		TokenEnable: group.TokenEnable,
		Valid:       group.Valid,
		Comment:     group.Comment,
		CreateTime:  group.CreateTime,
		ModifyTime:  group.ModifyTime,
		UserIds:     userIds,
	}
}

func convertForGroupDetail(group *groupForStore) *authcommon.UserGroupDetail {
	userIds := make(map[string]struct{}, len(group.UserIds))
	for id := range group.UserIds {
		userIds[id] = struct{}{}
	}

	return &authcommon.UserGroupDetail{
		UserGroup: &authcommon.UserGroup{
			ID:          group.ID,
			Name:        group.Name,
			Owner:       group.Owner,
			Token:       group.Token,
			TokenEnable: group.TokenEnable,
			Valid:       group.Valid,
			Comment:     group.Comment,
			CreateTime:  group.CreateTime,
			ModifyTime:  group.ModifyTime,
		},
		UserIds: userIds,
	}
}
