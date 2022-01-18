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

package model

import (
	"fmt"
	"time"
)

const (
	OperatorRoleKey       string = "operator_role"
	OperatorPrincipalType string = "operator_principal"
	OperatorIDKey         string = "operator_id"
	OperatorOwnerKey      string = "operator_owner"
	OperatorLinkStrategy  string = "operator_link_strategy"
	LinkUsersKey          string = "link_users"
	LinkGroupsKey         string = "link_groups"

	TokenDetailInfoKey string = "TokenInfo"
	TokenForUser       string = "uid"
	TokenForUserGroup  string = "groupid"

	ResourceAttachmentKey string = "resource_attachment"
)

type PrincipalType int

const (
	PrincipalUser      PrincipalType = 1
	PrincipalUserGroup PrincipalType = 2
)

var (
	PrincipalNames map[PrincipalType]string = map[PrincipalType]string{
		PrincipalUser:      "user",
		PrincipalUserGroup: "group",
	}
)

const (

	// 默认策略的名称前缀
	DefaultStrategyPrefix string = "__default__"
)

func BuildDefaultStrategyName(id string, uType PrincipalType) string {
	return fmt.Sprintf("%s%s_%s", DefaultStrategyPrefix, PrincipalNames[uType], id)
}

// ResourceOperation 资源操作
type ResourceOperation int16

const (

	// Read 只读动作
	Read ResourceOperation = 10

	// Create 创建动作
	Create ResourceOperation = 20

	// Modify 修改动作
	Modify ResourceOperation = 30

	// Delete 删除动作
	Delete ResourceOperation = 40
)

// BzModule 模块标识
type BzModule int16

const (

	// CoreModule 核心模块
	CoreModule BzModule = iota

	// DiscoverModule 服务模块
	DiscoverModule

	// ConfigModule 配置模块
	ConfigModule
)

type UserRoleType int

const (
	UnknownUserRole    UserRoleType = -1
	AdminUserRole      UserRoleType = 0
	OwnerUserRole      UserRoleType = 20
	SubAccountUserRole UserRoleType = 50
)

var (
	UserRoleNames map[UserRoleType]string = map[UserRoleType]string{
		AdminUserRole:      "admin",
		OwnerUserRole:      "main",
		SubAccountUserRole: "sub",
	}
)

type ResourceEntry struct {
	ID    string
	Owner string
}

// User
type User struct {
	ID          string
	Name        string
	Password    string
	Owner       string
	Source      string
	Type        UserRoleType
	Token       string
	TokenEnable bool
	Valid       bool
	Comment     string
	CreateTime  time.Time
	ModifyTime  time.Time
}

type ExpandUser struct {
	*User
	GroupName  string
	GroupToken string
}

// UserGroupDetail
type UserGroupDetail struct {
	*UserGroup

	// UserIds TODO 后续改为 map 的形式，加速下查询
	UserIds map[string]struct{}
}

func (ugd *UserGroupDetail) ToUserIdSlice() []string {
	uids := make([]string, 0, len(ugd.UserIds))
	for uid := range ugd.UserIds {
		uids = append(uids, uid)
	}

	return uids
}

// UserGroup
type UserGroup struct {
	ID          string
	Name        string
	Owner       string
	Token       string
	TokenEnable bool
	Valid       bool
	Comment     string
	CreateTime  time.Time
	ModifyTime  time.Time
}

type ModifyUserGroup struct {
	ID            string
	Owner         string
	Token         string
	TokenEnable   bool
	Comment       string
	AddUserIds    []string
	RemoveUserIds []string
}

// UserGroupRelation
type UserGroupRelation struct {
	GroupID    string
	UserIds    []string
	CreateTime time.Time
	ModifyTime time.Time
}

// Strategy
type StrategyDetail struct {
	ID         string
	Name       string
	Action     string
	Comment    string
	Principals []Principal
	Default    bool
	Owner      string
	Resources  []StrategyResource
	Valid      bool
	Revision   string
	CreateTime time.Time
	ModifyTime time.Time
}

type ModifyStrategyDetail struct {
	ID               string
	Action           string
	Comment          string
	AddPrincipals    []Principal
	RemovePrincipals []Principal

	AddResources    []StrategyResource
	RemoveResources []StrategyResource
	ModifyTime      time.Time
}

// Strategy
type Strategy struct {
	ID         string
	Name       string
	Principal  string
	Action     string
	Comment    string
	Owner      string
	Default    bool
	Valid      bool
	CreateTime time.Time
	ModifyTime time.Time
}

// StrategyResource
type StrategyResource struct {
	StrategyID string
	ResType    int32
	ResID      string
	Valid      bool
	CreateTime time.Time
	ModifyTime time.Time
}

type Principal struct {
	StrategyID    string
	PrincipalID   string
	PrincipalRole PrincipalType
}
