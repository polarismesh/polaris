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

	"go.uber.org/zap"

	authcommon "github.com/polarismesh/polaris/common/model/auth"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/store"
)

const (
	// IDAttribute is the name of the attribute that stores the ID of the object.
	IDAttribute string = "id"

	// NameAttribute will be used as the name of the attribute that stores the name of the object.
	NameAttribute string = "name"

	// FlagAttribute will be used as the name of the attribute that stores the flag of the object.
	FlagAttribute string = "flag"

	// GroupIDAttribute will be used as the name of the attribute that stores the group ID of the object.
	GroupIDAttribute string = "group_id"
)

var (
	groupAttribute map[string]string = map[string]string{
		"name":  "ug.name",
		"id":    "ug.id",
		"owner": "ug.owner",
	}
)

type groupStore struct {
	master *BaseDB
	slave  *BaseDB
}

// AddGroup 创建一个用户组
func (u *groupStore) AddGroup(tx store.Tx, group *authcommon.UserGroupDetail) error {
	if group.ID == "" || group.Name == "" || group.Token == "" {
		return store.NewStatusError(store.EmptyParamsErr, fmt.Sprintf(
			"add usergroup missing some params, groupId is %s, name is %s", group.ID, group.Name))
	}
	dbTx := tx.GetDelegateTx().(*BaseTx)

	// 先清理无效数据
	if err := cleanInValidGroup(dbTx, group.Name, group.Owner); err != nil {
		return store.Error(err)
	}

	addSql := `
	  INSERT INTO user_group (id, name, owner, token, token_enable, comment, flag, ctime, mtime)
	  VALUES (?, ?, ?, ?, ?, ?, ?, sysdate(), sysdate())
	  `

	tokenEnable := 1
	if !group.TokenEnable {
		tokenEnable = 0
	}

	if _, err := dbTx.Exec(addSql, []interface{}{
		group.ID,
		group.Name,
		group.Owner,
		group.Token,
		tokenEnable,
		group.Comment,
		0,
	}...); err != nil {
		log.Errorf("[Store][Group] add usergroup err: %s", err.Error())
		return err
	}

	if err := u.addGroupRelation(dbTx, group.ID, group.ToUserIdSlice()); err != nil {
		log.Errorf("[Store][Group] add usergroup relation err: %s", err.Error())
		return err
	}
	return nil
}

// UpdateGroup 更新用户组
func (u *groupStore) UpdateGroup(group *authcommon.ModifyUserGroup) error {
	if group.ID == "" {
		return store.NewStatusError(store.EmptyParamsErr, fmt.Sprintf(
			"update usergroup missing some params, groupId is %s", group.ID))
	}

	err := RetryTransaction("updateGroup", func() error {
		return u.updateGroup(group)
	})

	return store.Error(err)
}

func (u *groupStore) updateGroup(group *authcommon.ModifyUserGroup) error {
	tx, err := u.master.Begin()
	if err != nil {
		return err
	}

	defer func() { _ = tx.Rollback() }()

	tokenEnable := 1
	if !group.TokenEnable {
		tokenEnable = 0
	}

	// 更新用户-用户组关联数据
	if len(group.AddUserIds) != 0 {
		if err := u.addGroupRelation(tx, group.ID, group.AddUserIds); err != nil {
			log.Errorf("[Store][Group] add usergroup relation err: %s", err.Error())
			return err
		}
	}

	if len(group.RemoveUserIds) != 0 {
		if err := u.removeGroupRelation(tx, group.ID, group.RemoveUserIds); err != nil {
			log.Errorf("[Store][Group] remove usergroup relation err: %s", err.Error())
			return err
		}
	}

	modifySql := "UPDATE user_group SET token = ?, comment = ?, token_enable = ?, mtime = sysdate() " +
		" WHERE id = ? AND flag = 0"
	if _, err = tx.Exec(modifySql, []interface{}{
		group.Token,
		group.Comment,
		tokenEnable,
		group.ID,
	}...); err != nil {
		log.Errorf("[Store][Group] update usergroup main err: %s", err.Error())
		return err
	}

	if err := tx.Commit(); err != nil {
		log.Errorf("[Store][Group] update usergroup tx commit err: %s", err.Error())
		return err
	}

	return nil
}

// DeleteGroup 删除用户组
func (u *groupStore) DeleteGroup(tx store.Tx, group *authcommon.UserGroupDetail) error {
	if group.ID == "" || group.Name == "" {
		return store.NewStatusError(store.EmptyParamsErr, fmt.Sprintf(
			"delete usergroup missing some params, groupId is %s", group.ID))
	}

	dbTx := tx.GetDelegateTx().(*BaseTx)

	if _, err := dbTx.Exec("DELETE FROM user_group_relation WHERE group_id = ?", []interface{}{
		group.ID,
	}...); err != nil {
		log.Errorf("[Store][Group] clean usergroup relation err: %s", err.Error())
		return err
	}

	if _, err := dbTx.Exec("UPDATE user_group SET flag = 1, mtime = sysdate() WHERE id = ?", []interface{}{
		group.ID,
	}...); err != nil {
		log.Errorf("[Store][Group] remove usergroup err: %s", err.Error())
		return err
	}
	return nil
}

