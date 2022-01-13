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

	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/common/model"
	commontime "github.com/polarismesh/polaris-server/common/time"
	"github.com/polarismesh/polaris-server/common/utils"
	"github.com/polarismesh/polaris-server/store"
	"go.uber.org/zap"
)

var (
	userAttributeMapping map[string]string = map[string]string{
		OwnerAttribute:   "u.owner",
		NameAttribute:    "u.name",
		GroupIDAttribute: "group_id",
	}
)

type userStore struct {
	master *BaseDB
	slave  *BaseDB
}

func (u *userStore) AddUser(user *model.User) error {
	if user.ID == "" || user.Name == "" || user.Token == "" || user.Password == "" {
		return store.NewStatusError(store.EmptyParamsErr, fmt.Sprintf(
			"add user missing some params, id is %s, name is %s", user.ID, user.Name))
	}

	// 先清理无效数据
	if err := u.cleanInValidUser(user.Name); err != nil {
		return err
	}

	err := RetryTransaction("addUser", func() error {
		return u.addUser(user)
	})

	return store.Error(err)
}

func (u *userStore) addUser(user *model.User) error {

	tx, err := u.master.Begin()
	if err != nil {
		return err
	}

	defer func() { _ = tx.Rollback() }()

	addSql := "INSERT INTO user(`id`, `name`, `password`, `owner`, `source`, `token`, `comment`, `flag`, `user_type`, " +
		" `ctime`, `mtime`) VALUES (?,?,?,?,?,?,?,?,?,sysdate(),sysdate())"

	_, err = tx.Exec(addSql, []interface{}{
		user.ID,
		user.Name,
		user.Password,
		user.Owner,
		user.Source,
		user.Token,
		user.Comment,
		0,
		user.Type,
	}...)

	if err != nil {
		return err
	}

	if err := u.createDefaultStrategy(tx, model.PrincipalUser, user.ID, user.Owner); err != nil {
		return store.Error(err)
	}

	if err := tx.Commit(); err != nil {
		log.GetStoreLogger().Errorf("[Store][User] add user tx commit err: %s", err.Error())
		return err
	}
	return nil
}

func (u *userStore) UpdateUser(user *model.User) error {
	if user.ID == "" || user.Name == "" || user.Token == "" || user.Password == "" {
		return store.NewStatusError(store.EmptyParamsErr, fmt.Sprintf(
			"update user missing some params, id is %s, name is %s", user.ID, user.Name))
	}

	err := RetryTransaction("updateUser", func() error {
		return u.updateUser(user)
	})

	return store.Error(err)
}

func (u *userStore) updateUser(user *model.User) error {

	tx, err := u.master.Begin()
	if err != nil {
		return err
	}

	defer func() { _ = tx.Rollback() }()

	tokenEnable := 1
	if !user.TokenEnable {
		tokenEnable = 0
	}

	modifySql := "UPDATE user SET password = ?, token = ?, comment = ?, token_enable = ? WHERE id = ? AND flag = 0"

	_, err = tx.Exec(modifySql, []interface{}{
		user.Password,
		user.Token,
		user.Comment,
		tokenEnable,
		user.ID,
	}...)

	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		log.GetStoreLogger().Errorf("[Store][User] update user tx commit err: %s", err.Error())
		return err
	}

	return nil
}

// DeleteUser delete user by user id
func (u *userStore) DeleteUser(id string) error {
	if id == "" {
		return store.NewStatusError(store.EmptyParamsErr, "delete user id parameter missing")
	}

	err := RetryTransaction("deleteUser", func() error {
		return u.deleteUser(id)
	})

	return store.Error(err)
}

