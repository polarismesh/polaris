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

	addSql := "INSERT INTO user(`id`, `name`, `password`, `owner`, `source`, `token`, `comment`, `flag`) VALUES (?,?,?,?,?,?,?,?)"

	_, err = tx.Exec(addSql, []interface{}{
		user.ID,
		user.Name,
		user.Password,
		user.Owner,
		user.Source,
		user.Token,
		user.Comment,
		0,
	}...)

	if err != nil {
		return err
	}

	if err := u.createDefaultStrategy(tx, fmt.Sprintf("uid/%s", user.ID), user.ID, user.Owner); err != nil {
		return store.Error(err)
	}

	if err := tx.Commit(); err != nil {
		log.Errorf("[Store][database] add user tx commit err: %s", err.Error())
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

	ret, err := tx.Exec(modifySql, []interface{}{
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
		log.Errorf("[Store][database] update user tx commit err: %s", err.Error())
		return err
	}

	if err := checkAffectedRows("User", ret, 1); err != nil {
		if store.Code(err) == store.AffectedRowsNotMatch {
			return store.NewStatusError(store.NotFoundUser, "not found user")
		}
	}
	return nil
}

// DeleteUser delete user by user id
func (u *userStore) DeleteUser(id string) error {
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

	delSql := "UPDATE user SET flag = 1 WHERE id = ?"

	_, err = tx.Exec(delSql, []interface{}{
		id,
	}...)

	if err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		log.Errorf("[Store][database] delete user tx commit err: %s", err.Error())
		return err
	}
	return nil
}

func (u *userStore) GetUser(id string) (*model.User, error) {

	getSql := "SELECT u.id, u.name, u.password, u.owner, u.source, u.token FROM user as u WHERE u.flag = 0 AND u.id = ?"
	row := u.master.QueryRow(getSql, id)

	user := new(model.User)

	err := row.Scan(&user.ID, &user.Name, &user.Password, &user.Owner, &user.Source,
		&user.Token)

	if err != nil {
		return nil, err
	}

	return user, nil
}

func (u *userStore) GetUserByName(name string) (*model.User, error) {
	getSql := "SELECT u.id, u.name, u.password, u.owner, u.source, u.token FROM user as u WHERE u.flag = 0 AND u.name = ?"
	rows, err := u.master.Query(getSql, name)
	if err != nil {
		return nil, err
	}

	if rows.Next() {
		user := new(model.User)
		err := rows.Scan(&user.ID, &user.Name, &user.Password, &user.Owner, &user.Source,
			&user.Token)
		if err != nil {
			return nil, store.Error(err)
		}

		return user, nil
	}

	return nil, nil

}

func (u *userStore) GetUserByIDS(ids []string) ([]*model.User, error) {

	if len(ids) == 0 {
		return nil, nil
	}

	getSql := "SELECT u.id, u.name, u.password, u.owner, u.source, u.token, g.name, g.token, u.ctime, u.mtime, u.flag FROM user as u WHERE u.flag = 0 AND u.id in ("

	for i := range ids {
		getSql += " ? "
		if i != len(ids)-1 {
			getSql += ","
		}
	}
	getSql += ")"

	rows, err := u.master.Query(getSql, ids)
	if err != nil {
		return nil, err
	}

	users := make([]*model.User, 0)

	for rows.Next() {
		user, err := fetchRown2User(rows)
		if err != nil {
			log.Errorf("[Store][database] fetch user rows scan err: %s", err.Error())
			return nil, err
		}
		users = append(users, user)
	}

	return users, nil
}

// ListUsers Query user list information
func (u *userStore) ListUsers(filters map[string]string, offset uint32, limit uint32) (uint32, []*model.User, error) {

	countSql := "SELECT COUNT(*) FROM user AS u join user_group AS g ON u.id = g.user_id AND u.flag = 0 AND g.flag = 0 "
	getSql := "SELECT u.id, u.name, u.password, u.owner, u.source, u.token, g.name, g.token, u.ctime, u.mtime, u.flag FROM user AS u JOIN user_group AS g ON u.id = g.user_id AND u.flag = 0 AND g.flag = 0 "

	args := make([]interface{}, 0)

	if len(filters) != 0 {
		getSql += " WHERE "
		countSql += " WHERE "
		firstIndex := true
		for k, v := range filters {
			if !firstIndex {
				getSql += " AND "
				countSql += " AND "
			}
			if k == "name" {
				getSql += (" " + k + " like ? ")
				countSql += (" " + k + " like ? ")
				args = append(args, v+"%")
			} else {
				getSql += (" " + k + " = ? ")
				countSql += (" " + k + " = ? ")
				args = append(args, v)
			}

		}
	}

	getSql += " ORDER BY u.mtime LIMIT ? , ?"
	args = append(args, offset, limit)

	rows, err := u.master.Query(getSql, args...)
	if err != nil {
		return 0, nil, err
	}

	count, err := queryEntryCount(u.master, countSql, args)
	if err != nil {
		return 0, nil, err
	}

	users := make([]*model.User, 0)
	for rows.Next() {
		user, err := fetchRown2User(rows)
		if err != nil {
			log.Errorf("[Store][database] fetch user rows scan err: %s", err.Error())
			return 0, nil, err
		}
		users = append(users, user)
	}

	return count, users, nil
}

