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
	"sort"
	"strings"
	"time"

	"github.com/boltdb/bolt"
	"go.uber.org/zap"

	logger "github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/common/utils"
	"github.com/polarismesh/polaris-server/store"
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
func (gs *groupStore) AddGroup(group *model.UserGroupDetail) error {
	if group.ID == "" || group.Name == "" || group.Token == "" {
		return store.NewStatusError(store.EmptyParamsErr, fmt.Sprintf(
			"add usergroup missing some params, groupId is %s, name is %s", group.ID, group.Name))
	}

	proxy, err := gs.handler.StartTx()
	if err != nil {
		return err
	}
	tx := proxy.GetDelegateTx().(*bolt.Tx)

	defer tx.Rollback()

	if err := gs.cleanInValidGroup(tx, group.Name, group.Owner); err != nil {
		logger.StoreScope().Error("[Store][Group] clean invalid usergroup", zap.Error(err),
			zap.String("name", group.Name), zap.String("owner", group.Owner))
		return err
	}

	return gs.addGroup(tx, group)
}

// addGroup to boltdb
func (gs *groupStore) addGroup(tx *bolt.Tx, group *model.UserGroupDetail) error {

	group.Valid = true
	group.CreateTime = time.Now()
	group.ModifyTime = group.CreateTime

	data := convertForGroupStore(group)

	if err := saveValue(tx, tblGroup, data.ID, data); err != nil {
		logger.StoreScope().Error("[Store][Group] save usergroup", zap.Error(err),
			zap.String("name", group.Name), zap.String("owner", group.Owner))

		return err
	}

	if err := createDefaultStrategy(tx, model.PrincipalGroup, data.ID, data.Name,
		data.Owner); err != nil {
		logger.StoreScope().Error("[Store][Group] add usergroup default strategy", zap.Error(err),
			zap.String("name", group.Name), zap.String("owner", group.Owner))

		return err
	}

	if err := tx.Commit(); err != nil {
		logger.StoreScope().Error("[Store][Group] add usergroup tx commit", zap.Error(err),
			zap.String("name", group.Name), zap.String("owner", group.Owner))
		return err
	}

	return nil
}

// UpdateGroup update a group
func (gs *groupStore) UpdateGroup(group *model.ModifyUserGroup) error {
	if group.ID == "" {
		return store.NewStatusError(store.EmptyParamsErr, fmt.Sprintf(
			"update usergroup missing some params, groupId is %s", group.ID))
	}

	return gs.updateGroup(group)
}

func (gs *groupStore) updateGroup(group *model.ModifyUserGroup) error {
	proxy, err := gs.handler.StartTx()
	if err != nil {
		return err
	}
	tx := proxy.GetDelegateTx().(*bolt.Tx)

	defer tx.Rollback()

	values := make(map[string]interface{})

	if err := loadValues(tx, tblGroup, []string{group.ID}, &groupForStore{}, values); err != nil {
		logger.StoreScope().Error("[Store][Group] get usergroup by id", zap.Error(err), zap.String("id", group.ID))
	}

	if len(values) == 0 {
		return ErrorGroupNotFound
	}

	if len(values) > 1 {
		return ErrorMultipleGroupFound
	}

	var ret *model.UserGroupDetail
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
		logger.StoreScope().Error("[Store][Group] update usergroup", zap.Error(err), zap.String("id", ret.ID))
		return err
	}

	if err := tx.Commit(); err != nil {
		logger.StoreScope().Error("[Store][Group] update usergroup tx commit",
			zap.Error(err), zap.String("id", ret.ID))
		return err
	}
	return nil
}

// updateGroupRelation 更新用户组的关联关系数据
func updateGroupRelation(group *model.UserGroupDetail, modify *model.ModifyUserGroup) {
	for i := range modify.AddUserIds {
		group.UserIds[modify.AddUserIds[i]] = struct{}{}
	}

	for i := range modify.RemoveUserIds {
		delete(group.UserIds, modify.RemoveUserIds[i])
	}
}