func (u *userStore) deleteUser(id string) error {

	tx, err := u.master.Begin()
	if err != nil {
		return err
	}

	defer func() { _ = tx.Rollback() }()

	if _, err = tx.Exec("UPDATE user SET flag = 1 WHERE id = ?", []interface{}{
		id,
	}...); err != nil {
		return err
	}

	if _, err = tx.Exec("UPDATE auth_strategy SET flag = 1 WHERE name = ?", []interface{}{
		model.BuildDefaultStrategyName(id, model.PrincipalUserGroup),
	}...); err != nil {
		return err
	}

	if _, err = tx.Exec("DELETE FROM auth_strategy_resource WHERE strategy_id = (SELECT id FROM auth_strategy WHERE name = ?)", []interface{}{
		model.BuildDefaultStrategyName(id, model.PrincipalUserGroup),
	}...); err != nil {
		return err
	}

	if _, err = tx.Exec("DELETE FROM auth_principal WHERE strategy_id = (SELECT id FROM auth_strategy WHERE name = ?)", []interface{}{
		model.BuildDefaultStrategyName(id, model.PrincipalUserGroup),
	}...); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		log.GetStoreLogger().Errorf("[Store][User] delete user tx commit err: %s", err.Error())
		return err
	}
	return nil
}

func (u *userStore) GetUser(id string) (*model.User, error) {

	user := new(model.User)
	var tokenEnable, userType int
	getSql := "SELECT u.id, u.name, u.password, u.owner, u.source, u.token, u.token_enable, u.user_type FROM user as u WHERE u.flag = 0 AND u.id = ?"
	row := u.master.QueryRow(getSql, id)
	if err := row.Scan(&user.ID, &user.Name, &user.Password, &user.Owner, &user.Source,
		&user.Token, &tokenEnable, &userType); err != nil {
		switch err {
		case sql.ErrNoRows:
			return nil, nil
		default:
			return nil, store.Error(err)
		}
	}

	user.TokenEnable = tokenEnable == 1
	user.Type = model.UserRoleType(userType)

	return user, nil
}

func (u *userStore) GetUserByName(name, ownerId string) (*model.User, error) {
	getSql := "SELECT u.id, u.name, u.password, u.owner, u.source, u.token, u.token_enable, u.user_type FROM user as u WHERE u.flag = 0 AND u.name = ? AND u.owner = ?"

	user := new(model.User)
	var tokenEnable, userType int

	row := u.master.QueryRow(getSql, name, ownerId)
	if err := row.Scan(&user.ID, &user.Name, &user.Password, &user.Owner, &user.Source,
		&user.Token, &tokenEnable, &userType); err != nil {
		switch err {
		case sql.ErrNoRows:
			return nil, nil
		default:
			return nil, store.Error(err)
		}
	}

	user.TokenEnable = tokenEnable == 1
	user.Type = model.UserRoleType(userType)
	return user, nil

}

// GetUserByIDS
//  @receiver u
//  @param ids
//  @return []*model.User
//  @return error
func (u *userStore) GetUserByIDS(ids []string) ([]*model.User, error) {

	if len(ids) == 0 {
		return nil, nil
	}

	getSql := "SELECT u.id, u.name, u.password, u.owner, u.source, u.token, u.token_enable, u.user_type, UNIX_TIMESTAMP(u.ctime), UNIX_TIMESTAMP(u.mtime), u.flag FROM user as u WHERE u.flag = 0 AND u.id in ("

	for i := range ids {
		getSql += " ? "
		if i != len(ids)-1 {
			getSql += ","
		}
	}
	getSql += ")"

	args := make([]interface{}, 0, 8)
	for index := range ids {
		args = append(args, ids[index])
	}

	rows, err := u.master.Query(getSql, append(args, args...))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]*model.User, 0)
	for rows.Next() {
		user, err := fetchRown2User(rows)
		if err != nil {
			log.GetStoreLogger().Errorf("[Store][User] fetch user rows scan err: %s", err.Error())
			return nil, err
		}
		users = append(users, user)
	}

	return users, nil
}

