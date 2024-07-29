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
	"encoding/json"
	"time"

	"github.com/polarismesh/polaris/common/model"
	authcommon "github.com/polarismesh/polaris/common/model/auth"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/store"
	bolt "go.etcd.io/bbolt"
	"go.uber.org/zap"
)

var _ store.RoleStore = (*roleStore)(nil)

const (
	// tblRole 角色表数据
	tblRole string = "role"

	roleFieldUsers      string = "Users"
	roleFieldUserGroups string = "UserGroups"
	roleFieldMetadata   string = "Metadata"
	roleFieldType       string = "Type"
	roleFieldSource     string = "Source"
)

type roleStore struct {
	handle BoltHandler
}

// AddRole Add a role
func (s *roleStore) AddRole(role *authcommon.Role) error {
	if role.ID == "" || role.Name == "" {
		log.Error("[Store][role] create role missing some params")
		return ErrBadParam
	}

	data := newRoleData(role)
	data.CreateTime = time.Now()
	data.ModifyTime = time.Now()
	data.Valid = true

	err := s.handle.Execute(true, func(tx *bolt.Tx) error {
		if err := s.cleanInvalidRole(tx, data.ID); err != nil {
			return err
		}
		return saveValue(tx, tblRole, data.ID, data)
	})
	if err != nil {
		log.Error("[Store][role] create role failed", zap.String("name", role.Name), zap.Error(err))
		return store.Error(err)
	}
	return nil
}

// cleanInvalidRole 删除无效的角色数据信息
func (s *roleStore) cleanInvalidRole(tx *bolt.Tx, id string) error {
	err := deleteValues(tx, tblRole, []string{id})
	if err != nil {
		log.Errorf("[Store][role] delete invalid role error, %v", err)
		return err
	}
	return nil
}

// UpdateRole Update a role
func (s *roleStore) UpdateRole(role *authcommon.Role) error {
	if role.ID == "" {
		log.Error("[Store][role] update role missing some params")
		return ErrBadParam
	}

	data := newRoleData(role)

	err := s.handle.Execute(true, func(tx *bolt.Tx) error {
		properties := map[string]interface{}{
			CommonFieldValid:       true,
			CommonFieldModifyTime:  time.Now(),
			CommonFieldDescription: data.Description,
			roleFieldType:          data.Type,
			roleFieldMetadata:      data.Metadata,
			roleFieldSource:        data.Source,
		}
		return updateValue(tx, tblRole, data.ID, properties)
	})
	if err != nil {
		log.Error("[Store][role] update role failed", zap.String("name", role.Name), zap.Error(err))
		return store.Error(err)
	}
	return nil
}

// DeleteRole Delete a role
func (s *roleStore) DeleteRole(role *authcommon.Role) error {
	if role.ID == "" {
		log.Error("[Store][role] delete role missing some params")
		return ErrBadParam
	}

	data := newRoleData(role)

	err := s.handle.Execute(true, func(tx *bolt.Tx) error {
		properties := map[string]interface{}{
			CommonFieldValid:      false,
			CommonFieldModifyTime: time.Now(),
		}
		return updateValue(tx, tblRole, data.ID, properties)
	})
	if err != nil {
		log.Error("[Store][role] delete role failed", zap.String("name", role.Name), zap.Error(err))
		return store.Error(err)
	}
	return nil
}