// DeleteGroup 删除用户组
func (gs *groupStore) DeleteGroup(group *model.UserGroupDetail) error {
	if group.ID == "" {
		return store.NewStatusError(store.EmptyParamsErr, fmt.Sprintf(
			"delete usergroup missing some params, groupId is %s", group.ID))
	}

	return gs.deleteGroup(group)
}

func (gs *groupStore) deleteGroup(group *model.UserGroupDetail) error {
	proxy, err := gs.handler.StartTx()
	if err != nil {
		return err
	}
	tx := proxy.GetDelegateTx().(*bolt.Tx)

	defer tx.Rollback()

	properties := make(map[string]interface{})
	properties[GroupFieldValid] = false
	properties[GroupFieldModifyTime] = time.Now()

	if err := updateValue(tx, tblGroup, group.ID, properties); err != nil {
		logger.StoreScope().Error("[Store][Group] remove usergroup", zap.Error(err), zap.String("id", group.ID))
		return err
	}

	if err := cleanLinkStrategy(tx, model.PrincipalGroup, group.ID, group.Owner); err != nil {
		logger.StoreScope().Error("[Store][Group] clean usergroup default strategy",
			zap.Error(err), zap.String("id", group.ID))
		return err
	}

	if err := tx.Commit(); err != nil {
		logger.StoreScope().Error("[Store][Group] delete usergroupr tx commit",
			zap.Error(err), zap.String("id", group.ID))
		return err
	}

	return nil
}