// ListUsers Query user list information
//  @receiver u
//  @param filters
//  @param offset
//  @param limit
//  @return uint32
//  @return []*model.User
//  @return error
func (u *userStore) ListUsers(filters map[string]string, offset uint32, limit uint32) (uint32, []*model.User, error) {

	countSql := "SELECT COUNT(*) FROM user  WHERE flag = 0 "
	getSql := "SELECT id, name, password, owner, source, token, token_enable, user_type, UNIX_TIMESTAMP(ctime), UNIX_TIMESTAMP(mtime), flag FROM user  WHERE flag = 0 "

	args := make([]interface{}, 0)

	if len(filters) != 0 {
		for k, v := range filters {
			getSql += " AND "
			countSql += " AND "
			if k == NameAttribute {
				getSql += (" " + k + " like ? ")
				countSql += (" " + k + " like ? ")
				args = append(args, v+"%")
			} else if k == OwnerAttribute {
				getSql += "(id = ? OR owner = ?)"
				countSql += "(id = ? OR owner = ?)"
				args = append(args, v, v)
				continue
			} else {
				getSql += (" " + k + " = ? ")
				countSql += (" " + k + " = ? ")
				args = append(args, v)
			}
		}
	}

	getSql += " ORDER BY mtime LIMIT ? , ?"
	getArgs := append(args, offset, limit)

	log.GetStoreLogger().Debug("list user ", zap.String("sql", getSql), zap.Any("args", getArgs))
	log.GetStoreLogger().Debug("count list user", zap.String("sql", countSql), zap.Any("args", args))

	count, err := queryEntryCount(u.master, countSql, args)
	if err != nil {
		return 0, nil, err
	}

	rows, err := u.master.Query(getSql, getArgs...)
	if err != nil {
		return 0, nil, err
	}
	defer rows.Close()

	users := make([]*model.User, 0)
	for rows.Next() {
		user, err := fetchRown2User(rows)
		if err != nil {
			log.GetStoreLogger().Errorf("[Store][User] fetch user rows scan err: %s", err.Error())
			return 0, nil, err
		}
		users = append(users, user)
	}

	return count, users, nil
}

// GetUsersForCache
//  @receiver u
//  @param mtime
//  @param firstUpdate
//  @return []*model.User
//  @return error
func (u *userStore) GetUsersForCache(mtime time.Time, firstUpdate bool) ([]*model.User, error) {

	args := make([]interface{}, 0)

	querySql := "SELECT u.id, u.name, u.password, u.owner, u.source, u.token, u.token_enable, user_type, UNIX_TIMESTAMP(u.ctime), UNIX_TIMESTAMP(u.mtime), u.flag FROM user AS u"

	if !firstUpdate {
		querySql += " WHERE u.mtime >= ? "
		args = append(args, commontime.Time2String(mtime))
	}

	rows, err := u.master.Query(querySql, args...)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	users := make([]*model.User, 0)
	for rows.Next() {
		user, err := fetchRown2User(rows)
		if err != nil {
			log.GetStoreLogger().Errorf("[Store][User] fetch user rows scan err: %s", err.Error())
			return nil, err
		}
		users = append(users, user)
	}

	return users, nil
}

func (u *userStore) AddUserGroup(group *model.UserGroupDetail) error {
	if group.ID == "" || group.Name == "" || group.Token == "" {
		return store.NewStatusError(store.EmptyParamsErr, fmt.Sprintf(
			"add usergroup missing some params, id is %s, name is %s", group.ID, group.Name))
	}

	// 先清理无效数据
	if err := u.cleanInValidUserGroup(group.Name, group.Owner); err != nil {
		return store.Error(err)
	}

	err := RetryTransaction("addUserGroup", func() error {
		return u.addUserGroup(group)
	})

	return store.Error(err)
}

func (u *userStore) addUserGroup(group *model.UserGroupDetail) error {
	tx, err := u.master.Begin()
	if err != nil {
		return err
	}

	defer func() { _ = tx.Rollback() }()

	addSql := "INSERT INTO user_group(id, name, owner, token, comment, flag, ctime, mtime) VALUES (?,?,?,?,?,?,sysdate(),sysdate())"

	var flag int = 0
	if !group.Valid {
		flag = 1
	}

	if _, err = tx.Exec(addSql, []interface{}{
		group.ID,
		group.Name,
		group.Owner,
		group.Token,
		group.Comment,
		flag,
	}...); err != nil {
		return err
	}

	uids := make([]string, 0, len(group.UserIDs))
	for uid := range group.UserIDs {
		uids = append(uids, uid)
	}

	if err := u.addUserGroupRelation(tx, group.ID, uids); err != nil {
		return err
	}

	if err := u.createDefaultStrategy(tx, model.PrincipalUserGroup, group.ID, group.Owner); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		log.GetStoreLogger().Errorf("[Store][User] add usergroup tx commit err: %s", err.Error())
		return err
	}
	return nil
}

