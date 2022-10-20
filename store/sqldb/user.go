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
	"strings"
	"time"

	"go.uber.org/zap"

	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/store"
)

var (
	// 用户查询相关属性对应关系
	userAttributeMapping = map[string]string{
		"owner":    "u.owner",
		"name":     "u.name",
		"group_id": "group_id",
	}

	// 用户-用户组关系查询属性对应关系
	userLinkGroupAttributeMapping map[string]string = map[string]string{
		"user_id":    "ul.user_id",
		"group_name": "ug.name",
		"group_id":   "ug.group_id",
		"owner":      "ug.owner",
	}
)

type userStore struct {
	master *BaseDB
	slave  *BaseDB
}

// AddUser 添加用户
func (u *userStore) AddUser(user *model.User) error {
	if user.ID == "" || user.Name == "" || user.Token == "" || user.Password == "" {
		return store.NewStatusError(store.EmptyParamsErr, fmt.Sprintf(
			"add user missing some params, id is %s, name is %s", user.ID, user.Name))
	}

	// 先清理无效数据
	if err := u.cleanInValidUser(user.Name, user.Owner); err != nil {
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

	addSql := "INSERT INTO user(`id`, `name`, `password`, `owner`, `source`, `token`, " +
		" `comment`, `flag`, `user_type`, " +
		" `ctime`, `mtime`, `mobile`, `email`) VALUES (?,?,?,?,?,?,?,?,?,sysdate(),sysdate(),?,?)"

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
		user.Mobile,
		user.Email,
	}...)

	if err != nil {
		return store.Error(err)
	}

	owner := user.Owner
	if owner == "" {
		owner = user.ID
	}

	if err := createDefaultStrategy(tx, model.PrincipalUser, user.ID, user.Name, user.Owner); err != nil {
		log.Error("[Auth][User] create default strategy", zap.Error(err))
		return store.Error(err)
	}

	if err := tx.Commit(); err != nil {
		log.Errorf("[Store][User] add user tx commit err: %s", err.Error())
		return store.Error(err)
	}
	return nil
}

// UpdateUser 更新用户信息
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

	modifySql := "UPDATE user SET password = ?, token = ?, comment = ?, token_enable = ?, mobile = ?, email = ?, " +
		" mtime = sysdate() WHERE id = ? AND flag = 0"

	_, err = tx.Exec(modifySql, []interface{}{
		user.Password,
		user.Token,
		user.Comment,
		tokenEnable,
		user.Mobile,
		user.Email,
		user.ID,
	}...)

	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		log.Errorf("[Store][User] update user tx commit err: %s", err.Error())
		return err
	}

	return nil
}

// DeleteUser delete user by user id
func (u *userStore) DeleteUser(user *model.User) error {
	if user.ID == "" || user.Name == "" {
		return store.NewStatusError(store.EmptyParamsErr, "delete user id parameter missing")
	}

	err := RetryTransaction("deleteUser", func() error {
		return u.deleteUser(user)
	})

	return store.Error(err)
}

// deleteUser Specific deletion user steps
// step 1. Delete the user-associated policy information
//
//	a. Delete the user's default policy
//	b. Update the latest update time of related policies, make the Cache mechanism
//	c. Delete the association relationship of the user and policy
//
// step 2. Delete the user group associated with this user
func (u *userStore) deleteUser(user *model.User) error {

	tx, err := u.master.Begin()
	if err != nil {
		return err
	}

	defer func() { _ = tx.Rollback() }()

	if err := cleanLinkStrategy(tx, model.PrincipalUser, user.ID, user.Owner); err != nil {
		return err
	}

	if _, err = tx.Exec("UPDATE user SET flag = 1 WHERE id = ?", user.ID); err != nil {
		log.Error("[Store][User] update set user flag", zap.Error(err))
		return err
	}

	if _, err = tx.Exec("UPDATE user_group SET mtime = sysdate() WHERE id IN (SELECT DISTINCT group_id FROM "+
		" user_group_relation WHERE user_id = ?)", user.ID); err != nil {
		log.Error("[Store][User] update usergroup mtime", zap.Error(err))
		return err
	}

	if _, err = tx.Exec("DELETE FROM user_group_relation WHERE user_id = ?", user.ID); err != nil {
		log.Error("[Store][User] delete usergroup relation", zap.Error(err))
		return err
	}

	if err := tx.Commit(); err != nil {
		log.Error("[Store][User] delete user tx commit", zap.Error(err))
		return err
	}
	return nil
}