func (u *userStore) GetUsersForCache(mtime time.Time, firstUpdate bool) ([]*model.User, error) {

	args := make([]interface{}, 0)

	querySql := "SELECT u.id, u.name, u.password, u.owner, u.source, u.token, u.token_enable, UNIX_TIMESTAMP(u.ctime), UNIX_TIMESTAMP(u.mtime), u.flag FROM user AS u"

	if !firstUpdate {
		querySql += " WHERE u.mtime >= ? "
		args = append(args, commontime.Time2String(mtime))
	}

	rows, err := u.master.Query(querySql, args...)
	if err != nil {
		return nil, err
	}

	users := make([]*model.User, 0)
	for rows.Next() {
		user, err := fetchRown2User(rows)
		if err != nil {
			log.Errorf("[Store][database] fetch user rows scan err: %s", err.Error())
			return nil, err
		}
		users = append(users, user)
	}

	return users, nil
}

func (u *userStore) AddUserGroup(group *model.UserGroup) error {
	if group.ID == "" || group.Name == "" || group.Token == "" {
		return store.NewStatusError(store.EmptyParamsErr, fmt.Sprintf(
			"add usergroup missing some params, id is %s, name is %s", group.ID, group.Name))
	}

	// 先清理无效数据
	if err := u.cleanInValidUserGroup(group.Name); err != nil {
		return err
	}

	err := RetryTransaction("addUserGroup", func() error {
		return u.addUserGroup(group)
	})

	return store.Error(err)
}

func (u *userStore) addUserGroup(group *model.UserGroup) error {
	tx, err := u.master.Begin()
	if err != nil {
		return err
	}

	defer func() { _ = tx.Rollback() }()

	addSql := "INSERT INTO user_group(id, name, owner, token, comment, flag) VALUES (?,?,?,?,?,?)"

	var flag int = 0
	if !group.Valid {
		flag = 1
	}

	_, err = tx.Exec(addSql, []interface{}{
		group.ID,
		group.Name,
		group.Owner,
		group.Token,
		group.Comment,
		flag,
	}...)

	if err != nil {
		return err
	}

	if err := u.createDefaultStrategy(tx, fmt.Sprintf("groupid/%s", group.ID), group.ID, group.Owner); err != nil {
		return store.Error(err)
	}

	if err := tx.Commit(); err != nil {
		log.Errorf("[Store][database] add usergroup tx commit err: %s", err.Error())
		return err
	}
	return nil
}

func (u *userStore) UpdateUserGroup(group *model.UserGroup) error {
	if group.ID == "" {
		return store.NewStatusError(store.EmptyParamsErr, fmt.Sprintf(
			"update usergroup missing some params, id is %s", group.ID))
	}

	err := RetryTransaction("updateUserGroup", func() error {
		return u.updateUserGroup(group)
	})

	return store.Error(err)
}

func (u *userStore) updateUserGroup(group *model.UserGroup) error {

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

	ret, err := tx.Exec(modifySql, []interface{}{
		group.Token,
		group.Comment,
		tokenEnable,
		group.ID,
	}...)

	if err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		log.Errorf("[Store][database] delete usergroup tx commit err: %s", err.Error())
		return err
	}

	if err := checkAffectedRows("UserGroup", ret, 1); err != nil {
		if store.Code(err) == store.AffectedRowsNotMatch {
			return store.NewStatusError(store.NotFoundUser, "not found usergroup")
		}
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

	delSql := "UPDATE user_group SET flag = 1 WHERE id = ?"

	_, err = tx.Exec(delSql, []interface{}{
		id,
	}...)

	if err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		log.Errorf("[Store][database] delete usergroupr tx commit err: %s", err.Error())
		return err
	}
	return nil
}

func (u *userStore) GetUserGroup(id string) (*model.UserGroup, error) {

	var ctime, mtime int64

	getSql := "SELECT id, name, owner, token, UNIX_TIMESTAMP(ctime), UNIX_TIMESTAMP(mtime) FROM user_group WHERE flag = 0 AND id = ?"
	row := u.master.QueryRow(getSql, id)

	group := new(model.UserGroup)

	err := row.Scan(&group.ID, &group.Name, &group.Owner, &group.Token, &ctime, &mtime)

	if err != nil {
		return nil, err
	}

	group.CreateTime = time.Unix(ctime, 0)
	group.ModifyTime = time.Unix(mtime, 0)

	return group, nil
}