func (u *userStore) UpdateUserGroup(group *model.ModifyUserGroup) error {
	if group.ID == "" {
		return store.NewStatusError(store.EmptyParamsErr, fmt.Sprintf(
			"update usergroup missing some params, id is %s", group.ID))
	}

	if err := u.cleanInValidUserGroupRelation(group.ID); err != nil {
		return store.Error(err)
	}

	err := RetryTransaction("updateUserGroup", func() error {
		return u.updateUserGroup(group)
	})

	return store.Error(err)
}

func (u *userStore) updateUserGroup(group *model.ModifyUserGroup) error {

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
		return err
	}

	// 更新用户-用户组关联数据
	if len(group.AddUserIds) != 0 {
		if err := u.addUserGroupRelation(tx, group.ID, group.AddUserIds); err != nil {
			return err
		}
	}

	if len(group.RemoveUserIds) != 0 {
		if err := u.removeUserGroupRelation(tx, group.ID, group.RemoveUserIds); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		log.GetStoreLogger().Errorf("[Store][User] delete usergroup tx commit err: %s", err.Error())
		return err
	}

	return nil
}

func (u *userStore) DeleteUserGroup(id string) error {
	err := RetryTransaction("deleteUserGroup", func() error {
		return u.deleteUserGroup(id)
	})

	return store.Error(err)
}

func (u *userStore) deleteUserGroup(id string) error {

	tx, err := u.master.Begin()
	if err != nil {
		return err
	}

	defer func() { _ = tx.Rollback() }()

	if _, err = tx.Exec("UPDATE user_group SET flag = 1 WHERE id = ?", []interface{}{
		id,
	}...); err != nil {
		return err
	}

	if _, err = tx.Exec("UPDATE user_group_relation SET flag = 1 WHERE group_id = ?", []interface{}{
		id,
	}...); err != nil {
		return err
	}

	if _, err = tx.Exec("UPDATE auth_strategy SET flag = 1 WHERE name = ?", []interface{}{
		model.BuildDefaultStrategyName(id, model.PrincipalUserGroup),
	}...); err != nil {
		return err
	}

	if _, err = tx.Exec("DELETE FROM auth_strategy_resource WHERE strategy_id = (SELECT id FROM auth_strategy WHERE name = ?)", []interface{}{
		model.BuildDefaultStrategyName(id, model.PrincipalUserGroup),
	}...); err != nil {
		return err
	}

	if _, err = tx.Exec("DELETE FROM auth_principal WHERE strategy_id = (SELECT id FROM auth_strategy WHERE name = ?)", []interface{}{
		model.BuildDefaultStrategyName(id, model.PrincipalUserGroup),
	}...); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		log.GetStoreLogger().Errorf("[Store][User] delete usergroupr tx commit err: %s", err.Error())
		return err
	}
	return nil
}

func (u *userStore) GetUserGroup(id string) (*model.UserGroup, error) {

	var ctime, mtime int64

	getSql := "SELECT id, name, owner, token, token_enable, UNIX_TIMESTAMP(ctime), UNIX_TIMESTAMP(mtime) FROM user_group WHERE flag = 0 AND id = ?"
	row := u.master.QueryRow(getSql, id)

	group := new(model.UserGroup)

	var tokenEnable int
	if err := row.Scan(&group.ID, &group.Name, &group.Owner, &group.Token, &tokenEnable, &ctime, &mtime); err != nil {
		switch err {
		case sql.ErrNoRows:
			return nil, nil
		default:
			return nil, store.Error(err)
		}
	}

	group.TokenEnable = tokenEnable == 1
	group.CreateTime = time.Unix(ctime, 0)
	group.ModifyTime = time.Unix(mtime, 0)

	return group, nil
}