// GetGroup 根据用户组ID获取用户组
func (u *groupStore) GetGroup(groupId string) (*authcommon.UserGroupDetail, error) {
	if groupId == "" {
		return nil, store.NewStatusError(store.EmptyParamsErr, fmt.Sprintf(
			"get usergroup missing some params, groupId is %s", groupId))
	}

	getSql := `
	  SELECT ug.id, ug.name, ug.owner, ug.comment, ug.token, ug.token_enable
		  , UNIX_TIMESTAMP(ug.ctime), UNIX_TIMESTAMP(ug.mtime)
	  FROM user_group ug
	  WHERE ug.flag = 0
		  AND ug.id = ? 
	  `
	row := u.master.QueryRow(getSql, groupId)

	group := &authcommon.UserGroupDetail{
		UserGroup: &authcommon.UserGroup{},
	}
	var (
		ctime, mtime int64
		tokenEnable  int
	)

	if err := row.Scan(&group.ID, &group.Name, &group.Owner, &group.Comment, &group.Token, &tokenEnable,
		&ctime, &mtime); err != nil {
		switch err {
		case sql.ErrNoRows:
			return nil, nil
		default:
			return nil, store.Error(err)
		}
	}
	uids, err := u.getGroupLinkUserIds(group.ID)
	if err != nil {
		return nil, store.Error(err)
	}

	group.UserIds = uids
	group.TokenEnable = tokenEnable == 1
	group.CreateTime = time.Unix(ctime, 0)
	group.ModifyTime = time.Unix(mtime, 0)

	return group, nil
}