func (u *userStore) GetSubCount(user *model.User) (uint32, error) {
	countSql := "SELECT COUNT(*) FROM user WHERE owner = ? AND flag = 0"

	count, err := queryEntryCount(u.master, countSql, []interface{}{user.ID})
	if err != nil {
		log.Error("[Store][User] count sub-account", zap.String("owner", user.Owner), zap.Error(err))
	}

	return count, err
}

func (u *userStore) GetUser(id string) (*model.User, error) {

	var tokenEnable, userType int

	getSql := `
		 SELECT u.id, u.name, u.password, u.owner, u.comment, u.source, u.token, u.token_enable, 
		 	u.user_type, u.mobile, u.email
		 FROM user u
		 WHERE u.flag = 0 AND u.id = ? 
	  `
	row := u.master.QueryRow(getSql, id)

	user := new(model.User)
	if err := row.Scan(&user.ID, &user.Name, &user.Password, &user.Owner, &user.Comment, &user.Source,
		&user.Token, &tokenEnable, &userType, &user.Mobile, &user.Email); err != nil {
		switch err {
		case sql.ErrNoRows:
			return nil, nil
		default:
			return nil, store.Error(err)
		}
	}

	user.TokenEnable = (tokenEnable == 1)
	user.Type = model.UserRoleType(userType)

	return user, nil
}

// GetUserByName 根据用户名、owner 获取用户
func (u *userStore) GetUserByName(name, ownerId string) (*model.User, error) {
	getSql := `
		 SELECT u.id, u.name, u.password, u.owner, u.comment, u.source, u.token, u.token_enable, 
		 	u.user_type, u.mobile, u.email
		 FROM user u
		 WHERE u.flag = 0
			  AND u.name = ?
			  AND u.owner = ? 
	  `

	row := u.master.QueryRow(getSql, name, ownerId)

	user := new(model.User)
	var tokenEnable, userType int

	if err := row.Scan(&user.ID, &user.Name, &user.Password, &user.Owner, &user.Comment, &user.Source,
		&user.Token, &tokenEnable, &userType, &user.Mobile, &user.Email); err != nil {
		switch err {
		case sql.ErrNoRows:
			return nil, nil
		default:
			return nil, store.Error(err)
		}
	}

	user.TokenEnable = (tokenEnable == 1)
	user.Type = model.UserRoleType(userType)
	return user, nil

}

// GetUserByIds Get user list data according to user ID
func (u *userStore) GetUserByIds(ids []string) ([]*model.User, error) {

	if len(ids) == 0 {
		return nil, nil
	}

	getSql := `
	  SELECT u.id, u.name, u.password, u.owner, u.comment, u.source
		  , u.token, u.token_enable, u.user_type, UNIX_TIMESTAMP(u.ctime)
		  , UNIX_TIMESTAMP(u.mtime), u.flag, u.mobile, u.email
	  FROM user u
	  WHERE u.flag = 0 
		  AND u.id IN ( 
	  `

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

	rows, err := u.master.Query(getSql, args...)
	if err != nil {
		return nil, store.Error(err)
	}
	defer rows.Close()

	users := make([]*model.User, 0)
	for rows.Next() {
		user, err := fetchRown2User(rows)
		if err != nil {
			log.Errorf("[Store][User] fetch user rows scan err: %s", err.Error())
			return nil, store.Error(err)
		}
		users = append(users, user)
	}

	return users, nil
}

// GetUsers Query user list information
// Case 1. From the user's perspective, normal query conditions
// Case 2. From the perspective of the user group, query is the list of users involved under a user group.
func (u *userStore) GetUsers(filters map[string]string, offset uint32, limit uint32) (uint32,
	[]*model.User, error) {

	if _, ok := filters["group_id"]; ok {
		return u.listGroupUsers(filters, offset, limit)
	}

	return u.listUsers(filters, offset, limit)

}