// CleanPrincipalRoles clean principal roles
func (s *roleStore) CleanPrincipalRoles(tx store.Tx, p *authcommon.Principal) error {
	dbTx := tx.GetDelegateTx().(*bolt.Tx)
	fields := []string{roleFieldUsers, roleFieldUserGroups, CommonFieldValid, CommonFieldID}
	values := map[string]interface{}{}

	updateDatas := map[string]map[string]interface{}{}

	err := loadValuesByFilter(dbTx, tblRole, fields, &roleData{},
		func(m map[string]interface{}) bool {
			valid, _ := m[CommonFieldValid].(bool)
			if !valid {
				return false
			}
			switch p.PrincipalType {
			case authcommon.PrincipalUser:
				users := make([]*authcommon.User, 0, 4)
				_ = json.Unmarshal([]byte(m[roleFieldUsers].(string)), &users)
				finalUsers := make([]*authcommon.User, 0, len(users))
				for i := range users {
					if users[i].ID == p.PrincipalID {
						continue
					}
					finalUsers = append(finalUsers, users[i])
				}
				updateDatas[m[CommonFieldID].(string)] = map[string]interface{}{
					roleFieldUsers:        utils.MustJson(users),
					CommonFieldModifyTime: time.Now(),
				}
			case authcommon.PrincipalGroup:
				groups := make([]*authcommon.UserGroup, 0, 4)
				_ = json.Unmarshal([]byte(m[roleFieldUserGroups].(string)), &groups)
				finalGroups := make([]*authcommon.UserGroup, 0, len(groups))
				for i := range groups {
					if groups[i].ID == p.PrincipalID {
						continue
					}
					finalGroups = append(finalGroups, groups[i])
				}
				updateDatas[m[CommonFieldID].(string)] = map[string]interface{}{
					roleFieldUserGroups:   utils.MustJson(groups),
					CommonFieldModifyTime: time.Now(),
				}
			}
			return false
		}, values)

	if err != nil {
		log.Error("[Store][role] get principal all role", zap.String("principal", p.String()), zap.Error(err))
		return store.Error(err)
	}

	for id := range updateDatas {
		if err := updateValue(dbTx, tblRole, id, updateDatas[id]); err != nil {
			log.Error("[store][role] clean principal all roles", zap.String("principal", p.String()), zap.Error(err))
			return store.Error(err)
		}
	}
	return nil
}

// GetMoreRoles get more role for cache update
func (s *roleStore) GetMoreRoles(firstUpdate bool, mtime time.Time) ([]*authcommon.Role, error) {
	fields := []string{CommonFieldModifyTime, CommonFieldValid}

	ret, err := s.handle.LoadValuesByFilter(tblRole, fields, &model.RoutingConfig{},
		func(m map[string]interface{}) bool {
			if firstUpdate {
				valid, _ := m[CommonFieldValid].(bool)
				if valid {
					return true
				}
			}
			saveMtime, _ := m[CommonFieldModifyTime].(time.Time)
			return !saveMtime.Before(mtime)
		})
	if err != nil {
		log.Errorf("[Store][role] get more role for cache, %v", err)
		return nil, store.Error(err)
	}

	roles := make([]*authcommon.Role, 0, len(ret))
	for i := range ret {
		roles = append(roles, newRole(ret[i].(*roleData)))
	}
	return roles, nil
}

type roleData struct {
	ID          string
	Name        string
	Owner       string
	Source      string
	Type        string
	Metadata    map[string]string
	Valid       bool
	Description string
	CreateTime  time.Time
	ModifyTime  time.Time
	Users       string
	UserGroups  string
}

func newRoleData(r *authcommon.Role) *roleData {
	return &roleData{
		ID:          r.ID,
		Name:        r.Name,
		Owner:       r.Owner,
		Source:      r.Source,
		Type:        r.Type,
		Metadata:    r.Metadata,
		Description: r.Comment,
		Users:       utils.MustJson(r.Users),
		UserGroups:  utils.MustJson(r.UserGroups),
	}
}

func newRole(r *roleData) *authcommon.Role {
	users := make([]*authcommon.User, 0, 32)
	groups := make([]*authcommon.UserGroup, 0, 32)

	_ = json.Unmarshal([]byte(r.Users), &users)
	_ = json.Unmarshal([]byte(r.UserGroups), &groups)

	return &authcommon.Role{
		ID:         r.ID,
		Name:       r.Name,
		Owner:      r.Owner,
		Source:     r.Source,
		Type:       r.Type,
		Metadata:   r.Metadata,
		Comment:    r.Description,
		Users:      users,
		UserGroups: groups,
		CreateTime: r.CreateTime,
		ModifyTime: r.ModifyTime,
	}
}