// GetGroupByName 根据 owner、name 获取用户组
func (u *groupStore) GetGroupByName(name, owner string) (*authcommon.UserGroup, error) {
	if name == "" || owner == "" {
		return nil, store.NewStatusError(store.EmptyParamsErr, fmt.Sprintf(
			"get usergroup missing some params, name=%s, owner=%s", name, owner))
	}

	var ctime, mtime int64

	getSql := `
	  SELECT ug.id, ug.name, ug.owner, ug.comment, ug.token
		  , UNIX_TIMESTAMP(ug.ctime), UNIX_TIMESTAMP(ug.mtime)
	  FROM user_group ug
	  WHERE ug.flag = 0
		  AND ug.name = ?
		  AND ug.owner = ? 
	  `
	row := u.master.QueryRow(getSql, name, owner)

	group := new(authcommon.UserGroup)

	if err := row.Scan(&group.ID, &group.Name, &group.Owner, &group.Comment, &group.Token, &ctime, &mtime); err != nil {
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
func (u *groupStore) GetGroups(filters map[string]string, offset uint32, limit uint32) (uint32,
	[]*authcommon.UserGroup, error) {

	// 如果本次请求参数携带了 user_id，那么就是查询这个用户所关联的所有用户组
	if _, ok := filters["user_id"]; ok {
		return u.listGroupByUser(filters, offset, limit)
	}
	// 正常查询用户组信息
	return u.listSimpleGroups(filters, offset, limit)
}

// listSimpleGroups 正常的用户组查询
func (u *groupStore) listSimpleGroups(filters map[string]string, offset uint32, limit uint32) (uint32,
	[]*authcommon.UserGroup, error) {

	query := make(map[string]string)
	if _, ok := filters["id"]; ok {
		query["id"] = filters["id"]
	}
	if _, ok := filters["name"]; ok {
		query["name"] = filters["name"]
	}
	filters = query

	countSql := "SELECT COUNT(*) FROM user_group ug WHERE ug.flag = 0 "
	getSql := `
	  SELECT ug.id, ug.name, ug.owner, ug.comment, ug.token, ug.token_enable
		  , UNIX_TIMESTAMP(ug.ctime), UNIX_TIMESTAMP(ug.mtime)
		  , ug.flag
	  FROM user_group ug
	  WHERE ug.flag = 0 
	  `

	args := make([]interface{}, 0)

	if len(filters) != 0 {
		for k, v := range filters {
			getSql += " AND "
			countSql += " AND "
			if newK, ok := groupAttribute[k]; ok {
				k = newK
			}
			if utils.IsPrefixWildName(v) {
				getSql += (" " + k + " like ? ")
				countSql += (" " + k + " like ? ")
				args = append(args, "%"+v[:len(v)-1]+"%")
			} else {
				getSql += (" " + k + " = ? ")
				countSql += (" " + k + " = ? ")
				args = append(args, v)
			}
		}
	}

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

// listGroupByUser 查询某个用户下所关联的用户组信息
func (u *groupStore) listGroupByUser(filters map[string]string, offset uint32, limit uint32) (uint32,
	[]*authcommon.UserGroup, error) {
	countSql := "SELECT COUNT(*) FROM user_group_relation ul LEFT JOIN user_group ug ON " +
		" ul.group_id = ug.id WHERE ug.flag = 0 "
	getSql := "SELECT ug.id, ug.name, ug.owner, ug.comment, ug.token, ug.token_enable, UNIX_TIMESTAMP(ug.ctime), " +
		" UNIX_TIMESTAMP(ug.mtime), ug.flag " +
		" FROM user_group_relation ul LEFT JOIN user_group ug ON ul.group_id = ug.id WHERE ug.flag = 0 "

	args := make([]interface{}, 0)

	if len(filters) != 0 {
		for k, v := range filters {
			getSql += " AND "
			countSql += " AND "
			if newK, ok := userLinkGroupAttributeMapping[k]; ok {
				k = newK
			}
			if utils.IsPrefixWildName(v) {
				getSql += (" " + k + " like ? ")
				countSql += (" " + k + " like ? ")
				args = append(args, "%"+v[:len(v)-1]+"%")
			} else if k == "ug.owner" {
				getSql += " (ug.owner = ?) "
				countSql += " (ug.owner = ?) "
				args = append(args, v)
			} else {
				getSql += (" " + k + " = ? ")
				countSql += (" " + k + " = ? ")
				args = append(args, v)
			}
		}
	}

	count, err := queryEntryCount(u.master, countSql, args)
	if err != nil {
		return 0, nil, err
	}

	getSql += " GROUP BY ug.id ORDER BY ug.mtime LIMIT ? , ?"
	args = append(args, offset, limit)

	groups, err := u.collectGroupsFromRows(u.master.Query, getSql, args)
	if err != nil {
		return 0, nil, err
	}

	return count, groups, nil
}

// collectGroupsFromRows 查询用户组列表
func (u *groupStore) collectGroupsFromRows(handler QueryHandler, querySql string,
	args []interface{}) ([]*authcommon.UserGroup, error) {
	rows, err := u.master.Query(querySql, args...)
	if err != nil {
		log.Error("[Store][Group] list group", zap.String("query sql", querySql), zap.Any("args", args))
		return nil, err
	}
	defer rows.Close()

	groups := make([]*authcommon.UserGroup, 0)
	for rows.Next() {
		group, err := fetchRown2UserGroup(rows)
		if err != nil {
			log.Errorf("[Store][Group] list group by user fetch rows scan err: %s", err.Error())
			return nil, err
		}
		groups = append(groups, group)
	}

	return groups, nil
}

// GetGroupsForCache .
func (u *groupStore) GetGroupsForCache(mtime time.Time, firstUpdate bool) ([]*authcommon.UserGroupDetail, error) {
	tx, err := u.slave.Begin()
	if err != nil {
		return nil, store.Error(err)
	}

	defer func() { _ = tx.Commit() }()

	args := make([]interface{}, 0)
	querySql := "SELECT id, name, owner, comment, token, token_enable, UNIX_TIMESTAMP(ctime), UNIX_TIMESTAMP(mtime), " +
		" flag FROM user_group "
	if !firstUpdate {
		querySql += " WHERE mtime >= FROM_UNIXTIME(?)"
		args = append(args, timeToTimestamp(mtime))
	}

	rows, err := tx.Query(querySql, args...)
	if err != nil {
		return nil, store.Error(err)
	}
	defer rows.Close()

	ret := make([]*authcommon.UserGroupDetail, 0)
	for rows.Next() {
		detail := &authcommon.UserGroupDetail{
			UserIds: make(map[string]struct{}, 0),
		}
		group, err := fetchRown2UserGroup(rows)
		if err != nil {
			return nil, store.Error(err)
		}
		uids, err := u.getGroupLinkUserIds(group.ID)
		if err != nil {
			return nil, store.Error(err)
		}

		detail.UserIds = uids
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
		addSql := "INSERT INTO user_group_relation (group_id, user_id) VALUE (?,?)"
		args := []interface{}{groupId, uid}
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
		addSql := "DELETE FROM user_group_relation WHERE group_id = ? AND user_id = ?"
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
	idRows, err := u.slave.Query("SELECT user_id FROM user u JOIN user_group_relation ug ON "+
		" u.id = ug.user_id WHERE ug.group_id = ?", groupId)
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

func fetchRown2UserGroup(rows *sql.Rows) (*authcommon.UserGroup, error) {
	var ctime, mtime int64
	var flag, tokenEnable int
	group := new(authcommon.UserGroup)
	if err := rows.Scan(&group.ID, &group.Name, &group.Owner, &group.Comment, &group.Token, &tokenEnable,
		&ctime, &mtime, &flag); err != nil {
		return nil, err
	}

	group.Valid = flag == 0
	group.TokenEnable = tokenEnable == 1
	group.CreateTime = time.Unix(ctime, 0)
	group.ModifyTime = time.Unix(mtime, 0)

	return group, nil
}

// cleanInValidUserGroup 清理无效的用户组数据
func cleanInValidGroup(tx *BaseTx, name, owner string) error {
	log.Infof("[Store][User] clean usergroup(%s)", name)

	str := "delete from user_group where name = ? and flag = 1"
	if _, err := tx.Exec(str, name); err != nil {
		log.Errorf("[Store][User] clean usergroup(%s) err: %s", name, err.Error())
		return err
	}

	return nil
}