// listUsers Query user list information
func (u *userStore) listUsers(filters map[string]string, offset uint32, limit uint32) (uint32,
	[]*model.User, error) {

	countSql := "SELECT COUNT(*) FROM user WHERE flag = 0 "
	getSql := `
	  SELECT id, name, password, owner, comment, source
		  , token, token_enable, user_type, UNIX_TIMESTAMP(ctime)
		  , UNIX_TIMESTAMP(mtime), flag, mobile, email
	  FROM user
	  WHERE flag = 0 
	  `

	if val, ok := filters["hide_admin"]; ok && val == "true" {
		delete(filters, "hide_admin")
		countSql += "  AND user_type != 0 "
		getSql += "  AND user_type != 0 "
	}

	args := make([]interface{}, 0)

	if len(filters) != 0 {
		for k, v := range filters {
			getSql += " AND "
			countSql += " AND "
			if k == NameAttribute {
				if utils.IsWildName(v) {
					getSql += (" " + k + " like ? ")
					countSql += (" " + k + " like ? ")
					args = append(args, "%"+v[:len(v)-1]+"%")
				} else {
					getSql += (" " + k + " = ? ")
					countSql += (" " + k + " = ? ")
					args = append(args, v)
				}
			} else if k == OwnerAttribute {
				getSql += " (id = ? OR owner = ?) "
				countSql += " (id = ? OR owner = ?) "
				args = append(args, v, v)
				continue
			} else {
				getSql += (" " + k + " = ? ")
				countSql += (" " + k + " = ? ")
				args = append(args, v)
			}
		}
	}

	count, err := queryEntryCount(u.master, countSql, args)
	if err != nil {
		return 0, nil, store.Error(err)
	}

	getSql += " ORDER BY mtime LIMIT ? , ?"
	getArgs := append(args, offset, limit)

	users, err := u.collectUsers(u.master.Query, getSql, getArgs)
	if err != nil {
		return 0, nil, err
	}
	return count, users, nil
}

// listGroupUsers Check the user information under a user group
func (u *userStore) listGroupUsers(filters map[string]string, offset uint32, limit uint32) (uint32,
	[]*model.User, error) {
	if _, ok := filters[GroupIDAttribute]; !ok {
		return 0, nil, store.NewStatusError(store.EmptyParamsErr, "group_id is missing")
	}

	args := make([]interface{}, 0, len(filters))
	querySql := `
		  SELECT u.id, name, password, owner, u.comment, source
			  , token, token_enable, user_type, UNIX_TIMESTAMP(u.ctime)
			  , UNIX_TIMESTAMP(u.mtime), u.flag, u.mobile, u.email
		  FROM user_group_relation ug
			  LEFT JOIN user u ON ug.user_id = u.id AND u.flag = 0
		  WHERE 1=1 
	  `
	countSql := `
		  SELECT COUNT(*)
		  FROM user_group_relation ug
			  LEFT JOIN user u ON ug.user_id = u.id AND u.flag = 0 
		  WHERE 1=1 
	  `

	if val, ok := filters["hide_admin"]; ok && val == "true" {
		delete(filters, "hide_admin")
		countSql += " AND u.user_type != 0 "
		querySql += " AND u.user_type != 0 "
	}

	for k, v := range filters {
		if newK, ok := userLinkGroupAttributeMapping[k]; ok {
			k = newK
		}
		if utils.IsWildName(v) {
			querySql += " AND " + k + " like ?"
			countSql += " AND " + k + " like ?"
			args = append(args, v[:len(v)-1]+"%")
		} else {
			querySql += " AND " + k + " = ?"
			countSql += " AND " + k + " = ?"
			args = append(args, v)
		}
	}

	count, err := queryEntryCount(u.slave, countSql, args)
	if err != nil {
		return 0, nil, err
	}

	querySql += " ORDER BY u.mtime LIMIT ? , ?"
	args = append(args, offset, limit)

	users, err := u.collectUsers(u.master.Query, querySql, args)
	if err != nil {
		return 0, nil, err
	}

	return count, users, nil
}

