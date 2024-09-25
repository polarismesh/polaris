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
	"encoding/json"
	"time"

	authcommon "github.com/polarismesh/polaris/common/model/auth"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/store"
	"go.uber.org/zap"
)

type roleStore struct {
	master *BaseDB
	slave  *BaseDB
}

// AddRole Add a role
func (s *roleStore) AddRole(role *authcommon.Role) error {
	if role.ID == "" || role.Name == "" {
		return store.NewStatusError(store.EmptyParamsErr, "role id or name is empty")
	}
	err := s.master.processWithTransaction("add_role", func(tx *BaseTx) error {
		if _, err := tx.Exec("DELETE FROM auth_role WHERE id = ? AND flag = 1", role.ID); err != nil {
			log.Error("[store][role] delete invalid role", zap.String("name", role.Name), zap.Error(err))
			return err
		}
		addSql := `
INSERT INTO auth_role (id, name, owner, source, role_type
	, comment, flag, metadata, ctime, mtime)
VALUES (?, ?, ?, ?, ?
	, ?, 0, ?, sysdate(), sysdate())
		`
		args := []interface{}{role.ID, role.Name, role.Owner, role.Source, role.Type, role.Comment, utils.MustJson(role.Metadata)}
		if _, err := tx.Exec(addSql, args...); err != nil {
			log.Error("[store][role] add role main info", zap.String("name", role.Name), zap.Error(err))
			return err
		}

		if err := s.savePrincipals(tx, role); err != nil {
			log.Error("[store][role] save role principal info", zap.String("name", role.Name), zap.Error(err))
			return err
		}
		return nil
	})
	return store.Error(err)
}

func (s *roleStore) savePrincipals(tx *BaseTx, role *authcommon.Role) error {
	if _, err := tx.Exec("DELETE FROM auth_role_principal WHERE id = ?", role.ID); err != nil {
		log.Error("[store][role] clean role principal info", zap.String("name", role.Name), zap.Error(err))
		return err
	}

	insertTpl := "INSERT INTO auth_role_principal(role_id, principal_id, principal_role) VALUES (?, ?, ?)"

	for i := range role.Users {
		args := []interface{}{role.ID, role.Users[i].ID, authcommon.PrincipalUser}
		if _, err := tx.Exec(insertTpl, args...); err != nil {
			return err
		}
	}
	for i := range role.UserGroups {
		args := []interface{}{role.ID, role.UserGroups[i].ID, authcommon.PrincipalGroup}
		if _, err := tx.Exec(insertTpl, args...); err != nil {
			return err
		}
	}
	return nil
}

// UpdateRole Update a role
func (s *roleStore) UpdateRole(role *authcommon.Role) error {
	if role.ID == "" {
		return store.NewStatusError(store.EmptyParamsErr, "role id is empty")
	}
	err := s.master.processWithTransaction("update_role", func(tx *BaseTx) error {
		updateSql := `
UPDATE auth_role
SET source = ?, role_type = ?, comment = ?, metadata = ?, mtime = sysdate()
WHERE id = ?
				`
		args := []interface{}{role.Source, role.Type, role.Comment, utils.MustJson(role.Metadata), role.ID}
		if _, err := tx.Exec(updateSql, args...); err != nil {
			log.Error("[store][role] update role main info", zap.String("name", role.Name), zap.Error(err))
			return err
		}

		if err := s.savePrincipals(tx, role); err != nil {
			log.Error("[store][role] save role principal info", zap.String("name", role.Name), zap.Error(err))
			return err
		}
		return nil
	})
	return store.Error(err)
}

// DeleteRole Delete a role
func (s *roleStore) DeleteRole(role *authcommon.Role) error {
	if role.ID == "" {
		return store.NewStatusError(store.EmptyParamsErr, "role id is empty")
	}
	err := s.master.processWithTransaction("delete_role", func(tx *BaseTx) error {
		if _, err := tx.Exec("UPDATE auth_role SET flag = 1 WHERE id = ?", role.ID); err != nil {
			log.Error("[store][role] delete role", zap.String("name", role.Name), zap.Error(err))
			return err
		}
		return nil
	})
	return store.Error(err)
}

