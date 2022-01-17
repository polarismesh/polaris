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

package sqldb

import (
	"database/sql"
	"fmt"
	"time"

	logger "github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/common/model"
	commontime "github.com/polarismesh/polaris-server/common/time"
	"github.com/polarismesh/polaris-server/common/utils"
	"github.com/polarismesh/polaris-server/store"
	"go.uber.org/zap"
)

type groupStore struct {
	master *BaseDB
	slave  *BaseDB
}

// AddGroup 创建一个用户组
func (u *groupStore) AddGroup(group *model.UserGroupDetail) error {
	if group.ID == "" || group.Name == "" || group.Token == "" {
		return store.NewStatusError(store.EmptyParamsErr, fmt.Sprintf(
			"add usergroup missing some params, groupId is %s, name is %s", group.ID, group.Name))
	}

	// 先清理无效数据
	if err := u.cleanInValidGroup(group.Name, group.Owner); err != nil {
		return store.Error(err)
	}

	err := RetryTransaction("addGroup", func() error {
		return u.addGroup(group)
	})

	return store.Error(err)
}

func (u *groupStore) addGroup(group *model.UserGroupDetail) error {
	tx, err := u.master.Begin()
	if err != nil {
		return err
	}

	defer func() { _ = tx.Rollback() }()

	addSql := `
	INSERT INTO user_group (id, name, owner, token, token_enable
		, comment, flag, ctime, mtime)
	VALUES (?, ?, ?, ?, ?
		, ?, ?, sysdate(), sysdate())
	`

	if _, err = tx.Exec(addSql, []interface{}{
		group.ID,
		group.Name,
		group.Owner,
		group.Token,
		1,
		group.Comment,
		0,
	}...); err != nil {
		logger.AuthScope().Errorf("[Store][Group] add usergroup err: %s", err.Error())
		return err
	}

	if err := u.addGroupRelation(tx, group.ID, group.ToUserIdSlice()); err != nil {
		logger.AuthScope().Errorf("[Store][Group] add usergroup relation err: %s", err.Error())
		return err
	}

	if err := createDefaultStrategy(tx, model.PrincipalUserGroup, group.ID, group.Owner); err != nil {
		logger.AuthScope().Errorf("[Store][Group] add usergroup default strategy err: %s", err.Error())
		return err
	}

	if err := tx.Commit(); err != nil {
		logger.AuthScope().Errorf("[Store][Group] add usergroup tx commit err: %s", err.Error())
		return err
	}
	return nil
}

// UpdateGroup 更新用户组
func (u *groupStore) UpdateGroup(group *model.ModifyUserGroup) error {
	if group.ID == "" {
		return store.NewStatusError(store.EmptyParamsErr, fmt.Sprintf(
			"update usergroup missing some params, groupId is %s", group.ID))
	}

	if err := u.cleanInValidGroupRelation(group.ID); err != nil {
		return store.Error(err)
	}

	err := RetryTransaction("updateGroup", func() error {
		return u.updateGroup(group)
	})

	return store.Error(err)
}