// GetUsersForCache Get user information, mainly for cache
func (u *userStore) GetUsersForCache(mtime time.Time, firstUpdate bool) ([]*model.User, error) {

	args := make([]interface{}, 0)

	querySql := `
	  SELECT u.id, u.name, u.password, u.owner, u.comment, u.source
		  , u.token, u.token_enable, user_type, UNIX_TIMESTAMP(u.ctime)
		  , UNIX_TIMESTAMP(u.mtime), u.flag, u.mobile, u.email
	  FROM user u 
	  `

	if !firstUpdate {
		querySql += " WHERE u.mtime >= FROM_UNIXTIME(?) "
		args = append(args, timeToTimestamp(mtime))
	}

	users, err := u.collectUsers(u.master.Query, querySql, args)
	if err != nil {
		return nil, err
	}

	return users, nil
}

// collectUsers General query user list
func (u *userStore) collectUsers(handler QueryHandler, querySql string, args []interface{}) ([]*model.User, error) {

	rows, err := u.master.Query(querySql, args...)
	if err != nil {
		log.Error("[Store][User] list user ", zap.String("query sql", querySql), zap.Any("args", args))
		return nil, store.Error(err)
	}
	defer rows.Close()

	users := make([]*model.User, 0)
	for rows.Next() {
		user, err := fetchRown2User(rows)
		if err != nil {
			log.Errorf("[Store][User] fetch user rows scan err: %s", err.Error())
			return nil, store.Error(err)
		}
		users = append(users, user)
	}

	return users, nil
}

func createDefaultStrategy(tx *BaseTx, role model.PrincipalType, id, name, owner string) error {

	if strings.Compare(owner, "") == 0 {
		owner = id
	}

	// Create the user's default weight policy
	strategy := &model.StrategyDetail{
		ID:        utils.NewUUID(),
		Name:      model.BuildDefaultStrategyName(role, name),
		Action:    api.AuthAction_READ_WRITE.String(),
		Default:   true,
		Owner:     owner,
		Revision:  utils.NewUUID(),
		Resources: []model.StrategyResource{},
		Valid:     true,
		Comment:   "Default Strategy",
	}

	// 需要清理过期的 auth_strategy
	cleanInvalidRule := "DELETE FROM auth_strategy WHERE name = ? AND owner = ? AND flag = 1 AND `default` = ?"
	if _, err := tx.Exec(cleanInvalidRule, []interface{}{strategy.Name, strategy.Owner,
		strategy.Default}...); err != nil {
		return err
	}

	// Save policy master information
	saveMainSql := "INSERT INTO auth_strategy(`id`, `name`, `action`, `owner`, `comment`, `flag`, " +
		" `default`, `revision`) VALUES (?,?,?,?,?,?,?,?)"
	if _, err := tx.Exec(saveMainSql, []interface{}{strategy.ID, strategy.Name, strategy.Action,
		strategy.Owner, strategy.Comment,
		0, strategy.Default, strategy.Revision}...); err != nil {
		return err
	}

	// Insert User / Group and Policy Association
	savePrincipalSql := "INSERT INTO auth_principal(`strategy_id`, `principal_id`, `principal_role`) VALUES (?,?,?)"
	_, err := tx.Exec(savePrincipalSql, []interface{}{strategy.ID, id, role}...)
	return err
}

func fetchRown2User(rows *sql.Rows) (*model.User, error) {
	var ctime, mtime int64
	var flag, tokenEnable, userType int
	user := new(model.User)
	err := rows.Scan(&user.ID, &user.Name, &user.Password, &user.Owner, &user.Comment, &user.Source, &user.Token,
		&tokenEnable, &userType, &ctime, &mtime, &flag, &user.Mobile, &user.Email)

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

func (u *userStore) cleanInValidUser(name, owner string) error {
	log.Infof("[Store][User] clean user, name=(%s), owner=(%s)", name, owner)
	str := "delete from user where name = ? and owner = ? and flag = 1"
	if _, err := u.master.Exec(str, name, owner); err != nil {
		log.Errorf("[Store][User] clean user(%s) err: %s", name, err.Error())
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