// CleanPrincipalRoles clean principal roles
func (s *roleStore) CleanPrincipalRoles(tx store.Tx, p *authcommon.Principal) error {
	dbTx := tx.GetDelegateTx().(*BaseTx)
	listSql := "SELECT role_id FROM auth_role_principal WHERE principal_id = ? AND principal_role = ?"
	rows, err := dbTx.Query(listSql, p.PrincipalID, p.PrincipalType)
	if err != nil {
		log.Error("[store][role] list principal all roles", zap.String("principal", p.String()), zap.Error(err))
		return err
	}
	defer func() {
		_ = rows.Close()
	}()

	for rows.Next() {
		var roleId string
		if err := rows.Scan(&roleId); err != nil {
			log.Error("[store][role] fetch one record principal role", zap.String("principal", p.String()),
				zap.String("type", p.PrincipalType.String()), zap.Error(err))
			return err
		}

		if _, err := dbTx.Exec("UPDATE auth_role SET mtime = sysdate() WHERE id = ?", roleId); err != nil {
			log.Error("[store][role] update role when clean principal role", zap.String("id", roleId),
				zap.String("principal", p.String()), zap.Error(err))
			return err
		}
	}

	if _, err := dbTx.Exec("DELETE FROM auth_role_principal WHERE principal_id = ? AND principal_role = ?",
		p.PrincipalID, p.PrincipalType); err != nil {
		log.Error("[store][role] clean principal all roles", zap.String("principal", p.String()), zap.Error(err))
		return store.Error(err)
	}
	return nil
}

// GetRole get more role for cache update
func (s *roleStore) GetMoreRoles(firstUpdate bool, mtime time.Time) ([]*authcommon.Role, error) {
	tx, err := s.slave.Begin()
	if err != nil {
		return nil, store.Error(err)
	}

	defer func() { _ = tx.Commit() }()

	args := make([]interface{}, 0)
	querySql := "SELECT id, name, owner, source, role_type, comment, flag, metadata, UNIX_TIMESTAMP(ctime), " +
		" UNIX_TIMESTAMP(mtime) FROM auth_role "
	if !firstUpdate {
		querySql += " WHERE mtime >= FROM_UNIXTIME(?)"
		args = append(args, timeToTimestamp(mtime))
	} else {
		querySql += " WHERE flag = 0"
	}

	rows, err := tx.Query(querySql, args...)
	if err != nil {
		log.Error("[store][role] get more role for cache", zap.String("query sql", querySql),
			zap.Any("args", args), zap.Error(err))
		return nil, store.Error(err)
	}
	defer func() {
		_ = rows.Close()
	}()

	roles := make([]*authcommon.Role, 0, 32)
	for rows.Next() {
		var (
			ctime, mtime int64
			flag         int16
			metadata     string
		)
		ret := &authcommon.Role{
			Metadata:   map[string]string{},
			Users:      make([]*authcommon.User, 0, 4),
			UserGroups: make([]*authcommon.UserGroup, 0, 4),
		}

		if err := rows.Scan(&ret.ID, &ret.Name, &ret.Owner, &ret.Source, &ret.Type, &ret.Comment,
			&flag, &metadata, &ctime, &mtime); err != nil {
			log.Error("[store][role] fetch one record role info", zap.Error(err))
			return nil, store.Error(err)
		}

		ret.CreateTime = time.Unix(ctime, 0)
		ret.ModifyTime = time.Unix(mtime, 0)
		ret.Valid = flag == 0
		_ = json.Unmarshal([]byte(metadata), &ret.Metadata)

		if err := s.fetchRolePrincipals(tx, ret); err != nil {
			return nil, store.Error(err)
		}

		// fetch link user or groups
		roles = append(roles, ret)
	}
	return roles, nil
}

func (s *roleStore) fetchRolePrincipals(tx *BaseTx, role *authcommon.Role) error {
	rows, err := tx.Query("SELECT role_id, principal_id, principal_role FROM auth_role_principal WHERE rold_id = ?", role.ID)
	if err != nil {
		log.Error("[store][role] fetch role principals", zap.String("name", role.Name), zap.Error(err))
		return store.Error(err)
	}
	defer func() {
		_ = rows.Close()
	}()

	for rows.Next() {
		var (
			roleID, principalID string
			principalRole       int
		)
		if err := rows.Scan(&roleID, &principalID, &principalRole); err != nil {
			log.Error("[store][role] fetch one record role principal", zap.String("name", role.Name), zap.Error(err))
			return store.Error(err)
		}

		if principalRole == int(authcommon.PrincipalUser) {
			role.Users = append(role.Users, &authcommon.User{
				ID: principalID,
			})
		} else {
			role.UserGroups = append(role.UserGroups, &authcommon.UserGroup{
				ID: principalID,
			})
		}
	}
	return nil
}