func (u *groupStore) updateGroup(group *model.ModifyUserGroup) error {

	tx, err := u.master.Begin()
	if err != nil {
		return err
	}

	defer func() { _ = tx.Rollback() }()

	tokenEnable := 1
	if !group.TokenEnable {
		tokenEnable = 0
	}

	modifySql := "UPDATE user_group SET token = ?, comment = ?, token_enable = ? WHERE id = ? AND flag = 0"

	if _, err = tx.Exec(modifySql, []interface{}{
		group.Token,
		group.Comment,
		tokenEnable,
		group.ID,
	}...); err != nil {
		logger.AuthScope().Errorf("[Store][Group] update usergroup main err: %s", err.Error())
		return err
	}

	// 更新用户-用户组关联数据
	if len(group.AddUserIds) != 0 {
		if err := u.addGroupRelation(tx, group.ID, group.AddUserIds); err != nil {
			logger.AuthScope().Errorf("[Store][Group] add usergroup relation err: %s", err.Error())
			return err
		}
	}

	if len(group.RemoveUserIds) != 0 {
		if err := u.removeGroupRelation(tx, group.ID, group.RemoveUserIds); err != nil {
			logger.AuthScope().Errorf("[Store][Group] remove usergroup relation err: %s", err.Error())
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		logger.AuthScope().Errorf("[Store][Group] update usergroup tx commit err: %s", err.Error())
		return err
	}

	return nil
}

// DeleteGroup 删除用户组
func (u *groupStore) DeleteGroup(groupId string) error {
	if groupId == "" {
		return store.NewStatusError(store.EmptyParamsErr, fmt.Sprintf(
			"delete usergroup missing some params, groupId is %s", groupId))
	}

	err := RetryTransaction("deleteUserGroup", func() error {
		return u.deleteUserGroup(groupId)
	})

	return store.Error(err)
}

func (u *groupStore) deleteUserGroup(id string) error {

	tx, err := u.master.Begin()
	if err != nil {
		return err
	}

	defer func() { _ = tx.Rollback() }()

	if _, err = tx.Exec("UPDATE user_group SET flag = 1 WHERE id = ?", []interface{}{
		id,
	}...); err != nil {
		logger.AuthScope().Errorf("[Store][Group] remove usergroup err: %s", err.Error())
		return err
	}

	if _, err = tx.Exec("UPDATE user_group_relation SET flag = 1 WHERE group_id = ?", []interface{}{
		id,
	}...); err != nil {
		logger.AuthScope().Errorf("[Store][Group] clean usergroup relation err: %s", err.Error())
		return err
	}

	if err := cleanLinkStrategy(tx, model.PrincipalUserGroup, id); err != nil {
		logger.AuthScope().Errorf("[Store][Group] clean usergroup default strategy err: %s", err.Error())
		return err
	}

	if err := tx.Commit(); err != nil {
		logger.AuthScope().Errorf("[Store][Group] delete usergroupr tx commit err: %s", err.Error())
		return err
	}
	return nil
}

// GetGroup 根据用户组ID获取用户组
func (u *groupStore) GetGroup(groupId string) (*model.UserGroup, error) {
	if groupId == "" {
		return nil, store.NewStatusError(store.EmptyParamsErr, fmt.Sprintf(
			"get usergroup missing some params, groupId is %s", groupId))
	}

	getSql := `
	SELECT id, name, owner, token, token_enable
		, UNIX_TIMESTAMP(ctime), UNIX_TIMESTAMP(mtime)
	FROM user_group
	WHERE flag = 0
		AND id = ? 
	`
	row := u.master.QueryRow(getSql, groupId)

	group := new(model.UserGroup)
	var (
		ctime, mtime int64
		tokenEnable  int
	)

	if err := row.Scan(&group.ID, &group.Name, &group.Owner, &group.Token, &tokenEnable, &ctime, &mtime); err != nil {
		switch err {
		case sql.ErrNoRows:
			return nil, nil
		default:
			return nil, store.Error(err)
		}
	}

	group.TokenEnable = (tokenEnable == 1)
	group.CreateTime = time.Unix(ctime, 0)
	group.ModifyTime = time.Unix(mtime, 0)

	return group, nil
}

// GetGroupByName 根据 owner、name 获取用户组
func (u *groupStore) GetGroupByName(name, owner string) (*model.UserGroup, error) {

	var ctime, mtime int64

	getSql := `
	SELECT id, name, owner, token
		, UNIX_TIMESTAMP(ctime), UNIX_TIMESTAMP(mtime)
	FROM user_group
	WHERE flag = 0
		AND name = ?
		AND owner = ? 
	`
	row := u.master.QueryRow(getSql, name, owner)

	group := new(model.UserGroup)

	if err := row.Scan(&group.ID, &group.Name, &group.Owner, &group.Token, &ctime, &mtime); err != nil {
		switch err {
		case sql.ErrNoRows:
			return nil, nil
		default:
			return nil, store.Error(err)
		}
	}

	group.CreateTime = time.Unix(ctime, 0)
	group.ModifyTime = time.Unix(mtime, 0)

	return group, nil
}

// GetGroups 根据不同的请求情况进行不同的用户组列表查询
func (u *groupStore) GetGroups(filters map[string]string, offset uint32, limit uint32) (uint32, []*model.UserGroup, error) {

	// 如果本次请求参数携带了 user_id，那么就是查询这个用户所关联的所有用户组
	if _, ok := filters["user_id"]; ok {
		return u.listGroupByUser(filters, offset, limit)
	} else {

		// 正常查询用户组信息
		return u.listSimpleGroups(filters, offset, limit)
	}
}

// listSimpleGroups 正常的用户组查询
func (u *groupStore) listSimpleGroups(filters map[string]string, offset uint32, limit uint32) (uint32, []*model.UserGroup, error) {

	countSql := "SELECT COUNT(*) FROM user_group WHERE flag = 0 "
	getSql := `
	SELECT id, name, owner, token, token_enable
		, UNIX_TIMESTAMP(ctime), UNIX_TIMESTAMP(mtime)
		, flag
	FROM user_group
	WHERE flag = 0 
	`

	args := make([]interface{}, 0)

	if len(filters) != 0 {
		for k, v := range filters {
			getSql += " AND "
			countSql += " AND "
			if k == NameAttribute && utils.IsWildName(v) {
				getSql += (" " + k + " like ? ")
				countSql += (" " + k + " like ? ")
				args = append(args, v[:len(v)-1]+"%")
				continue
			}
			getSql += (" " + k + " = ? ")
			countSql += (" " + k + " = ? ")
			args = append(args, v)
		}
	}

	count, err := queryEntryCount(u.master, countSql, args)
	if err != nil {
		return 0, nil, err
	}

	getSql += " ORDER BY mtime LIMIT ? , ?"
	args = append(args, offset, limit)

	groups, err := u.collectGroupsFromRows(u.master.Query, getSql, args)
	if err != nil {
		return 0, nil, err
	}

	return count, groups, nil
}

// listGroupByUser 查询某个用户下所关联的用户组信息
func (u *groupStore) listGroupByUser(filters map[string]string, offset uint32, limit uint32) (uint32, []*model.UserGroup, error) {
	countSql := "SELECT COUNT(*) FROM user_group_relation ul LEFT JOIN user_group ug ON ul.group_id = ug.id WHERE ug.flag = 0  "
	getSql := "SELECT ug.id, ug.name, ug.owner, ug.token, ug.token_enable, UNIX_TIMESTAMP(ug.ctime), UNIX_TIMESTAMP(ug.mtime), ug.flag " +
		" FROM user_group_relation ul LEFT JOIN user_group ug ON ul.group_id = ug.id WHERE ug.flag = 0 "

	args := make([]interface{}, 0)

	if len(filters) != 0 {
		for k, v := range filters {
			if _, ok := userLinkGroupAttributeMapping[k]; ok {
				k = userLinkGroupAttributeMapping[k]
			}
			getSql += " AND  " + k + " = ? "
			countSql += " AND  " + k + " = ? "
			args = append(args, v)
		}
	}

	logger.AuthScope().Debug("[Store][Group] list group by user", zap.String("count sql", countSql), zap.Any("args", args))
	count, err := queryEntryCount(u.master, countSql, args)
	if err != nil {
		return 0, nil, err
	}

	getSql += " ORDER BY ug.mtime LIMIT ? , ?"
	args = append(args, offset, limit)

	groups, err := u.collectGroupsFromRows(u.master.Query, getSql, args)
	if err != nil {
		return 0, nil, err
	}

	return count, groups, nil
}

// collectGroupsFromRows 查询用户组列表
func (u *groupStore) collectGroupsFromRows(handler QueryHandler, querySql string, args []interface{}) ([]*model.UserGroup, error) {

	logger.AuthScope().Debug("[Store][Group] list group", zap.String("query sql", querySql), zap.Any("args", args))

	rows, err := u.master.Query(querySql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	groups := make([]*model.UserGroup, 0)
	for rows.Next() {
		group, err := fetchRown2UserGroup(rows)
		if err != nil {
			logger.AuthScope().Errorf("[Store][Group] list group by user fetch rows scan err: %s", err.Error())
			return nil, err
		}
		groups = append(groups, group)
	}

	return groups, nil
}

func (u *groupStore) GetGroupsForCache(mtime time.Time, firstUpdate bool) ([]*model.UserGroupDetail, error) {
	tx, err := u.slave.Begin()
	if err != nil {
		return nil, store.Error(err)
	}

	defer func() { _ = tx.Commit() }()

	args := make([]interface{}, 0)
	querySql := "SELECT id, name, owner, token, token_enable, UNIX_TIMESTAMP(ctime), UNIX_TIMESTAMP(mtime), flag FROM user_group "
	if !firstUpdate {
		querySql += " WHERE mtime >= ?"
		args = append(args, commontime.Time2String(mtime))
	}

	rows, err := tx.Query(querySql, args...)
	if err != nil {
		return nil, store.Error(err)
	}
	defer rows.Close()

	ret := make([]*model.UserGroupDetail, 0)
	for rows.Next() {
		detail := &model.UserGroupDetail{
			UserIDs: make(map[string]struct{}, 0),
		}
		group, err := fetchRown2UserGroup(rows)
		if err != nil {
			return nil, store.Error(err)
		}
		uids, err := u.getGroupLinkUserIds(group.ID)
		if err != nil {
			return nil, store.Error(err)
		}

		detail.UserIDs = uids
		detail.UserGroup = group

		ret = append(ret, detail)
	}

	return ret, nil
}

func (u *groupStore) addGroupRelation(tx *BaseTx, groupId string, userIds []string) error {
	if groupId == "" {
		return store.NewStatusError(store.EmptyParamsErr, fmt.Sprintf(
			"add user relation missing some params, groupid is %s", groupId))
	}
	if len(userIds) > utils.MaxBatchSize {
		return store.NewStatusError(store.InvalidUserIDSlice, fmt.Sprintf(
			"user id slice is invalid, len=%d", len(userIds)))
	}

	for i := range userIds {
		uid := userIds[i]
		addSql := "INSERT INTO user_group_relation (group_id, user_id, flag) VALUE (?,?,?)"
		args := []interface{}{groupId, uid, 0}
		_, err := tx.Exec(addSql, args...)
		if err != nil {
			err = store.Error(err)
			// 之前的用户已经存在，直接忽略
			if store.Code(err) == store.DuplicateEntryErr {
				continue
			}
			return err
		}
	}
	return nil
}

func (u *groupStore) removeGroupRelation(tx *BaseTx, groupId string, userIds []string) error {
	if groupId == "" {
		return store.NewStatusError(store.EmptyParamsErr, fmt.Sprintf(
			"delete user relation missing some params, groupid is %s", groupId))
	}
	if len(userIds) > utils.MaxBatchSize {
		return store.NewStatusError(store.InvalidUserIDSlice, fmt.Sprintf(
			"user id slice is invalid, len=%d", len(userIds)))
	}

	for i := range userIds {
		uid := userIds[i]
		addSql := "UPDATE user_group_relation SET flag = 1 WHERE group_id = ? AND user_id = ?"
		args := []interface{}{groupId, uid}
		if _, err := tx.Exec(addSql, args...); err != nil {
			return err
		}
	}

	return nil
}

func (u *groupStore) getGroupLinkUserIds(groupId string) (map[string]struct{}, error) {

	ids := make(map[string]struct{})

	// 拉取该分组下的所有 user
	idRows, err := u.slave.Query("SELECT user_id FROM user u JOIN user_group_relation ug ON u.id = ug.user_id AND u.flag = 0 WHERE ug.group_id = ?", groupId)
	if err != nil {
		return nil, err
	}
	defer idRows.Close()
	for idRows.Next() {
		var uid string
		if err := idRows.Scan(&uid); err != nil {
			return nil, err
		}
		ids[uid] = struct{}{}
	}

	return ids, nil
}

func fetchRown2UserGroup(rows *sql.Rows) (*model.UserGroup, error) {
	var ctime, mtime int64
	var flag, tokenEnable int
	group := new(model.UserGroup)
	err := rows.Scan(&group.ID, &group.Name, &group.Owner, &group.Token, &tokenEnable, &ctime, &mtime, &flag)

	if err != nil {
		return nil, err
	}

	group.Valid = flag == 0
	group.TokenEnable = tokenEnable == 1
	group.CreateTime = time.Unix(ctime, 0)
	group.ModifyTime = time.Unix(mtime, 0)

	return group, nil
}

// cleanInValidUserGroup 清理无效的用户组数据
func (u *groupStore) cleanInValidGroup(name, owner string) error {
	logger.AuthScope().Infof("[Store][User] clean usergroup(%s)", name)
	str := "delete from user_group_relation where group_id = (select id from user_group where name = ? and owner = ? and flag = 1) and flag = 1"
	_, err := u.master.Exec(str, name, owner)
	if err != nil {
		logger.AuthScope().Errorf("[Store][User] clean usergroup(%s) err: %s", name, err.Error())
		return err
	}

	str = "delete from user_group where name = ? and flag = 1"
	_, err = u.master.Exec(str, name)
	if err != nil {
		logger.AuthScope().Errorf("[Store][User] clean usergroup(%s) err: %s", name, err.Error())
		return err
	}

	return nil
}

// cleanInValidUserGroupRelation 清理无效的用户-用户组关联数据
func (u *groupStore) cleanInValidGroupRelation(id string) error {
	logger.AuthScope().Infof("[Store][User] clean usergroup(%s)", id)
	str := "delete from user_group_relation where group_id = ? and flag = 1"
	_, err := u.master.Exec(str, id)
	if err != nil {
		logger.AuthScope().Errorf("[Store][User] clean usergroup(%s) err: %s", id, err.Error())
		return err
	}
	return nil
}
