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
	"context"
	"time"

	api "github.com/polarismesh/polaris-server/common/api/v1"
)

const (

	// 默认策略的名称前缀
	DefaultStrategyPrefix string = "__default__"
)

type ResourceOperation int16

const (
	Read   ResourceOperation = 10
	Create ResourceOperation = 20
	Modify ResourceOperation = 30
	Delete ResourceOperation = 40
)

type BzModule int16

const (
	CoreModule BzModule = iota
	DiscoverModule
	ConfigModule
)

// AcquireContext 每次鉴权请求上下文信息
type AcquireContext struct {
	
	// RequestContext 请求上下文
	RequestContext context.Context

	// Token 本次请求的访问凭据
	Token string

	// Module 来自那个业务层（服务注册与服务治理、配置模块）
	Module BzModule

	// Operation 本次操作涉及的动作
	Operation ResourceOperation

	// Resources 本次
	Resources map[api.ResourceType][]string

	// Attachment 携带信息，用于操作完权限检查和资源操作的后置处理逻辑，解决信息需要二次查询问题
	Attachment map[string]interface{}
}

// User
type User struct {
	ID          string
	Name        string
	Password    string
	Owner       string
	Source      string
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
	UserIDs []string
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
	Principal  string
	Action     string
	Comment    string
	Default    bool
	Owner      string
	Resources  []StrategyResource
	Valid      bool
	CreateTime time.Time
	ModifyTime time.Time
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