// GetGroup get a group
func (gs *groupStore) GetGroup(groupID string) (*model.UserGroupDetail, error) {
	if groupID == "" {
		return nil, store.NewStatusError(store.EmptyParamsErr, fmt.Sprintf(
			"get usergroup missing some params, groupID is %s", groupID))
	}

	values, err := gs.handler.LoadValues(tblGroup, []string{groupID}, &groupForStore{})
	if err != nil {
		logger.StoreScope().Error("[Store][Group] get usergroup by id", zap.Error(err), zap.String("id", groupID))
		return nil, err
	}

	if len(values) == 0 {
		return nil, nil
	}

	if len(values) > 1 {
		return nil, ErrorMultipleGroupFound
	}

	var ret *model.UserGroupDetail
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
func (gs *groupStore) GetGroupByName(name, owner string) (*model.UserGroup, error) {

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

			saveName, _ := m[GroupFieldName]
			saveOwner, _ := m[GroupFieldOwner]

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
	var ret *model.UserGroupDetail
	for _, v := range values {
		ret = convertForGroupDetail(v.(*groupForStore))
		break
	}
	return ret.UserGroup, nil
}

// GetGroups get groups
func (gs *groupStore) GetGroups(filters map[string]string, offset uint32,
	limit uint32) (uint32, []*model.UserGroup, error) {

	// 如果本次请求参数携带了 user_id，那么就是查询这个用户所关联的所有用户组
	if _, ok := filters["user_id"]; ok {
		return gs.listGroupByUser(filters, offset, limit)
	}
	// 正常查询用户组信息
	return gs.listSimpleGroups(filters, offset, limit)

}

// listSimpleGroups Normal user group query
func (gs *groupStore) listSimpleGroups(filters map[string]string, offset uint32, limit uint32) (uint32,
	[]*model.UserGroup, error) {

	fields := []string{GroupFieldID, GroupFieldOwner, GroupFieldName, GroupFieldValid}

	values, err := gs.handler.LoadValuesByFilter(tblGroup, fields, &groupForStore{},
		func(m map[string]interface{}) bool {
			valid, ok := m[GroupFieldValid].(bool)
			if ok && !valid {
				return false
			}

			saveId, _ := m[GroupFieldID].(string)
			saveName, _ := m[GroupFieldName].(string)
			saveOwner, _ := m[GroupFieldOwner].(string)

			if sId, ok := filters["id"]; ok && sId != saveId {
				return false
			}
			if sName, ok := filters["name"]; ok {
				if utils.IsWildName(sName) {
					sName = sName[:len(sName)-1]
				}
				if !strings.Contains(saveName, sName) {
					return false
				}
			}

			if sOwner, ok := filters["owner"]; ok && sOwner != saveOwner {
				return false
			}

			return true
		})

	if err != nil {
		return 0, nil, err
	}

	total := uint32(len(values))

	return total, doGroupPage(values, offset, limit), nil
}

// listGroupByUser 查询某个用户下所关联的用户组信息
func (gs *groupStore) listGroupByUser(filters map[string]string, offset uint32, limit uint32) (uint32, []*model.UserGroup, error) {

	var (
		userID            = filters["user_id"]
		owner, existOwner = filters["owner"]
		fields            = []string{GroupFieldUserIds, GroupFieldOwner, GroupFieldValid}
	)

	values, err := gs.handler.LoadValuesByFilter(tblGroup, fields, &groupForStore{},
		func(m map[string]interface{}) bool {
			valid, ok := m[GroupFieldValid].(bool)
			if ok && !valid {
				return false
			}

			if sName, ok := filters["name"]; ok {
				saveName, _ := m[GroupFieldName].(string)
				if utils.IsWildName(sName) {
					sName = sName[:len(sName)-1]
				}
				if !strings.Contains(saveName, sName) {
					return false
				}
			}

			saveOwner, _ := m[GroupFieldOwner]
			saveVal, ok := m[GroupFieldUserIds]
			if !ok {
				return false
			}

			saveUserIds := saveVal.(map[string]string)
			_, exist := saveUserIds[userID]

			if existOwner {
				return exist || saveOwner == owner
			}
			return exist
		})

	if err != nil {
		return 0, nil, err
	}

	total := uint32(len(values))

	return total, doGroupPage(values, offset, limit), nil
}

func doGroupPage(ret map[string]interface{}, offset uint32, limit uint32) []*model.UserGroup {

	groups := make([]*model.UserGroup, 0, len(ret))

	beginIndex := offset
	endIndex := beginIndex + limit
	totalCount := uint32(len(ret))

	if totalCount == 0 {
		return groups
	}
	if beginIndex >= endIndex {
		return groups
	}
	if beginIndex >= totalCount {
		return groups
	}
	if endIndex > totalCount {
		endIndex = totalCount
	}
	for k := range ret {
		groups = append(groups, convertForGroupDetail(ret[k].(*groupForStore)).UserGroup)
	}

	sort.Slice(groups, func(i, j int) bool {
		return groups[i].ModifyTime.After(groups[j].ModifyTime)
	})

	return groups[beginIndex:endIndex]
}

// GetGroupsForCache 查询用户分组数据，主要用于Cache更新
func (gs *groupStore) GetGroupsForCache(mtime time.Time, firstUpdate bool) ([]*model.UserGroupDetail, error) {
	ret, err := gs.handler.LoadValuesByFilter(tblGroup, []string{GroupFieldModifyTime}, &groupForStore{},
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
		groups = append(groups, convertForGroupDetail(val.(*groupForStore)))
	}

	return groups, nil
}

// cleanInValidGroup 清理无效的用户组数据
func (gs *groupStore) cleanInValidGroup(tx *bolt.Tx, name, owner string) error {
	logger.StoreScope().Infof("[Store][User] clean usergroup(%s)", name)

	fields := []string{GroupFieldName, GroupFieldValid, GroupFieldOwner}

	values := make(map[string]interface{})

	err := loadValuesByFilter(tx, tblGroup, fields, &groupForStore{},
		func(m map[string]interface{}) bool {
			valid, ok := m[GroupFieldValid].(bool)
			// 如果数据是 valid 的，则不能被清理
			if ok && valid {
				return false
			}

			saveName, _ := m[GroupFieldName]
			saveOwner, _ := m[GroupFieldOwner]

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

	return deleteValues(tx, tblGroup, keys, false)
}

func convertForGroupStore(group *model.UserGroupDetail) *groupForStore {

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

func convertForGroupDetail(group *groupForStore) *model.UserGroupDetail {
	userIds := make(map[string]struct{}, len(group.UserIds))
	for id := range group.UserIds {
		userIds[id] = struct{}{}
	}

	return &model.UserGroupDetail{
		UserGroup: &model.UserGroup{
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