func (u *userStore) GetUserGroupByName(name string) (*model.UserGroup, error) {

	var ctime, mtime int64

	getSql := "SELECT id, name, owner, token, UNIX_TIMESTAMP(ctime), UNIX_TIMESTAMP(mtime) FROM user_group WHERE flag = 0 AND name = ?"
	row := u.master.QueryRow(getSql, name)

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

func (u *userStore) ListUserGroups(filters map[string]string, offset uint32, limit uint32) (uint32, []*model.UserGroup, error) {

	countSql := "SELECT COUNT(*) FROM user_group WHERE flag = 0 "
	getSql := "SELECT id, name, owner, token, token_enable, UNIX_TIMESTAMP(ctime), UNIX_TIMESTAMP(mtime), flag FROM user_group WHERE flag = 0 "

	args := make([]interface{}, 0)

	if len(filters) != 0 {
		for k, v := range filters {
			getSql += " AND "
			countSql += " AND "
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

	rows, err := u.master.Query(getSql, args...)
	if err != nil {
		return 0, nil, err
	}
	defer rows.Close()

	groups := make([]*model.UserGroup, 0)
	for rows.Next() {
		group, err := fetchRown2UserGroup(rows)
		if err != nil {
			log.GetStoreLogger().Errorf("[Store][User] fetch usergroup rows scan err: %s", err.Error())
			return 0, nil, err
		}
		groups = append(groups, group)
	}

	return count, groups, nil
}

func (u *userStore) addUserGroupRelation(tx *BaseTx, groupId string, userIds []string) error {
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

func (u *userStore) removeUserGroupRelation(tx *BaseTx, groupId string, userIds []string) error {
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

func (u *userStore) GetUserGroupsForCache(mtime time.Time, firstUpdate bool) ([]*model.UserGroupDetail, error) {
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

// ListUserByGroup
//  @receiver u
//  @param filters
//  @param offset
//  @param limit
//  @return uint32
//  @return []*model.User
//  @return error
func (u *userStore) ListUserByGroup(filters map[string]string, offset uint32, limit uint32) (uint32, []*model.User, error) {
	if _, ok := filters[GroupIDAttribute]; !ok {
		return 0, nil, store.NewStatusError(store.EmptyParamsErr, "group_id is missing")
	}
	filters["ug.group_id"] = filters[GroupIDAttribute]
	delete(filters, GroupIDAttribute)

	args := make([]interface{}, 0, len(filters))
	querySql := `
		SELECT u.id, name, password, owner, source
			, token, token_enable, user_type, UNIX_TIMESTAMP(u.ctime)
			, UNIX_TIMESTAMP(u.mtime), u.flag
		FROM user_group_relation ug
			LEFT JOIN user u ON ug.user_id = u.id AND u.flag = 0 AND ug.flag = 0
		WHERE 1=1
	`
	countSql := `
		SELECT COUNT(*)
		FROM user_group_relation ug
			LEFT JOIN user u ON ug.user_id = u.id AND u.flag = 0 AND ug.flag = 0
		WHERE 1=1
	`

	for k, v := range filters {
		querySql += " " + And + " " + k + " = ?"
		countSql += " " + And + " " + k + " = ?"

		args = append(args, v)
	}

	count, err := queryEntryCount(u.slave, countSql, args)
	log.GetStoreLogger().Debug("count list user", zap.String("sql", countSql), zap.Any("args", args))

	if err != nil {
		return 0, nil, err
	}

	querySql += " ORDER BY u.mtime LIMIT ? , ?"
	args = append(args, offset, limit)
	log.GetStoreLogger().Debug("list user by group", zap.String("sql", querySql), zap.Any("args", args))

	rows, err := u.slave.Query(querySql, args...)
	if err != nil {
		return 0, nil, err
	}

	defer rows.Close()
	users := make([]*model.User, 0)
	for rows.Next() {
		user, err := fetchRown2User(rows)
		if err != nil {
			log.GetStoreLogger().Errorf("[Store][User] fetch user rows scan err: %s", err.Error())
			return 0, nil, err
		}
		users = append(users, user)
	}

	return count, users, nil
}

func (u *userStore) getGroupLinkUserIds(groupId string) (map[string]struct{}, error) {

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

func (u *userStore) createDefaultStrategy(tx *BaseTx, role model.PrincipalType, id, owner string) error {
	// 创建该用户的默认权限策略
	strategy := &model.StrategyDetail{
		ID:        utils.NewUUID(),
		Name:      model.BuildDefaultStrategyName(id, role),
		Action:    api.AuthAction_READ_WRITE.String(),
		Comment:   "default user auth_strategy",
		Default:   true,
		Owner:     owner,
		Revision:  utils.NewUUID(),
		Resources: []model.StrategyResource{},
		Valid:     true,
	}

	// 保存策略主信息
	saveMainSql := "INSERT INTO auth_strategy(`id`, `name`, `action`, `owner`, `comment`, `flag`, " +
		" `default`, `revision`) VALUES (?,?,?,?,?,?,?,?)"
	_, err := tx.Exec(saveMainSql, []interface{}{strategy.ID, strategy.Name, strategy.Action, strategy.Owner, strategy.Comment,
		0, strategy.Default, strategy.Revision}...)

	if err != nil {
		return err
	}

	savePrincipalSql := "INSERT INTO auth_principal(`strategy_id`, `principal_id`, `principal_role`) VALUES (?,?,?)"
	_, err = tx.Exec(savePrincipalSql, []interface{}{strategy.ID, id, role}...)
	if err != nil {
		return err
	}

	return nil
}

func fetchRown2User(rows *sql.Rows) (*model.User, error) {
	var ctime, mtime int64
	var flag, tokenEnable, userType int
	user := new(model.User)
	err := rows.Scan(&user.ID, &user.Name, &user.Password, &user.Owner, &user.Source, &user.Token,
		&tokenEnable, &userType, &ctime, &mtime, &flag)

	if err != nil {
		return nil, err
	}

	user.Valid = flag == 0
	user.TokenEnable = tokenEnable == 1
	user.CreateTime = time.Unix(ctime, 0)
	user.ModifyTime = time.Unix(mtime, 0)
	user.Type = model.UserRoleType(userType)

	return user, nil
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

func (u *userStore) cleanInValidUser(name string) error {
	log.GetStoreLogger().Infof("[Store][User] clean user(%s)", name)
	str := "delete from user where name = ? and flag = 1"
	_, err := u.master.Exec(str, name)
	if err != nil {
		log.GetStoreLogger().Errorf("[Store][User] clean user(%s) err: %s", name, err.Error())
		return err
	}

	return nil
}

func (u *userStore) cleanInValidUserGroup(name, owner string) error {
	log.GetStoreLogger().Infof("[Store][User] clean usergroup(%s)", name)
	str := "delete from user_group_relation where group_id = (select id from user_group where name = ? and owner = ? and flag = 1) and flag = 1"
	_, err := u.master.Exec(str, name, owner)
	if err != nil {
		log.GetStoreLogger().Errorf("[Store][User] clean usergroup(%s) err: %s", name, err.Error())
		return err
	}

	str = "delete from user_group where name = ? and flag = 1"
	_, err = u.master.Exec(str, name)
	if err != nil {
		log.GetStoreLogger().Errorf("[Store][User] clean usergroup(%s) err: %s", name, err.Error())
		return err
	}

	return nil
}

func (u *userStore) cleanInValidUserGroupRelation(id string) error {
	log.GetStoreLogger().Infof("[Store][User] clean usergroup(%s)", id)
	str := "delete from user_group_relation where group_id = ? and flag = 1"
	_, err := u.master.Exec(str, id)
	if err != nil {
		log.GetStoreLogger().Errorf("[Store][User] clean usergroup(%s) err: %s", id, err.Error())
		return err
	}
	return nil
}

func checkAffectedRows(label string, result sql.Result, count int64) error {
	n, err := result.RowsAffected()
	if err != nil {
		log.GetStoreLogger().Errorf("[Store][%s] get rows affected err: %s", label, err.Error())
		return err
	}

	if n == count {
		return nil
	}
	log.GetStoreLogger().Errorf("[Store][%s] get rows affected result(%d) is not match expect(%d)", label, n, count)
	return store.NewStatusError(store.AffectedRowsNotMatch, "affected rows not match")
}