func (u *userStore) GetUserGroupByName(name string) (*model.UserGroup, error) {

	var ctime, mtime int64

	getSql := "SELECT id, name, owner, token, UNIX_TIMESTAMP(ctime), UNIX_TIMESTAMP(mtime) FROM user_group WHERE flag = 0 AND name = ?"
	row := u.master.QueryRow(getSql, name)

	group := new(model.UserGroup)

	err := row.Scan(&group.ID, &group.Name, &group.Owner, &group.Token, &ctime, &mtime)

	if err != nil {
		return nil, err
	}

	group.CreateTime = time.Unix(ctime, 0)
	group.ModifyTime = time.Unix(mtime, 0)

	return group, nil
}

func (u *userStore) ListUserGroups(filters map[string]string, offset uint32, limit uint32) (uint32, []*model.UserGroup, error) {

	countSql := "SELECT COUNT(*) FROM user_group WHERE flag = 0 "
	getSql := "SELECT id, name, owner, token, UNIX_TIMESTAMP(ctime), UNIX_TIMESTAMP(mtime) FROM user_group WHERE flag = 0 "

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

	getSql += " ORDER BY u.mtime LIMIT ? , ?"
	args = append(args, offset, limit)

	rows, err := u.master.Query(getSql, args...)
	if err != nil {
		return 0, nil, err
	}

	count, err := queryEntryCount(u.master, countSql, args)
	if err != nil {
		return 0, nil, err
	}

	groups := make([]*model.UserGroup, 0)
	for rows.Next() {
		group, err := fetchRown2UserGroup(rows)
		if err != nil {
			log.Errorf("[Store][database] fetch usergroup rows scan err: %s", err.Error())
			return 0, nil, err
		}
		groups = append(groups, group)
	}

	return count, groups, nil
}

func (u *userStore) AddUserGroupRelation(relations *model.UserGroupRelation) error {
	if relations.GroupID == "" {
		return store.NewStatusError(store.EmptyParamsErr, fmt.Sprintf(
			"add user relation missing some params, groupid is %s", relations.GroupID))
	}
	if len(relations.UserIds) == 0 || len(relations.UserIds) > utils.MaxBatchSize {
		return store.NewStatusError(store.InvalidUserIDSlice, fmt.Sprintf(
			"user id slice is invalid, len=%d", len(relations.UserIds)))
	}

	tx, err := u.master.Begin()
	if err != nil {
		return err
	}

	defer func() { _ = tx.Rollback() }()

	userIds := relations.UserIds

	for i := range userIds {
		uid := userIds[i]
		addSql := "INSERT INTO user_group_relation (group_id, user_id, flag) VALUE (?,?,?)"
		args := []interface{}{relations.GroupID, uid, 0}
		ret, err := tx.Exec(addSql, args...)
		if err != nil {
			err = store.Error(err)
			// 之前的用户已经存在，直接忽略
			if store.Code(err) == store.DuplicateEntryErr {
				continue
			}
			return err
		}

		if err := checkAffectedRows("AddUserGroupRelation", ret, 1); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		log.Errorf("[Store][database] add user relation with group tx commit err: %s", err.Error())
		return err
	}

	return nil
}

func (u *userStore) RemoveUserGroupRelation(relations *model.UserGroupRelation) error {
	if relations.GroupID == "" {
		return store.NewStatusError(store.EmptyParamsErr, fmt.Sprintf(
			"delete user relation missing some params, groupid is %s", relations.GroupID))
	}
	if len(relations.UserIds) == 0 || len(relations.UserIds) > utils.MaxBatchSize {
		return store.NewStatusError(store.InvalidUserIDSlice, fmt.Sprintf(
			"user id slice is invalid, len=%d", len(relations.UserIds)))
	}

	tx, err := u.master.Begin()
	if err != nil {
		return err
	}

	defer func() { _ = tx.Rollback() }()

	userIds := relations.UserIds

	for i := range userIds {
		uid := userIds[i]
		addSql := "UPDATE user_group_relation SET flag = 1 WHERE group_id = ? AMD user_id = ?"
		args := []interface{}{relations.GroupID, uid}
		_, err = tx.Exec(addSql, args...)
		if err != nil {
			return store.Error(err)
		}
	}

	if err := tx.Commit(); err != nil {
		log.Errorf("[Store][database] delete user relation with group tx commit err: %s", err.Error())
		return err
	}

	return nil
}

func (u *userStore) GetUserGroupsForCache(mtime time.Time, firstUpdate bool) ([]*model.UserGroupDetail, error) {

	args := make([]interface{}, 0)

	tx, err := u.slave.Begin()
	if err != nil {
		return nil, err
	}

	defer func() { _ = tx.Commit() }()

	querySql := "SELECT id, name, owner, token, token_enable, UNIX_TIMESTAMP(ctime), UNIX_TIMESTAMP(mtime), flag FROM user_group "
	if !firstUpdate {
		querySql += " WHERE mtime >= ?"
		args = append(args, commontime.Time2String(mtime))
	}

	rows, err := tx.Query(querySql, args...)
	if err != nil {
		return nil, err
	}

	ret := make([]*model.UserGroupDetail, 0)
	for rows.Next() {
		detail := &model.UserGroupDetail{
			UserIDs: make([]string, 0),
		}
		group, err := fetchRown2UserGroup(rows)
		if err != nil {
			return nil, err
		}
		detail.UserGroup = group

		// 拉取该分组下的所有 user
		pullAllUserIds := "SELECT user_id FROM user u JOIN user_group_relation ug ON u.id = ug.user_id AND u.flag = 0 WHERE ug.group_id = ?"

		idRows, err := tx.Query(pullAllUserIds, group.ID)

		for idRows.Next() {
			var uid string
			if err := idRows.Scan(&uid); err != nil {
				return nil, store.Error(err)
			}
			detail.UserIDs = append(detail.UserIDs, uid)
		}

		ret = append(ret, detail)
	}

	return ret, nil
}

func (u *userStore) createDefaultStrategy(tx *BaseTx, principal, id, owner string) error {
	// 创建该用户的默认权限策略
	strategy := &model.StrategyDetail{
		ID:        utils.NewUUID(),
		Name:      fmt.Sprintf("%s%s", model.DefaultStrategyPrefix, id),
		Principal: principal,
		Action:    api.AuthAction_READ_WRITE.String(),
		Comment:   "default user auth_strategy",
		Default:   true,
		Owner:     owner,
		Resources: []model.StrategyResource{},
		Valid:     true,
	}

	// 保存策略主信息
	saveMainSql := "INSERT INTO auth_strategy(`id`, `name`, `principal`, `action`, `owner`, `comment`, `flag`, `default`) VALUES (?,?,?,?,?,?,?,?)"
	_, err := tx.Exec(saveMainSql, []interface{}{strategy.ID, strategy.Name, strategy.Principal, strategy.Action, strategy.Owner, strategy.Comment, 0, strategy.Default}...)

	if err != nil {
		return err
	}

	return nil
}

func fetchRown2User(rows *sql.Rows) (*model.User, error) {
	var ctime, mtime int64
	var flag, tokenEnable int
	user := new(model.User)
	err := rows.Scan(&user.ID, &user.Name, &user.Password, &user.Owner, &user.Source, &user.Token, &tokenEnable, &ctime, &mtime, &flag)

	if err != nil {
		return nil, err
	}

	if flag == 1 {
		user.Valid = false
	}
	if tokenEnable == 1 {
		user.TokenEnable = true
	}

	user.CreateTime = time.Unix(ctime, 0)
	user.ModifyTime = time.Unix(mtime, 0)

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

	if flag == 1 {
		group.Valid = false
	}
	if tokenEnable == 1 {
		group.TokenEnable = true
	}

	group.CreateTime = time.Unix(ctime, 0)
	group.ModifyTime = time.Unix(mtime, 0)

	return group, nil
}

func (u *userStore) cleanInValidUser(name string) error {
	log.Infof("[Store][database] clean user(%s)", name)
	str := "delete from user where name = ? and flag = 1"
	_, err := u.master.Exec(str, name)
	if err != nil {
		log.Errorf("[Store][database] clean user(%s) err: %s", name, err.Error())
		return err
	}

	return nil
}

func (u *userStore) cleanInValidUserGroup(name string) error {
	log.Infof("[Store][database] clean usergroup(%s)", name)
	str := "delete from user_group_relation where user_id = (select id from user_group where name = ? and flag = 1) and flag = 1"
	_, err := u.master.Exec(str, name)
	if err != nil {
		log.Errorf("[Store][database] clean usergroup(%s) err: %s", name, err.Error())
		return err
	}

	str = "delete from user_group where name = ? and flag = 1"
	_, err = u.master.Exec(str, name)
	if err != nil {
		log.Errorf("[Store][database] clean usergroup(%s) err: %s", name, err.Error())
		return err
	}

	return nil
}

func checkAffectedRows(label string, result sql.Result, count int64) error {
	n, err := result.RowsAffected()
	if err != nil {
		log.Errorf("[Store][%s] get rows affected err: %s", label, err.Error())
		return err
	}

	if n == count {
		return nil
	}
	log.Errorf("[Store][%s] get rows affected result(%d) is not match expect(%d)", label, n, count)
	return store.NewStatusError(store.AffectedRowsNotMatch, "affected rows not match")
}
